// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package iscsi

import (
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	log "github.com/hpe-storage/common-host-libs/logger"
)

const (
	nimbleVendorProduct     = "Nimble  Server          " // Nimble Server Vendor ID / Product ID
	nimbleTargetScopeOffset = 0x2E                       // Offset in Inquiry page where target scope is stored
)

const (
	// Shared error messages
	errorMessageEmptyIqnFound       = "empty iqn found"
	errorMessageFailedInquiry       = "failed Inquiry with scsiStatus=%v, len(inquiryBuffer)=%v"
	errorMessageInvalidTargetScope  = "invalid target scope %v"
	errorMessageIscsiPathNotFound   = "%s not found to determine iscsi initiator name"
	errorMessageNoActiveConnections = "no active connections on sessionId %x-%x"
	errorMessageNoTargetScope       = "no sessions could report the target scope"
	errorMessageNonNimbleTarget     = "non-Nimble target %v"
)

type IscsiPlugin struct {
}

func NewIscsiPlugin() *IscsiPlugin {
	return &IscsiPlugin{}
}

func (plugin *IscsiPlugin) GetDiscoveredTargets() ([]*model.IscsiTarget, error) {
	return nil, nil
}
func (plugin *IscsiPlugin) GetAllLoggedInTargets() ([]*model.IscsiTarget, error) {
	return nil, nil
}
func (plugin *IscsiPlugin) GetTargetPortals(targetName string, ipv4Only bool) ([]*model.TargetPortal, error) {
	log.Infof(">>> GetTargetPortals, targetName=%v, ipv4Only=%v", targetName, ipv4Only)
	defer log.Infoln("<<< GetTargetPortals")
	return plugin.getTargetPortals(targetName, ipv4Only)
}
func (plugin *IscsiPlugin) GetSessionProperties(targetName string, sessionId string) (map[string]string, error) {
	return nil, nil
}
func (plugin *IscsiPlugin) DiscoverTargets(portal string) ([]*model.IscsiTarget, error) {
	return nil, nil
}
func (plugin *IscsiPlugin) LoginTarget(targetName string) error {
	return nil
}
func (plugin *IscsiPlugin) IsTargetLoggedIn(targetName string, portal string) (bool, error) {
	return false, nil
}
func (plugin *IscsiPlugin) LogoutTarget(targetName string, portal string) error {
	return nil
}
func (plugin *IscsiPlugin) GetIscsiInitiators() (*model.Initiator, error) {
	return getIscsiInitiators()
}
func (plugin *IscsiPlugin) GetTargetScope(targetName string) (string, error) {
	return getTargetScope(targetName)
}

// RescanIscsiTarget rescans host ports for iSCSI devices
func (plugin *IscsiPlugin) RescanIscsiTarget(lunID string) error {
	log.Infof(">>> RescanIscsiTarget initiated for lunID %v", lunID)
	defer log.Infoln("<<< RescanIscsiTarget")
	return rescanIscsiTarget(lunID)
}
