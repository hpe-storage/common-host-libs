// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package wmi handles WMI queries
package wmi

import (
	"strconv"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// Win32_DiskPartition WMI class
type Win32_DiskPartition struct {
	Access                      uint16
	Availability                uint16
	BlockSize                   uint64
	Bootable                    bool
	BootPartition               bool
	Caption                     string
	ConfigManagerErrorCode      uint32 `wmi:",nil=0xFFFFFFFF"` // If property not available, use 0xFFFFFFFF
	ConfigManagerUserConfig     bool
	CreationClassName           string
	Description                 string
	DeviceID                    string
	DiskIndex                   uint32 `wmi:",nil=0xFFFFFFFF"` // If property not available, use 0xFFFFFFFF
	ErrorCleared                bool
	ErrorDescription            string
	ErrorMethodology            string
	HiddenSectors               uint32
	Index                       uint32 `wmi:",nil=0xFFFFFFFF"` // If property not available, use 0xFFFFFFFF
	InstallDate                 string
	LastErrorCode               uint32
	Name                        string
	NumberOfBlocks              uint64
	PNPDeviceID                 string
	PowerManagementCapabilities []uint16
	PowerManagementSupported    bool
	PrimaryPartition            bool
	Purpose                     string
	RewritePartition            bool
	Size                        uint64
	StartingOffset              uint64
	Status                      string
	StatusInfo                  uint16
	SystemCreationClassName     string
	SystemName                  string
	Type                        string
}

// GetWin32DiskPartition enumerates this host's Win32_DiskPartition objects
func GetWin32DiskPartition(whereOperator string) (diskPartitions []*Win32_DiskPartition, err error) {
	log.Tracef(">>>>> GetWin32DiskPartition, whereOperator=%v", whereOperator)
	defer log.Trace("<<<<< GetWin32DiskPartition")

	// Form the WMI query
	wmiQuery := "SELECT * FROM Win32_DiskPartition"
	if whereOperator != "" {
		wmiQuery += " WHERE " + whereOperator
	}

	// Execute the WMI query
	err = ExecQuery(wmiQuery, rootCIMV2, &diskPartitions)
	return diskPartitions, err
}

// GetWin32DiskPartitionForDiskIndex enumerates only the given disk's partitions
func GetWin32DiskPartitionForDiskIndex(diskIndex int) ([]*Win32_DiskPartition, error) {
	whereOperator := "DiskIndex=" + strconv.Itoa(diskIndex)
	return GetWin32DiskPartition(whereOperator)
}
