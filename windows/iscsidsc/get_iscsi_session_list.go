// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package iscsidsc wraps the Windows iSCSI Discovery Library API
package iscsidsc

import (
	"syscall"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// GetIscsiSessionList - Go wrapped Win32 API - GetIScsiSessionListW()
// https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-getiscsisessionlistw
func GetIscsiSessionList() (iscsiSessions []*ISCSI_SESSION_INFO, err error) {
	log.Trace(">>>>> GetIscsiSessionList")
	defer log.Trace("<<<<< GetIscsiSessionList")

	// NWT-3303.  We're seeing Win32 GetIscsiSessionList return ERROR_INSUFFICIENT_BUFFER during
	// an array failover. The first call to GetIScsiSessionListW returns ERROR_INSUFFICIENT_BUFFER
	// with the required buffer size.  By the time the second call is made, with the required
	// buffer size, the buffer size requirements have now increased.  In order to avoid this
	// scenario, we'll retry up to 5 times until ERROR_INSUFFICIENT_BUFFER is no longer returned.

	const RetryCount int = 5
	iscsiErr := uintptr(syscall.ERROR_INSUFFICIENT_BUFFER)
	for retry := 0; (retry < RetryCount) && (iscsiErr == uintptr(syscall.ERROR_INSUFFICIENT_BUFFER)); retry++ {

		// Call the Win32 GetIScsiSessionListW API to find the required buffer size
		var bufferSizeNeeded, sessionCount uint32
		iscsiErr, _, _ = procGetIScsiSessionListW.Call(uintptr(unsafe.Pointer(&bufferSizeNeeded)), uintptr(unsafe.Pointer(&sessionCount)), uintptr(0))

		if (iscsiErr == uintptr(syscall.ERROR_INSUFFICIENT_BUFFER)) && ((sessionCount > 0) || (bufferSizeNeeded > 0)) {

			// NWT-3303.  Additional safety measure we'll add to deal with NWT-3303 is to bump
			// up the size of the buffer we'll allocate.  That way, in the unlikely event the
			// size has grown from the first invocation to the second invocation, we'll have
			// additional buffer space to try and compensate for that.  This workaround,
			// coupled with the retry loop, helps eliminate NWT-3303 from occurring.
			bufferSizeNeeded += (16 * 1024)

			// Allocate the necessary buffer size and retry the request
			dataBuffer := make([]uint8, bufferSizeNeeded)
			iscsiErr, _, _ = procGetIScsiSessionListW.Call(uintptr(unsafe.Pointer(&bufferSizeNeeded)), uintptr(unsafe.Pointer(&sessionCount)), uintptr(unsafe.Pointer(&dataBuffer[0])))
			if iscsiErr == ERROR_SUCCESS {

				// Convert the raw buffer into an array of ISCSI_SESSION_INFO_RAW structs
				iscsiSessionsRaw := (*[1024]ISCSI_SESSION_INFO_RAW)(unsafe.Pointer(&dataBuffer[0]))[:sessionCount]

				// Loop through each enumerated session and append to our session array
				for _, iscsiSessionRaw := range iscsiSessionsRaw {
					iscsiSessions = append(iscsiSessions, iscsiSessionInfoFromRaw(&iscsiSessionRaw))
				}
			}
		}
	}

	if iscsiErr != ERROR_SUCCESS {
		// If an unexpected error occurs, initialize error object and log failure
		err = syscall.Errno(iscsiErr)
		log.Error(logIscsiFailure, err.Error())
	} else {
		// Log the sessions
		for index, iscsiSession := range iscsiSessions {
			log.Tracef("iscsiSession[%v], SessionId=%x-%x, TargetName=%v, ConnectionCount=%v",
				index, iscsiSession.SessionID.AdapterUnique, iscsiSession.SessionID.AdapterSpecific,
				iscsiSession.TargetName, len(iscsiSession.Connections))
			// Log the connections per session
			for _, iscsiConnection := range iscsiSession.Connections {
				log.Tracef("    ConnectionIdId=%x-%x, InitiatorAddress=%v, InitiatorSocket=%v, TargetAddress=%v, TargetSocket=%v",
					iscsiConnection.ConnectionID.AdapterUnique, iscsiConnection.ConnectionID.AdapterSpecific,
					iscsiConnection.InitiatorAddress, iscsiConnection.InitiatorSocket,
					iscsiConnection.TargetAddress, iscsiConnection.TargetSocket)
			}
		}
	}

	return iscsiSessions, err
}

// Internal function to convert an ISCSI_SESSION_INFO_RAW struct to ISCSI_SESSION_INFO
func iscsiSessionInfoFromRaw(iscsiSessionRaw *ISCSI_SESSION_INFO_RAW) (iscsiSession *ISCSI_SESSION_INFO) {

	// Convert each ISCSI_SESSION_INFO_RAW to ISCSI_SESSION_INFO
	iscsiSession = new(ISCSI_SESSION_INFO)
	iscsiSession.SessionID = iscsiSessionRaw.SessionID
	iscsiSession.InitiatorName = safeUTF16PtrToString(iscsiSessionRaw.InitiatorName)
	iscsiSession.TargetNodeName = safeUTF16PtrToString(iscsiSessionRaw.TargetNodeName)
	iscsiSession.TargetName = safeUTF16PtrToString(iscsiSessionRaw.TargetName)
	iscsiSession.ISID = iscsiSessionRaw.ISID
	iscsiSession.TSID = iscsiSessionRaw.TSID

	// Loop through the session's connections
	iscsiConnectionsRaw := (*[1024]ISCSI_CONNECTION_INFO_RAW)(unsafe.Pointer(iscsiSessionRaw.Connections))[:iscsiSessionRaw.ConnectionCount]
	for _, iscsiConnectionRaw := range iscsiConnectionsRaw {

		// Convert each ISCSI_CONNECTION_INFO_RAW to ISCSI_CONNECTION_INFO
		iscsiConnection := new(ISCSI_CONNECTION_INFO)
		iscsiConnection.ConnectionID = iscsiConnectionRaw.ConnectionID
		iscsiConnection.InitiatorAddress = safeUTF16PtrToString(iscsiConnectionRaw.InitiatorAddress)
		iscsiConnection.TargetAddress = safeUTF16PtrToString(iscsiConnectionRaw.TargetAddress)
		iscsiConnection.InitiatorSocket = iscsiConnectionRaw.InitiatorSocket
		iscsiConnection.TargetSocket = iscsiConnectionRaw.TargetSocket
		iscsiConnection.CID = iscsiConnectionRaw.CID

		// Append connection to connection array
		iscsiSession.Connections = append(iscsiSession.Connections, iscsiConnection)
	}

	return iscsiSession
}
