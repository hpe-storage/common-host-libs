// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package wmi handles WMI queries
package wmi

import (
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/windows/ioctl"
)

// Win32_DiskDrive WMI class
type Win32_DiskDrive struct {
	Availability                uint16
	BytesPerSector              uint32
	Capabilities                []uint16
	CapabilityDescriptions      []string
	Caption                     string
	CompressionMethod           string
	ConfigManagerErrorCode      uint32 `wmi:",nil=0xFFFFFFFF"` // If property not available, use 0xFFFFFFFF
	ConfigManagerUserConfig     bool
	CreationClassName           string
	DefaultBlockSize            uint64
	Description                 string
	DeviceID                    string
	ErrorCleared                bool
	ErrorDescription            string
	ErrorMethodology            string
	FirmwareRevision            string
	Index                       uint32 `wmi:",nil=0xFFFFFFFF"` // If property not available, use 0xFFFFFFFF
	InstallDate                 string
	InterfaceType               string
	LastErrorCode               uint32
	Manufacturer                string
	MaxBlockSize                uint64
	MaxMediaSize                uint64
	MediaLoaded                 bool
	MediaType                   string
	MinBlockSize                uint64
	Model                       string
	Name                        string
	NeedsCleaning               bool
	NumberOfMediaSupported      uint32
	Partitions                  uint32
	PNPDeviceID                 string
	PowerManagementCapabilities []uint16
	PowerManagementSupported    bool
	SCSIBus                     uint32 `wmi:",nil=0xFFFFFFFF"` // If property not available, use 0xFFFFFFFF
	SCSILogicalUnit             uint16 `wmi:",nil=0xFFFF"`     // If property not available, use 0xFFFF
	SCSIPort                    uint16 `wmi:",nil=0xFFFF"`     // If property not available, use 0xFFFF
	SCSITargetId                uint16 `wmi:",nil=0xFFFF"`     // If property not available, use 0xFFFF
	SectorsPerTrack             uint32
	SerialNumber                string
	Signature                   uint32
	Size                        uint64
	Status                      string
	StatusInfo                  uint16
	SystemCreationClassName     string
	SystemName                  string
	TotalCylinders              uint64
	TotalHeads                  uint32
	TotalSectors                uint64
	TotalTracks                 uint64
	TracksPerCylinder           uint32
}

// GetWin32DiskDrive enumerates this host's Win32_DiskDrive objects
func GetWin32DiskDrive(whereOperator string) (diskDevices []*Win32_DiskDrive, err error) {
	log.Tracef(">>>>> GetWin32DiskDrive, whereOperator=%v", whereOperator)
	defer log.Trace("<<<<< GetWin32DiskDrive")

	// Form the WMI query
	wmiQuery := "SELECT * FROM Win32_DiskDrive"
	if whereOperator != "" {
		wmiQuery += " WHERE " + whereOperator
	}

	// Execute the WMI query
	err = ExecQuery(wmiQuery, rootCIMV2, &diskDevices)

	// NWT-3428.  As detailed in the JIRA ticket, the Win32_DiskDrive class can report the disk
	// disk capacity as being smaller than the actual capacity.  It's usually off by several MiB.
	// Here we're going to query the disk capacity directly.  If the request is successful, we'll
	// use that capacity for this disk, else we'll leave the WMI enumerated value.
	for _, diskDevice := range diskDevices {
		if diskDevice.Index != ^uint32(0) {
			capacity, _ := ioctl.GetDiskCapacity(diskDevice.Index)
			if capacity > diskDevice.Size {
				log.Tracef("Adjusting disk capacity from %v to true size of %v", diskDevice.Size, capacity)
				diskDevice.Size = capacity
			}
		}
	}

	return diskDevices, err
}

// GetNimbleWin32DiskDrive enumerates only Nimble volumes
func GetNimbleWin32DiskDrive(serialNumber string) ([]*Win32_DiskDrive, error) {
	whereOperator := `(PNPDeviceID LIKE "%VEN_NIMBLE&PROD_SERVER%")`
	if serialNumber != "" {
		whereOperator += ` AND (SerialNumber="` + serialNumber + `")`
	}
	return GetWin32DiskDrive(whereOperator)
}
