// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package iscsidsc wraps the Windows iSCSI Discovery Library API
package iscsidsc

import (
	"net"
	"syscall"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// ReportIScsiTargetPortals - Go wrapped Win32 API - ReportIScsiTargetPortalsW()
// https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-reportiscsitargetportalsw
func ReportIScsiTargetPortals(initiatorName string, targetName string, ipv4Only bool) (targetPortals []*ISCSI_TARGET_PORTAL, err error) {
	log.Tracef(">>>>> ReportIScsiTargetPortals, initiatorName=%v, targetName=%v, ipv4Only=%v", initiatorName, targetName, ipv4Only)
	defer log.Trace("<<<<< ReportIScsiTargetPortals")

	// Get UTF16 versions of initiatorName and targetName
	initiatorNameUTF16 := syscall.StringToUTF16(initiatorName)
	targetNameUTF16 := syscall.StringToUTF16(targetName)

	// Determine the buffer size we'll need to allocate
	var elementCount uint32
	iscsiErr, _, _ := procReportIScsiTargetPortalsW.Call(uintptr(unsafe.Pointer(&initiatorNameUTF16[0])), uintptr(unsafe.Pointer(&targetNameUTF16[0])), uintptr(0), uintptr(unsafe.Pointer(&elementCount)), uintptr(0))
	if (iscsiErr == uintptr(syscall.ERROR_INSUFFICIENT_BUFFER)) && (elementCount > 0) {

		// Allocate a data buffer large enough to hold all the target portals and resubmit the request
		targetPortalsRaw := make([]ISCSI_TARGET_PORTAL_RAW, elementCount)
		iscsiErr, _, _ = procReportIScsiTargetPortalsW.Call(uintptr(0), uintptr(unsafe.Pointer(&targetNameUTF16[0])), uintptr(0), uintptr(unsafe.Pointer(&elementCount)), uintptr(unsafe.Pointer(&targetPortalsRaw[0])))
		if iscsiErr == ERROR_SUCCESS {

			// Adjust array to the actual target portal count
			targetPortalsRaw = targetPortalsRaw[:elementCount]

			// Loop through each enumerated target portal
			for _, targetPortalRaw := range targetPortalsRaw {

				// Get the ipv4 target portal address
				ipv4String := syscall.UTF16ToString(targetPortalRaw.Address[:])
				ip := net.ParseIP(ipv4String)

				if (ipv4Only == false) || (ip.To4() != nil) {
					// Append target portal to our return array
					targetPortals = append(targetPortals, iscsiTargetPortalFromRaw(&targetPortalRaw))
				}
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
			log.Tracef("targetPortals[%v], Address=%v, Socket=%v", index, targetPortal.Address, targetPortal.Socket)
		}
	}

	return targetPortals, err
}

// Internal function to convert an ISCSI_TARGET_PORTAL_RAW struct to ISCSI_TARGET_PORTAL
func iscsiTargetPortalFromRaw(targetPortalRaw *ISCSI_TARGET_PORTAL_RAW) (targetPortal *ISCSI_TARGET_PORTAL) {
	targetPortal = new(ISCSI_TARGET_PORTAL)
	targetPortal.SymbolicName = syscall.UTF16ToString(targetPortalRaw.SymbolicName[:])
	targetPortal.Address = syscall.UTF16ToString(targetPortalRaw.Address[:])
	targetPortal.Socket = targetPortalRaw.Socket
	return targetPortal
}

// Internal function to convert an ISCSI_TARGET_PORTAL struct to ISCSI_TARGET_PORTAL_RAW
func iscsiTargetPortalToRaw(targetPortal *ISCSI_TARGET_PORTAL) (targetPortalRaw *ISCSI_TARGET_PORTAL_RAW) {
	targetPortalRaw = new(ISCSI_TARGET_PORTAL_RAW)
	for i, c := range targetPortal.SymbolicName {
		targetPortalRaw.SymbolicName[i] = uint16(c)
	}
	for i, c := range targetPortal.Address {
		targetPortalRaw.Address[i] = uint16(c)
	}
	targetPortalRaw.Socket = targetPortal.Socket
	return targetPortalRaw
}
