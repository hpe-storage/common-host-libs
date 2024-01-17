// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package wmi handles WMI queries
package wmi

import (
	log "github.com/hpe-storage/common-host-libs/logger"
)

// MSFC_FibrePortHBAAttributes WMI class
type MSFC_FibrePortHBAAttributes struct {
	InstanceName string
	Active       bool
	UniquePortId uint64
	HBAStatus    uint32
	Attributes   *MSFC_HBAPortAttributesResults
}

// MSFC_HBAPortAttributesResults WMI class
type MSFC_HBAPortAttributesResults struct {
	NodeWWN                     [8]uint8
	PortWWN                     [8]uint8
	PortFcId                    uint32
	PortType                    uint32
	PortState                   uint32
	PortSupportedClassofService uint32
	PortSupportedFc4Types       [32]uint8
	PortActiveFc4Types          [32]uint8
	PortSupportedSpeed          uint32
	PortSpeed                   uint32
	PortMaxFrameSize            uint32
	FabricName                  [8]uint8
	NumberofDiscoveredPorts     uint32
}

// GetMSFC_FibrePortHBAAttributes enumerates this host's MSFC_FibrePortHBAAttributes objects
func GetMSFC_FibrePortHBAAttributes() (fcPorts []*MSFC_FibrePortHBAAttributes, err error) {
	log.Trace(">>>>> GetMSFC_FibrePortHBAAttributes")
	defer log.Trace("<<<<< GetMSFC_FibrePortHBAAttributes")

	// Execute the WMI query
	err = ExecQuery("SELECT * FROM MSFC_FibrePortHBAAttributes", rootWMI, &fcPorts)
	return fcPorts, err
}
