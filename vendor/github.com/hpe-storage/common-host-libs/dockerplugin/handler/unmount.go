// Copyright 2019 Hewlett Packard Enterprise Development LP

package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/hpe-storage/common-host-libs/chapi"
	"github.com/hpe-storage/common-host-libs/connectivity"
	"github.com/hpe-storage/common-host-libs/dockerplugin/provider"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/model"
)

var (
	unmountRequestsChan = make(chan string, defaultChannelCapacity)
)

//@APIVersion 1.0.0
//@Title  implement the Nimble Volume Driver Unmount for docker
//@Description implement the /VolumeDriver.Unmount Docker end point
//@Accept json
//@Resource /VolumeDriver.Unmount
//@Success 200 DriverResponse
//@Router /VolumeDriver.Unmount [post]
//@BasePath http:/VolumeDriver.Unmount
//nolint: gocyclo
// VolumeDriverUnmount implement the /VolumeDriver.Unmount Docker end point
func VolumeDriverUnmount(w http.ResponseWriter, r *http.Request) {
	log.Debug("/VolumeDriver.Unmount called ")
	volResp := &VolumeUnmountResponse{}
	log.Trace("volResp ", volResp)
	var dr DriverResponse
	// Populate Host Context to the Plugin Request
	pluginReq, err := preparePluginRequest(r)
	if err != nil {
		dr = DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(dr)
		return
	}

	// obtain new chapi client
	chapiClient, err := chapi.NewChapiClient()
	if err != nil {
		err = errors.New("unable to setup the chapi client " + err.Error())
		resp := &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(resp)
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
	log.Debugf("taken lock for volume %s in Unmount", pluginReq.Name)
	defer mapMutex.Unlock(pluginReq.Name)

	unmountRequestsChan <- pluginReq.Name
	log.Tracef("taken channel for volume %s in unmount with channel length :%d channel capacity :%d", pluginReq.Name, len(unmountRequestsChan), cap(unmountRequestsChan))
	defer unblockChannelHandler("unmount", pluginReq.Name, unmountRequestsChan)

	//1. container-provider /VolumeDriver.Unmount called
	_, err = providerClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.UnmountURI, Payload: &pluginReq, Response: &volResp, ResponseError: &volResp})
	log.Tracef("/VolumeDriver.Unmount for volume %s response=%+v", pluginReq.Name, volResp)
	if volResp.Err != "" {
		log.Errorf("unmount error (%s) on volume(%s) ", volResp.Err, pluginReq.Name)
		dr = DriverResponse{Err: volResp.Err}
		json.NewEncoder(w).Encode(dr)
		return
	}
	if err != nil {
		dr = DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(dr)
		return
	}
	volume := volResp.Volume
	log.Tracef("volResp Message %s", volResp.Message)

	//2. check for message for other mounts
	if volResp.Message == donotUnmount {
		log.Infof("%s is mounted on other containers", volume.Name)
		dr = DriverResponse{}
		json.NewEncoder(w).Encode(dr)
		return
	}

	//3. unmount the volume
	err = chapiClient.UnmountDevice(volume)
	if err != nil {
		dr = DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(dr)
		return
	}

	//4. Offline the device
	device, _ := chapiClient.GetDeviceFromVolume(volume)
	if device != nil {
		err = chapiClient.OfflineDevice(device)
		// return error only on Group Scoped Volume, Ignore for VST
		if err != nil && volResp.Volume.TargetScope == model.GroupScope.String() {
			dr = DriverResponse{Err: err.Error()}
			json.NewEncoder(w).Encode(dr)
			return
		}
	}

	//5. call detach on array
	err = nimbleDetach(volume, pluginReq)
	if err != nil {
		dr = DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(dr)
		return
	}

	//6. Delete the device
	if device != nil {
		err = chapiClient.DeleteDevice(device)
		if err != nil {
			if !strings.Contains(err.Error(), "object was not found on the system") {
				dr = DriverResponse{Err: err.Error()}
				json.NewEncoder(w).Encode(dr)
				return
			} else {
				log.Errorln("Delete device called failed, chapi returned ", err.Error())
			}

		}
	}

	//7. if destroyondetach is present in message, invoke /VolumeDriver.Remove
	if volResp.Message == destroyOnDetach {
		log.Debugf("destroy %s on detach", volume.Name)
		prefs := make(map[string]interface{})
		prefs["destroyOnDetach"] = "true"
		pluginReq.Preferences = prefs
		_, err = providerClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.RemoveURI, Payload: &pluginReq, Response: &dr, ResponseError: nil})
	}
	if err != nil {
		log.Debugf(err.Error())
	}
	log.Infof("%s: request=(%+v) response=(%+v)", provider.UnmountURI, pluginReq, volResp.VolumeResponse)
	json.NewEncoder(w).Encode(dr)
	return
}

// container provider /Nimble.Detach to clean up and remove access to the volume
func nimbleDetach(vol *model.Volume, req *PluginRequest) error {
	log.Trace("nimbleDetach called")
	nimbleDetach := NimbleDetachRequest{
		Volume: vol,
		Host:   req.Host,
		User:   req.User,
	}
	var dr DriverResponse
	providerClient, err := provider.GetProviderClient()
	if err != nil {
		return err
	}
	_, err = providerClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.NimbleDetachURI, Payload: &nimbleDetach, Response: &dr, ResponseError: nil})
	return err
}
