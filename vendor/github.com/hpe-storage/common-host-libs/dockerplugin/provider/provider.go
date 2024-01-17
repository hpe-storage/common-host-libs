// Copyright 2019 Hewlett Packard Enterprise Development LP

package provider

import (
	"fmt"
	"github.com/hpe-storage/common-host-libs/connectivity"
	log "github.com/hpe-storage/common-host-libs/logger"
	"os"
	"sync"
	"time"
)

var (
	providerClient *connectivity.Client
	clientLock     sync.Mutex
	// VersionLimits indicates options limited by provider versions
	VersionLimits = map[string][]optionLimit{
		"0.0": []optionLimit{
			{"importVol", []string{"reverseRepl", "takeover", "snapshot", "restore"}},
		},
	}
)

const (
	// URI's
	// ActivateURI represents activate endpoint
	ActivateURI = "/Plugin.Activate"
	// CreateURI represents create endpoint
	CreateURI = "/VolumeDriver.Create"
	// UpdateURI represents update endpoint
	UpdateURI = "/VolumeDriver.Update"
	// ListURI represents list endpoint
	ListURI = "/VolumeDriver.List"
	// CapabilitiesURI represents capabilities endpoint
	CapabilitiesURI = "/VolumeDriver.Capabilities"
	// MountURI represents mount endpoint
	MountURI = "/VolumeDriver.Mount"
	// UnmountURI represents unmount endpoint
	UnmountURI = "/VolumeDriver.Unmount"
	// VolumeDriverGetURI represents get endpoint
	VolumeDriverGetURI = "/VolumeDriver.Get"

	// NimbleDetachURI represents nimble detach endpoint
	NimbleDetachURI = "/Nimble.Detach"
	// NimbleGetURI represents nimble get endpoint
	NimbleGetURI = "/Nimble.Get"
	// NimbleConfURI represents nimble config endpoint
	NimbleConfURI = "/HPEVolume.Config"
	//NimbleLoginURI :
	NimbleLoginURI = "/Nimble.Login"
	//NimbleRemoveURI  represent cert remove endpoint
	NimbleRemoveURI = "/Nimble.RemoveCert"
	// RemoveURI represents volume remove endpoint
	RemoveURI = "/VolumeDriver.Remove"
	// HPEVolumeVersionURI version URI
	HPEVolumeVersionURI = "/HPEVolume.Version"

	// timeouts
	providerClientTimeout = time.Duration(300) * time.Second

	// env params
	// EnvIP represents provider IP env
	EnvIP = "PROVIDER_IP"
	// EnvService represents service name when provider running as k8s service
	EnvService = "PROVIDER_SERVICE"
	// EnvUsername represents provider username env
	EnvUsername = "PROVIDER_USERNAME"
	// EnvPassword represents provider password env
	EnvPassword = "PROVIDER_PASSWORD"
	// EnvPort represents provider port env
	EnvPort = "PROVIDER_PORT"
	// EnvInsecure represents http or https mode
	EnvInsecure = "INSECURE"
)

// User : provide HPE User API keys
type User struct {
	AccessKey    string `json:"access_key,omitempty"`
	AccessSecret string `json:"access_secret,omitempty"`
}

// GetProviderClient returns container-storage-provider client based on the plugin type
func GetProviderClient() (*connectivity.Client, error) {
	log.Trace(">>> getProviderClient")
	defer log.Trace("<<< getProviderClient")
	var err error

	// see if we have already created one
	if providerClient != nil {
		return providerClient, nil
	}

	clientLock.Lock()
	defer clientLock.Unlock()

	if IsHPECloudVolumesPlugin() {
		// get hpe cloud volumes client
		providerClient, err = getCloudContainerProviderClient()
	} else if IsSimplivityPlugin() {
		// get simplivity client
		providerClient, err = getSimplivityContainerProviderClient()
	} else {
		// assume nimble client by default
		providerClient, err = getNimbleContainerProviderClient()
	}
	if err != nil {
		log.Errorf("unable to get container provider client, err %s", err.Error())
		return nil, err
	}
	return providerClient, err
}

// GetProviderIP returns container-storage-provider IP
func GetProviderIP() (ip string, err error) {
	if ip = os.Getenv(EnvIP); ip != "" {
		return ip, nil
	}
	return "", fmt.Errorf("%s env is not set", EnvIP)
}

// GetProviderAccessKeys returns api access keys for the provider configured in env
func GetProviderAccessKeys() (*User, error) {
	// read from environment variables
	accessKey := os.Getenv(EnvUsername)
	if accessKey == "" {
		return nil, fmt.Errorf("env variable %s is not provided", EnvUsername)
	}
	accessSecret := os.Getenv(EnvPassword)
	if accessKey == "" {
		return nil, fmt.Errorf("env variable %s is not provided", EnvPassword)
	}
	return &User{AccessKey: accessKey, AccessSecret: accessSecret}, nil
}

// GetProviderURI returns container storage provider URI based on env set or using passed in defaults
func GetProviderURI(defaultProviderPortal, defaultProviderPort, basePath string) (providerURI string, err error) {
	// Assume defaults
	portal := defaultProviderPortal
	port := defaultProviderPort

	// Override ip:port if specified from env
	if envportal := os.Getenv(EnvIP); envportal != "" {
		portal = envportal
	}

	if envport := os.Getenv(EnvPort); envport != "" {
		port = envport
	}

	// if service name is provided, then handle container-provider running as k8s service
	if envService := os.Getenv(EnvService); envService != "" {
		// override with service name
		portal = envService
		// allow http connection to service
		os.Setenv(EnvInsecure, "true")
	}

	if port == "" || portal == "" {
		return "", fmt.Errorf("unable to get provider uri as environment param %s/%s are not set", EnvIP, EnvPort)
	}

	if os.Getenv(EnvInsecure) == "true" {
		providerURI = fmt.Sprintf("http://%s:%s", portal, port)
	} else {
		providerURI = fmt.Sprintf("https://%s:%s", portal, port)
	}
	if basePath != "" {
		providerURI = providerURI + basePath
	}
	log.Debugf("using container provider URI %s", providerURI)
	return providerURI, nil
}
