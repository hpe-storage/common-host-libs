// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package wmi handles WMI queries
package wmi

import (
	log "github.com/hpe-storage/common-host-libs/logger"
)

const (
	ISCSI_IP_ADDRESS_TEXT = iota
	ISCSI_IP_ADDRESS_IPV4
	ISCSI_IP_ADDRESS_IPV6
	ISCSI_IP_ADDRESS_EMPTY
)

type ISCSI_IP_Address struct {
	Type         uint32
	IpV4Address  uint32
	IpV6Address  [16]uint8
	IpV6FlowInfo uint32
	IpV6ScopeId  uint32
	TextAddress  string
}

// ISCSI_PortalInfo WMI class
type ISCSI_PortalInfo struct {
	Index      uint32
	PortalType uint8
	Protocol   uint8
	Reserved1  uint8
	Reserved2  uint8
	IPAddr     *ISCSI_IP_Address
	Port       uint32
	PortalTag  uint16
}

// MSiSCSI_PortalInfoClass WMI class
type MSiSCSI_PortalInfoClass struct {
	InstanceName      string
	Active            bool
	PortalInfoCount   uint32
	PortalInformation []*ISCSI_PortalInfo
}

// GetMSiSCSIPortalInfoClass enumerates this host's MSiSCSI_PortalInfoClass object
func GetMSiSCSIPortalInfoClass() (portals *MSiSCSI_PortalInfoClass, err error) {
	log.Trace(">>>>> GetMSiSCSIPortalInfoClass")
	defer log.Trace("<<<<< GetMSiSCSIPortalInfoClass")

	// Execute the WMI query
	err = ExecQuery("SELECT * FROM MSiSCSI_PortalInfoClass", rootWMI, &portals)
	return portals, err
}
