// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package ioctl provides Windows IOCTL support
package ioctl

import (
	"fmt"
	"syscall"
	"unsafe"

	uuid "github.com/satori/go.uuid"
	log "github.com/hpe-storage/common-host-libs/logger"
)

// PARTITION_STYLE enumeration
type PARTITION_STYLE uint32

const (
	PARTITION_STYLE_MBR = iota
	PARTITION_STYLE_GPT
	PARTITION_STYLE_RAW
)

// MEDIA_TYPE enumeration
type MEDIA_TYPE uint32

const (
	Unknown        = iota // Format is unknown
	F5_1Pt2_512           // 5.25", 1.2MB,  512 bytes/sector
	F3_1Pt44_512          // 3.5",  1.44MB, 512 bytes/sector
	F3_2Pt88_512          // 3.5",  2.88MB, 512 bytes/sector
	F3_20Pt8_512          // 3.5",  20.8MB, 512 bytes/sector
	F3_720_512            // 3.5",  720KB,  512 bytes/sector
	F5_360_512            // 5.25", 360KB,  512 bytes/sector
	F5_320_512            // 5.25", 320KB,  512 bytes/sector
	F5_320_1024           // 5.25", 320KB,  1024 bytes/sector
	F5_180_512            // 5.25", 180KB,  512 bytes/sector
	F5_160_512            // 5.25", 160KB,  512 bytes/sector
	RemovableMedia        // Removable media other than floppy
	FixedMedia            // Fixed hard disk media
	F3_120M_512           // 3.5", 120M Floppy
	F3_640_512            // 3.5" ,  640KB,  512 bytes/sector
	F5_640_512            // 5.25",  640KB,  512 bytes/sector
	F5_720_512            // 5.25",  720KB,  512 bytes/sector
	F3_1Pt2_512           // 3.5" ,  1.2Mb,  512 bytes/sector
	F3_1Pt23_1024         // 3.5" ,  1.23Mb, 1024 bytes/sector
	F5_1Pt23_1024         // 5.25",  1.23MB, 1024 bytes/sector
	F3_128Mb_512          // 3.5" MO 128Mb   512 bytes/sector
	F3_230Mb_512          // 3.5" MO 230Mb   512 bytes/sector
	F8_256_128            // 8",     256KB,  128 bytes/sector
	F3_200Mb_512          // 3.5",   200M Floppy (HiFD)
	F3_240M_512           // 3.5",   240Mb Floppy (HiFD)
	F3_32M_512            // 3.5",   32Mb Floppy
)

type DISK_PARTITION_INFO_MBR struct {
	SizeOfPartitionInfo uint32
	PartitionStyle      PARTITION_STYLE
	Signature           uint32
	CheckSum            uint32
}

type DISK_PARTITION_INFO_GPT struct {
	SizeOfPartitionInfo uint32
	PartitionStyle      PARTITION_STYLE
	DiskId              uuid.UUID
}

type DISK_GEOMETRY struct {
	Cylinders         uint64
	MediaType         MEDIA_TYPE
	TracksPerCylinder uint32
	SectorsPerTrack   uint32
	BytesPerSector    uint32
}

type DISK_GEOMETRY_EX_RAW struct {
	Geometry DISK_GEOMETRY
	DiskSize uint64
}

type DISK_GEOMETRY_EX struct {
	Geometry         DISK_GEOMETRY
	DiskSize         uint64
	DiskPartitionMBR *DISK_PARTITION_INFO_MBR
	DiskPartitionGPT *DISK_PARTITION_INFO_GPT
}

// GetDiskGeometry issues an IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS to the given volume and
// returns back the volume's geometry details.
func GetDiskGeometry(diskNumber uint32) (diskGeometry *DISK_GEOMETRY_EX, err error) {
	log.Tracef(">>>>> GetDiskGeometry, diskNumber=%v", diskNumber)
	defer log.Trace("<<<<< GetDiskGeometry")

	// Convert disk number to a disk path UTF16 string
	diskPathUTF16 := syscall.StringToUTF16(diskPathFromNumber(diskNumber))

	// Get a handle to the disk object
	var handle syscall.Handle
	handle, err = syscall.CreateFile(&diskPathUTF16[0], syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)

	if handle == syscall.Handle(INVALID_HANDLE_VALUE) {
		// Return rile not found if INVALID_HANDLE_VALUE returned
		if err == nil {
			err = syscall.ERROR_FILE_NOT_FOUND
		}
	} else {
		// Close the volume handle when we're done
		defer syscall.CloseHandle(handle)

		// The DISK_GEOMETRY_EX structure is comprised of a DISK_GEOMETRY_EX structure, DISK_PARTITION_INFO
		// structure, and DISK_DETECTION_INFO structure which totals 112 bytes.  We'll allocate
		// a 128 byte buffer to submit with the IOCTL.
		dataBuffer := make([]uint8, 0x80)

		// Issue the IOCTL
		var bytesReturned uint32
		err = syscall.DeviceIoControl(handle, IOCTL_DISK_GET_DRIVE_GEOMETRY_EX, nil, 0, &dataBuffer[0], uint32(len(dataBuffer)), &bytesReturned, nil)

		// If IOCTL was successful, extract the DISK_GEOMETRY_EX
		if err == nil {
			// Extract the raw structures
			diskGeometryBase := (*DISK_GEOMETRY_EX_RAW)(unsafe.Pointer(&dataBuffer[0x00]))
			diskPartitionMBR := (*DISK_PARTITION_INFO_MBR)(unsafe.Pointer(&dataBuffer[0x20]))
			diskPartitionGPT := (*DISK_PARTITION_INFO_GPT)(unsafe.Pointer(&dataBuffer[0x20]))

			// Build up and populate the return DISK_GEOMETRY_EX object
			diskGeometry = new(DISK_GEOMETRY_EX)
			diskGeometry.Geometry = diskGeometryBase.Geometry
			diskGeometry.DiskSize = diskGeometryBase.DiskSize
			switch diskPartitionMBR.PartitionStyle {
			case PARTITION_STYLE_MBR:
				diskGeometry.DiskPartitionMBR = diskPartitionMBR
			case PARTITION_STYLE_GPT:
				diskGeometry.DiskPartitionGPT = diskPartitionGPT
			}
		}
	}

	// Log the results
	if err == nil {
		var partitionDetails string
		if diskGeometry.DiskPartitionMBR != nil {
			partitionDetails = fmt.Sprintf("MBR, CheckSum=%v, Signature=%v", diskGeometry.DiskPartitionMBR.CheckSum, diskGeometry.DiskPartitionMBR.Signature)
		} else if diskGeometry.DiskPartitionGPT != nil {
			partitionDetails = fmt.Sprintf("GPT, DiskId=%v", diskGeometry.DiskPartitionGPT.DiskId.String())
		}
		log.Tracef("DiskSize=%v, Partition={%v}", diskGeometry.DiskSize, partitionDetails)
	} else {
		// Log error on failure
		log.Errorf("Error=%v", err)
	}

	return diskGeometry, err
}

// GetDiskCapacity returns back the given disk's capacity (in bytes)
func GetDiskCapacity(diskNumber uint32) (diskCapacity uint64, err error) {
	log.Tracef(">>>>> GetDiskCapacity, diskNumber=%v", diskNumber)
	defer log.Trace("<<<<< GetDiskCapacity")

	var diskGeometry *DISK_GEOMETRY_EX
	if diskGeometry, err = GetDiskGeometry(diskNumber); err == nil && diskGeometry != nil {
		diskCapacity = diskGeometry.DiskSize
	}
	return
}
