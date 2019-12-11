// Copyright 2019 Hewlett Packard Enterprise Development LP

package plugin

import (
	"fmt"
	"net"
	"os"

	"github.com/hpe-storage/common-host-libs/linux"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/util"
)

const (
	// PluginBaseDir represents base directory for plugin install
	PluginBaseDir = "/opt/hpe-storage/"
	// PluginSocketPath represents flexvolume plugin sockets directory
	PluginSocketPath = "/etc/hpe-storage/"
	// ManagedPluginSocketPath represents plugin socket path for docker managed plugins
	ManagedPluginSocketPath = "/run/docker/plugins/"
	// PluginSpecPath represents docker plugin spec directory
	PluginSpecPath = "/etc/docker/plugins/"
	// ConfigBaseDir represents user facing config directory for plugin
	ConfigBaseDir = "/etc/hpe-storage/"
	// MountBaseDir represents base directory for plugin volume mounts
	MountBaseDir = "/var/lib/kubelet/plugins/hpe.com/mounts/"
	// PluginLogFile represents plugin log location
	PluginLogFile = "/var/log/hpe-docker-plugin.log"
	// ManagedPluginSocketName represents plugin socket name for managed plugins
	ManagedPluginSocketName = "hpe-plugin.sock"
)

var (
	// PluginConfigDir represents config directory for plugin
	PluginConfigDir = ""
	// MountDir represents volume mount directory for the plugin
	MountDir = ""
	// SupportedFileSystems represent filesystem types supported for formatting with our plugin
	SupportedFileSystems = []string{"xfs", "btrfs", "ext2", "ext3", "ext4"}
)

// GetOrCreatePluginConfigDirectory get or create plugin config directory
func GetOrCreatePluginConfigDirectory() (string, error) {
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

// GetOrCreatePluginConfigDirectory get or create plugin mount directory
// TODO: handle mountDir from volume-driver.json
func GetOrCreatePluginMountDirectory() (string, error) {
	exists, _, _ := util.FileExists(MountBaseDir)
	if !exists {
		err := os.MkdirAll(MountBaseDir, 0644)
		if err != nil {
			log.Errorf("unable to create plugin mount directory %s, err %s", MountBaseDir, err.Error())
			return "", err
		}
	}
	// ignore our mount directory to locate file using mlocate
	excludeMountDirFromUpdateDb()
	MountDir = MountBaseDir
	return MountDir, nil
}

//PreparePluginSocket will setup socket path and listener
func PreparePluginSocket() (listener net.Listener, err error) {
	// determine socket path for listener
	socketPath := PluginSocketPath
	if IsManagedPlugin() {
		socketPath = ManagedPluginSocketPath
	}

	// create the socket directory if needed
	_, isDir, err := util.FileExists(socketPath)
	if err != nil {
		log.Trace("Err", err)
		return nil, err
	}
	if !isDir {
		log.Trace("creating plugin directory ", socketPath)
		os.MkdirAll(socketPath, 0700)
	}

	// get socket name based on plugin type
	pluginSocketFileName := getPluginSocketFileName()

	// cleanup already existing plugin socket file
	os.Remove(socketPath + pluginSocketFileName)

	listener, err = net.Listen("unix", socketPath+pluginSocketFileName)
	if err != nil {
		log.Fatalf("unable to create listener on plugin socket file %s, err %s", socketPath+pluginSocketFileName, err.Error())
		return nil, err
	}
	return listener, nil
}

// returns socket file name this plugin will listen on
// for managed plugins, socket file is scoped within plugin(container), so common name can be used
// so that same config.json can be maintained across all with one name.
func getPluginSocketFileName() (socket string) {
	if IsManagedPlugin() {
		return ManagedPluginSocketName
	}
	pluginType := GetPluginType()
	return fmt.Sprintf("%s.sock", pluginType.String())
}

// invoke call to exclude docker plugin mountDir during mlocate file search
func excludeMountDirFromUpdateDb() error {
	return linux.ExcludeMountDirFromUpdateDb(MountDir)
}

// set default filesystem
func setDefaultFilesystem(reqOpts map[string]interface{}) (err error) {
	// dont need to set the default filesystem for linux.
	return nil
}

func setDefaultVolumeDir(reqOpts map[string]interface{}) (err error) {
	// dont need to se the default here for linux
	return nil
}
