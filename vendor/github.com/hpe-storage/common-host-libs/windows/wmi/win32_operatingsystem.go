// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package wmi handles WMI queries
package wmi

import (
	log "github.com/hpe-storage/common-host-libs/logger"
)

// Win32_OperatingSystem WMI class
type Win32_OperatingSystem struct {
	BootDevice                                string
	BuildNumber                               string
	BuildType                                 string
	Caption                                   string
	CodeSet                                   string
	CountryCode                               string
	CreationClassName                         string
	CSCreationClassName                       string
	CSDVersion                                string
	CSName                                    string
	CurrentTimeZone                           int16 `wmi:",nil=0x7FFF"` // If property not available, use 0x7FFF
	DataExecutionPrevention_Available         bool
	DataExecutionPrevention_32BitApplications bool
	DataExecutionPrevention_Drivers           bool
	DataExecutionPrevention_SupportPolicy     uint8
	Debug                                     bool
	Description                               string
	Distributed                               bool
	EncryptionLevel                           uint32 `wmi:",nil=0xFFFFFFFF"` // If property not available, use 0xFFFFFFFF
	ForegroundApplicationBoost                uint8  `wmi:",nil=0xFF"`       // If property not available, use 0xFF
	FreePhysicalMemory                        uint64
	FreeSpaceInPagingFiles                    uint64
	FreeVirtualMemory                         uint64
	InstallDate                               string
	LargeSystemCache                          uint32
	LastBootUpTime                            string
	LocalDateTime                             string
	Locale                                    string
	Manufacturer                              string
	MaxNumberOfProcesses                      uint32
	MaxProcessMemorySize                      uint64
	MUILanguages                              []string
	Name                                      string
	NumberOfLicensedUsers                     uint32 `wmi:",nil=0xFFFFFFFF"` // If property not available, use 0xFFFFFFFF
	NumberOfProcesses                         uint32
	NumberOfUsers                             uint32
	OperatingSystemSKU                        uint32
	Organization                              string
	OSArchitecture                            string
	OSLanguage                                uint32
	OSProductSuite                            uint32
	OSType                                    uint16
	OtherTypeDescription                      string
	PAEEnabled                                bool
	PlusProductID                             string
	PlusVersionNumber                         string
	PortableOperatingSystem                   bool
	Primary                                   bool
	ProductType                               uint32
	RegisteredUser                            string
	SerialNumber                              string
	ServicePackMajorVersion                   uint16
	ServicePackMinorVersion                   uint16
	SizeStoredInPagingFiles                   uint64
	Status                                    string
	SuiteMask                                 uint32
	SystemDevice                              string
	SystemDirectory                           string
	SystemDrive                               string
	TotalSwapSpaceSize                        uint64
	TotalVirtualMemorySize                    uint64
	TotalVisibleMemorySize                    uint64
	Version                                   string
	WindowsDirectory                          string
}

// GetWin32OperatingSystem enumerates this host's Win32_OperatingSystem object
func GetWin32OperatingSystem() (operatingSystem *Win32_OperatingSystem, err error) {
	log.Trace(">>>>> GetWin32OperatingSystem")
	defer log.Trace("<<<<< GetWin32OperatingSystem")

	// Execute the WMI query
	err = ExecQuery("SELECT * FROM Win32_OperatingSystem", rootCIMV2, &operatingSystem)
	return operatingSystem, err
}
