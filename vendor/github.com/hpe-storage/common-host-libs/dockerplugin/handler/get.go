// Copyright 2019 Hewlett Packard Enterprise Development LP

package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hpe-storage/common-host-libs/chapi"
	"github.com/hpe-storage/common-host-libs/connectivity"
	"github.com/hpe-storage/common-host-libs/dockerplugin/provider"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/model"
	"net/http"
	"strings"
)

//@APIVersion 1.0.0
//@Title  implement the Nimble Volume Driver get for docker plugin
//@Description implement the /VolumeDriver.Get Docker end point
//@Accept json
//@Resource /VolumeDriver.Get
//@Success 200 VolumeResponse
//@Router /VolumeDriver.Get [post]
//@BasePath http:/VolumeDriver.Get
// VolumeDriverGet implement the /VolumeDriver.Get Docker end point
func VolumeDriverGet(w http.ResponseWriter, r *http.Request) {
	log.Tracef("VolumeDriver.Get")
	volumeResp := &VolumeResponse{}
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

	mapMutex.Lock(pluginReq.Name)
	log.Debugf("taken lock for volume %s in /VolumeDriver.Get", pluginReq.Name)
	defer mapMutex.Unlock(pluginReq.Name)

	// container provider /VolumeDriver.Get called
	_, err = providerClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.VolumeDriverGetURI, Payload: &pluginReq, Response: &volumeResp, ResponseError: &volumeResp})

	if err != nil {
		if volumeResp.Err != "" {
			vr := VolumeResponse{Err: err.Error()}
			json.NewEncoder(w).Encode(vr)
			return
		}
		vr := VolumeResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(vr)
		return
	}

	// Now get the mounts from the host and add MountPoint to response
	if volumeResp.Volume != nil {
		// obtain chapi client
		chapiClient, err := chapi.NewChapiClient()
		if err != nil {
			vr := VolumeResponse{Err: "unable to get chapi client" + err.Error()}
			json.NewEncoder(w).Encode(vr)
			return
		}
		var respMount []*model.Mount

		err = chapiClient.GetMounts(&respMount, volumeResp.Volume.SerialNumber)
		if err != nil && !(strings.Contains(err.Error(), "object was not found")) {
			vr := VolumeResponse{Err: err.Error()}
			json.NewEncoder(w).Encode(vr)
			return
		}
		setVolumeStatus(respMount, volumeResp)
	}
	log.Debugf("%s: request=(%+v) response=(%+v)", provider.VolumeDriverGetURI, pluginReq, volumeResp.Volume)
	json.NewEncoder(w).Encode(volumeResp)
	return
}

func setVolumeStatus(respMount []*model.Mount, volumeResp *VolumeResponse) {
	for _, mount := range respMount {
		if mount.Device.SerialNumber == volumeResp.Volume.SerialNumber {
			log.Trace("Mount Point found ", mount.Mountpoint)
			volumeResp.Volume.MountPoint = mount.Mountpoint
			if mount.Device != nil && mount.Device.AltFullPathName != "" {
				volumeResp.Volume.Status["devicePath"] = mount.Device.AltFullPathName
			} else {
				volumeResp.Volume.Status["devicePath"] = ""
			}
		}
	}
}

// nolint : dupl
func getVolumeInfo(providerClient *connectivity.Client, pluginReq *PluginRequest) (volume *model.Volume, err error) {
	log.Tracef(">>>>> getVolumeInfo called with %s", pluginReq.Name)
	defer log.Tracef("<<<<< getVolumeInfo")
	volResp := &VolumeResponse{}
	_, err = providerClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.VolumeDriverGetURI, Payload: &pluginReq, Response: &volResp, ResponseError: &volResp})
	if err != nil {
		if volResp.Err != "" {
			log.Errorf("getVolumeInfo err %s", volResp.Err)
			return nil, fmt.Errorf(volResp.Err)
		}
		return nil, err
	}
	if volResp.Volume == nil {
		return nil, fmt.Errorf("unable to retrieve volume with name %s", pluginReq.Name)
	}
	volume = volResp.Volume
	log.Debugf("retrieved volume %s ", volResp.Volume.Name)
	return volume, nil
}

// nolint : dupl
func nimbleGetVolumeInfo(providerClient *connectivity.Client, pluginReq *PluginRequest) (volume *model.Volume, err error) {
	log.Tracef(">>>>> nimbleGetVolumeInfo called with %s", pluginReq.Name)
	defer log.Trace("<<<<< nimbleGetVolumeInfo")
	volResp := &VolumeResponse{}
	_, err = providerClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.NimbleGetURI, Payload: &pluginReq, Response: &volResp, ResponseError: nil})
	if err != nil {
		if volResp.Err != "" {
			log.Trace(volResp.Err)
			return nil, fmt.Errorf(volResp.Err)
		}
		return nil, err
	}
	if volResp.Volume == nil {
		return nil, fmt.Errorf("unable to retrieve volume with name %s", pluginReq.Name)
	}
	volume = volResp.Volume
	log.Debugf("retrieved volume %s ", volResp.Volume.Name)
	return volume, nil
}
