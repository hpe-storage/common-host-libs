// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package model

import (
	"github.com/hpe-storage/common-host-libs/windows/iscsidsc"
	"github.com/hpe-storage/common-host-libs/windows/wmi"
)

// KeyFileInfo : Windows keyfile object
type KeyFileInfo struct {
	Path string `json:"path,omitempty"`
}

// NetworkPrivate provides model.Network platform specific private data
type NetworkPrivate struct {
	InitiatorInstance   string // WMI MSiSCSI_PortalInfoClass->InstanceName
	InitiatorPortNumber uint32 // WMI MSiSCSI_PortalInfoClass->ISCSI_PortalInfo->Index
}

// TargetPortalPrivate provides model.TargetPortal platform specific private data
type TargetPortalPrivate struct {
	WindowsTargetPortal *iscsidsc.ISCSI_TARGET_PORTAL `json:"-"` // Windows iSCSI target portal object
}

// DevicePrivate provides model.Device platform specific private data
type DevicePrivate struct {
	WindowsDisk *wmi.MSFT_Disk `json:"-"` // Windows device details
}

// MountPrivate provides model.Mount platform specific private data
type MountPrivate struct {
	WindowsDisk      *wmi.MSFT_Disk      `json:"-"` // Windows device details
	WindowsPartition *wmi.MSFT_Partition `json:"-"` // Windows partition details
}
