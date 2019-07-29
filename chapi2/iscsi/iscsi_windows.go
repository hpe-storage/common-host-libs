// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package iscsi

import (
	"strings"

	"github.com/hpe-storage/common-host-libs/chapi2/cerrors"
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/windows/iscsidsc"
	"github.com/hpe-storage/common-host-libs/windows/wmi"
)

func getIscsiInitiators() (init *model.Initiator, err error) {
	log.Info(">>>>> getIscsiInitiators")
	defer log.Info("<<<<< getIscsiInitiators")

	var initiatorNodeName string
	initiatorNodeName, err = iscsidsc.GetIScsiInitiatorNodeName()
	if err != nil {
		return nil, err
	}

	if initiatorNodeName == "" {
		err = cerrors.NewChapiError(cerrors.NotFound, errorMessageEmptyIqnFound)
		log.Error(err)
		return nil, err
	}
	log.Infof("got iscsi initiator name as %s", initiatorNodeName)

	initiators := []string{initiatorNodeName}
	init = &model.Initiator{AccessProtocol: "iscsi", Init: initiators}
	return init, err
}

// getTargetScope enumerates the target scope for the given iSCSI target.  An empty string is
// returned if we were unable to determine the target scope.
func getTargetScope(targetName string) (string, error) {
	log.Infof(">>>>> getTargetScope, targetName=%v", targetName)
	defer log.Info("<<<<< getTargetScope")

	// Enumerate all the iSCSI sessions
	iscsiSessions, err := iscsidsc.GetIscsiSessionList()
	if err != nil {
		log.Error(err.Error())
		return "", err
	}

	// Keep track of the last enumeration error.  If all attempted queries fail, we'll return
	// this error to the caller.
	var lastErr error

	// Loop through all the iSCSI sessions
	for _, iscsiSession := range iscsiSessions {

		// If the session isn't for our target, skip it
		if !strings.EqualFold(targetName, iscsiSession.TargetName) {
			continue
		}

		// If there are no session connections (e.g. reconnecting), skip this session
		if len(iscsiSession.Connections) == 0 {
			lastErr = cerrors.NewChapiErrorf(cerrors.NotFound, errorMessageNoActiveConnections, iscsiSession.SessionID.AdapterUnique, iscsiSession.SessionID.AdapterSpecific)
			log.Info(lastErr.Error())
			continue
		}

		// Issue an Inquiry request on the current session
		scsiStatus, inquiryBuffer, _, inquiryErr := iscsidsc.SendScsiInquiry(iscsiSession.SessionID, 0, 0, 0)
		if len(inquiryBuffer) >= nimbleTargetScopeOffset {

			// Convert the vendor/product ID into a string
			vendorProduct := string(inquiryBuffer[8:32])

			// If this isn't a Nimble target, log an error and fail request
			if vendorProduct != nimbleVendorProduct {
				lastErr = cerrors.NewChapiErrorf(cerrors.Internal, errorMessageNonNimbleTarget, vendorProduct)
				log.Error(lastErr.Error())
				return "", lastErr
			}

			// Get the target scope value from the Inquiry data
			var targetScope string
			targetScopeBits := inquiryBuffer[nimbleTargetScopeOffset] & 0x03
			switch targetScopeBits {
			case 0:
				targetScope = model.TargetScopeVolume
			case 1:
				targetScope = model.TargetScopeGroup
			default:
				// If an unexpected target scope is returned, log an error and fail request
				lastErr = cerrors.NewChapiErrorf(cerrors.Internal, errorMessageInvalidTargetScope, targetScopeBits)
				log.Error(lastErr.Error())
				return "", lastErr
			}

			// Successfully enumerated target scope on this session.  Log target scope and return to the caller
			log.Infof("targetName=%v, targetScope=%v", targetName, targetScope)
			return targetScope, nil
		}

		// Inquiry request failed, update our lastErr
		lastErr = inquiryErr
		if lastErr == nil {
			lastErr = cerrors.NewChapiErrorf(cerrors.NotFound, errorMessageFailedInquiry, scsiStatus, len(inquiryBuffer))
		}
	}

	// We were unable to enumerate the target scope from any target session; return last error detected
	if lastErr == nil {
		// If we couldn't find any session for our target, we could end up here.  In that case,
		// we'll log a generic error.
		lastErr = cerrors.NewChapiErrorf(cerrors.NotFound, errorMessageNoTargetScope)
	}
	log.Error(lastErr.Error())
	return "", lastErr
}

// rescanIscsiTarget rescans host ports for iSCSI devices
func rescanIscsiTarget(lunID string) error {
	// Unlike Linux, Windows does not have Target/LUN specific rescan capabilities so a synchronous
	// disk rescan is initiated and the lunID is ignored.
	return wmi.RescanDisks()
}
