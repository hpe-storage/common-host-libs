// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package multipath

import (
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	log "github.com/hpe-storage/common-host-libs/logger"
)

// getDevices enumerates all the Nimble volumes while only providing basic details (e.g. serial number).
// If a "serialNumber" is passed in, only that specific serial number is enumerated.
// getDevices enumerates all the Nimble volumes while only providing basic details (e.g. serial number).
// If a "serialNumber" is passed in, only that specific serial number is enumerated.
func (plugin *MultipathPlugin) getDevices(serialNumber string) ([]*model.Device, error) {
	log.Infof(">>>>> getDevices, serialNumber=%v", serialNumber)
	defer log.Info("<<<<< getDevices")
	// TODO
	return nil, nil
}

// getDevices enumerates all the Nimble volumes while providing full details about the device.
// If a "serialNumber" is passed in, only that specific serial number is enumerated.
func (plugin *MultipathPlugin) getAllDeviceDetails(serialNumber string) ([]*model.Device, error) {
	log.Info(">>>>> getAllDeviceDetails")
	defer log.Info("<<<<< getAllDeviceDetails")
	// TODO
	return nil, nil
}

// getPartitionInfo enumerates the partitions on the given volume
func (plugin *MultipathPlugin) getPartitionInfo(serialNumber string) ([]*model.DevicePartition, error) {
	log.Infof(">>>>> getPartitionInfo, serialNumber=%v", serialNumber)
	defer log.Info("<<<<< getPartitionInfo")
	// TODO
	return nil, nil
}

// offlineDevice is called to offline the given device
func (plugin *MultipathPlugin) offlineDevice(device model.Device) error {
	log.Infof(">>>>> offlineDevice")
	defer log.Info("<<<<< offlineDevice")

	// TODO
	return nil
}

// createFileSystem is called to create a file system on the given device
func (plugin *MultipathPlugin) createFileSystem(device model.Device, filesystem string) error {
	log.Infof(">>>>> createFileSystem")
	defer log.Info("<<<<< createFileSystem")

	// TODO
	return nil
}
