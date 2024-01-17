// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package ioctl provides Windows IOCTL support
package ioctl

import (
	"encoding/binary"
	"strings"
	"syscall"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

type DISK_EXTENT struct {
	DiskNumber     uint32
	StartingOffset uint64
	ExtentLength   uint64
}

// GetVolumeDiskExtents issues an IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS to the given volume and
// returns back the volume's DISK_EXTENT array.
func GetVolumeDiskExtents(volumePathID string) (diskExtents []DISK_EXTENT, err error) {
	log.Tracef(">>>>> GetVolumeDiskExtents, volumePathID=%v", volumePathID)
	defer log.Trace("<<<<< GetVolumeDiskExtents")

	// Convert volume path to a UTF16 string (strip any trailing backslash)
	volumePathID = strings.TrimRight(volumePathID, `\`)
	volumePathIDUTF16 := syscall.StringToUTF16(volumePathID)

	// Get a handle to the volume object
	var handle syscall.Handle
	handle, err = syscall.CreateFile(&volumePathIDUTF16[0], syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)

	if handle == syscall.Handle(INVALID_HANDLE_VALUE) {
		// Return rile not found if INVALID_HANDLE_VALUE returned
		if err == nil {
			err = syscall.ERROR_FILE_NOT_FOUND
		}
	} else {
		// Close the volume handle when we're done
		defer syscall.CloseHandle(handle)

		// We'll start with a buffer size of 256 bytes and grow it if IOCTL request indicates additional
		// space is needed.  Note that we default to 32 bytes in the C# code.  Extremely unlikely we'll
		// ever go beyond 256 bytes.
		dataBuffer := make([]uint8, 256)

		// Issue the IOCTL
		var bytesReturned uint32
		err = syscall.DeviceIoControl(handle, IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS, nil, 0, &dataBuffer[0], uint32(len(dataBuffer)), &bytesReturned, nil)

		// If the buffer we passed in wasn't large enough, allocate a larger buffer and try once more
		if (err == syscall.ERROR_INSUFFICIENT_BUFFER) || (err == syscall.ERROR_MORE_DATA) {
			if bytesReturned >= 4 {

				// Extract the number of disk extents and determine required IOCTL buffer length
				numberOfDiskExtents := binary.LittleEndian.Uint32(dataBuffer[0:4])
				dataBufferLen := 8 + (numberOfDiskExtents * uint32(unsafe.Sizeof(diskExtents[0])))

				// We should only have one extent per volume so our buffer requirements should be
				// extremely small.  If there is a bug in the IOCTL handler, and it returns an
				// unreasonably large value, we'll limit the next request to 4K and log an error.
				const maxBufferLen = uint32(4096)
				if dataBufferLen > maxBufferLen {
					log.Errorf("Buffer limits exceeded, numberOfDiskExtents=%v, dataBufferLen=%v, maxBufferLen=%v", numberOfDiskExtents, dataBufferLen, maxBufferLen)
					dataBufferLen = maxBufferLen
				}

				// Resize the buffer and try once more
				dataBuffer = make([]uint8, dataBufferLen)
				err = syscall.DeviceIoControl(handle, IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS, nil, 0, &dataBuffer[0], uint32(len(dataBuffer)), &bytesReturned, nil)
			}
		}

		// If IOCTL was successful, extract the DISK_EXTENT array
		if (err == nil) && (bytesReturned >= 4) {
			numberOfDiskExtents := binary.LittleEndian.Uint32(dataBuffer[0:4])
			diskExtents = (*[1024]DISK_EXTENT)(unsafe.Pointer(&dataBuffer[8]))[:numberOfDiskExtents]
		}
	}

	// Log the results
	if err == nil {
		for _, extent := range diskExtents {
			log.Tracef("DiskNumber=%v, StartingOffset=%v, ExtentLength=%v", extent.DiskNumber, extent.StartingOffset, extent.ExtentLength)
		}
	} else {
		// Log error on failure
		log.Errorf("Error=%v", err)
	}

	return diskExtents, err
}
