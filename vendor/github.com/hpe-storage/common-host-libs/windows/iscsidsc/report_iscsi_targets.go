// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package iscsidsc wraps the Windows iSCSI Discovery Library API
package iscsidsc

import (
	"syscall"
	"time"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// ReportIscsiTargets - Go wrapped Win32 API - ReportIscsiTargets()
// https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-reportiscsitargetsw
func ReportIscsiTargets(forceUpdate bool) (targets []string, err error) {
	log.Tracef(">>>>> ReportIscsiTargets, forceUpdate=%v", forceUpdate)
	defer log.Trace("<<<<< ReportIscsiTargets")

	//
	// Note that the algorithm employed below is ported from C# ReportTargetIqns() including
	// the safety checks / workarounds previously employed.
	//

	// Convert ForceUpdate boolean into a uint8 variable
	uint8ForceUpdate := uint8(0)
	if forceUpdate {
		uint8ForceUpdate = 1
	}

	const RetryCount int = 5
	var iscsiErr uintptr
	var bufferSizeNeeded uint32

	for retry := 0; retry < RetryCount; retry++ {
		// Do an initial call to the function to find the buffersize
		iscsiErr, _, _ = procReportIScsiTargetsW.Call(uintptr(uint8ForceUpdate), uintptr(unsafe.Pointer(&bufferSizeNeeded)), uintptr(0))

		// ReportIScsiTargetsA() is giving a bogus error occasionally but still returning the required buffer size and
		// successfully completing the ReportIScsiTargetsA() function.  So I removed the ERROR_INSUFFICIENT_BUFFER
		// check for this test...
		// the 64000 is a safety net in case ReportIScsiTargetsA() returns a bogus size needed
		//     ^^^^^ Unsure what "64000" is referenced in the C# code

		// BZ30145 we have seen a potential race condition where this function returns ERROR_INSUFFICIENT_BUFFER
		// and for a brief time period bufferSizeNeeded=0, then it has non-zero value hack
		if (bufferSizeNeeded > 0) || ((iscsiErr == ERROR_SUCCESS) && (bufferSizeNeeded == 0)) {
			break
		} else {
			uint8ForceUpdate = 0 // Do not do deep discovery
			time.Sleep(5 * time.Millisecond)
			if bufferSizeNeeded > 0 {
				break
			}
		}
	}

	if bufferSizeNeeded > 0 {
		// Allocate memory for that list of targets
		// Occasionally, perhaps when snapshots are being created, onlined, ... we get an
		// ERROR_INSUFFICIENT_BUFFER error on the next call.  This will add the ability to
		// handle a few more targets if one arrives between these two calls. Bug 2444
		bufferSizeNeeded += 8192
		byteBuffer := make([]uint16, bufferSizeNeeded)

		// We have done a deep discovery going to array in the previous call to get buffer size. So we just need to get cached data. Otherwise
		// we may again do a deep discovery on an changing # of targets on the array and get insufficient buffer
		iscsiErr, _, _ = procReportIScsiTargetsW.Call(uintptr(0), uintptr(unsafe.Pointer(&bufferSizeNeeded)), uintptr(unsafe.Pointer(&byteBuffer[0])))
		if iscsiErr == ERROR_SUCCESS {
			// Scan through the list of strings and append to our return array
			startIndex := 0
			for index, element := range byteBuffer {

				if element != 0 {
					// Keep looping until we hit a null terminator
					continue
				} else if startIndex == index {
					// Break out if we hit the double null terminator
					break
				}

				// Convert target from UTF16 to string and append to targets array
				target := syscall.UTF16ToString(byteBuffer[startIndex:index])
				targets = append(targets, target)

				// Move to the next string
				startIndex = index + 1
			}
		}
	}

	if iscsiErr != ERROR_SUCCESS {
		// If an unexpected error occurs, initialize error object and log failure
		err = syscall.Errno(iscsiErr)
		log.Error(logIscsiFailure, err.Error())
	} else {
		// Log the enumerated targets
		for index, target := range targets {
			log.Tracef("targets[%v]=%v", index, target)
		}
	}

	return targets, err
}
