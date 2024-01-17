// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package wmi handles WMI queries
package wmi

import (
	log "github.com/hpe-storage/common-host-libs/logger"
)

// MSiSCSIInitiator_TargetClass WMI class
type MSiSCSIInitiator_TargetClass struct {
	TargetName         string
	DiscoveryMechanism string
	InitiatorName      string
	ProtocolType       uint32
	TargetAlias        string
	PortalGroups       []*MSIscsiInitiator_PortalGroup
	Mappings           *MSIscsiInitiator_TargetMappings
	TargetFlags        uint32
	LoginOptions       *MSIscsiInitiator_TargetLoginOptions
}

// MSIscsiInitiator_PortalGroup WMI class
type MSIscsiInitiator_PortalGroup struct {
	Index   uint32
	Portals []*MSIscsiInitiator_Portal
}

// MSIscsiInitiator_Portal WMI class
type MSIscsiInitiator_Portal struct {
	Index        uint32
	SymbolicName string
	Address      string
	Port         uint16
}

// MSIscsiInitiator_TargetMappings WMI class
type MSIscsiInitiator_TargetMappings struct {
	InitiatorName  string
	TargetName     string
	OSDeviceName   string
	OSBusNumber    uint32
	OSTargetNumber uint32
	LUNList        []*MSIscsiInitiator_LUNList
}

// MSIscsiInitiator_LUNList WMI class
type MSIscsiInitiator_LUNList struct {
	OSLunNumber uint32
	TargetLun   uint64
}

// MSIscsiInitiator_TargetLoginOptions WMI class
type MSIscsiInitiator_TargetLoginOptions struct {
	Version                  uint32
	InformationSpecified     uint32
	LoginFlags               uint32
	AuthType                 uint32
	HeaderDigest             uint32
	DataDigest               uint32
	MaximumConnectionsuint32 uint32
	DefaultTime2Waituint32   uint32
	DefaultTime2Retainuint32 uint32
	Username                 []uint8
	Password                 []uint8
}

// GetMSIscsiInitiatorTargetClass enumerates this host's MSiSCSIInitiator_TargetClass objects
func GetMSIscsiInitiatorTargetClass(whereOperator string) (targets []*MSiSCSIInitiator_TargetClass, err error) {
	log.Trace(">>>>> GetMSIscsiInitiatorTargetClass")
	defer log.Trace("<<<<< GetMSIscsiInitiatorTargetClass")

	// Form the WMI query
	wmiQuery := "SELECT * FROM MSiSCSIInitiator_TargetClass"
	if whereOperator != "" {
		wmiQuery += " WHERE " + whereOperator
	}

	// Execute the WMI query
	err = ExecQuery(wmiQuery, rootWMI, &targets)
	return targets, err
}

// GetMSIscsiInitiatorTargetClassForTarget enumerates the specific target's MSiSCSIInitiator_TargetClass object.
func GetMSIscsiInitiatorTargetClassForTarget(target string) ([]*MSiSCSIInitiator_TargetClass, error) {
	log.Tracef(">>>>> GetMSIscsiInitiatorTargetClass, target=%v", target)
	defer log.Trace("<<<<< GetMSIscsiInitiatorTargetClass")

	whereOperator := `TargetName = "` + target + `"`
	return GetMSIscsiInitiatorTargetClass(whereOperator)
}
