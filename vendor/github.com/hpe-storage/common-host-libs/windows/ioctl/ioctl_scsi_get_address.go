// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package ioctl provides Windows IOCTL support
package ioctl

import (
	"strings"
	"syscall"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

type SCSI_ADDRESS struct {
	Length     uint32
	PortNumber uint8
	PathId     uint8
	TargetId   uint8
	Lun        uint8
}

// GetScsiAddress issues an IOCTL_SCSI_GET_ADDRESS to the given device and returns back the
// device's SCSI_ADDRESS struct.
func GetScsiAddress(devicePathID string) (scsiAddress *SCSI_ADDRESS, err error) {
	log.Tracef(">>>>> GetScsiAddress, devicePathID=%v", devicePathID)
	defer log.Trace("<<<<< GetScsiAddress")

	// Convert device path to a UTF16 string (strip any trailing backslash)
	devicePathID = strings.TrimRight(devicePathID, `\`)
	devicePathIDUTF16 := syscall.StringToUTF16(devicePathID)

	// Get a handle to the device object
	var handle syscall.Handle
	handle, err = syscall.CreateFile(&devicePathIDUTF16[0], syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)

	if handle == syscall.Handle(INVALID_HANDLE_VALUE) {
		// Return rile not found if INVALID_HANDLE_VALUE returned
		if err == nil {
			err = syscall.ERROR_FILE_NOT_FOUND
		}
	} else {
		// Close the volume handle when we're done
		defer syscall.CloseHandle(handle)

		// We only need an 8 byte buffer to populate the SCSI_ADDRESS struct
		dataBuffer := make([]uint8, 8)

		// Issue the IOCTL
		var bytesReturned uint32
		err = syscall.DeviceIoControl(handle, IOCTL_SCSI_GET_ADDRESS, nil, 0, &dataBuffer[0], uint32(len(dataBuffer)), &bytesReturned, nil)

		// If IOCTL was successful, extract the SCSI_ADDRESS
		if err == nil {
			scsiAddress = (*SCSI_ADDRESS)(unsafe.Pointer(&dataBuffer[0]))
		}
	}

	// Log the results
	if err == nil {
		log.Tracef("SCSI_ADDRESS Lengh=%v, ID=%02X:%02X:%02X:%02X", scsiAddress.Length, scsiAddress.PortNumber, scsiAddress.PathId, scsiAddress.TargetId, scsiAddress.Lun)
	} else {
		// Log error on failure
		log.Errorf("Error=%v", err)
	}

	return scsiAddress, err
}
