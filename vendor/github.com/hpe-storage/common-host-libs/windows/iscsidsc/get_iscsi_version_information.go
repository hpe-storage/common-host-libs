// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package iscsidsc wraps the Windows iSCSI Discovery Library API
package iscsidsc

import (
	"syscall"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// GetIScsiVersionInformation - Go wrapped Win32 API - GetIScsiVersionInformation()
// https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-getiscsiversioninformation
func GetIScsiVersionInformation() (versionInfo ISCSI_VERSION_INFO, err error) {
	log.Trace(">>>>> GetIScsiVersionInformation")
	defer log.Trace("<<<<< GetIScsiVersionInformation")

	// Call the Win32 API
	if iscsiErr, _, _ := procGetIScsiVersionInformation.Call(uintptr(unsafe.Pointer(&versionInfo))); iscsiErr == ERROR_SUCCESS {
		// Log the API results
		log.Tracef("versionInfo=%v.%v.%v", versionInfo.MajorVersion, versionInfo.MinorVersion, versionInfo.BuildNumber)
	} else {
		// If an unexpected error occurs, initialize error object and log failure
		err = syscall.Errno(iscsiErr)
		log.Error(logIscsiFailure, err.Error())
	}

	return versionInfo, err
}
