// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package iscsi

import (
	"github.com/hpe-storage/common-host-libs/chapi2/cerrors"
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	log "github.com/hpe-storage/common-host-libs/logger"
)

const (
	nimbleVendorProduct     = "Nimble  Server          " // Nimble Server Vendor ID / Product ID
	nimbleTargetScopeOffset = 0x2E                       // Offset in Inquiry page where target scope is stored
	loginTimeout            = 5 * 60                     // Host has up to 5 minutes to make optimal iSCSI connections
)

const (
	// Shared error messages
	errorMessageConnectionFailed       = "connection failed"
	errorMessageEmptyIqnFound          = "empty iqn found"
	errorMessageFailedInquiry          = "failed Inquiry with scsiStatus=%v, len(inquiryBuffer)=%v"
	errorMessageInvalidConnectionType  = `invalid connection type "%v"`
	errorMessageInvalidTargetScope     = "invalid target scope %v"
	errorMessageIscsiPathNotFound      = "%s not found to determine iscsi initiator name"
	errorMessageLoginTimeout           = "logins not completed in time"
	errorMessageMissingIscsiAccessInfo = "missing IscsiAccessInfo object"
	errorMessageMissingIscsiTargetName = "missing iscsi target name"
	errorMessageNoAvailableConnections = "no available connections"
	errorMessageNoActiveConnections    = "no active connections on sessionId %x-%x"
	errorMessageNoTargetScope          = "no sessions could report the target scope"
	errorMessageNonNimbleTarget        = "non-Nimble target %v"
	errorMessageTargetNotFound         = "target not found"
)

// ITNexus - Initiator Port and Target Port
type ITNexus struct {
	initiatorPort *model.Network
	targetPort    *model.TargetPortal
}

type IscsiPlugin struct {
}

func NewIscsiPlugin() *IscsiPlugin {
	return &IscsiPlugin{}
}

func (plugin *IscsiPlugin) GetDiscoveredTargets() ([]*model.IscsiTarget, error) {
	// TODO
	return nil, nil
}

func (plugin *IscsiPlugin) GetAllLoggedInTargets() ([]*model.IscsiTarget, error) {
	// TODO
	return nil, nil
}

func (plugin *IscsiPlugin) GetTargetPortals(targetName string, ipv4Only bool) ([]*model.TargetPortal, error) {
	log.Tracef(">>>>> GetTargetPortals, targetName=%v, ipv4Only=%v", targetName, ipv4Only)
	defer log.Traceln("<<<<< GetTargetPortals")
	return plugin.getTargetPortals(targetName, ipv4Only)
}

func (plugin *IscsiPlugin) GetSessionProperties(targetName string, sessionId string) (map[string]string, error) {
	// TODO
	return nil, nil
}

func (plugin *IscsiPlugin) DiscoverTargets(portal string) ([]*model.IscsiTarget, error) {
	// TODO
	return nil, nil
}

// LoginTarget ensures that the provided iSCSI device is logged into this host
func (plugin *IscsiPlugin) LoginTarget(blockDev model.BlockDeviceAccessInfo) (err error) {
	log.Tracef(">>>>> LoginTarget, TargetName=%v", blockDev.TargetName)
	defer log.Traceln("<<<<< LoginTarget")

	// If the iSCSI iqn is not provided, fail the request
	if blockDev.TargetName == "" {
		err := cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageMissingIscsiTargetName)
		log.Error(err)
		return err
	}

	// If the IscsiAccessInfo object is not provided, fail the request
	if blockDev.IscsiAccessInfo == nil {
		err := cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageMissingIscsiAccessInfo)
		log.Error(err)
		return err
	}

	// Use the platform specific routine to login to the iSCSI target
	err = plugin.loginTarget(blockDev)

	// If there was an error logging into the iSCSI target, but connections remain, clean up
	// after ourselves by logging out the target.
	if err != nil {
		if loggedIn, _ := plugin.IsTargetLoggedIn(blockDev.TargetName); loggedIn == true {
			plugin.LogoutTarget(blockDev.TargetName)
		}
		return err
	}

	// Success!!!
	return nil
}

// IsTargetLoggedIn checks to see if the given iSCSI target is already logged in
func (plugin *IscsiPlugin) IsTargetLoggedIn(targetName string) (bool, error) {
	log.Tracef(">>>>> IsTargetLoggedIn, targetName=%v", targetName)
	defer log.Traceln("<<<<< IsTargetLoggedIn")

	// Call platform specific module
	return plugin.isTargetLoggedIn(targetName)
}

// LogoutTarget logs out the given iSCSI target
func (plugin *IscsiPlugin) LogoutTarget(targetName string) error {
	log.Tracef(">>>>> LogoutTarget, TargetName=%v", targetName)
	defer log.Traceln("<<<<< LogoutTarget")

	// Call platform specific module
	return plugin.logoutTarget(targetName)
}

// GetIscsiInitiators returns the host's iSCSI initiator object
func (plugin *IscsiPlugin) GetIscsiInitiators() (*model.Initiator, error) {
	return getIscsiInitiators()
}

// GetTargetScope returns the target's scope if known ("volume", "group", or empty string)
func (plugin *IscsiPlugin) GetTargetScope(targetName string) (string, error) {
	return getTargetScope(targetName)
}

// RescanIscsiTarget rescans host ports for iSCSI devices
func (plugin *IscsiPlugin) RescanIscsiTarget(lunID string) error {
	log.Tracef(">>>>> RescanIscsiTarget initiated for lunID %v", lunID)
	defer log.Traceln("<<<<< RescanIscsiTarget")
	return rescanIscsiTarget(lunID)
}
