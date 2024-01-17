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

// GetDevicesForIScsiSession - Go wrapped Win32 API - GetDevicesForIScsiSessionW()
// https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-getdevicesforiscsisessionw
func GetDevicesForIScsiSession(sessionID ISCSI_UNIQUE_SESSION_ID) (devicesOnSession []*ISCSI_DEVICE_ON_SESSION, err error) {
	log.Trace(">>>>> GetDevicesForIScsiSession")
	defer log.Trace("<<<<< GetDevicesForIScsiSession")

	//
	// Note that the algorithm employed below is ported from C# GetDevicesForSession() including
	// the safety checks / workarounds previously employed.
	//

	var iscsiErr uintptr

	// NWT-3137.  After a session is newly connected, we're seeing requests to that
	// session fail with ISDSC_DEVICE_BUSY_ON_SESSION.  We're going to retry up to
	// 5 times with a one second delay between retries to give the iSCSI API time
	// to return the devices for the iSCSI session.
	const RetryCount int = 5
	const DelayBetweenRetries time.Duration = 1000 * time.Millisecond

	for i := 0; i < RetryCount; i++ {

		var devAllocCount, devCount uint32

		// Do an initial call to GetDevicesForIscsiSession to find the device count
		iscsiErr, _, _ = procGetDevicesForIScsiSessionW.Call(uintptr(unsafe.Pointer(&sessionID)), uintptr(unsafe.Pointer(&devAllocCount)), uintptr(0))
		devCount = devAllocCount

		if (iscsiErr == uintptr(syscall.ERROR_INSUFFICIENT_BUFFER)) && (devCount > 0) {

			// In the C# code, we used to allocate 2x the buffer size (see C# for details).  We are not
			// going to allocate 2x the buffer in the Go code because we are using the Unicode structure
			// versions intead of ASCII.  That means the structures are already nearly twice as large as
			// C# and we do not want to allocate 2x buffers.  Instead, we'll limit ourselves to 20
			// additional devices.
			devCount += 20

			// Allocate buffer for the session's device structures
			devicesOnSessionRaw := make([]ISCSI_DEVICE_ON_SESSION_RAW, devCount)

			// Call the Win32 GetDevicesForIScsiSessionW  API to find the required buffer size
			iscsiErr, _, _ = procGetDevicesForIScsiSessionW.Call(uintptr(unsafe.Pointer(&sessionID)), uintptr(unsafe.Pointer(&devCount)), uintptr(unsafe.Pointer(&devicesOnSessionRaw[0])))
			if iscsiErr == ERROR_SUCCESS {
				// Adjust array to the actual device count
				devicesOnSessionRaw = devicesOnSessionRaw[:devCount]

				// Convert ISCSI_DEVICE_ON_SESSION_RAW to array of ISCSI_DEVICE_ON_SESSION
				for _, deviceOnSessionRaw := range devicesOnSessionRaw {
					devicesOnSession = append(devicesOnSession, iscsiDeviceOnSessionFromRaw(&deviceOnSessionRaw))
				}
			} else if iscsiErr == uintptr(syscall.ERROR_INSUFFICIENT_BUFFER) {
				log.Errorf("GetDevicesForIScsiSession failed, iscsiErr=%v, InitialCount=%v, FinalCount=%v", iscsiErr, devAllocCount, devCount)
			}
		}

		// We have seen that alloc returns a count, but actual call returns 0 when the sessions are in "Reconnecting..." state.
		if iscsiErr == ERROR_SUCCESS && devAllocCount != 0 && len(devicesOnSession) == 0 {
			iscsiErr = ERROR_FAIL
		}

		if (iscsiErr == ISDSC_DEVICE_BUSY_ON_SESSION) || (iscsiErr == ISDSC_SESSION_BUSY) || (iscsiErr == uintptr(syscall.ERROR_INSUFFICIENT_BUFFER)) {
			log.Tracef("GetDevicesForIScsiSession status returned, iscsiErr=%v, attempt %v of %v", iscsiErr, i+1, RetryCount)

			// In PRT-312 and PRT-309, we know that a bug in Microsoft's iSCSI library can
			// return ERROR_INSUFFICIENT_BUFFER.  If this error is detected, we will not
			// sleep one second between retries.
			if iscsiErr != uintptr(syscall.ERROR_INSUFFICIENT_BUFFER) {
				time.Sleep(DelayBetweenRetries)
			}
		} else {
			// If we see any other failure, break out of our retry loop
			break
		}
	}

	if iscsiErr != ERROR_SUCCESS {
		// If an unexpected error occurs, initialize error object and log failure
		err = syscall.Errno(iscsiErr)
		log.Error(logIscsiFailure, err.Error())
	} else if len(devicesOnSession) > 0 {
		// Log the enumerated devices per session
		log.Tracef("SessionID=%x-%x", sessionID.AdapterUnique, sessionID.AdapterSpecific)
		for _, element := range devicesOnSession {
			log.Tracef("    ScsiAddress=%02X:%02X:%02X:%02X, DeviceNumber=%v, LegacyName=%v, TargetName=%v",
				element.ScsiAddress.PortNumber, element.ScsiAddress.PathID, element.ScsiAddress.TargetID, element.ScsiAddress.Lun,
				element.StorageDeviceNumber.DeviceNumber, element.LegacyName, element.TargetName)
		}
	}

	return devicesOnSession, err
}

// Internal function to convert an ISCSI_SESSION_INFO_RAW struct to ISCSI_SESSION_INFO
func iscsiDeviceOnSessionFromRaw(deviceOnSessionRaw *ISCSI_DEVICE_ON_SESSION_RAW) (deviceOnSession *ISCSI_DEVICE_ON_SESSION) {
	deviceOnSession = new(ISCSI_DEVICE_ON_SESSION)
	deviceOnSession.InitiatorName = syscall.UTF16ToString(deviceOnSessionRaw.InitiatorName[:])
	deviceOnSession.TargetName = syscall.UTF16ToString(deviceOnSessionRaw.TargetName[:])
	deviceOnSession.ScsiAddress = deviceOnSessionRaw.ScsiAddress
	deviceOnSession.DeviceInterfaceType = deviceOnSessionRaw.DeviceInterfaceType
	deviceOnSession.DeviceInterfaceName = syscall.UTF16ToString(deviceOnSessionRaw.DeviceInterfaceName[:])
	deviceOnSession.LegacyName = syscall.UTF16ToString(deviceOnSessionRaw.LegacyName[:])
	deviceOnSession.StorageDeviceNumber = deviceOnSessionRaw.StorageDeviceNumber
	deviceOnSession.DeviceInstance = deviceOnSessionRaw.DeviceInstance
	return deviceOnSession
}
