// Copyright 2019 Hewlett Packard Enterprise Development LP

package handler

import (
	"encoding/json"
	"errors"
	"github.com/hpe-storage/common-host-libs/connectivity"
	"github.com/hpe-storage/common-host-libs/dockerplugin/plugin"
	"github.com/hpe-storage/common-host-libs/dockerplugin/provider"
	log "github.com/hpe-storage/common-host-libs/logger"
	"net/http"
)

//@APIVersion 1.0.0
//@Title  implement the Nimble Volume Driver capabilities for docker
//@Description implement the /VolumeDriver.Capabilities Docker end point
//@Accept json
//@Resource /VolumeDriver.Capabilities
//@Success 200 PluginCapability
//@Router /VolumeDriver.Capabilities [post]
//@BasePath http:/VolumeDriver.Capabilities
// VolumeDriverCapabilities implement the /VolumeDriver.Capabilities Docker end point
func VolumeDriverCapabilities(w http.ResponseWriter, r *http.Request) {
	log.Tracef("VolumeDriver.Capabilities")
	var pluginReq PluginRequest
	capability := &PluginCapability{}

	pluginReq.Scope = plugin.IsLocalScopeDriver()
	var hostContext Host
	pluginReq.Host = &hostContext
	log.Trace("HostContext :", pluginReq.Host)

	// Add user credentials to the request
	user, err := provider.GetProviderAccessKeys()
	if err != nil {
		dr := DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(dr)
		return
	}
	if user != nil {
		pluginReq.User = user
	}

	//get containerProviderClient
	client, err := provider.GetProviderClient()
	if err != nil {
		err = errors.New("unable to setup the container-provider client " + err.Error())
		resp := &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	// container-provider /VolumeDriver.Capabilities called
	_, err = client.DoJSON(&connectivity.Request{Action: "POST", Path: provider.CapabilitiesURI, Payload: &pluginReq, Response: &capability, ResponseError: nil})
	if err != nil {
		resp := &DriverResponse{
			Err: err.Error(),
		}
		json.NewEncoder(w).Encode(resp)
		return
	}
	log.Debugf("%s: request=(%+v) response=(%+v)", provider.CapabilitiesURI, pluginReq, capability)
	json.NewEncoder(w).Encode(capability)
	return
}
