// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package driver

import (
	"fmt"

	"github.com/hpe-storage/common-host-libs/chapi2/cerrors"
	"github.com/hpe-storage/common-host-libs/chapi2/fc"
	"github.com/hpe-storage/common-host-libs/chapi2/host"
	"github.com/hpe-storage/common-host-libs/chapi2/iscsi"
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	"github.com/hpe-storage/common-host-libs/chapi2/mount"
	"github.com/hpe-storage/common-host-libs/chapi2/multipath"
	"github.com/hpe-storage/common-host-libs/chapi2/virtualdevice"
	log "github.com/hpe-storage/common-host-libs/logger"
)

const (
	// Shared error messages
	errorMessageEmptyIqnFound         = "empty iqn found"
	errorMessageMultipleDevices       = "multiple (%v) devices enumerated"
	errorMessageMultipleDeviceObjects = "multiple device access objects provided"
	errorMessageNoDeviceObject        = "device access object not provided"
	errorMessageNoDevicesOnHost       = "no devices found on host"
	errorMessageNoInitiatorsFound     = "neither of iscsi or fc initiators are found on host"
	errorMessageNoMountPointsFound    = "no mount points found"
	errorMessageNoNetworkInterfaces   = "no network interfaces found on host"
	errorMessageNoPartitionsOnVolume  = "no partitions found on volume"
	errorMessageNotYetImplemented     = "not yet implemented"
	errorMessageVolumeMounted         = "volume mounted"
)

// Driver provides a common interface for host related operations
type Driver interface {
	///////////////////////////////////////////////////////////////////////////////////////////
	// Host Methods
	///////////////////////////////////////////////////////////////////////////////////////////

	GetHostInfo() (*model.Host, error)              // GET /api/v1/hosts
	GetHostInitiators() ([]*model.Initiator, error) // GET /api/v1/initiators
	GetHostNetworks() ([]*model.Network, error)     // GET /api/v1/networks

	///////////////////////////////////////////////////////////////////////////////////////////
	// Device Methods
	///////////////////////////////////////////////////////////////////////////////////////////

	// GET /api/v1/devices or
	// GET /api/v1/devices?serial=serial
	GetDevices(serialNumber string) ([]*model.Device, error)

	// GET /api/v1/devices/details or
	// GET /api/v1/devices/details?serial=serial
	GetAllDeviceDetails(serialNumber string) ([]*model.Device, error)

	// GET /api/v1/devices/{serialnumber}/partitions
	GetPartitionInfo(serialNumber string) ([]*model.DevicePartition, error)

	// POST /api/v1/devices
	CreateDevice(publishInfo model.PublishInfo) (*model.Device, error)

	// DELETE /api/v1/devices/{serialnumber}
	DeleteDevice(serialNumber string) error

	// PUT /api/v1/devices/{serialnumber}/actions/offline
	OfflineDevice(serialNumber string) error

	// PUT /api/v1/devices/{serialnumber}/filesystem/{filesystem}
	CreateFileSystem(serialNumber string, filesystem string) error

	///////////////////////////////////////////////////////////////////////////////////////////
	// Mount Methods
	///////////////////////////////////////////////////////////////////////////////////////////

	// GET /api/v1/mounts or
	// GET /api/v1/mounts?serial=serial
	GetMounts(serialNumber string) ([]*model.Mount, error)

	// GET /api/v1/mounts/details  or filter by serial using
	// GET /api/v1/mounts/details?serial=serial or filter by serial and specific mount using
	// GET /api/v1/mounts/details?serial=serial,mountId=mount
	GetAllMountDetails(serialNumber, mountPointID string) ([]*model.Mount, error)

	// POST /api/v1/mounts
	CreateMount(serialNumber string, mountPoint string, fsOptions *model.FileSystemOptions) (*model.Mount, error)

	// DELETE /api/v1/mounts/{mountId}
	DeleteMount(serialNumber, mountPointID string) error

	// TODO: check with George/Suneeth on this
	// POST /api/v1/mounts/bind
	CreateBindMount(sourceMount string, targetMount string, bindType string) (*model.Mount, error)
}

// ChapiServer ... Implements the "Driver" interfaces
type ChapiServer struct {
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Host methods
///////////////////////////////////////////////////////////////////////////////////////////////////

// GetHostInfo returns host name, domain, and network interfaces
func (driver *ChapiServer) GetHostInfo() (*model.Host, error) {
	log.Trace(">>>>> GetHostInfo called")
	defer log.Trace("<<<<< GetHostInfo")
	hostPlugin := host.NewHostPlugin()

	log.Info("Get Host Information")

	id, err := hostPlugin.GetUuid()
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}
	log.Infof("Host UUID - %v", id)

	hostName, err := hostPlugin.GetHostName()
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}
	log.Infof("Host Name - %v", hostName)

	domainName, err := hostPlugin.GetDomainName()
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}
	log.Infof("Domain Name - %v", domainName)

	return &model.Host{UUID: id, Name: hostName, Domain: domainName}, nil
}

// GetHostNetworks reports the networks on this host
func (driver *ChapiServer) GetHostNetworks() ([]*model.Network, error) {
	log.Trace(">>>>> GetHostNetworks called")
	defer log.Trace("<<<<< GetHostNetworks")
	hostPlugin := host.NewHostPlugin()

	log.Info("Get Host Networks")

	networks, err := hostPlugin.GetNetworks()
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}
	if len(networks) == 0 {
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageNoNetworkInterfaces)
	}
	driver.logNetworks(networks)
	return networks, nil
}

// GetHostInitiators reports the initiators on this host
func (driver *ChapiServer) GetHostInitiators() ([]*model.Initiator, error) {
	log.Trace(">>>>> GetHostInitiators called")
	defer log.Trace("<<<<< GetHostInitiators")
	//var inits Initiators
	var inits []*model.Initiator

	log.Info("Get Host Initiators")

	// fetch iscsi initiator details
	iscsiPlugin := iscsi.NewIscsiPlugin()

	iscsiInits, err := iscsiPlugin.GetIscsiInitiators()
	if err != nil {
		log.Trace("Error getting iscsiInitiator: ", err)
	}

	// fetch fc initiator details
	fcPlugin := fc.NewFcPlugin()

	fcInits, err := fcPlugin.GetFcInitiators()
	if err != nil {
		log.Trace("Error getting FcInitiator: ", err)
	}
	if fcInits != nil {
		inits = append(inits, fcInits)
	}
	if iscsiInits != nil {
		inits = append(inits, iscsiInits)
	}

	if fcInits == nil && iscsiInits == nil {
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageNoInitiatorsFound)
	}

	// Log enumerated iSCSI and FC initiators
	for _, initiator := range inits {
		for _, init := range initiator.Init {
			log.Infof("AccessProtocol=%v, Initiator=%v", initiator.AccessProtocol, init)
		}
	}

	return inits, nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Device methods
///////////////////////////////////////////////////////////////////////////////////////////////////

// GetDevices enumerates all the Nimble volumes with basic details.
// If serialNumber is non-empty then only specified device is returned
func (driver *ChapiServer) GetDevices(serialNumber string) ([]*model.Device, error) {
	log.Tracef(">>>>> GetDevices called, serialNumber=%v", serialNumber)
	defer log.Trace("<<<<< GetDevices")
	multipathPlugin := multipath.NewMultipathPlugin()

	log.Infof("Get Devices, serialNumber=%v", serialNumber)

	// Enumerate all the Nimble volumes on this host (basic details only)
	devices, err := multipathPlugin.GetDevices(serialNumber)
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}

	// Fail request if no Nimble devices found on this host
	if len(devices) == 0 {
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageNoDevicesOnHost)
	}

	// Log enumerated device serial numbers
	for _, device := range devices {
		log.Infof("Device SerialNumber=%v", device.SerialNumber)
	}

	return devices, nil
}

// GetAllDeviceDetails enumerates all the Nimble volumes with detailed information.
// If serialNumber is non-empty then only specified device is returned
func (driver *ChapiServer) GetAllDeviceDetails(serialNumber string) ([]*model.Device, error) {
	log.Tracef(">>>>> GetAllDeviceDetails called, serialNumber=%v", serialNumber)
	defer log.Trace("<<<<< GetAllDeviceDetails")
	multipathPlugin := multipath.NewMultipathPlugin()

	log.Infof("Get All Device Details, serialNumber=%v", serialNumber)

	// Enumerate all the Nimble volumes on this host (full details)
	devices, err := multipathPlugin.GetAllDeviceDetails(serialNumber)
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}

	// Fail request if no Nimble devices found on this host
	if len(devices) == 0 {
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageNoDevicesOnHost)
	}

	// Log enumerated device details
	driver.logDeviceArrayDetails(devices)

	return devices, nil
}

// GetPartitionInfo reports the partitions on the provided device
func (driver *ChapiServer) GetPartitionInfo(serialNumber string) ([]*model.DevicePartition, error) {
	log.Tracef(">>>>> GetPartitionInfo called, serialNumber=%v", serialNumber)
	defer log.Trace("<<<<< GetPartitionInfo")
	multipathPlugin := multipath.NewMultipathPlugin()

	log.Infof("Get Partition Information, serialNumber=%v", serialNumber)

	// Enumerate all the Nimble volume's partition
	partitions, err := multipathPlugin.GetPartitionInfo(serialNumber)
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}

	// Fail request if no partitions found on this host
	if len(partitions) == 0 {
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageNoPartitionsOnVolume)
	}

	// Log enumerated partition details
	for _, partition := range partitions {
		log.Infof("Partition Name=%v, PartitionType=%v, Size=%v", partition.Name, partition.PartitionType, partition.Size)
	}

	return partitions, nil
}

// CreateDevice will attach device on this host based on the details provided
func (driver *ChapiServer) CreateDevice(publishInfo model.PublishInfo) (*model.Device, error) {
	log.Tracef(">>>>> CreateDevice called, publishInfo=%v", publishInfo)
	defer log.Trace("<<<<< CreateDevice")

	log.Info("Create Device")

	// Invalid request if no device access object provided
	if (publishInfo.BlockDev == nil) && (publishInfo.VirtualDev == nil) {
		err := cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageNoDeviceObject)
		log.Error(err)
		return nil, err
	}

	// Invalid request if multiple device access objects provided
	if (publishInfo.BlockDev != nil) && (publishInfo.VirtualDev != nil) {
		err := cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageMultipleDeviceObjects)
		log.Error(err)
		return nil, err
	}

	// Attach the virtual device
	if publishInfo.VirtualDev != nil {
		virtualDevPlugin := virtualdevice.NewVirtualDevPlugin()
		_ = virtualDevPlugin // Avoid "declared and not used" errors until feature is implemented
		return nil, cerrors.NewChapiError(cerrors.Unimplemented, errorMessageNotYetImplemented)
	}

	// Attach the block device
	multipathPlugin := multipath.NewMultipathPlugin()
	device, err := multipathPlugin.AttachDevice(publishInfo.SerialNumber, *publishInfo.BlockDev)
	if err != nil {
		return nil, err
	}

	driver.logDeviceDetails(device)
	return device, nil
}

// DeleteDevice will delete the given device from the host
func (driver *ChapiServer) DeleteDevice(serialNumber string) error {
	log.Tracef(">>>>> DeleteDevice called, serialNumber=%v", serialNumber)
	defer log.Trace("<<<<< DeleteDevice")
	multipathPlugin := multipath.NewMultipathPlugin()

	log.Infof("Delete Device, serialNumber=%v", serialNumber)

	// TODO - handle VirtualDev vs BlockDev

	// Find the device serial number details.  If the device is not present on this host (i.e.
	// cerrors.NotFound), there is no device to detach so we return no error.
	devices, err := multipathPlugin.GetAllDeviceDetails(serialNumber)
	if len(devices) == 0 {
		log.Infof("Serial number %v not present, returning success", serialNumber)
		return nil
	} else if err != nil {
		return err
	}

	// Fail request if device is mounted.  We only allow deleting the device if it isn't already
	// mounted.  Caller should dismount the device before attempting to delete the device.
	if mounts, _ := driver.GetMounts(serialNumber); len(mounts) > 0 {
		err = cerrors.NewChapiError(cerrors.PermissionDenied, errorMessageVolumeMounted)
		log.Error(err)
		return err
	}

	// Detach the block device
	driver.logDeviceDetails(devices[0])
	if err := multipathPlugin.DetachDevice(*devices[0]); err != nil {
		return err
	}

	// Success!!!
	log.Infof("Device Deleted, SerialNumber=%v", serialNumber)
	return nil
}

// OfflineDevice will offline the given device from the host
func (driver *ChapiServer) OfflineDevice(serialNumber string) error {
	log.Tracef(">>>>> OfflineDevice called, serialNumber=%v", serialNumber)
	defer log.Trace("<<<<< OfflineDevice")
	multipathPlugin := multipath.NewMultipathPlugin()

	log.Infof("Offline Device, serialNumber=%v", serialNumber)

	// Enumerate basic details for the serial number
	device, err := driver.getSingleDeviceSummary(serialNumber)
	if err != nil {
		return err
	}

	// Offline the device
	if err := multipathPlugin.OfflineDevice(*device); err != nil {
		return err
	}

	// Success!!!
	log.Infof("Device Offlined, SerialNumber=%v", serialNumber)
	return nil
}

// CreateFileSystem writes the given file system to the device with the given serial number
func (driver *ChapiServer) CreateFileSystem(serialNumber string, filesystem string) error {
	log.Tracef(">>>>> CreateFileSystem called, serialNumber=%v, filesystem=%v", serialNumber, filesystem)
	defer log.Trace("<<<<< CreateFileSystem")
	multipathPlugin := multipath.NewMultipathPlugin()

	log.Infof("Create File System, serialNumber=%v, filesystem=%v", serialNumber, filesystem)

	// Enumerate basic details for the serial number
	device, err := driver.getSingleDeviceSummary(serialNumber)
	if err != nil {
		return err
	}

	// Format the device
	driver.logDeviceDetails(device)
	return multipathPlugin.CreateFileSystem(*device, filesystem)
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Mount point methods
///////////////////////////////////////////////////////////////////////////////////////////////////

// GetMounts reports all mounts on this host for the specified Nimble volume
func (driver *ChapiServer) GetMounts(serialNumber string) ([]*model.Mount, error) {
	log.Tracef(">>>>> GetMounts called, serialNumber=%v", serialNumber)
	defer log.Trace("<<<<< GetMounts")

	log.Infof("Get Mounts, serialNumber=%v", serialNumber)

	// Route request to the mount package to get the mounts
	mountPlugin := mount.NewMounter()
	mounts, err := mountPlugin.GetMounts(serialNumber)
	if err != nil {
		return nil, err
	}

	// Fail request if no mount points detected
	if len(mounts) == 0 {
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageNoMountPointsFound)
	}

	driver.logMountArray(mounts)
	return mounts, nil
}

// GetAllMountDetails enumerates the specified mount point ID
func (driver *ChapiServer) GetAllMountDetails(serialNumber string, mountPointID string) ([]*model.Mount, error) {
	log.Tracef(">>>>> GetAllMountDetails called, serialNumber=%v, mountPointID=%v", serialNumber, mountPointID)
	defer log.Trace("<<<<< GetAllMountDetails")

	log.Infof("Get All Mount Details, serialNumber=%v, mountPointID=%v", serialNumber, mountPointID)

	// Route request to the mount package to get the mounts
	mountPlugin := mount.NewMounter()
	mounts, err := mountPlugin.GetAllMountDetails(serialNumber, mountPointID)
	if err != nil {
		return nil, err
	}

	// Fail request if no mount points detected
	if len(mounts) == 0 {
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageNoMountPointsFound)
	}

	driver.logMountArray(mounts)
	return mounts, nil
}

// CreateMount mounts the given device to the given mount point
func (driver *ChapiServer) CreateMount(serialNumber string, mountPoint string, fsOptions *model.FileSystemOptions) (*model.Mount, error) {
	log.Tracef(">>>>> CreateMount called, serialNumber=%v, mountPoint=%v, fsOptions=%v", serialNumber, mountPoint, fsOptions)
	defer log.Trace("<<<<< CreateMount")

	log.Infof("Create Mount, serialNumber=%v, mountPoint=%v", serialNumber, mountPoint)

	// Route request to the mount package to create the mount point
	mountPlugin := mount.NewMounter()
	mount, err := mountPlugin.CreateMount(serialNumber, mountPoint, fsOptions)
	if err != nil {
		return nil, err
	}

	driver.logMount(mount)
	return mount, nil
}

// DeleteMount unmounts the given mount point, serialNumber can be optional in the body
func (driver *ChapiServer) DeleteMount(serialNumber string, mountPointId string) error {
	log.Tracef(">>>>> DeleteMount called, serialNumber=%v, mountPointID=%v", serialNumber, mountPointId)
	defer log.Trace("<<<<< DeleteMount")

	log.Infof("Delete Mount, serialNumber=%v, mountPointId=%v", serialNumber, mountPointId)

	// Route request to the mount package to delete the mount point
	mountPlugin := mount.NewMounter()
	if err := mountPlugin.DeleteMount(serialNumber, mountPointId); err != nil {
		return err
	}

	// Success!!!
	log.Infof("Mount Point ID %v successfully deleted", mountPointId)
	return nil
}

// CreateBindMount creates the given bind mount
func (driver *ChapiServer) CreateBindMount(sourceMount string, targetMount string, bindType string) (*model.Mount, error) {
	log.Tracef(">>>>> CreateBindMount called, sourceMount=%s, targetMount=%s bindType=%s", sourceMount, targetMount, bindType)
	defer log.Trace("<<<<< CreateBindMount")

	log.Infof("Create Bind Mount, sourceMount=%v, targetMount=%v, bindType=%v", sourceMount, targetMount, bindType)

	return nil, cerrors.NewChapiError(cerrors.Unimplemented, errorMessageNotYetImplemented)
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Internal helper methods
///////////////////////////////////////////////////////////////////////////////////////////////////

// getSingleDeviceSummary uses the driver.GetDevices() endpoint to query basic summary details
// about the given serial number.  If multiple volumes share that serial number (e.g. multipath
// not configured properly), this routine will fail the request.
func (driver *ChapiServer) getSingleDeviceSummary(serialNumber string) (*model.Device, error) {
	log.Tracef(">>>>> getSingleDeviceSummary called, serialNumber=%v", serialNumber)
	defer log.Trace("<<<<< getSingleDeviceSummary")
	multipathPlugin := multipath.NewMultipathPlugin()

	// Enumerate the device details for the provided serial number
	devices, err := multipathPlugin.GetDevices(serialNumber)
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}

	// Fail request if no Nimble devices found on this host
	if len(devices) == 0 {
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageNoDevicesOnHost)
	}

	// If we did not enumerate a single volume, with the provided serial number, the host is likely
	// misconfigured (e.g. multipath misconfigured)
	if len(devices) != 1 {
		err = cerrors.NewChapiErrorf(cerrors.Internal, errorMessageMultipleDevices, len(devices))
		log.Errorf(err.Error())
		return nil, cerrors.NewChapiError(err)
	}

	// Return the single enumerated volume
	return devices[0], nil
}

// logNetworks records the host NIC details, one line for NIC, to the information log
func (driver *ChapiServer) logNetworks(networks []*model.Network) {
	for _, network := range networks {
		log.Infof("Network=%v, AddressV4=%v, MaskV4=%v, Up=%v", network.Name, network.AddressV4, network.MaskV4, network.Up)
	}
}

// logDeviceArrayDetails records the device array details to the information log
func (driver *ChapiServer) logDeviceArrayDetails(devices []*model.Device) {
	for _, device := range devices {
		driver.logDeviceDetails(device)
	}
}

// logDeviceDetails records the device details to the information log
func (driver *ChapiServer) logDeviceDetails(device *model.Device) {
	if device == nil {
		log.Error("logDeviceDetails called with nil device")
		return
	}
	msg := fmt.Sprintf("Device SerialNumber=%v, Pathname=%v, Size=%v, State=%v", device.SerialNumber, device.Pathname, device.Size, device.State)
	if device.IscsiTarget != nil {
		msg += fmt.Sprintf(", IscsiTargetName=%v, TargetScope=%v", device.IscsiTarget.Name, device.IscsiTarget.TargetScope)
	}
	log.Infoln(msg)
}

// logMountArray records the mount details to the information log
func (driver *ChapiServer) logMountArray(mounts []*model.Mount) {
	for _, mount := range mounts {
		driver.logMount(mount)
	}
}

// logMount records the single mount details to the information log
func (driver *ChapiServer) logMount(mount *model.Mount) {
	if mount == nil {
		log.Error("logMount called with nil mount")
		return
	}
	msg := fmt.Sprintf("Mount ID=%v", mount.ID)
	if (mount.MountPoint != "") || (mount.SerialNumber != "") {
		msg += fmt.Sprintf(", MountPoint=%v, SerialNumber=%v", mount.MountPoint, mount.SerialNumber)
	}
	log.Infoln(msg)
}
