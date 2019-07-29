// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package multipath

import (
	"sync"

	"github.com/hpe-storage/common-host-libs/chapi2/cerrors"
	"github.com/hpe-storage/common-host-libs/chapi2/fc"
	"github.com/hpe-storage/common-host-libs/chapi2/iscsi"
	"github.com/hpe-storage/common-host-libs/chapi2/model"
)

const (
	// Shared error messages
	errorMessageUnableLocateIscsiTarget = "unable to locate iSCSI target"
	errorMessageVolumeNotFound          = "volume not found"
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

func (plugin *MultipathPlugin) AttachDevice(serialNumber string, blockDev model.BlockDeviceAccessInfo) (*model.Device, error) {
	return nil, cerrors.NewChapiError(cerrors.Unimplemented)
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
