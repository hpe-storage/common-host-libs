// Copyright 2019 Hewlett Packard Enterprise Development LP

package plugin

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hpe-storage/common-host-libs/jconfig"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/util"
)

const (
	// DriverConfigFile represents volume driver config file
	DriverConfigFile = "volume-driver.json"
	// EnvManagedPlugin represents if running as docker managed plugin
	EnvManagedPlugin = "MANAGED_PLUGIN"
	// EnvPluginType represents underlying storage platform type which plugin is servicing
	EnvPluginType = "PLUGIN_TYPE"
	// EnvScope represents plugin scope, i.e global or local
	EnvScope = "SCOPE"
	// DeleteConflictDelayKey represents key name for wait on conflicts during remove
	DeleteConflictDelayKey = "deleteConflictDelay"
	// DefaultDeleteConflictDelay represents delay to wait on conflicts during remove
	DefaultDeleteConflictDelay = 150
	// MountConflictDelayKey represents the key name for wait on conflicts for mount
	MountConflictDelayKey = "mountConflictDelay"
	// DefaultMountConflictDelay represents the default delay to wait on conflicts during mount
	DefaultMountConflictDelay = 120
)

var (
	// VolumeDriverConfig represent cache of volume-driver.json loaded
	VolumeDriverConfig *ConfigCache
	configLock         sync.Mutex
	// Version of Plugin
	Version = "dev"
	// DeleteConflictDelay represent conflict delay to wait during remove
	DeleteConflictDelay = DefaultDeleteConflictDelay
	// MountConflictDelay represent conflict delay to wait during mount
	MountConflictDelay = DefaultMountConflictDelay
)

// ConfigCache to store config options
type ConfigCache struct {
	cache      *jconfig.Config
	updateTime time.Time
}

// GetCache returns volume-driver.json config
func (c *ConfigCache) GetCache() *jconfig.Config {
	return c.cache
}

// Section different config section
type Section int

const (
	// Global section type
	Global = iota + 1
	// Defaults section type
	Defaults
	// Overrides section type
	Overrides
)

func (s Section) String() string {
	switch s {
	case Global:
		return "global"
	case Defaults:
		return "defaults"
	case Overrides:
		return "overrides"
	}
	return ""
}

// PluginType indicates the docker volume plugin type
type PluginType int

const (
	// nimble on-prem plugin type
	Nimble PluginType = 1 + iota
	// hpe cloud volumes plugin type
	Cv
	// simplivity plugin type
	Simplivity
)

func (plugin PluginType) String() string {
	switch plugin {
	case Nimble:
		return "nimble"
	case Cv:
		return "cv"
	case Simplivity:
		return "simplivity"
	default:
		return ""
	}
}

func GetPluginType() (plugin PluginType) {
	switch strings.ToLower(os.Getenv(EnvPluginType)) {
	case Nimble.String():
		return Nimble
	case Cv.String():
		return Cv
	case Simplivity.String():
		return Simplivity
	default:
		log.Infof("%s env is not set, assuming nimble type by default", EnvPluginType)
		return Nimble
	}
}

// IsManagedPlugin returns true if the plugin is deployed as docker managed plugin
func IsManagedPlugin() bool {
	managedPlugin := os.Getenv(EnvManagedPlugin)
	if managedPlugin != "" && strings.EqualFold(managedPlugin, "true") {
		return true
	}
	return false
}

// IsLocalScopeDriver return true if its a local scoped driver, false otherwise
func IsLocalScopeDriver() bool {
	scope := GetDriverScope()
	if scope == "local" {
		return true
	}
	return false
}

// GetDriverScope returns plugin scope configured
func GetDriverScope() (scope string) {
	scope, ok := os.LookupEnv(EnvScope)
	if !ok || len(scope) == 0 {
		// if not set assume global scope
		log.Tracef("plugin %s env is not specified, assuming global scope as default", EnvScope)
		return "global"
	}
	log.Tracef("obtained volume driver scope as %s", scope)
	return scope
}

// LoadHPEVolConfig loads the container config json file
func LoadHPEVolConfig() (err error) {
	PluginConfigDir, _ = GetOrCreatePluginConfigDirectory()
	exists, _, _ := util.FileExists(PluginConfigDir)
	if !exists {
		err = os.MkdirAll(PluginConfigDir, 0644)
		if err != nil {
			return fmt.Errorf("unable to create plugin config directory %s, err %s", PluginConfigDir, err.Error())
		}
	}

	volumeDriverConfFile := PluginConfigDir + DriverConfigFile
	exists, _, _ = util.FileExists(volumeDriverConfFile)
	if !exists {
		return fmt.Errorf("%s not present", volumeDriverConfFile)
	}
	log.Tracef("loading volumedriver config file %s", volumeDriverConfFile)
	configLock.Lock()
	defer configLock.Unlock()
	localConfig := &ConfigCache{}
	localConfig.cache, err = jconfig.NewConfig(volumeDriverConfFile)
	if err != nil {
		log.Error("unable to load volume driver config options", err.Error())
		return err
	}
	localConfig.updateTime = time.Now()

	// if no errors update volumeDriverConfig
	VolumeDriverConfig = localConfig
	log.Debugf("volumeDriverConfig cache :%v updatetime :%v", VolumeDriverConfig.cache, VolumeDriverConfig.updateTime)
	_, err = VolumeDriverConfig.cache.GetMap(Section.String(Global))
	if err != nil {
		return err
	}
	return nil
}

// UpdateVolumeDriverConfigCache updates the volume driver config cache
func UpdateVolumeDriverConfigCache(volumeDriverConfigFile string) {
	log.Tracef("updateVolumeDriverConfigCache called with %s", volumeDriverConfigFile)
	is, _, _ := util.FileExists(volumeDriverConfigFile)
	if !is {
		log.Tracef("config file not present to update cache")
		return
	}
	if VolumeDriverConfig != nil {
		// check the last modidication time stamp of the config file if existing cache is not nil
		file, err := os.Stat(volumeDriverConfigFile)
		if err != nil {
			log.Tracef("unable to read config file :%s", err.Error())
			return
		}
		modifiedTime := file.ModTime()
		if modifiedTime.Sub(VolumeDriverConfig.updateTime) > 0 {
			// cache is dirty update it
			log.Tracef("volumeDriverConfig cache is dirty. cache last updated at (%v), config file modified at(%v), updating cache", VolumeDriverConfig.updateTime.String(), modifiedTime.String())
			err = LoadHPEVolConfig()
			if err != nil {
				// if there is any error to update cache reuse the dirty cache
				log.Trace("error updating cache ", err.Error())
				return
			}
		}
	}
}

// GetUpdatedOptsFromConfig returns updated options after combining options from config files
func GetUpdatedOptsFromConfig(reqOpts map[string]interface{}) (updatedOpts map[string]interface{}, err error) {
	// get global options
	err = setDefaultFilesystem(reqOpts)
	updatedOpts, err = populateConfigOptsBySection(Section.String(Global), reqOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to obtain %s options %s", Section.String(Global), err.Error())
	}

	// get default options and populate
	updatedOpts, err = populateConfigOptsBySection(Section.String(Defaults), updatedOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to obtain %s options %s", Section.String(Defaults), err.Error())
	}
	err = setDefaultVolumeDir(reqOpts)
	// get override options and populate
	updatedOpts, err = populateConfigOptsBySection(Section.String(Overrides), updatedOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to obtain %s options %s", Section.String(Overrides), err.Error())
	}
	return updatedOpts, nil
}

func populateConfigOptsBySection(section string, reqOpts map[string]interface{}) (opts map[string]interface{}, err error) {
	log.Tracef("populateConfigOptsBySection called with section %s", section)
	// get config options map based on section name as key
	if VolumeDriverConfig == nil {
		return nil, fmt.Errorf("no config cache present to populate")
	}
	optionMap, err := VolumeDriverConfig.GetCache().GetMap(section)
	if err != nil {
		return nil, err
	}
	for key, value := range optionMap {
		// change to sizeInGiB if short form size is given in config file
		if key == "size" {
			key = "sizeInGiB"
		}
		// apply default or global options only when not present in command line
		if _, present := reqOpts[key]; present && section != Section.String(Overrides) {
			continue
		}
		reqOpts[key] = value
	}
	return reqOpts, nil
}

//CreateConfDirectory :
func CreateConfDirectory(confDir string) error {
	log.Trace("createConfDirectory called with ", confDir)
	_, isDir, err := util.FileExists(confDir)
	if err != nil {
		log.Errorf("CreateConfDirectory failed for %s, err %s", confDir, err.Error())
		return err
	}
	if isDir == false {
		log.Trace("creating conf directory ", confDir)
		os.MkdirAll(confDir, 0700)
	}
	log.Tracef("%s exists", confDir)
	return nil
}

// InitializeMountConflictDelay initializes mountConflictDelay
//nolint : dupl
func InitializeMountConflictDelay() {
	MountConflictDelay = DefaultMountConflictDelay
	if VolumeDriverConfig == nil {
		log.Debugf("unable to load hpe volume config")
		return
	}
	optsMap, err := VolumeDriverConfig.cache.GetMap(Section.String(Global))
	if err != nil {
		log.Debugf("failed to read from config file with err %s", err.Error())
		return
	}
	if val, ok := optsMap[MountConflictDelayKey]; ok {
		switch v := val.(type) {
		case string:
			intVal, err := strconv.Atoi(v)
			if err != nil {
				log.Warnf("unable to parse %s from config file, setting mountConflictDelay=%d", MountConflictDelayKey, DefaultMountConflictDelay)
				return
			}
			MountConflictDelay = intVal
		case int:
			MountConflictDelay = v
		}
	}
	log.Debugf("%s is set to %d", MountConflictDelayKey, MountConflictDelay)

}

// InitializeDeleteConflictDelay initializes deleteConflictDelay
//nolint : dupl
func InitializeDeleteConflictDelay() {
	DeleteConflictDelay = DefaultDeleteConflictDelay
	if VolumeDriverConfig == nil {
		log.Debugf("unable to load hpe volume config")
		return
	}
	optsMap, err := VolumeDriverConfig.cache.GetMap(Section.String(Global))
	if err != nil {
		log.Debugf("failed to read from config file with err %s", err.Error())
		return
	}
	if val, ok := optsMap[DeleteConflictDelayKey]; ok {
		switch v := val.(type) {
		case string:
			intVal, err := strconv.Atoi(v)
			if err != nil {
				log.Warnf("unable to parse %s setting deleteConflictDelay=%d", DeleteConflictDelayKey, DefaultDeleteConflictDelay)
				return
			}
			DeleteConflictDelay = intVal
		case int:
			DeleteConflictDelay = v
		}
	}
	log.Debugf("%s is set to %d", DeleteConflictDelayKey, DeleteConflictDelay)
}
