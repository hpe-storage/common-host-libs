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
//@Title  implement the Nimble Volume Driver Update for docker
//@Description implement the /VolumeDriver.Update Docker end point
//@Accept json
//@Resource /VolumeDriver.Update
//@Success 200 DriverResponse
//@Router /VolumeDriver.Update [put]
//@BasePath http:/VolumeDriver.Update
// VolumeDriverUpdate implement the /VolumeDriver.Update Docker end point
func VolumeDriverUpdate(w http.ResponseWriter, r *http.Request) {
	log.Debugf("volumeUpdate called")
	cr := &CreateResponse{}
	// Populate Host Context to the Plugin Request
	pluginReq, err := preparePluginRequest(r)
	if err != nil {
		dr := DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(dr)
		return
	}
	//get containerProviderClient
	providerClient, err := provider.GetProviderClient()
	if err != nil {
		err = errors.New("unable to setup the container-provider client " + err.Error())
		resp := &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}
	//container-provider /VolumeDriver.Update called
	_, err = providerClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.UpdateURI, Payload: &pluginReq, Response: &cr, ResponseError: &cr})
	if cr.Err != "" {
		cr = &CreateResponse{Err: cr.Err}
		json.NewEncoder(w).Encode(cr)
		return
	}
	if err != nil {
		log.Trace("err: ", err.Error())
		cr = &CreateResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(cr)
		return
	}
	log.Debugf("%s: request=(%+v) response=(%+v)", provider.UpdateURI, pluginReq, cr)
	json.NewEncoder(w).Encode(cr)
	return
}
