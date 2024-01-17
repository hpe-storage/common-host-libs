// Copyright 2019 Hewlett Packard Enterprise Development LP

package handler

import (
	"encoding/json"
	"errors"
	"github.com/hpe-storage/common-host-libs/connectivity"
	"github.com/hpe-storage/common-host-libs/dockerplugin/provider"
	log "github.com/hpe-storage/common-host-libs/logger"
	"net/http"
)

//@APIVersion 1.0.0
//@Title  implement the Nimble Volume Driver List for docker
//@Description implement the /VolumeDriver.List Docker end point
//@Accept json
//@Resource /VolumeDriver.List
//@Success 200 ListResponse
//@Router /VolumeDriver.Lists [post]
//@BasePath http:/VolumeDriver.List
// VolumeDriverList implement the /VolumeDriver.List Docker end point
func VolumeDriverList(w http.ResponseWriter, r *http.Request) {
	log.Trace("volumeDriverList called")
	//Login to the Nimble Group
	listResp := &ListResponse{}
	pluginReq, err := populateHostContextAndScope(r)
	if err != nil {
		resp := &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Add user credentials for cloud/simplivity container-provider
	user, err := provider.GetProviderAccessKeys()
	if err != nil {
		dr := DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(dr)
		return
	}
	if user != nil {
		pluginReq.User = user
	}

	//container-provider /VolumeDriver.List called
	errResp := &ErrorResponse{}
	//get containerProviderClient
	providerClient, err := provider.GetProviderClient()
	if err != nil {
		err = errors.New("unable to setup the container-provider client " + err.Error())
		resp := &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}
	_, err = providerClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.ListURI, Payload: &pluginReq, Response: &listResp, ResponseError: &errResp})
	if err != nil {
		if errResp != nil {
			log.Error(errResp.Info)
			listResp = &ListResponse{Err: errResp.Info}
			json.NewEncoder(w).Encode(listResp)
			return
		}
		log.Trace("Err: ", err)
		listResp = &ListResponse{Volumes: nil, Err: err.Error()}
		json.NewEncoder(w).Encode(listResp)
		return
	}
	log.Tracef("response: %+v %s", listResp.Volumes, listResp.Err)
	json.NewEncoder(w).Encode(listResp)
	return
}
