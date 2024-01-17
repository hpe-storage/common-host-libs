// Copyright 2019 Hewlett Packard Enterprise Development LP

package handler

import (
	"encoding/json"
	"errors"
	"github.com/hpe-storage/common-host-libs/chapi"
	"github.com/hpe-storage/common-host-libs/connectivity"
	"github.com/hpe-storage/common-host-libs/dockerplugin/plugin"
	"github.com/hpe-storage/common-host-libs/dockerplugin/provider"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/model"
	"net/http"
	"strings"
)

//@APIVersion 1.0.0
//@Title  implement the Nimble Volume Driver Path for docker
//@Description implement the /VolumeDriver.Path Docker end point
//@Accept json
//@Resource /VolumeDriver.Path
//@Success 200 MountResponse
//@Router /VolumeDriver.Path [post]
//@BasePath http:/VolumeDriver.Path
// VolumeDriverPath implement the /VolumeDriver.Path Docker end point
func VolumeDriverPath(w http.ResponseWriter, r *http.Request) {
	log.Trace("/VolumeDriver.Path called")
	volResp := &VolumeResponse{}
	var mr MountResponse
	// Populate Host Context to the Plugin Request
	pluginReq, err := populateHostContextAndScope(r)
	if err != nil {
		mr = MountResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(mr)
		return
	}

	// obtain chapi client
	chapiClient, err := chapi.NewChapiClient()
	if err != nil {
		mr = MountResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(mr)
		return
	}
	var respMount []*model.Mount

	// Add user credentials for request
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
	providerClient, err := provider.GetProviderClient()
	if err != nil {
		err = errors.New("unable to setup the container-provider client " + err.Error())
		resp := &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}
	//1. container-provider /Nimble.Get called
	_, err = providerClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.NimbleGetURI, Payload: &pluginReq, Response: &volResp, ResponseError: nil})
	if volResp.Err != "" {
		mr = MountResponse{Err: volResp.Err}
		json.NewEncoder(w).Encode(mr)
		return
	}

	//2. get mount for the volume object obtained from array
	err = chapiClient.GetMounts(&respMount, volResp.Volume.SerialNumber)
	if err != nil && !(strings.Contains(err.Error(), "object was not found")) {
		mr = MountResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(mr)
		return
	}
	mountPoint := plugin.MountDir + volResp.Volume.Name
	if respMount != nil {
		for _, mts := range respMount {
			if mts.Mountpoint == mountPoint {
				log.Trace("Mount of volume found")
				mr = MountResponse{MountPoint: mts.Mountpoint, Err: ""}
				json.NewEncoder(w).Encode(mr)
				return
			}
		}
	}
	log.Infof("%s: request=(%+v) response=(%+v)", "VolumeDriver.Path", pluginReq, mr)
	json.NewEncoder(w).Encode(mr)
	return
}
