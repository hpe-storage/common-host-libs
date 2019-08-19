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
	log.Trace(">>>>> getIscsiInitiators")
	defer log.Trace("<<<<< getIscsiInitiators")

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
	log.Tracef("got iscsi initiator name as %s", initiators[0])
	init = &model.Initiator{AccessProtocol: model.AccessProtocolIscsi, Init: initiators}
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

// getTargetPortals enumerates the target portals for the given iSCSI target
func (plugin *IscsiPlugin) getTargetPortals(targetName string, ipv4Only bool) ([]*model.TargetPortal, error) {
	// TODO
	return nil, nil
}

// loginTarget is called to connect to the given iSCSI target.  The parent LoginTarget() routine
// has already validated that target iqn and blockDev.IscsiAccessInfo are provided.
func (plugin *IscsiPlugin) loginTarget(blockDev model.BlockDeviceAccessInfo) (err error) {
	// TODO
	return nil
}

// logoutTarget is called to disconnect the given iSCSI target from this host.
func (plugin *IscsiPlugin) logoutTarget(targetName string) (err error) {
	// TODO
	return nil
}

// isTargetLoggedIn checks to see if the given iSCSI target is already logged in.
func (plugin *IscsiPlugin) isTargetLoggedIn(targetName string) (bool, error) {
	// TODO
	return false, nil
}
