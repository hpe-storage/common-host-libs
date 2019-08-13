// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package iscsidsc wraps the Windows iSCSI Discovery Library API
package iscsidsc

import (
	"strings"
	"syscall"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// RemoveIScsiPersistentTarget - Go wrapped Win32 API - RemoveIScsiPersistentTargetW()
// https://docs.microsoft.com/is-is/windows/desktop/api/iscsidsc/nf-iscsidsc-removeiscsipersistenttargetw
func RemoveIScsiPersistentTarget(initiatorInstance string, initiatorPortNumber uint32, targetName string, targetPortal ISCSI_TARGET_PORTAL) (err error) {
	log.Tracef(">>>>> RemoveIScsiPersistentTarget, initiatorInstance=%v, initiatorPortNumber=%v, targetName=%v", initiatorInstance, initiatorPortNumber, targetName)
	defer log.Trace("<<<<< RemoveIScsiPersistentTarget")

	// Convert initiatorInstance, targetName, and targetPortal into raw equivalents so that we can
	// send them to the iSCSI API.
	initiatorNameUTF16 := syscall.StringToUTF16(initiatorInstance)
	targetNameUTF16 := syscall.StringToUTF16(targetName)
	targetPortalRaw := iscsiTargetPortalToRaw(&targetPortal)

	// Call the Win32 RemoveIScsiPersistentTargetW API
	iscsiErr, _, _ := procRemoveIScsiPersistentTargetW.Call(uintptr(unsafe.Pointer(&initiatorNameUTF16[0])), uintptr(initiatorPortNumber), uintptr(unsafe.Pointer(&targetNameUTF16[0])), uintptr(unsafe.Pointer(targetPortalRaw)))
	if iscsiErr != ERROR_SUCCESS {
		// If an unexpected error occurs, initialize error object and log failure
		err = syscall.Errno(iscsiErr)
		log.Error(logIscsiFailure, err.Error())
	}

	return err
}

// RemoveIScsiPersistentTargetAll is a extended wrapper around the RemoveIScsiPersistentTarget API.
// What it does is remove *all* persistent logins for the specified target.
func RemoveIScsiPersistentTargetAll(targetName string) (err error) {
	log.Tracef(">>>>> RemoveIScsiPersistentTargetAll, targetName=%v", targetName)
	defer log.Trace("<<<<< RemoveIScsiPersistentTargetAll")

	// Start by querying all the iSCSI persistent logins on this host
	persistentLogins, err := ReportIScsiPersistentLogins()
	if err == nil {
		// Loop through the enumerated persistent logins
		for _, persistentLogin := range persistentLogins {
			// If it isn't a target match (case insensitive comparison), skip it
			if !strings.EqualFold(persistentLogin.TargetName, targetName) {
				continue
			}

			// Remove this persistent login
			errTemp := RemoveIScsiPersistentTarget(persistentLogin.InitiatorInstance, persistentLogin.InitiatorPortNumber, persistentLogin.TargetName, persistentLogin.TargetPortal)

			// We keep trying to remove all persistent logins for our target but only return
			// the first failure (if any) to the caller.
			if (err == nil) && (errTemp != nil) {
				err = errTemp
			}
		}
	}

	return err
}
