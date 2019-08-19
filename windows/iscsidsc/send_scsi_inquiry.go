// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package iscsidsc wraps the Windows iSCSI Discovery Library API
package iscsidsc

import (
	"syscall"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// SendScsiInquiry - Go wrapped Win32 API - SendScsiInquiry()
// https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-sendscsiinquiry
func SendScsiInquiry(sessionID ISCSI_UNIQUE_SESSION_ID, lun uint64, evpdCmddt uint8, pageCode uint8) (scsiStatus uint8, inquiryBuffer []uint8, senseBuffer []uint8, err error) {
	log.Tracef(">>>>> SendScsiInquiry, sessionID=%x-%x, lun=%v, evpdCmddt=%v, pageCode=%v", sessionID.AdapterUnique, sessionID.AdapterSpecific, lun, evpdCmddt, pageCode)

	// Set our Inquiry and Sense buffer sizes.  We never need more than 255 bytes from any Inquiry
	// request we make.  Rather than adding the burden to the caller to pass in an Inquiry size,
	// we'll just use the max inquiry length (back when Inquiry allocation length was one byte).
	const InquiryBufferSize = 255
	const SenseBufferSize = 18
	var inquiryBufferSize, senseBufferSize uint32 = InquiryBufferSize, SenseBufferSize
	inquiryBuffer, senseBuffer = make([]uint8, inquiryBufferSize), make([]uint8, senseBufferSize)

	// Issue the SCSI Inquiry command to the requested session / lun
	iscsiErr, _, _ := procSendScsiInquiry.Call(
		uintptr(unsafe.Pointer(&sessionID)),
		uintptr(lun),
		uintptr(evpdCmddt),
		uintptr(pageCode),
		uintptr(unsafe.Pointer(&scsiStatus)),
		uintptr(unsafe.Pointer(&inquiryBufferSize)),
		uintptr(unsafe.Pointer(&inquiryBuffer[0])),
		uintptr(unsafe.Pointer(&senseBufferSize)),
		uintptr(unsafe.Pointer(&senseBuffer[0])))

	// If a check condition was returned, set the Sense data and log the data
	if scsiStatus == SCSISTAT_CHECK_CONDITION {
		// Only return the data that the iSCSI initiator claims was returned by the target
		senseBuffer = senseBuffer[:senseBufferSize]
		logTraceHexDump(senseBuffer, "Sense Data")
	} else {
		// Empty sense buffer if no check condition
		senseBuffer = nil
	}

	if iscsiErr != ERROR_SUCCESS {
		// If an unexpected error occurs, initialize error object and log failure
		err = syscall.Errno(iscsiErr)
		log.Error(logIscsiFailure, err.Error())

		// Clear the return buffer
		inquiryBuffer = nil
	} else {
		// Only return the data that the iSCSI initiator claims was returned by the target
		inquiryBuffer = inquiryBuffer[:inquiryBufferSize]

		// Log the Inquiry data
		logTraceHexDump(inquiryBuffer, "Inquiry Data")
	}

	// Log the SCSI status
	log.Tracef("<<<<< SendScsiInquiry, scsiStatus=%v", scsiStatus)

	// Return SCSI status, Inquiry buffer, Sense buffer, and error
	return scsiStatus, inquiryBuffer, senseBuffer, err
}
