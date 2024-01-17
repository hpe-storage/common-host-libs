// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package iscsidsc wraps the Windows iSCSI Discovery Library API
package iscsidsc

import (
	"syscall"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// ReportActiveIScsiTargetMappings - Go wrapped Win32 API - ReportActiveIScsiTargetMappingsW()
// https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-reportactiveiscsitargetmappingsw
func ReportActiveIScsiTargetMappings() (targetMappings []*ISCSI_TARGET_MAPPING, err error) {
	log.Trace(">>>>> ReportActiveIScsiTargetMappings")
	defer log.Trace("<<<<< ReportActiveIScsiTargetMappings")

	// Call ReportActiveIScsiTargetMappingsW  to find the required buffer size
	var bufferSize, mappingCount uint32
	iscsiErr, _, _ := procReportActiveIScsiTargetMappingsW.Call(uintptr(unsafe.Pointer(&bufferSize)), uintptr(unsafe.Pointer(&mappingCount)), uintptr(0))

	if (iscsiErr == uintptr(syscall.ERROR_INSUFFICIENT_BUFFER)) && (mappingCount > 0) {

		// Allocate the necessary buffer size and retry the request
		dataBuffer := make([]uint8, bufferSize)
		iscsiErr, _, _ = procReportActiveIScsiTargetMappingsW.Call(uintptr(unsafe.Pointer(&bufferSize)), uintptr(unsafe.Pointer(&mappingCount)), uintptr(unsafe.Pointer(&dataBuffer[0])))
		if iscsiErr == ERROR_SUCCESS {

			// Convert the raw buffer into an array of IscsiTargetMappingRaw structs
			iscsiTargetMappingsRaw := (*[1024]ISCSI_TARGET_MAPPING_RAW)(unsafe.Pointer(&dataBuffer[0]))[:mappingCount]

			// Loop through and append each enumerated mapping
			for _, iscsiTargetMappingRaw := range iscsiTargetMappingsRaw {
				targetMappings = append(targetMappings, iscsiTargetMappingFromRaw(&iscsiTargetMappingRaw))
			}
		}
	}

	if iscsiErr != ERROR_SUCCESS {
		// If an unexpected error occurs, initialize error object and log failure
		err = syscall.Errno(iscsiErr)
		log.Error(logIscsiFailure, err.Error())
	} else {
		// Log the target mappings
		for index, targetMapping := range targetMappings {
			log.Tracef("targetMappings[%v], TargetName=%v, OSDeviceName=%v, SessionId=%x-%x, OSBusNumber=%02Xh, OSTargetNumber=%02Xh",
				index, targetMapping.TargetName, targetMapping.OSDeviceName, targetMapping.SessionId.AdapterUnique, targetMapping.SessionId.AdapterSpecific,
				targetMapping.OSBusNumber, targetMapping.OSTargetNumber)
			for index2, lun := range targetMapping.LUNList {
				log.Tracef("    LUNList[%v], OSLUN=%02Xh, TargetLUN=%04Xh", index2, lun.OSLUN, lun.TargetLUN)
			}

		}
	}

	return targetMappings, err
}

// Internal function to convert an ISCSI_TARGET_MAPPING_RAW struct to ISCSI_TARGET_MAPPING
func iscsiTargetMappingFromRaw(targetMappingRaw *ISCSI_TARGET_MAPPING_RAW) (targetMapping *ISCSI_TARGET_MAPPING) {
	targetMapping = new(ISCSI_TARGET_MAPPING)

	targetMapping.InitiatorName = syscall.UTF16ToString(targetMappingRaw.InitiatorName[:])
	targetMapping.TargetName = syscall.UTF16ToString(targetMappingRaw.TargetName[:])
	targetMapping.OSDeviceName = syscall.UTF16ToString(targetMappingRaw.OSDeviceName[:])
	targetMapping.SessionId = targetMappingRaw.SessionId
	targetMapping.OSBusNumber = targetMappingRaw.OSBusNumber
	targetMapping.OSTargetNumber = targetMappingRaw.OSTargetNumber

	// Loop through the LUN List
	iscsiLuns := (*[1024]SCSI_LUN_LIST)(unsafe.Pointer(targetMappingRaw.LUNList))[:targetMappingRaw.LUNCount]
	for _, iscsiLun := range iscsiLuns {
		targetMapping.LUNList = append(targetMapping.LUNList, iscsiLun)
	}

	return targetMapping
}
