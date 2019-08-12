// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package wmi handles WMI queries
package wmi

import (
	log "github.com/hpe-storage/common-host-libs/logger"
)

// MSFT_iSCSITargetPortal WMI class
type MSFT_iSCSITargetPortal struct {
	InitiatorInstanceName  string
	InitiatorPortalAddress string
	IsDataDigest           bool
	IsHeaderDigest         bool
	TargetPortalAddress    string
	TargetPortalPortNumber uint16
}

// GetMSFTiSCSITargetPortal enumerates this host's MSFT_iSCSITargetPortal objects
func GetMSFTiSCSITargetPortal(whereOperator string) (portals []*MSFT_iSCSITargetPortal, err error) {
	log.Tracef(">>>>> GetMSFTiSCSITargetPortal, whereOperator=%v", whereOperator)
	log.Trace("<<<<< GetMSFTiSCSITargetPortal")

	// Form the WMI query
	wmiQuery := "SELECT * FROM MSFT_iSCSITargetPortal"
	if whereOperator != "" {
		wmiQuery += " WHERE " + whereOperator
	}

	// Execute the WMI query
	err = ExecQuery(wmiQuery, rootMicrosoftWindowsStorage, &portals)
	return portals, err
}
