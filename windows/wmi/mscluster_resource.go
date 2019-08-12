// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package wmi handles WMI queries
package wmi

import (
	log "github.com/hpe-storage/common-host-libs/logger"
)

// MSCluster_Resource_IP_Address is a customer MSCluster_Resource WMI class where PrivateProperties
// is defined as MSCluster_Property_Resource_IP_Address.
type MSCluster_Resource_IP_Address struct {
	Caption                   string
	InstallDate               string
	Status                    string
	Flags                     uint32
	Characteristics           uint32
	Name                      string
	Id                        string
	Description               string
	IsAlivePollInterval       uint32
	LooksAlivePollInterval    uint32
	PendingTimeout            uint32
	MonitorProcessId          uint32
	PersistentState           bool
	RestartAction             uint32
	RestartPeriod             uint32
	RestartThreshold          uint32
	EmbeddedFailureAction     uint32
	RetryPeriodOnFailure      uint32
	SeparateMonitor           bool
	Type                      string
	State                     uint32
	ResourceClass             uint32
	Subclass                  uint32
	PrivateProperties         *MSCluster_Property_Resource_IP_Address
	CryptoCheckpoints         []string
	RegistryCheckpoints       []string
	QuorumCapable             bool
	LocalQuorumCapable        bool
	DeleteRequiresAllNodes    bool
	CoreResource              bool
	DeadlockTimeout           uint32
	StatusInformation         uint64
	LastOperationStatusCode   uint64
	ResourceSpecificData1     uint64
	ResourceSpecificData2     uint64
	ResourceSpecificStatus    string
	RestartDelay              uint32
	IsClusterSharedVolume     bool
	RequiredDependencyTypes   []string
	RequiredDependencyClasses []uint32
	OwnerGroup                string
	OwnerNode                 string
}

// MSCluster_Property_Resource_IP_Address WMI class
type MSCluster_Property_Resource_IP_Address struct {
	Address               string
	DhcpAddress           string
	DhcpServer            string
	DhcpSubnetMask        string
	EnableDhcp            uint32
	EnableNetBIOS         uint32
	LeaseExpiresTime      string
	LeaseObtainedTime     string
	Network               string
	OverrideAddressMatch  uint32
	ProbeFailureThreshold uint32
	ProbePort             uint32
	SubnetMask            string
}

// GetClusterIPs enumerates this host's cluster IPs
func GetClusterIPs() (clusterIPs []*MSCluster_Resource_IP_Address, err error) {
	log.Trace(">>>>> GetClusterIPs")
	defer log.Trace("<<<<< GetClusterIPs")

	// Form the WMI query
	wmiQuery := `SELECT * FROM MSCluster_Resource WHERE Type="IP Address"`

	// Execute the WMI query
	err = ExecQuery(wmiQuery, rootMSCluster, &clusterIPs)

	// Log the cluster IPs
	if err == nil {
		for _, clusterIP := range clusterIPs {
			if clusterIP.PrivateProperties != nil {
				log.Tracef("Cluster IP address detected, %v", clusterIP.PrivateProperties.Address)
			}
		}
	}

	return clusterIPs, err
}
