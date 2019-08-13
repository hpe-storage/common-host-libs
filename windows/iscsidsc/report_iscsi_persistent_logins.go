// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package iscsidsc wraps the Windows iSCSI Discovery Library API
package iscsidsc

import (
	"syscall"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// ReportIScsiPersistentLogins - Go wrapped Win32 API - ReportIScsiPersistentLoginsW()
// https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-reportiscsipersistentloginsw
func ReportIScsiPersistentLogins() (persistentLogins []*PERSISTENT_ISCSI_LOGIN_INFO, err error) {
	log.Trace(">>>>> ReportIScsiPersistentLogins")
	defer log.Trace("<<<<< ReportIScsiPersistentLogins")

	// Determine the persistent login count on this host
	var count, bufferSizeNeeded uint32
	iscsiErr, _, _ := procReportIScsiPersistentLoginsW.Call(uintptr(unsafe.Pointer(&count)), uintptr(0), uintptr(unsafe.Pointer(&bufferSizeNeeded)))
	if iscsiErr == uintptr(syscall.ERROR_INSUFFICIENT_BUFFER) {

		// We sometimes see this routine return ERROR_INSUFFICIENT_BUFFER with no buffer size needed.
		// Should that occur, we return no error to the caller.
		if bufferSizeNeeded == 0 {
			iscsiErr = ERROR_SUCCESS
		} else {
			// Allocate memory to hold the list of portals.
			// The extra size is to accomodate new persistent connections between the get size and actual call
			bufferSizeNeeded += 8192
			dataBuffer := make([]uint8, bufferSizeNeeded)

			// Get the list of persistent logins from the iSCSI initiator.
			iscsiErr, _, _ = procReportIScsiPersistentLoginsW.Call(uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&dataBuffer[0])), uintptr(unsafe.Pointer(&bufferSizeNeeded)))
			if iscsiErr == ERROR_SUCCESS {

				// Convert the raw buffer into an array of PERSISTENT_ISCSI_LOGIN_INFO_RAW structs
				persistentLoginsRaw := (*[1024]PERSISTENT_ISCSI_LOGIN_INFO_RAW)(unsafe.Pointer(&dataBuffer[0]))[:count]

				// Convert each PERSISTENT_ISCSI_LOGIN_INFO_RAW to PERSISTENT_ISCSI_LOGIN_INFO and append to our array
				for _, persistentLoginRaw := range persistentLoginsRaw {
					persistentLogins = append(persistentLogins, persistentIscsiLoginInfoFromRaw(&persistentLoginRaw))
				}
			}
		}
	}

	if iscsiErr != ERROR_SUCCESS {
		// If an unexpected error occurs, initialize error object and log failure
		err = syscall.Errno(iscsiErr)
		log.Error(logIscsiFailure, err.Error())
	} else {
		// Log the enumerated persistent logins
		for index, persistentLogin := range persistentLogins {
			log.Tracef("persistentLogins[%v], TargetName=%v, InitiatorPortNumber=%v, TargetPortal.Address=%v, Mapping=%v, InformationSpecified=%v, LoginFlags=%v",
				index, persistentLogin.TargetName,
				persistentLogin.InitiatorPortNumber, persistentLogin.TargetPortal.Address, persistentLogin.Mappings != nil,
				persistentLogin.LoginOptions.InformationSpecified, persistentLogin.LoginOptions.LoginFlags)
		}
	}

	return persistentLogins, err
}

// Internal function to convert a PERSISTENT_ISCSI_LOGIN_INFO_RAW struct to PERSISTENT_ISCSI_LOGIN_INFO
func persistentIscsiLoginInfoFromRaw(persistentLoginRaw *PERSISTENT_ISCSI_LOGIN_INFO_RAW) (persistentLogin *PERSISTENT_ISCSI_LOGIN_INFO) {
	persistentLogin = new(PERSISTENT_ISCSI_LOGIN_INFO)
	persistentLogin.TargetName = syscall.UTF16ToString(persistentLoginRaw.TargetName[:])
	persistentLogin.IsInformationalSession = persistentLoginRaw.IsInformationalSession
	persistentLogin.InitiatorInstance = syscall.UTF16ToString(persistentLoginRaw.InitiatorInstance[:])
	persistentLogin.InitiatorPortNumber = persistentLoginRaw.InitiatorPortNumber
	persistentLogin.TargetPortal.SymbolicName = syscall.UTF16ToString(persistentLoginRaw.TargetPortal.SymbolicName[:])
	persistentLogin.TargetPortal.Address = syscall.UTF16ToString(persistentLoginRaw.TargetPortal.Address[:])
	persistentLogin.TargetPortal.Socket = persistentLoginRaw.TargetPortal.Socket
	persistentLogin.SecurityFlags = persistentLoginRaw.SecurityFlags
	if persistentLoginRaw.Mappings != nil {
		persistentLogin.Mappings = iscsiTargetMappingFromRaw(persistentLoginRaw.Mappings)
	}
	persistentLogin.LoginOptions = persistentLoginRaw.LoginOptions
	return persistentLogin
}
