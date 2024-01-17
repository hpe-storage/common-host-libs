// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package wmi handles WMI queries
package wmi

import (
	log "github.com/hpe-storage/common-host-libs/logger"
)

// STORAGE_BUS_TYPE defines the supported bus types
// https://docs.microsoft.com/en-us/previous-versions/windows/hardware/drivers/ff566356(v=vs.85)
type STORAGE_BUS_TYPE uint16

const (
	BusTypeUnknown           STORAGE_BUS_TYPE = 0
	BusTypeScsi              STORAGE_BUS_TYPE = 1
	BusTypeAtapi             STORAGE_BUS_TYPE = 2
	BusTypeAta               STORAGE_BUS_TYPE = 3
	BusType1394              STORAGE_BUS_TYPE = 4
	BusTypeSsa               STORAGE_BUS_TYPE = 5
	BusTypeFibre             STORAGE_BUS_TYPE = 6
	BusTypeUsb               STORAGE_BUS_TYPE = 7
	BusTypeRAID              STORAGE_BUS_TYPE = 8
	BusTypeiScsi             STORAGE_BUS_TYPE = 9
	BusTypeSas               STORAGE_BUS_TYPE = 10
	BusTypeSata              STORAGE_BUS_TYPE = 11
	BusTypeSd                STORAGE_BUS_TYPE = 12
	BusTypeMmc               STORAGE_BUS_TYPE = 13
	BusTypeVirtual           STORAGE_BUS_TYPE = 14
	BusTypeFileBackedVirtual STORAGE_BUS_TYPE = 15
	BusTypeSpaces            STORAGE_BUS_TYPE = 16
	BusTypeNvme              STORAGE_BUS_TYPE = 17
	BusTypeSCM               STORAGE_BUS_TYPE = 18
	BusTypeUfs               STORAGE_BUS_TYPE = 19
	BusTypeMax               STORAGE_BUS_TYPE = 20
	BusTypeMaxReserved       STORAGE_BUS_TYPE = 0x7F
)

// MSFT_Disk WMI class
type MSFT_Disk struct {
	// MSFT_StorageObject base class (in the future we might moved supported contained objects)
	ObjectId             string
	PassThroughClass     string
	PassThroughIds       string
	PassThroughNamespace string
	PassThroughServer    string
	UniqueId             string

	// MSFT_Disk
	Path                string
	Location            string
	FriendlyName        string
	UniqueIdFormat      uint16 `wmi:",nil=0xFFFF"`     // If property not available, use 0xFFFF
	Number              uint32 `wmi:",nil=0xFFFFFFFF"` // If property not available, use 0xFFFFFFFF
	SerialNumber        string
	AdapterSerialNumber string
	FirmwareVersion     string
	Manufacturer        string
	Model               string
	Size                uint64
	AllocatedSize       uint64
	LogicalSectorSize   uint32
	PhysicalSectorSize  uint32
	LargestFreeExtent   uint64
	NumberOfPartitions  uint32
	ProvisioningType    uint16
	OperationalStatus   []uint16
	HealthStatus        uint16 `wmi:",nil=0xFFFF"` // If property not available, use 0xFFFF
	BusType             uint16 // See STORAGE_BUS_TYPE
	PartitionStyle      uint16
	Signature           uint32
	Guid                string
	IsOffline           bool
	OfflineReason       uint16
	IsReadOnly          bool
	IsSystem            bool
	IsClustered         bool
	IsBoot              bool
	BootFromDisk        bool
	IsHighlyAvailable   bool
	IsScaleOut          bool
}

// GetMSFTDisk enumerates this host's MSFT_Disk objects
func GetMSFTDisk(whereOperator string) (diskDevices []*MSFT_Disk, err error) {
	log.Tracef(">>>>> GetMSFTDisk, whereOperator=%v", whereOperator)
	defer log.Trace("<<<<< GetMSFTDisk")

	// Form the WMI query
	wmiQuery := "SELECT * FROM MSFT_Disk"
	if whereOperator != "" {
		wmiQuery += " WHERE " + whereOperator
	}

	// Execute the WMI query
	err = ExecQuery(wmiQuery, rootMicrosoftWindowsStorage, &diskDevices)
	return diskDevices, err
}

// GetNimbleMSFTDisk enumerates only Nimble volumes
func GetNimbleMSFTDisk(serialNumber string) ([]*MSFT_Disk, error) {
	query := `(Path LIKE "%ven_nimble&prod_server%")`
	if serialNumber != "" {
		query += ` AND (SerialNumber="` + serialNumber + `")`
	}
	return GetMSFTDisk(query)
}
