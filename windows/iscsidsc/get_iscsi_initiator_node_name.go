// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package iscsidsc wraps the Windows iSCSI Discovery Library API
package iscsidsc

import (
	"syscall"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// GetIScsiInitiatorNodeName - Go wrapped Win32 API - GetIScsiInitiatorNodeNameW()
// https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-getiscsiinitiatornodenamew
func GetIScsiInitiatorNodeName() (initiatorNodeName string, err error) {
	log.Trace(">>>>> GetIScsiInitiatorNodeName")
	defer log.Trace("<<<<< GetIScsiInitiatorNodeName")

	// Allocate a data buffer large enough to hold the initiator name
	dataBuffer := make([]uint16, MAX_ISCSI_NAME_LEN+1)

	// Call the Win32 API
	if iscsiErr, _, _ := procGetIScsiInitiatorNodeNameW.Call(uintptr(unsafe.Pointer(&dataBuffer[0]))); iscsiErr == ERROR_SUCCESS {
		// Convert initiator name from UTF16 into a Go string
		initiatorNodeName = syscall.UTF16ToString(dataBuffer[:])
		log.Trace("initiatorNodeName=", initiatorNodeName)
	} else {
		// If an unexpected error occurs, initialize error object and log failure
		err = syscall.Errno(iscsiErr)
		log.Error(logIscsiFailure, err.Error())
	}

	return initiatorNodeName, err
}
