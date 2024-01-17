// Copyright 2019 Hewlett Packard Enterprise Development LP

package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/hpe-storage/common-host-libs/chapi"
	"github.com/hpe-storage/common-host-libs/connectivity"
	"github.com/hpe-storage/common-host-libs/dockerplugin/plugin"
	"github.com/hpe-storage/common-host-libs/dockerplugin/provider"
	log "github.com/hpe-storage/common-host-libs/logger"
)

//@APIVersion 1.0.0
//@Title  implement the Nimble Volume Driver Remove for docker
//@Description implement the /VolumeDriver.Remove Docker end point
//@Accept json
//@Resource /VolumeDriver.Remove
//@Success 200 DriverResponse
//@Router /VolumeDriver.Remove [post]
//@BasePath http:/VolumeDriver.Remove
//nolint: gocyclo
// VolumeDriverRemove implement the /VolumeDriver.Remove Docker end point
func VolumeDriverRemove(w http.ResponseWriter, r *http.Request) {
	log.Debug("VolumeDriver.Remove called")
	volResp := &VolumeResponse{}
	dr := &DriverResponse{}
	// Populate Host Context to the Plugin Request
	pluginReq, err := populateHostContextAndScope(r)
	if err != nil {
		dr = &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(dr)
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

	//get containerProviderClient
	providerClient, err := provider.GetProviderClient()
	if err != nil {
		err = errors.New("unable to setup the container-provider client " + err.Error())
		resp := &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}
	//1. container-provider /Nimble.Get called
	_, err = providerClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.VolumeDriverGetURI, Payload: &pluginReq, Response: &volResp, ResponseError: &volResp})
	if volResp.Err != "" {
		dr = &DriverResponse{Err: volResp.Err}
		json.NewEncoder(w).Encode(dr)
		return
	}
	if err != nil {
		log.Trace(err.Error())
		dr = &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(dr)
		return
	}

	volume := volResp.Volume
	log.Tracef("Volume found :%+v", volume)

	// Check if the volume has ACR before we attempt to delete the volume
	log.Tracef("volume status is %+v ", volResp.Volume.Status)
	inUseValue, ok := volResp.Volume.Status[inUseKey]
	if ok {
		switch v := inUseValue.(type) {
		case bool:
			if v == true {
				log.Tracef("value of %s is %v", inUseKey, v)
				processDeleteConflictDelay(volume.Name, providerClient, pluginReq, plugin.DeleteConflictDelay)
			} else {
				log.Infof("%s is false for %s,ignoring processDeleteConflictDelay", plugin.DeleteConflictDelayKey, volume.Name)
			}
		}
	} else {
		log.Infof("%s not present for %s,ignoring processDeleteConflictDelay", plugin.DeleteConflictDelayKey, volume.Name)
	}

	// obtain chapi client
	chapiClient, err := chapi.NewChapiClient()
	if err != nil {
		dr = &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(dr)
		return
	}

	//2.Perform host side remove workflow
	err = chapiClient.UnmountDevice(volume)
	if err != nil {
		dr = &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(dr)
		return
	}

	// 3 . Offline the device (if present)
	device, _ := chapiClient.GetDeviceFromVolume(volume)
	if device != nil {
		log.Tracef("best effort to ofline device for %+v", volume)
		chapiClient.OfflineDevice(device)
	}

	// 4. Finally call Nimble.Detach (remove acl's). It should not have acl's so don't fail the request but do our best attempt
	log.Tracef("best effort to remove acl for %+v", volume)
	nimbleDetach(volume, pluginReq)

	// 5 . Delete the device (if present)
	if device != nil {
		log.Tracef("best effort to remove device for %+v", volume)
		chapiClient.DeleteDevice(device)
	}

	// 6. container-provider /VolumeDriver.Remove called
	_, err = providerClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.RemoveURI, Payload: &pluginReq, Response: &dr, ResponseError: nil})

	if err != nil {
		dr = &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(dr)
		return
	}
	log.Infof("%s: request=(%+v) response=(%+v)", provider.RemoveURI, pluginReq, dr)
	json.NewEncoder(w).Encode(dr)
	return
}

/* processDeleteConflictDelay
   The method checks the volume status to check if it is currently inUse.
   If the volume is inUse, we poll every tick ( 5 secs) to recheck if the volume is not inUse.
   Eventually after timeout (deleteDelay) we return
*/
func processDeleteConflictDelay(volName string, containerProviderClient *connectivity.Client, pluginReq *PluginRequest, deleteDelay int) {
	log.Tracef(">>>>> processDeleteConflictDelay called for %s with a timeout of %d seconds", volName, deleteDelay)
	defer log.Tracef("<<<<<< processDeleteConflictDelay")
	tick := time.Tick(5 * time.Second)
	timeout := time.After(time.Duration(deleteDelay) * time.Second)
	// Keep trying until we're timed out or got a result or got an error
	try := 0
	for {
		select {
		// Got a timeout! return
		case <-timeout:
			log.Tracef("timeout occurred after %v seconds for %s. Returning", timeout, volName)
			return
		// Got a tick, we should check on getVolumeInfo()
		case <-tick:
			try++
			volume, err := getVolumeInfo(containerProviderClient, pluginReq)
			if val, ok := volume.Status[inUseKey]; ok {
				switch v := val.(type) {
				case bool:
					if v == false {
						// if it is false it is safe to return
						log.Tracef("%d volume %s is not currently in use. Returning.", try, volName)
						return
					}
				}
				log.Debugf("%d: volume %s is still in use. Continuing.", try, volName)
			} else {
				// if the inUse field is absent return and not process
				log.Debugf("%d: %s absent from volume %s. Returning.", try, inUseKey, volName)
				return
			}
			// Error from getVolumeInfo(), we should bail
			if err != nil {
				log.Debugf("%d: unable to process deleteConflictDelay for %s, continue %s", try, volName, err.Error())
			}
		}
	}
}
