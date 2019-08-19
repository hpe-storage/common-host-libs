// Copyright 2019 Hewlett Packard Enterprise Development LP

package plugin

import (
	"fmt"

	"github.com/hpe-storage/common-host-libs/dockerplugin/provider"
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
	if PluginConfigDir != "" {
		return PluginConfigDir, nil
	}
	ip, err := provider.GetProviderIP()
	if err != nil {
		return "", err
	}
	pluginType := GetPluginType()
	PluginConfigDir = fmt.Sprintf("%s%s\\%s\\%s\\", ConfigBaseDir, ip, pluginType.String(), GetDriverScope())
	exists, _, _ := util.FileExists(PluginConfigDir)
	if !exists {
		err := os.MkdirAll(PluginConfigDir, 0644)
		if err != nil {
			log.Errorf("unable to create plugin config directory %s, err %s", PluginConfigDir, err.Error())
			return "", err
		}
	}
	return PluginConfigDir, nil
}

// GetOrCreatePluginConfigDirectory get or create plugin mount directory based on ip address and scope
// TODO: handle custom mountDir from volume-driver.json
func GetOrCreatePluginMountDirectory() (pluginMountDir string, err error) {
	if MountDir != "" {
		return MountDir, nil
	}
	ip, err := provider.GetProviderIP()
	if err != nil {
		return "", err
	}
	pluginType := GetPluginType()
	MountDir = fmt.Sprintf("%s%s\\%s\\%s\\", MountBaseDir, ip, pluginType.String(), GetDriverScope())
	exists, _, _ := util.FileExists(MountDir)
	if !exists {
		err := os.MkdirAll(MountDir, 0744)
		if err != nil {
			log.Errorf("unable to create volume mount directory %s, err %s", MountDir, err.Error())
			return "", err
		}
	}
	return MountDir, nil
}

// PreparePluginSocket creates docker plugin socket directory and listener for our plugin
func PreparePluginSocket() (localListner net.Listener, globalListner net.Listener, err error) {
	// Listener for plugin
	log.Info("Local plugin listening port", windows.PluginListenPort)
	// local listen
	localListner, err = net.Listen(windows.Proto, windows.Hostname+":"+windows.PluginListenPort)
	if err != nil {
		log.Fatal("Listen err on local http port :", err.Error())
		return nil, nil, err
	}
	// global listen
	log.Info("Global plugin listening on ", windows.GlobalPluginListenPort)
	globalListner, err = net.Listen(windows.Proto, windows.Hostname+":"+windows.GlobalPluginListenPort)
	if err != nil {
		log.Fatal("Listen err on global http port :", err.Error())
		return nil, nil, err
	}
	return localListner, globalListner, nil
}

//GetDeviceSerialNumber :  Get the host device serial Number from the Volume SN
func GetDeviceSerialNumber(arraySn string) string {
	log.Tracef("GetDeviceSerialNumber called with %s", arraySn)
	return arraySn
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
