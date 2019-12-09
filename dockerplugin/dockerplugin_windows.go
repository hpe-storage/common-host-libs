// Copyright 2019 Hewlett Packard Enterprise Development LP

package dockerplugin

import (
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/dockerplugin/plugin"
)

// RunNimbledockerd runs listeners fordocker sockets
func RunNimbledockerd(c chan error, version string) (err error) {
	// version from build process
	plugin.Version = version
	// create listener for the socket
	listener, err := plugin.PreparePluginSocket()
	if err != nil {
		return err
	}
	// check and create config directory
	_, err = plugin.GetOrCreatePluginConfigDirectory()
	if err != nil {
		return nil
	}
	// check and create mount directory
	_, err = plugin.GetOrCreatePluginMountDirectory()
	if err != nil {
		return nil
	}
	// load the HPE Volume Config Cache
	err = plugin.LoadHPEVolConfig()
	if err != nil {
		log.Errorf("unable to load hpe volume config %s", err.Error())
		return err
	}
	// initialize the DeleteConflictDelay timeout
	//Fix : this is causing crash and not really required for Windows
	// since windows doesnt support K8s yet.
	//plugin.InitializeDeleteConflictDelay()

	// Control the mountConflictDelay behavior as it is causing default timeout 120 sec.
	plugin.InitializeMountConflictDelay()
	// listen on the http port
	router := NewRouter()

	//use channel to listen to multiple ports simultaneously
	go runNimbledockerd(listener, router, c)
	return nil
}
