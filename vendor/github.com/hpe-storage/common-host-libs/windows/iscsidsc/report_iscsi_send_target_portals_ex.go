// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package iscsidsc wraps the Windows iSCSI Discovery Library API
package iscsidsc

import (
	"syscall"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// ReportIScsiSendTargetPortalsEx - Go wrapped Win32 API - ReportIScsiSendTargetPortalsExW()
// https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-reportiscsisendtargetportalsexw
func ReportIScsiSendTargetPortalsEx() (targetPortals []*ISCSI_TARGET_PORTAL_INFO_EX, err error) {
	log.Trace(">>>>> ReportIScsiSendTargetPortalsEx")
	defer log.Trace("<<<<< ReportIScsiSendTargetPortalsEx")

	// Determine the target portal count on this host
	var portalCount, portalInfoSize uint32
	iscsiErr, _, _ := procReportIScsiSendTargetPortalsExW.Call(uintptr(unsafe.Pointer(&portalCount)), uintptr(unsafe.Pointer(&portalInfoSize)), uintptr(0))
	if (iscsiErr == uintptr(syscall.ERROR_INSUFFICIENT_BUFFER)) && (portalCount > 0) && (portalInfoSize > 0) {

		// Allocate a data buffer large enough to hold all the target portals and resubmit the request
		dataBuffer := make([]uint8, portalInfoSize)
		iscsiErr, _, _ = procReportIScsiSendTargetPortalsExW.Call(uintptr(unsafe.Pointer(&portalCount)), uintptr(unsafe.Pointer(&portalInfoSize)), uintptr(unsafe.Pointer(&dataBuffer[0])))
		if iscsiErr == ERROR_SUCCESS {

			// Convert the raw buffer into an array of ISCSI_TARGET_PORTAL_INFO_EX_RAW structs
			portals := (*[1024]ISCSI_TARGET_PORTAL_INFO_EX_RAW)(unsafe.Pointer(&dataBuffer[0]))[:portalCount]

			// Loop through and append each target portal to array
			for _, portal := range portals {
				targetPortals = append(targetPortals, iscsiTargetPortalInfoExFromRaw(&portal))
			}
		}
	}

	if iscsiErr != ERROR_SUCCESS {
		// If an unexpected error occurs, initialize error object and log failure
		err = syscall.Errno(iscsiErr)
		log.Error(logIscsiFailure, err.Error())
	} else {
		// Log the enumerated target portals
		for index, targetPortal := range targetPortals {
			log.Tracef("targetPortals[%v], Address=%v, InitiatorPortNumber=%v", index, targetPortal.Address, int32(targetPortal.InitiatorPortNumber))
		}
	}

	return targetPortals, err
}

// Internal function to convert an ISCSI_TARGET_PORTAL_INFO_EX_RAW struct to ISCSI_TARGET_PORTAL_INFO_EX
func iscsiTargetPortalInfoExFromRaw(targetPortalRaw *ISCSI_TARGET_PORTAL_INFO_EX_RAW) (targetPortal *ISCSI_TARGET_PORTAL_INFO_EX) {
	targetPortal = new(ISCSI_TARGET_PORTAL_INFO_EX)
	targetPortal.InitiatorName = syscall.UTF16ToString(targetPortalRaw.InitiatorName[:])
	targetPortal.InitiatorPortNumber = targetPortalRaw.InitiatorPortNumber
	targetPortal.SymbolicName = syscall.UTF16ToString(targetPortalRaw.SymbolicName[:])
	targetPortal.Address = syscall.UTF16ToString(targetPortalRaw.Address[:])
	targetPortal.Socket = targetPortalRaw.Socket
	targetPortal.SecurityFlags = targetPortalRaw.SecurityFlags
	targetPortal.LoginOptions = targetPortalRaw.LoginOptions
	return targetPortal
}
