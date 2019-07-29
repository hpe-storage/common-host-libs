// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package driver

import (
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
	// GET /api/v1/devices?serialNumber=serial
	GetDevices(serialNumber string) ([]*model.Device, error)

	// GET /api/v1/devices/detail or
	// GET /api/v1/devices/detail?serialNumber=serial
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
	CreateFileSystem(serialNumber string, fsType string) error

	///////////////////////////////////////////////////////////////////////////////////////////
	// Filesystem Methods
	///////////////////////////////////////////////////////////////////////////////////////////

	// GET /api/v1/mounts or
	// GET /api/v1/mounts?serialNumber=serial
	GetMounts(serialNumber string) ([]*model.Mount, error)

	// GET /api/v1/mounts/details  or filter by serial using
	// GET /api/v1/mounts/details?serialNumber=serial or filter by serial and specific mount using
	// GET /api/v1/mounts/details?serialNumber=serial,mountId=mount
	GetAllMountDetails(serialNumber, mountPointId string) ([]*model.Mount, error)

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
	log.Info(">>>>> GetHostInfo called")
	defer log.Info("<<<<< GetHostInfo")
	hostPlugin := host.NewHostPlugin()

	id, err := hostPlugin.GetUuid()
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}

	hostName, err := hostPlugin.GetHostName()
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}

	domainName, err := hostPlugin.GetDomainName()
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}

	networks, err := hostPlugin.GetNetworks()
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}
	return &model.Host{UUID: id, Name: hostName, Domain: domainName, Networks: networks}, nil
}

// GetHostNetworks reports the networks on this host
func (driver *ChapiServer) GetHostNetworks() ([]*model.Network, error) {
	log.Info(">>>>> GetHostNetworks called")
	defer log.Info("<<<<< GetHostNetworks")
	hostPlugin := host.NewHostPlugin()

	networks, err := hostPlugin.GetNetworks()
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}
	if len(networks) == 0 {
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageNoNetworkInterfaces)
	}
	return networks, nil
}

// GetHostInitiators reports the initiators on this host
func (driver *ChapiServer) GetHostInitiators() ([]*model.Initiator, error) {
	log.Info(">>>>> GetHostInitiators called")
	defer log.Info("<<<<< GetHostInitiators")
	//var inits Initiators
	var inits []*model.Initiator

	// fetch iscsi initiator details
	iscsiPlugin := iscsi.NewIscsiPlugin()

	iscsiInits, err := iscsiPlugin.GetIscsiInitiators()
	if err != nil {
		log.Info("Error getting iscsiInitiator: ", err)
	}

	// fetch fc initiator details
	fcPlugin := fc.NewFcPlugin()

	fcInits, err := fcPlugin.GetFcInitiators()
	if err != nil {
		log.Info("Error getting FcInitiator: ", err)
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

	log.Info("initiators ", inits)
	return inits, nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Device methods
///////////////////////////////////////////////////////////////////////////////////////////////////

// GetDevices enumerates all the Nimble volumes with basic details.
// If serialNumber is non-empty then only specified device is returned
func (driver *ChapiServer) GetDevices(serialNumber string) ([]*model.Device, error) {
	log.Info(">>>>> GetDevices called")
	defer log.Info("<<<<< GetDevices")
	multipathPlugin := multipath.NewMultipathPlugin()

	// Enumerate all the Nimble volumes on this host (basic details only)
	devices, err := multipathPlugin.GetDevices(serialNumber)
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}

	// Fail request if no Nimble devices found on this host
	if len(devices) == 0 {
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageNoDevicesOnHost)
	}

	return devices, nil
}

// GetAllDeviceDetails enumerates all the Nimble volumes with detailed information.
// If serialNumber is non-empty then only specified device is returned
func (driver *ChapiServer) GetAllDeviceDetails(serialNumber string) ([]*model.Device, error) {
	log.Infof(">>>>> GetAllDeviceDetails called, serialNumber=%v", serialNumber)
	defer log.Info("<<<<< GetAllDeviceDetails")
	multipathPlugin := multipath.NewMultipathPlugin()

	// Enumerate all the Nimble volumes on this host (full details)
	devices, err := multipathPlugin.GetAllDeviceDetails(serialNumber)
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}

	// Fail request if no Nimble devices found on this host
	if len(devices) == 0 {
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageNoDevicesOnHost)
	}

	return devices, nil
}

// GetPartitionInfo reports the partitions on the provided device
func (driver *ChapiServer) GetPartitionInfo(serialNumber string) ([]*model.DevicePartition, error) {
	log.Infof(">>>>> GetPartitionInfo called, serialNumber=%v", serialNumber)
	defer log.Info("<<<<< GetPartitionInfo")
	multipathPlugin := multipath.NewMultipathPlugin()

	// Enumerate all the Nimble volume's partition
	partitions, err := multipathPlugin.GetPartitionInfo(serialNumber)
	if err != nil {
		return nil, cerrors.NewChapiError(err)
	}

	// Fail request if no partitions found on this host
	if len(partitions) == 0 {
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageNoPartitionsOnVolume)
	}

	return partitions, nil
}

// CreateDevice will attach device on this host based on the details provided
func (driver *ChapiServer) CreateDevice(publishInfo model.PublishInfo) (*model.Device, error) {
	log.Infof(">>>>> CreateDevice called, publishInfo=%v", publishInfo)
	defer log.Info("<<<<< CreateDevice")

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
	return multipathPlugin.AttachDevice(publishInfo.SerialNumber, *publishInfo.BlockDev)
}

// DeleteDevice will delete the given device from the host
func (driver *ChapiServer) DeleteDevice(serialNumber string) error {
	log.Infof(">>>>> DeleteDevice called, serialNumber=%v", serialNumber)
	defer log.Info("<<<<< DeleteDevice")

	return cerrors.NewChapiError(cerrors.Unimplemented, errorMessageNotYetImplemented)
}

// OfflineDevice will offline the given device from the host
func (driver *ChapiServer) OfflineDevice(serialNumber string) error {
	log.Infof(">>>>> OfflineDevice called, serialNumber=%v", serialNumber)
	defer log.Info("<<<<< OfflineDevice")
	multipathPlugin := multipath.NewMultipathPlugin()

	// Enumerate basic details for the serial number
	device, err := driver.getSingleDeviceSummary(serialNumber)
	if err != nil {
		return err
	}

	// Offline the device
	return multipathPlugin.OfflineDevice(*device)
}

// CreateFileSystem writes the given file system to the device with the given serial number
func (driver *ChapiServer) CreateFileSystem(serialNumber string, filesystem string) error {
	log.Infof(">>>>> CreateFileSystem called, serialNumber=%v, filesystem=%v", serialNumber, filesystem)
	defer log.Info("<<<<< CreateFileSystem")
	multipathPlugin := multipath.NewMultipathPlugin()

	// Enumerate basic details for the serial number
	device, err := driver.getSingleDeviceSummary(serialNumber)
	if err != nil {
		return err
	}

	// Format the device
	return multipathPlugin.CreateFileSystem(*device, filesystem)
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Mount point methods
///////////////////////////////////////////////////////////////////////////////////////////////////

// GetMounts reports all mounts on this host for the specified Nimble volume
func (driver *ChapiServer) GetMounts(serialNumber string) ([]*model.Mount, error) {
	log.Infof(">>>>> GetMounts called, serialNumber=%v", serialNumber)
	defer log.Info("<<<<< GetMounts")

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

	return mounts, nil
}

// GetAllMountDetails enumerates the specified mount point ID
func (driver *ChapiServer) GetAllMountDetails(serialNumber string, mountId string) ([]*model.Mount, error) {
	log.Infof(">>>>> GetMount called, serialNumber=%v, mountPointID=%v", serialNumber, mountId)
	defer log.Info("<<<<< GetMount")

	// Route request to the mount package to get the mounts
	mountPlugin := mount.NewMounter()
	mounts, err := mountPlugin.GetAllMountDetails(serialNumber, mountId)
	if err != nil {
		return nil, err
	}

	// Fail request if no mount points detected
	if len(mounts) == 0 {
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageNoMountPointsFound)
	}

	return mounts, nil
}

// CreateMount mounts the given device to the given mount point
func (driver *ChapiServer) CreateMount(serialNumber string, mountPoint string, fsOptions *model.FileSystemOptions) (*model.Mount, error) {
	log.Infof(">>>>> MountDevice called, serialNumber=%v, mountPoint=%v, fsOptions=%v", serialNumber, mountPoint, fsOptions)
	defer log.Info("<<<<< MountDevice")

	// Route request to the mount package to create the mount point
	mountPlugin := mount.NewMounter()
	mount, err := mountPlugin.CreateMount(serialNumber, mountPoint, fsOptions)
	if err != nil {
		return nil, err
	}

	return mount, nil
}

// DeleteMount unmounts the given mount point, serialNumber can be optional in the body
func (driver *ChapiServer) DeleteMount(serialNumber string, mountPointId string) error {
	log.Infof(">>>>> DeleteMount called, serialNumber=%v, mountPointID=%v", serialNumber, mountPointId)
	defer log.Info("<<<<< DeleteMount")

	// Route request to the mount package to delete the mount point
	mountPlugin := mount.NewMounter()
	return mountPlugin.DeleteMount(serialNumber, mountPointId)
}

// CreateBindMount unmounts the given mount point
func (driver *ChapiServer) CreateBindMount(sourceMount string, targetMount string, bindType string) (*model.Mount, error) {
	log.Infof(">>>>> CreateBindMount called, sourceMount=%s, targetMount=%s bindType=%s", sourceMount, targetMount, bindType)
	defer log.Info("<<<<< CreateBindMount")

	return nil, cerrors.NewChapiError(cerrors.Unimplemented, errorMessageNotYetImplemented)
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Internal helper methods
///////////////////////////////////////////////////////////////////////////////////////////////////

// getSingleDeviceSummary uses the driver.GetDevices() endpoint to query basic summary details
// about the given serial number.  If multiple volumes share that serial number (e.g. multipath
// not configured properly), this routine will fail the request.
func (driver *ChapiServer) getSingleDeviceSummary(serialNumber string) (*model.Device, error) {
	log.Infof(">>>>> getSingleDeviceSummary called, serialNumber=%v", serialNumber)
	defer log.Info("<<<<< getSingleDeviceSummary")

	// Enumerate the device details for the provided serial number
	devices, err := driver.GetDevices(serialNumber)
	if err != nil {
		return nil, err
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
