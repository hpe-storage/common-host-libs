// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package wmi handles WMI queries
package wmi

import (
	log "github.com/hpe-storage/common-host-libs/logger"
)

// Win32_Volume WMI class
type Win32_Volume struct {
	Access                       uint16
	Automount                    bool
	Availability                 uint16
	BlockSize                    uint64
	BootVolume                   bool
	Capacity                     uint64
	Caption                      string
	Compressed                   bool
	ConfigManagerErrorCode       uint32 `wmi:",nil=0xFFFFFFFF"` // If property not available, use 0xFFFFFFFF
	ConfigManagerUserConfig      bool
	CreationClassName            string
	Description                  string
	DeviceID                     string
	DirtyBitSet                  bool
	DriveLetter                  string
	DriveType                    uint32
	ErrorCleared                 bool
	ErrorDescription             string
	ErrorMethodology             string
	FileSystem                   string
	FreeSpace                    uint64
	IndexingEnabled              bool
	InstallDate                  string
	Label                        string
	LastErrorCode                uint32
	MaximumFileNameLength        uint32
	Name                         string
	NumberOfBlocks               uint64
	PageFilePresent              bool
	PNPDeviceID                  string
	PowerManagementCapabilities  []uint16
	PowerManagementSupported     bool
	Purpose                      string
	QuotasEnabled                bool
	QuotasIncomplete             bool
	QuotasRebuilding             bool
	SerialNumber                 uint32
	Status                       string
	StatusInfo                   uint16
	SupportsDiskQuotas           bool
	SupportsFileBasedCompression bool
	SystemCreationClassName      string
	SystemName                   string
	SystemVolume                 bool
}

// GetWin32Volume enumerates this host's Win32_Volume objects
func GetWin32Volume() (volumes []*Win32_Volume, err error) {
	log.Tracef(">>>>> GetWin32Volume")
	defer log.Trace("<<<<< GetWin32Volume")

	// Execute the WMI query
	err = ExecQuery("SELECT * FROM Win32_Volume", rootCIMV2, &volumes)
	return volumes, err
}
