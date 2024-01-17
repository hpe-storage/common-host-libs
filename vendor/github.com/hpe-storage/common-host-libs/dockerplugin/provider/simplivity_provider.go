// Copyright 2019 Hewlett Packard Enterprise Development LP

package provider

import (
	"github.com/hpe-storage/common-host-libs/connectivity"
	log "github.com/hpe-storage/common-host-libs/logger"
	"os"
)

const (
	defaultSimplivityProviderPort   = "8080"
	defaultSimplivityProviderPortal = "simplivity.hpe.com"
	defaultSimplivityBasePath       = "/docker-simplivity-plugin"
)

// IsSimplivityPlugin returns true if plugin_type is simplivity
func IsSimplivityPlugin() bool {
	if os.Getenv("PLUGIN_TYPE") == "simplivity" {
		return true
	}
	return false
}

func getSimplivityContainerProviderClient() (*connectivity.Client, error) {
	log.Trace(">>> getSimplivityContainerProviderClient")
	defer log.Trace("<<< getSimplivityContainerProviderClient")

	providerURI, err := GetProviderURI(defaultSimplivityProviderPortal, defaultSimplivityProviderPort, defaultSimplivityBasePath)
	if err != nil {
		return nil, err
	}
	return connectivity.NewHTTPClientWithTimeout(providerURI, providerClientTimeout), nil
}
