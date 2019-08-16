// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package multipath

import (
	"sync"

	"github.com/hpe-storage/common-host-libs/chapi2/cerrors"
	"github.com/hpe-storage/common-host-libs/chapi2/fc"
	"github.com/hpe-storage/common-host-libs/chapi2/iscsi"
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	log "github.com/hpe-storage/common-host-libs/logger"
)

const (
	// Shared error messages
	errorMessageDeviceNotFound           = "device not found"
	errorMessageInvalidAccessProtocol    = `invalid AccessProtocol "%v"`
	errorMessageMisconfiguredMultipathIO = `misconfigured multipath I/O - multiple instances of serial number "%v" detected`
	errorMessageSerialNumberNotProvided  = "serial number not provided"
	errorMessageUnableLocateIscsiTarget  = "unable to locate iSCSI target"
)

var (
	lock            = &sync.Mutex{}
	targetTypeCache *TargetTypeCache // Global target type cache
)

type MultipathPlugin struct {
	fcPlugin    *fc.FcPlugin
	iscsiPlugin *iscsi.IscsiPlugin
}

func NewMultipathPlugin() *MultipathPlugin {
	return &MultipathPlugin{
		fcPlugin:    fc.NewFcPlugin(),
		iscsiPlugin: iscsi.NewIscsiPlugin(),
	}
}

func (plugin *MultipathPlugin) GetDevices(serialNumber string) ([]*model.Device, error) {
	devices, err := plugin.getDevices(serialNumber)
	if err != nil {
		return nil, err
	}
	return devices, nil
}

func (plugin *MultipathPlugin) GetAllDeviceDetails(serialNumber string) ([]*model.Device, error) {
	devices, err := plugin.getAllDeviceDetails(serialNumber)
	if err != nil {
		return nil, err
	}
	return devices, nil
}

func (plugin *MultipathPlugin) GetPartitionInfo(serialNumber string) ([]*model.DevicePartition, error) {
	partitions, err := plugin.getPartitionInfo(serialNumber)
	if err != nil {
		return nil, err
	}
	return partitions, nil
}

func (plugin *MultipathPlugin) OfflineDevice(device model.Device) error {
	return plugin.offlineDevice(device)
}

func (plugin *MultipathPlugin) CreateFileSystem(device model.Device, filesystem string) error {
	return plugin.createFileSystem(device, filesystem)
}

// AttachDevice attaches the given block device to this host.  If the device is successfully
// attached, a model.Device object is returned for the attached device.
func (plugin *MultipathPlugin) AttachDevice(serialNumber string, blockDev model.BlockDeviceAccessInfo) (device *model.Device, err error) {
	log.Trace(">>>>> AttachDevice called")
	defer log.Trace("<<<<< AttachDevice")

	log.Infof("Attach device, serialNumber=%v, protocol=%v", serialNumber, blockDev.AccessProtocol)

	// Fail request if no serial number provided
	if serialNumber == "" {
		err := cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageSerialNumberNotProvided)
		log.Error(err)
		return nil, err
	}

	// If it's an FC volume, all we need to do is an FC rescan.  If it's iSCSI, we need to
	// ensure the target is logged in.  Any other AccessProtocol is invalid and unsupported.
	switch blockDev.AccessProtocol {
	case model.AccessProtocolFC:
		err = fc.NewFcPlugin().RescanFcTarget(blockDev.LunID)
	case model.AccessProtocolIscsi:
		err = iscsi.NewIscsiPlugin().LoginTarget(blockDev)
	default:
		err = cerrors.NewChapiErrorf(cerrors.InvalidArgument, errorMessageInvalidAccessProtocol, blockDev.AccessProtocol)
		log.Error(err)
	}

	// Exit if FC rescan or iSCSI login failure
	if err != nil {
		return nil, err
	}

	// Enumerate the device with the provided serial number
	var devices []*model.Device
	devices, err = plugin.GetAllDeviceDetails(serialNumber)
	if err != nil {
		return nil, err
	}

	// If device was not found, fail the request
	if len(devices) == 0 {
		err = cerrors.NewChapiError(cerrors.NotFound, errorMessageDeviceNotFound)
		log.Error(err)
		return nil, err
	}

	// Return the enumerated serial number.  No need to check for duplicate serial number
	// entries as the GetAllDeviceDetails() routine already performs this check.
	log.Infof("Device successfully attached, SerialNumber=%v, PathName=%v, Size=%v", devices[0].SerialNumber, devices[0].Pathname, devices[0].Size)
	return devices[0], nil
}

// getTargetTypeCache returns the global TargetTypeCache object
func getTargetTypeCache() *TargetTypeCache {
	lock.Lock()
	defer lock.Unlock()
	if targetTypeCache == nil {
		targetTypeCache = NewTargetTypeCache()
	}
	return targetTypeCache
}

// TODO, Remove or implement member functions below that are not utilized

// func (plugin *MultipathPlugin) GetDeviceName(serial string) (*string, error) {
// 	return nil, nil
// }

// func (plugin *MultipathPlugin) GetFriendlyName(serial string) (*string, error) {
// 	return nil, nil
// }

// func (plugin *MultipathPlugin) GetAllPathOfDevice(serial string) ([]model.Path, error) {
// 	return nil, nil
// }

// func (plugin *MultipathPlugin) DetachDevice(device model.Device) error {
// 	return nil
// }

// func (plugin *MultipathPlugin) IsDeviceReady(serial string) error {
// 	return nil
// }

// func (plugin *MultipathPlugin) GetAllMaps(format string) ([]string, error) {
// 	return nil, nil
// }

// func (plugin *MultipathPlugin) GetAllPaths(format string) ([]string, error) {
// 	return nil, nil
// }

// func (plugin *MultipathPlugin) ReloadMaps() error {
// 	return nil
// }

// func (plugin *MultipathPlugin) ReconfigureMaps() error {
// 	return nil
// }

// func (plugin *MultipathPlugin) DeleteDevicePaths(serial string, lunId string) error {
// 	return nil
// }
