// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package iscsidsc wraps the Windows iSCSI Discovery Library API
package iscsidsc

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// LogoutIScsiTarget - Go wrapped Win32 API - LogoutIScsiTarget()
// https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-logoutiscsitarget
func LogoutIScsiTarget(sessionID ISCSI_UNIQUE_SESSION_ID) (err error) {
	log.Tracef(">>>>> LogoutIScsiTarget, sessionID=%x-%x", sessionID.AdapterUnique, sessionID.AdapterSpecific)
	defer log.Trace("<<<<< LogoutIScsiTarget")

	// Call the Win32 LogoutIScsiTarget API to logout the session.
	// In our C# code, we would terminate the thread if it hung more than 60 seconds.  We can't simply
	// move this code into a Go routine and then try to kill it.  A Go routine is not a thread and thus
	// cannot be killed.  Also, since it's hung in an OS API call, we can't even signal the thread to
	// try and abort.  The timeout may be longer than 60 seconds but eventually the Win32 API will
	// return.  In the unlikely event that this API proves troublesome, we could move to executing a
	// Windows cmdlet to logout the target and then abort the process if the cmdlet gets hung.  We
	// also have a WMI logout method that we can utilize.
	iscsiErr, _, _ := procLogoutIScsiTarget.Call(uintptr(unsafe.Pointer(&sessionID)))

	if iscsiErr != ERROR_SUCCESS {
		// If an unexpected error occurs, initialize error object and log failure
		err = syscall.Errno(iscsiErr)
		log.Error(logIscsiFailure, err.Error())
	}

	return err
}

// LogoutIScsiTargetAll enumerates all the target's sessions and then tries to log them all out.
// The removePersistentTarget flag can be set to true if the caller also wants any persistent
// logins to be removed as well.
func LogoutIScsiTargetAll(targetName string, removePersistentTarget bool) (err error) {
	log.Tracef(">>>>> LogoutIScsiTargetAll, targetName=%v, removePersistentTarget=%v", targetName, removePersistentTarget)
	defer log.Trace("<<<<< LogoutIScsiTargetAll")

	// Remove persistent logins?
	if removePersistentTarget {
		err = RemoveIScsiPersistentTargetAll(targetName)
	}

	// Even if RemoveIScsiPersistentTargetAll failed, we're still going to proceed with
	// the logout of the targets.  We'll return whatever error occurs first.

	// Query the current iSCSI sessions
	iscsiSessions, errSessionQuery := GetIscsiSessionList()

	if errSessionQuery != nil {
		// If unable to query the sessions, update the return error (unless it's already set)
		if err != nil {
			err = errSessionQuery
		}
	} else {

		// For each session, we'll retry up to 10 times to logout the session
		maxLogoutRetryCount := 10

		// For each session, we'll give it no more than 60 seconds to complete
		// before moving to the next session.
		const maxDurationPerSessionLogout time.Duration = 60 * time.Second

		// If we're spending more than 2 minutes in this routine, we'll fail
		// the request even if we have not yet completed all the logouts.
		const maxDurationTotal time.Duration = 2 * time.Minute

		// We're going to keep track of each session's logout error (if any)
		sessionErrors := make(map[string]error)

		// Time that we started the logout process for all sessions
		startLogoutTarget := time.Now()

		// Flag is set to true only when this routine's timeout detection checks have hit a timeout
		timeoutDetection := false

		// Loop through each iSCSI session
		for _, iscsiSession := range iscsiSessions {

			// Break out of our session logout loop if a timeout has been detected
			if timeoutDetection {
				break
			}

			// If this session isn't for our target (case-insensitive comparison required), then
			// skip this session.
			if !strings.EqualFold(iscsiSession.TargetName, targetName) {
				continue
			}

			// Set a "nil" entry for our session.  We want to keep track of the last logout session
			// failure in case we want to return that to the caller.
			sessionIDString := fmt.Sprintf("%x-%x", iscsiSession.SessionID.AdapterUnique, iscsiSession.SessionID.AdapterSpecific)
			sessionErrors[sessionIDString] = nil

			// Time that we started the logout process for the current session
			startLogoutSession := time.Now()

			// Keep retrying up to "maxLogoutRetryCount" times
			for retry := 0; retry < maxLogoutRetryCount; retry++ {

				// How how time has elapsed since we started the logout process for our target?
				elapsedLogoutTarget := time.Since(startLogoutTarget)

				// How how time has elapsed since we started the logout process for the current session?
				elapsedLogoutSession := time.Since(startLogoutSession)

				// If we're past our alloted timeouts, set timeoutDetection flag and break out of loop
				if (elapsedLogoutSession >= maxDurationPerSessionLogout) || (elapsedLogoutSession >= maxDurationTotal) {
					log.Tracef("Logout timeout expired, elapsedLogoutSession=%v, elapsedLogoutTarget=%v", elapsedLogoutSession, elapsedLogoutTarget)
					timeoutDetection = true
					break
				}

				// Logout of the iSCSI session; update our session error map
				sessionErrors[sessionIDString] = LogoutIScsiTarget(iscsiSession.SessionID)

				if sessionErrors[sessionIDString] == nil {
					// Clear error map entry and break out of loop if logout successful
					break
				} else {
					// If we get a BUSY status, sleep 500 msec before retrying
					switch sessionErrors[sessionIDString] {
					case syscall.Errno(ISDSC_DEVICE_BUSY_ON_SESSION), syscall.Errno(ISDSC_SESSION_BUSY):
						time.Sleep(500 * time.Millisecond)
					}
				}
			}
		}

		// If our return error is not set, check each of our logout sessions to see if any session
		// logout failed.  If so, update return error with one of the session failures.
		if err == nil {
			for _, v := range sessionErrors {
				if v != nil {
					err = v
					break
				}
			}
		}
	}

	return err
}
