// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package iscsidsc wraps the Windows iSCSI Discovery Library API
package iscsidsc

import (
	"syscall"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// AddIScsiSendTargetPortal - Go wrapped Win32 API - AddIScsiSendTargetPortalW()
// https://docs.microsoft.com/en-us/windows/win32/api/iscsidsc/nf-iscsidsc-addiscsisendtargetportalw
func AddIScsiSendTargetPortal(initiatorInstance string, initiatorPortNumber uint32, address string) (err error) {
	log.Tracef(">>>>> AddIScsiSendTargetportal, initiatorInstance=%v, initiatorPortNumber=%v, address=%v", initiatorInstance, initiatorPortNumber, address)
	defer log.Traceln("<<<<< AddIScsiSendTargetPortal")

	// Convert initiatorInstance into a raw equivalent so that we can send it to the iSCSI API
	initiatorNameUTF16 := syscall.StringToUTF16(initiatorInstance)

	// Allocate and initialize an ISCSI_TARGET_PORTAL_RAW object
	targetPortal := ISCSI_TARGET_PORTAL{Address: address, Socket: 3260}
	targetPortalRaw := iscsiTargetPortalToRaw(&targetPortal)

	// Call the Win32 AddIScsiSendTargetPortalW API
	iscsiErr, _, _ := procAddIScsiSendTargetPortalW.Call(uintptr(unsafe.Pointer(&initiatorNameUTF16[0])), uintptr(initiatorPortNumber), uintptr(0), uintptr(0), uintptr(unsafe.Pointer(targetPortalRaw)))
	if iscsiErr != ERROR_SUCCESS {
		// If an unexpected error occurs, initialize error object and log failure
		err = syscall.Errno(iscsiErr)
		log.Errorln(logIscsiFailure, err.Error())
	}

	return err
}
