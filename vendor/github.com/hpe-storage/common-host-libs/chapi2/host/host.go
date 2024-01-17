// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package host

import (
	"os"

	"github.com/hpe-storage/common-host-libs/chapi2/model"
)

const (
	// Shared error messages
	errorMessageInvalidIpv4Address        = "invalid ipv4 address or mask provided to get network address"
	errorMessageUnableToDetermineHostName = "unable to determine host domain name"
	errorMessageUnableToParseIP           = "unable to parse ip address. Error:  %s"
	errorMessageUnableToParseMask         = "unable to parse network mask %s"
)

type HostPlugin struct {
}

func NewHostPlugin() *HostPlugin {
	return &HostPlugin{}
}

func (plugin *HostPlugin) GetUuid() (string, error) {
	uuid, err := getHostId()
	if err != nil {
		return "", err
	}
	return uuid, nil
}

func (plugin *HostPlugin) GetHostName() (string, error) {
	name, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return name, nil
}

func (plugin *HostPlugin) GetDomainName() (string, error) {
	domainName, err := getDomainName()
	if err != nil {
		return "", err
	}
	return domainName, nil
}

func (plugin *HostPlugin) GetNetworks() ([]*model.Network, error) {
	networks, err := getNetworkInterfaces()
	if err != nil {
		return nil, err
	}
	return networks, nil
}
