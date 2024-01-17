// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package virtualdevice

import (
	"github.com/hpe-storage/common-host-libs/chapi2/model"
)

type VirtualDevPlugin struct {
}

func NewVirtualDevPlugin() *VirtualDevPlugin {
	return &VirtualDevPlugin{}
}

func (plugin *VirtualDevPlugin) GetDeviceName(serial string) (*string, error) {
	return nil, nil
}

func (plugin *VirtualDevPlugin) AttachDevice(publishInfo *model.PublishInfo) error {
	return nil
}

func (plugin *VirtualDevPlugin) DetachDevice(device model.Device) error {
	return nil
}

func (plugin *VirtualDevPlugin) IsDeviceReady(serial string) error {
	return nil
}

func (plugin *VirtualDevPlugin) OfflineDevice(device model.Device) error {
	return nil
}
