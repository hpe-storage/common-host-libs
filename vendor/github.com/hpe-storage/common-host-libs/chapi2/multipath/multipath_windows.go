// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package multipath

import (
	"fmt"

	"github.com/hpe-storage/common-host-libs/chapi2/cerrors"
	"github.com/hpe-storage/common-host-libs/chapi2/iscsi"
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/windows/ioctl"
	"github.com/hpe-storage/common-host-libs/windows/iscsidsc"
	"github.com/hpe-storage/common-host-libs/windows/powershell"
	"github.com/hpe-storage/common-host-libs/windows/wmi"
)

// getDevices enumerates all the Nimble volumes while only providing basic details (e.g. serial number).
// If a "serialNumber" is passed in, only that specific serial number is enumerated.
func (plugin *MultipathPlugin) getDevices(serialNumber string) ([]*model.Device, error) {
	log.Tracef(">>>>> getDevices, serialNumber=%v", serialNumber)
	defer log.Trace("<<<<< getDevices")

	// Enumerate all Nimble volumes
	nimbleDisks, err := wmi.GetNimbleMSFTDisk(serialNumber)
	if err != nil {
		return nil, err
	}

	// Create an array of model.Device objects where only the serial number is populated
	var devices []*model.Device
	for _, nimbleDisk := range nimbleDisks {
		device := &model.Device{
			SerialNumber: nimbleDisk.SerialNumber,
			Private:      &model.DevicePrivate{WindowsDisk: nimbleDisk},
		}
		log.Tracef("SerialNumber=%v, Number=%v, IsOffline=%v, IsReadOnly=%v", nimbleDisk.SerialNumber, nimbleDisk.Number, nimbleDisk.IsOffline, nimbleDisk.IsReadOnly)
		devices = append(devices, device)
	}

	// Make sure duplicate serial numbers are not detected (e.g. misconfigured MPIO)
	if err = plugin.checkDuplicateSerialNumbers(devices); err != nil {
		return nil, err
	}

	return devices, nil
}

// getAllDeviceDetails enumerates all the Nimble volumes while providing full details about the
// device.  If a "serialNumber" is passed in, only that specific serial number is enumerated.
func (plugin *MultipathPlugin) getAllDeviceDetails(serialNumber string) ([]*model.Device, error) {
	log.Trace(">>>>> getAllDeviceDetails")
	defer log.Trace("<<<<< getAllDeviceDetails")

	// Enumerate all Nimble volumes
	nimbleDisks, err := wmi.GetNimbleMSFTDisk(serialNumber)
	if err != nil {
		return nil, err
	}

	// If an iSCSI device was detected, enumerate the iSCSI target mappings
	var targetMappings []*iscsidsc.ISCSI_TARGET_MAPPING
	for _, nimbleDisk := range nimbleDisks {
		if wmi.STORAGE_BUS_TYPE(nimbleDisk.BusType) == wmi.BusTypeiScsi {
			targetMappings, _ = iscsidsc.ReportActiveIScsiTargetMappings()
			break
		}
	}

	// On a Group Scoped Target (GST), a single target could have multiple LUNs.  To speed the
	// enumerate of a device's target ports, we'll cache the iqn target ports so that they can
	// be used on other GST LUNs (if present).
	cachedTargetPortals := make(map[string][]*model.TargetPortal)

	// Loop through and create a fully populated array of model.Device objects
	var devices []*model.Device
	for deviceIndex, nimbleDisk := range nimbleDisks {

		// Start by populating the protocol independent properties
		device := &model.Device{
			SerialNumber:    nimbleDisk.SerialNumber,
			Pathname:        fmt.Sprintf("Disk%v", nimbleDisk.Number),
			AltFullPathName: nimbleDisk.Path,
			Size:            nimbleDisk.Size,
			Private:         &model.DevicePrivate{WindowsDisk: nimbleDisk},
		}

		// Is this an iSCSI volume?  If so, we want to populate the device iSCSI details.
		if wmi.STORAGE_BUS_TYPE(nimbleDisk.BusType) == wmi.BusTypeiScsi {

			// If we were not provided an iSCSI plugin object, log an error and skip volume
			if plugin.iscsiPlugin == nil {
				log.Errorf("iscsiPlugin object not provided, skipping iSCSI device, Number=%v, Path=%v", nimbleDisk.Number, nimbleDisk.Path)
				continue
			}

			// Enumerate the IscsiTarget for our device
			device.IscsiTarget, _ = plugin.getIscsiTarget(nimbleDisk.Path, targetMappings, cachedTargetPortals)
		}

		// Log the device details
		log.Tracef("Device %v, SerialNumber=%v, Pathname=%v, BusType=%v, Size=%v, IsOffline=%v, IsReadOnly=%v",
			deviceIndex, device.SerialNumber, device.Pathname, nimbleDisk.BusType, device.Size, nimbleDisk.IsOffline, nimbleDisk.IsReadOnly)

		// If it's an iSCSI target, log the iSCSI details
		if device.IscsiTarget != nil {
			log.Tracef("    IQN   - %v", device.IscsiTarget.Name)
			log.Tracef("    Scope - %v", device.IscsiTarget.TargetScope)
			for _, targetPortal := range device.IscsiTarget.TargetPortals {
				log.Tracef("    Port  - %v:%v", targetPortal.Address, targetPortal.Port)
			}
		}

		// Append the enumerated device to our return device array
		devices = append(devices, device)
	}

	// Make sure duplicate serial numbers are not detected (e.g. misconfigured MPIO)
	if err = plugin.checkDuplicateSerialNumbers(devices); err != nil {
		return nil, err
	}

	return devices, nil
}

// checkDuplicateSerialNumbers scans the array of CHAPI devices for any duplicate serial numbers.
// If any are found, an error object is returned (e.g. misconfigured MPIO) else nil is returned.
func (plugin *MultipathPlugin) checkDuplicateSerialNumbers(devices []*model.Device) error {
	m := make(map[string]bool)
	for _, device := range devices {
		if m[device.SerialNumber] == true {
			err := cerrors.NewChapiErrorf(cerrors.Internal, errorMessageMisconfiguredMultipathIO, device.SerialNumber)
			log.Error(err)
			return err
		}
		m[device.SerialNumber] = true
	}
	return nil
}

// getPartitionInfo enumerates the partitions on the given volume
func (plugin *MultipathPlugin) getPartitionInfo(serialNumber string) ([]*model.DevicePartition, error) {
	log.Tracef(">>>>> getPartitionInfo, serialNumber=%v", serialNumber)
	defer log.Trace("<<<<< getPartitionInfo")

	// Enumerate the one serial number
	device, err := plugin.getDevices(serialNumber)
	if err != nil {
		return nil, err
	}

	// Fail request if volume not found
	if len(device) != 1 {
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageDeviceNotFound)
	}

	// Enumerate the volume's partitions
	var win32Partitions []*wmi.Win32_DiskPartition
	win32Partitions, err = wmi.GetWin32DiskPartitionForDiskIndex(int(device[0].Private.WindowsDisk.Number))
	if err != nil {
		return nil, err
	}

	// Convert []*wmi.Win32_DiskPartition into []*model.DevicePartition
	var partitions []*model.DevicePartition
	for _, win32Partition := range win32Partitions {
		partition := &model.DevicePartition{
			Name:          win32Partition.Name,
			PartitionType: win32Partition.Type,
			Size:          win32Partition.Size,
		}
		log.Tracef("Name=%v, PartitionType=%v, Size=%v", partition.Name, partition.PartitionType, partition.Size)
		partitions = append(partitions, partition)
	}

	// Return the enumerated partitions (or empty list if no partitions present)
	return partitions, nil
}

// offlineDevice is called to offline the given device
func (plugin *MultipathPlugin) offlineDevice(device model.Device) error {
	log.Tracef(">>>>> offlineDevice, Path=%v", device.Private.WindowsDisk.Path)
	defer log.Trace("<<<<< offlineDevice")

	// Use PowerShell to offline the disk
	_, _, err := powershell.SetDiskOffline(device.Private.WindowsDisk.Path, true)
	return err
}

// createFileSystem is called to create a file system on the given device
func (plugin *MultipathPlugin) createFileSystem(device model.Device, filesystem string) error {
	log.Tracef(">>>>> createFileSystem, Path=%v, filesystem=%v", device.Private.WindowsDisk.Path, filesystem)
	defer log.Trace("<<<<< createFileSystem")

	// Make sure disk is online and writable before attempting the format
	if err := plugin.MakeDiskOnlineAndWritable(device.Private.WindowsDisk.Path, true, true); err != nil {
		return err
	}

	// Determine partition style to use
	partitionStyle := powershell.PartitionStyleGPT
	if device.Size < powershell.MinimumGPTSize {
		log.Tracef("Disk not large enough for GPT (%v bytes), using MBR", device.Size)
		partitionStyle = powershell.PartitionStyleMBR
	}

	// Initialize the disk
	if _, _, err := powershell.InitializeDisk(device.Private.WindowsDisk.Path, partitionStyle); err != nil {
		return err
	}

	// Use PowerShell to format the disk
	_, _, err := powershell.PartitionAndFormatVolume(device.Private.WindowsDisk.Path, filesystem)
	return err
}

// getIscsiTarget enumerates the IscsiTarget object for the "devicePathID" device.  The caller needs
// to pass in the current target mappings (targetMappings object) and pass in cache objects where
// this routine can cache the last enumerated target ports.  This routine first checks the cache to
// see if the target values are known.  If not, then the target is queried to retrieve this
// information and update the cache.
func (plugin *MultipathPlugin) getIscsiTarget(devicePathID string, targetMappings []*iscsidsc.ISCSI_TARGET_MAPPING, cachedTargetPortals map[string][]*model.TargetPortal) (*model.IscsiTarget, error) {
	log.Tracef(">>>>> getIscsiTarget, devicePathID=%v", devicePathID)
	defer log.Trace("<<<<< getIscsiTarget")

	// Start by enumerating the device SCSI address; abort if unable to enumerate
	scsiAddress, err := ioctl.GetScsiAddress(devicePathID)
	if err != nil {
		return nil, err
	}

	// Define the IscsiTarget object we'll return
	var iscsiTarget *model.IscsiTarget

	// Allocate an iSCSI plugin object
	iscsiPlugin := iscsi.NewIscsiPlugin()

	// Loop through our enumerated iSCSI target mappings looking for a match
	for _, targetMapping := range targetMappings {

		// If either the target number or bus/path number do not match, then keep
		// looping until we find a match
		if (targetMapping.OSTargetNumber != uint32(scsiAddress.TargetId)) || (targetMapping.OSBusNumber != uint32(scsiAddress.PathId)) {
			continue
		}

		// We found the iSCSI target for the device.  Populate the IscsiTarget object
		// with the target iqn.
		iscsiTarget = &model.IscsiTarget{Name: targetMapping.TargetName}

		// See if we have a cached target scope for the iqn.  If we do not, enumerate
		// the scope from the device.
		iscsiTarget.TargetScope = getTargetTypeCache().GetTargetType(targetMapping.TargetName)
		if iscsiTarget.TargetScope == "" {
			iscsiTarget.TargetScope, _ = plugin.iscsiPlugin.GetTargetScope(targetMapping.TargetName)
			if iscsiTarget.TargetScope != "" {
				getTargetTypeCache().SetTargetType(targetMapping.TargetName, iscsiTarget.TargetScope)
			}
		}

		// See if we have cached target portals for the iqn.  If we do not, enumerate
		// the target portals from the device.
		iscsiTarget.TargetPortals = cachedTargetPortals[targetMapping.TargetName]
		if iscsiTarget.TargetPortals == nil {
			iscsiTarget.TargetPortals, _ = iscsiPlugin.GetTargetPortals(targetMapping.TargetName, true)
			if iscsiTarget.TargetPortals != nil {
				cachedTargetPortals[targetMapping.TargetName] = iscsiTarget.TargetPortals
			}
		}

		// We found the iSCSI mapping so we can break out of our target mapping loop
		break
	}

	// Return an error if we were unable to locate the iSCSI target
	if iscsiTarget == nil {
		err = cerrors.NewChapiError(cerrors.NotFound, errorMessageUnableLocateIscsiTarget)
		log.Error(err)
		return nil, err
	}

	return iscsiTarget, nil
}

// MakeDiskOnlineAndWritable is a helper routine that will make a disk online and/or writable
func (plugin *MultipathPlugin) MakeDiskOnlineAndWritable(path string, makeOnline bool, makeWritable bool) error {
	log.Tracef(">>>>> MakeDiskOnlineAndWritable, path=%v, makeOnline=%v, makeWritable=%v", path, makeOnline, makeWritable)
	defer log.Trace("<<<<< MakeDiskOnlineAndWritable")

	// The make writable / online ordering here is important.  We have found that if you attach a
	// cloned volume, and if it has a signature collision with its already mounted parent volume,
	// making the cloned volume online will fail and render the volume inaccessible.  You need to
	// clear the volume's read only flag first before you online the volume.  This allows Windows
	// to resignature the cloned volume to avoid the disk signature collision.

	// Start by ensuring the disk is not marked as read only
	if makeWritable {
		if _, _, err := powershell.SetDiskReadOnly(path, false); err != nil {
			return err
		}
	}

	// Make the disk online
	if makeOnline {
		if _, _, err := powershell.SetDiskOffline(path, false); err != nil {
			return err
		}
	}

	// If we had to online the disk, or make it writable, we'll need to update the cached information
	// about the disk.  We need to do this otherwise a re-enumeration of the partition might not
	// pickup the current mount point(s).
	if makeOnline || makeWritable {
		if _, _, err := powershell.UpdateDisk(path); err != nil {
			return err
		}
	}

	return nil
}
