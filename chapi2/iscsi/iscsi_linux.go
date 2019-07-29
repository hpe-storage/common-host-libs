// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package iscsi

import (
	"github.com/hpe-storage/common-host-libs/chapi2/cerrors"
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/util"
)

const (
	initiatorPath        = "/etc/iscsi/initiatorname.iscsi"
	initiatorNamePattern = "^InitiatorName=(?P<iscsiinit>.*)$"
)

func getIscsiInitiators() (init *model.Initiator, err error) {
	log.Info(">>>>> getIscsiInitiators")
	defer log.Info("<<<<< getIscsiInitiators")

	exists, _, err := util.FileExists(initiatorPath)
	if !exists {
		return nil, cerrors.NewChapiErrorf(cerrors.NotFound, errorMessageIscsiPathNotFound, initiatorPath)
		return nil, nil
	}
	initiators, err := util.FileGetStringsWithPattern(initiatorPath, initiatorNamePattern)
	if err != nil {
		log.Errorf("failed to get iqn from %s error %s", initiatorPath, err.Error())
		return nil, err
	}
	if len(initiators) == 0 {
		log.Errorf("empty iqn found from %s", initiatorPath)
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageEmptyIqnFound)
	}
	log.Infof("got iscsi initiator name as %s", initiators[0])
	init = &model.Initiator{AccessProtocol: "iscsi", Init: initiators}
	return init, err
}

// getTargetScope enumerates the target scope for the given iSCSI target.  An empty string is
// returned if we were unable to determine the target scope.
func getTargetScope(targetName string) (targetScope string, err error) {
	// TODO
	return "", nil
}

// rescanIscsiTarget rescans host ports for iSCSI devices
func rescanIscsiTarget(lunID string) error {
	// TODO
	return nil
}
