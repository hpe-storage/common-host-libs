// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package iscsidsc wraps the Windows iSCSI Discovery Library API
package iscsidsc

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// LoginIScsiTargetEx is similar to our C# WindowsBridge.cs LoginTarget() routine.  It will both log
// into the iSCSI target as well as make a persistent connection (if requested)
func LoginIScsiTargetEx(targetName string, initiatorInstance string, initiatorPortNumber uint32, targetPortal *ISCSI_TARGET_PORTAL, headerDigest ISCSI_DIGEST_TYPES, dataDigest ISCSI_DIGEST_TYPES, chapUsername string, chapPassword string, isPersistent bool) (uniqueSessionID *ISCSI_UNIQUE_SESSION_ID, uniqueConnectionID *ISCSI_UNIQUE_CONNECTION_ID, err error) {
	log.Trace(">>>>> LoginIScsiTargetEx")
	defer log.Trace("<<<<< LoginIScsiTargetEx")

	// Start by calling our base loginIScsiTarget routine with "isPersistent" set to false.  We do this
	// so that we actually make a connection.  If we set "isPersistent" to true, it would just make a
	// persistent connection without actually logging into the target.
	uniqueSessionID, uniqueConnectionID, err = loginIScsiTarget(targetName, initiatorInstance, initiatorPortNumber, targetPortal, headerDigest, dataDigest, chapUsername, chapPassword, false)

	// If the connection was successful, and "isPersistent" was set to true, now try to make it a
	// persistent connection.
	if (err == nil) && isPersistent {

		// NWT-3305: One second delay before setting persistent connection to work around Windows API issue
		time.Sleep(1000 * time.Millisecond)

		// Now make the persistent connection.  We don't do anything with the error should it occur.  The
		// routine we're calling will log the error in the unlikely event it does occur.  The more critical
		// task, that of logging into the target, was successful.
		loginIScsiTarget(targetName, initiatorInstance, initiatorPortNumber, targetPortal, headerDigest, dataDigest, chapUsername, chapPassword, true)
	}

	return uniqueSessionID, uniqueConnectionID, err
}

// loginIScsiTarget wraps the iSCSI discovery LoginIScsiTarget() API.  It's only for internal
// package use as we recommend the public LoginIScsiTarget() function be used instead.
func loginIScsiTarget(targetName string, initiatorInstance string, initiatorPortNumber uint32, targetPortal *ISCSI_TARGET_PORTAL, headerDigest ISCSI_DIGEST_TYPES, dataDigest ISCSI_DIGEST_TYPES, chapUsername string, chapPassword string, isPersistent bool) (uniqueSessionID *ISCSI_UNIQUE_SESSION_ID, uniqueConnectionID *ISCSI_UNIQUE_CONNECTION_ID, err error) {
	log.Tracef(">>>>> loginIScsiTarget, targetName=%v, initiatorPortNumber=%v, targetPortal=%v, headerDigest=%v, dataDigest=%v, CHAP=%v:%v, isPersistent=%v",
		targetName, initiatorPortNumber, targetPortal, headerDigest, dataDigest, chapUsername != "", chapPassword != "", isPersistent)
	defer log.Trace("<<<<< loginIScsiTarget")

	// iscsiLoginOptions is composed of ISCSI_LOGIN_OPTIONS plus additional storage for any CHAP
	// username/password we need to pass to the Windows iSCSI initiator.  Nimble Storage has a
	// limit of no more than 65 characters for the username and between 12-16 characters for
	// the CHAP password/secret.  We'll use up to 256 characters.
	type iscsiLoginOptions struct {
		ISCSI_LOGIN_OPTIONS
		UsernameStorage [256 + 1]uint8
		PasswordStorage [256 + 1]uint8
	}

	// Convert the target name to a UTF16 string
	targetNameUTF16 := syscall.StringToUTF16(targetName)

	// Convert the initiator instance to a UTF16 string
	initiatorInstanceUTF16 := syscall.StringToUTF16(initiatorInstance)

	// Convert the target portal to an ISCSI_TARGET_PORTAL_RAW object
	var targetPortalRaw *ISCSI_TARGET_PORTAL_RAW
	if targetPortal != nil {
		targetPortalRaw = iscsiTargetPortalToRaw(targetPortal)
	}

	// Build up an iscsiLoginOptions object
	var loginOptions iscsiLoginOptions
	loginOptions.Version = ISCSI_LOGIN_OPTIONS_VERSION
	loginOptions.LoginFlags = ISCSI_LOGIN_FLAG_MULTIPATH_ENABLED

	// Copy the header digest settings
	if headerDigest != ISCSI_DIGEST_TYPE_NONE {
		loginOptions.InformationSpecified |= ISCSI_LOGIN_OPTIONS_HEADER_DIGEST
		loginOptions.HeaderDigest = headerDigest
	}

	// Copy the data digest settings
	if dataDigest != ISCSI_DIGEST_TYPE_NONE {
		loginOptions.InformationSpecified |= ISCSI_LOGIN_OPTIONS_DATA_DIGEST
		loginOptions.DataDigest = dataDigest
	}

	// If CHAP credentials are specified, fill the CHAP details into the login structure
	if (chapUsername != "") && (chapPassword != "") {
		loginOptions.InformationSpecified |= (ISCSI_LOGIN_OPTIONS_AUTH_TYPE + ISCSI_LOGIN_OPTIONS_USERNAME + ISCSI_LOGIN_OPTIONS_PASSWORD)
		loginOptions.AuthType = ISCSI_CHAP_AUTH_TYPE

		// Convert username/password into ASCII arrays and determine their lengths
		chapUsernameASCII := []uint8(chapUsername)
		chapPasswordASCII := []uint8(chapPassword)
		chapUsernameLen := len(chapUsernameASCII)
		chapPasswordLen := len(chapPasswordASCII)

		// How large are our object's username/password arrays (less one character for NULL terminator)
		maxUsernameLen := len(loginOptions.UsernameStorage) - 1
		maxPasswordLen := len(loginOptions.PasswordStorage) - 1

		// If the CHAP username exceeds the maximum length, log error and fail request
		if chapUsernameLen > maxUsernameLen {
			errorMessage := fmt.Sprintf("CHAP username length of %v exceeds maximum of %v", chapUsernameLen, maxUsernameLen)
			log.Error(errorMessage)
			return nil, nil, fmt.Errorf(errorMessage)
		}

		// If the CHAP password exceeds the maximum length, log error and fail request
		if chapPasswordLen > maxPasswordLen {
			errorMessage := fmt.Sprintf("CHAP password length of %v exceeds maximum of %v", chapPasswordLen, maxPasswordLen)
			log.Error(errorMessage)
			return nil, nil, fmt.Errorf(errorMessage)
		}

		// Copy the CHAPI username/password into our login options structure
		copy(loginOptions.UsernameStorage[0:chapUsernameLen], chapUsernameASCII[0:chapUsernameLen])
		copy(loginOptions.PasswordStorage[0:chapPasswordLen], chapPasswordASCII[0:chapPasswordLen])
		loginOptions.Username = uintptr(unsafe.Pointer(&loginOptions.UsernameStorage[0]))
		loginOptions.Password = uintptr(unsafe.Pointer(&loginOptions.PasswordStorage[0]))
		loginOptions.UsernameLength = uint32(chapUsernameLen)
		loginOptions.PasswordLength = uint32(chapPasswordLen)
	}

	// Enumerate the "IsPersistent" value
	isPersistentInt := 0
	if isPersistent {
		isPersistentInt = 1
	}

	// Allocate session/connection ID structures
	uniqueSessionID = new(ISCSI_UNIQUE_SESSION_ID)
	uniqueConnectionID = new(ISCSI_UNIQUE_CONNECTION_ID)

	// Call the iSCSI discovery API (LoginIScsiTargetW)
	iscsiErr, _, _ := procLoginIScsiTargetW.Call(
		uintptr(unsafe.Pointer(&targetNameUTF16[0])),
		uintptr(0), // IsInformationalSession is not supported
		uintptr(unsafe.Pointer(&initiatorInstanceUTF16[0])),
		uintptr(initiatorPortNumber),
		uintptr(unsafe.Pointer(targetPortalRaw)),
		uintptr(0), // Security flags are not supported
		uintptr(0), // Target mapping is not supported
		uintptr(unsafe.Pointer(&loginOptions)),
		uintptr(0), // KeySize is not supported
		uintptr(0), // key is not supported
		uintptr(isPersistentInt),
		uintptr(unsafe.Pointer(uniqueSessionID)),
		uintptr(unsafe.Pointer(uniqueConnectionID)))

	if iscsiErr != ERROR_SUCCESS {
		// If an unexpected error occurs, initialize error object and log failure
		err = syscall.Errno(iscsiErr)
		log.Error(logIscsiFailure, err.Error())
	} else {
		// Log the return data
		log.Tracef("uniqueSessionID=%x-%x", uniqueSessionID.AdapterUnique, uniqueSessionID.AdapterSpecific)
		log.Tracef("uniqueConnectionID=%x-%x", uniqueConnectionID.AdapterUnique, uniqueConnectionID.AdapterSpecific)
	}

	// If an error occurred, we'll clear the session/connection IDs
	if err != nil {
		uniqueSessionID = nil
		uniqueConnectionID = nil
	}

	return uniqueSessionID, uniqueConnectionID, err
}
