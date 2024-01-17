// Copyright 2019 Hewlett Packard Enterprise Development LP

package plugin

import (
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/util"
	"github.com/hpe-storage/common-host-libs/windows"
	"net"
	"os"
)

const (
	// socketpath doesnt exist on windows. declare it for compilation only.
	jsonType = "json"
	// PluginBaseDir represents base directory for plugin install
	PluginBaseDir = "C:\\ProgramData\\hpe-storage\\"
	// PluginSocketPath represents docker plugin sockets directory
	PluginSocketPath = "C:\\ProgramData\\Docker\\plugins\\"
	// PluginSpecPath represents docker plugin spec directory
	PluginSpecPath = "C:\\ProgramData\\Docker\\plugins\\"
	// ConfigBaseDir represents user facing config directory for plugin
	ConfigBaseDir = "C:\\ProgramData\\hpe-storage\\conf\\"
	// MountBaseDir represents base directory for plugin volume mounts
	MountBaseDir = "C:\\ProgramData\\hpe-storage-mounts\\"
	// PluginLogFile represents plugin log location
	PluginLogFile = "C:\\ProgramData\\hpe-storage\\log\\hpe-docker-plugin.log"
	// Default filesystem
	DefaultFileSystem = "ntfs"
)

var (
	// PluginConfigDir represents config directory for plugin
	PluginConfigDir = ""
	// MountDir represents volume mount directory for the plugin
	MountDir = ""
	// SupportedFileSystems represent filesystem types supported for formatting with our plugin
	SupportedFileSystems = []string{"ntfs", "refs"}
)

// placeholder for any windows plugin specific stuff

// GetOrCreatePluginConfigDirectory get or create plugin config directory based on ip address and scope
func GetOrCreatePluginConfigDirectory() (pluginConfigDir string, err error) {
	exists, _, _ := util.FileExists(ConfigBaseDir)
	if !exists {
		err := os.MkdirAll(ConfigBaseDir, 0644)
		if err != nil {
			log.Errorf("unable to create plugin config directory %s, err %s", ConfigBaseDir, err.Error())
			return "", err
		}
	}
	PluginConfigDir = ConfigBaseDir
	return PluginConfigDir, nil
}

// GetOrCreatePluginConfigDirectory get or create plugin mount directory based on ip address and scope
// TODO: handle custom mountDir from volume-driver.json
func GetOrCreatePluginMountDirectory() (pluginMountDir string, err error) {
	exists, _, _ := util.FileExists(MountBaseDir)
	if !exists {
		err := os.MkdirAll(MountBaseDir, 0644)
		if err != nil {
			log.Errorf("unable to create plugin mount directory %s, err %s", MountBaseDir, err.Error())
			return "", err
		}
	}

	MountDir = MountBaseDir
	return MountDir, nil
}

// PreparePluginSocket creates docker plugin socket directory and listener for our plugin
func PreparePluginSocket() (listner net.Listener, err error) {
	// Listener for plugin
	log.Info("Plugin listening port %s", windows.PluginListenPort)
	// local listen
	listner, err = net.Listen(windows.Proto, windows.Hostname+":"+windows.PluginListenPort)
	if err != nil {
		log.Fatal("Listen err on http port %s, err %s:", windows.PluginListenPort, err.Error())
		return nil, err
	}

	return listner, nil
}

// set default filesystem
func setDefaultFilesystem(reqOpts map[string]interface{}) (err error) {
	if _, present := reqOpts["filesystem"]; !present {
		reqOpts["filesystem"] = DefaultFileSystem
	}
	return nil
}

// set default mount point
func setDefaultVolumeDir(reqOpts map[string]interface{}) (err error) {
	if _, present := reqOpts["volumeDir"]; !present {
		reqOpts["volumeDir"] = MountBaseDir
	}
	return nil
}
