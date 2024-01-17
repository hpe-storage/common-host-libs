// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package wmi handles WMI queries
package wmi

import (
	"fmt"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// MSFT_Partition WMI class
type MSFT_Partition struct {
	// MSFT_StorageObject base class (in the future we might moved supported contained objects)
	ObjectId             string
	PassThroughClass     string
	PassThroughIds       string
	PassThroughNamespace string
	PassThroughServer    string
	UniqueId             string

	// MSFT_Partition
	DiskId               string
	DiskNumber           uint32 `wmi:",nil=0xFFFFFFFF"` // If property not available, use 0xFFFFFFFF
	PartitionNumber      uint32 `wmi:",nil=0xFFFFFFFF"` // If property not available, use 0xFFFFFFFF
	DriveLetter          uint16
	AccessPaths          []string
	OperationalStatus    uint16
	TransitionState      uint16
	Offset               uint64
	Size                 uint64
	MbrType              uint16
	GptType              string
	Guid                 string
	IsReadOnly           bool
	IsOffline            bool
	IsSystem             bool
	IsBoot               bool
	IsActive             bool
	IsHidden             bool
	IsShadowCopy         bool
	IsDAX                bool
	NoDefaultDriveLetter bool
}

// GetMSFTPartition enumerates this host's MSFTPartition objects
func GetMSFTPartition(whereOperator string) (diskPartitions []*MSFT_Partition, err error) {
	log.Tracef(">>>>> GetMSFTPartition, whereOperator=%v", whereOperator)
	defer log.Trace("<<<<< GetMSFTPartition")

	// Form the WMI query
	wmiQuery := "SELECT * FROM MSFT_Partition"
	if whereOperator != "" {
		wmiQuery += " WHERE " + whereOperator
	}

	// Execute the WMI query
	err = ExecQuery(wmiQuery, rootMicrosoftWindowsStorage, &diskPartitions)
	return diskPartitions, err
}

// GetMSFTPartitionForDiskNumber enumerates only the given disk's partitions
func GetMSFTPartitionForDiskNumber(diskNumber uint32) (diskPartitions []*MSFT_Partition, err error) {
	whereOperator := fmt.Sprintf("DiskNumber=%v", diskNumber)
	return GetMSFTPartition(whereOperator)
}
