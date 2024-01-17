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
//@Title  activate the docker volume plugin
//@Description implement the /Plugin.Activate Docker end point
//@Accept json
//@Resource /Plugin.Activate
//@Success 200 PluginActivate
//@Router /Plugin.Activate [post]
//@BasePath http:/Plugin.Activate
// ActivatePlugin implement the /Plugin.Activate Docker end point
func ActivatePlugin(w http.ResponseWriter, r *http.Request) {
	log.Tracef("Plugin.Activate called")
	actPlugResp := &PluginActivate{}
	var pluginReq PluginRequest

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

	pluginReq.Scope = plugin.IsLocalScopeDriver()
	var hostContext Host
	hostContext.NodeID = "" // race condition is hit if we tried to connect to docker socket when it is not ready
	pluginReq.Host = &hostContext
	//get containerProviderClient
	providerClient, err := provider.GetProviderClient()
	if err != nil {
		err = errors.New("unable to setup the container-provider client " + err.Error())
		resp := &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}
	// container-provider /Plugin.Activate called
	log.Trace(pluginReq)
	_, err = providerClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.ActivateURI, Payload: &pluginReq, Response: &actPlugResp, ResponseError: nil})
	if err != nil {
		resp := &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}
	log.Infof("%s: request=(%+v) response=(%+v)", provider.ActivateURI, pluginReq, actPlugResp.Activate)
	json.NewEncoder(w).Encode(actPlugResp)
}
