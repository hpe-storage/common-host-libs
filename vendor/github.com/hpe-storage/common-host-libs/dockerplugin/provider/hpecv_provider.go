// Copyright 2019 Hewlett Packard Enterprise Development LP

package provider

import (
	"github.com/hpe-storage/common-host-libs/connectivity"
	log "github.com/hpe-storage/common-host-libs/logger"
	"os"
)

const (
	defaultHpecvProviderPort   = "8090"
	defaultHpecvProviderPortal = "cloudvolumes.hpe.com"
)

//IsHPECloudVolumesPlugin returns true if plugin type is hpecv
func IsHPECloudVolumesPlugin() bool {
	if os.Getenv("PLUGIN_TYPE") == "cv" {
		return true
	}
	return false
}

func getCloudContainerProviderClient() (*connectivity.Client, error) {
	log.Trace(">>> getCloudContainerProviderClient")
	defer log.Trace("<<< getCloudContainerProviderClient")

	uri, err := GetProviderURI(defaultHpecvProviderPortal, defaultHpecvProviderPort, "")
	if err != nil {
		return nil, err
	}
	return connectivity.NewHTTPClientWithTimeout(uri, providerClientTimeout), nil
}
