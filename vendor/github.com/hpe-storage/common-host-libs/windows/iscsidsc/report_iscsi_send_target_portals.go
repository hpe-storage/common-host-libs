// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package iscsidsc wraps the Windows iSCSI Discovery Library API
package iscsidsc

import (
	"syscall"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// ReportIScsiSendTargetPortals - Go wrapped Win32 API - ReportIScsiSendTargetPortalsW()
// https://docs.microsoft.com/is-is/windows/desktop/api/iscsidsc/nf-iscsidsc-reportiscsisendtargetportalsw
func ReportIScsiSendTargetPortals() (targetPortals []*ISCSI_TARGET_PORTAL_INFO, err error) {
	log.Trace(">>>>> ReportIScsiSendTargetPortals")
	defer log.Trace("<<<<< ReportIScsiSendTargetPortals")

	// Determine the target portal count on this host
	var portalCount uint32
	iscsiErr, _, _ := procReportIScsiSendTargetPortalsW.Call(uintptr(unsafe.Pointer(&portalCount)), uintptr(0))
	if (iscsiErr == uintptr(syscall.ERROR_INSUFFICIENT_BUFFER)) && (portalCount > 0) {

		// Allocate a data buffer large enough to hold all the target portals and resubmit the request
		portals := make([]ISCSI_TARGET_PORTAL_INFO_RAW, portalCount)
		iscsiErr, _, _ = procReportIScsiSendTargetPortalsW.Call(uintptr(unsafe.Pointer(&portalCount)), uintptr(unsafe.Pointer(&portals[0])))
		if iscsiErr == ERROR_SUCCESS {

			// Adjust array to the actual portal count
			portals = portals[:portalCount]

			// Loop through and append each target portal to array
			for _, portal := range portals {
				targetPortals = append(targetPortals, iscsiTargetPortalInfoFromRaw(&portal))
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

// Internal function to convert an ISCSI_TARGET_PORTAL_INFO_RAW struct to ISCSI_TARGET_PORTAL_INFO
func iscsiTargetPortalInfoFromRaw(targetPortalRaw *ISCSI_TARGET_PORTAL_INFO_RAW) (targetPortal *ISCSI_TARGET_PORTAL_INFO) {
	targetPortal = new(ISCSI_TARGET_PORTAL_INFO)
	targetPortal.InitiatorName = syscall.UTF16ToString(targetPortalRaw.InitiatorName[:])
	targetPortal.InitiatorPortNumber = targetPortalRaw.InitiatorPortNumber
	targetPortal.SymbolicName = syscall.UTF16ToString(targetPortalRaw.SymbolicName[:])
	targetPortal.Address = syscall.UTF16ToString(targetPortalRaw.Address[:])
	targetPortal.Socket = targetPortalRaw.Socket
	return targetPortal
}
