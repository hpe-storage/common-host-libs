// Copyright 2019 Hewlett Packard Enterprise Development LP

package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/hpe-storage/common-host-libs/logger"

	"github.com/hpe-storage/common-host-libs/chapi"
	"github.com/hpe-storage/common-host-libs/connectivity"
	"github.com/hpe-storage/common-host-libs/dockerplugin/plugin"
	"github.com/hpe-storage/common-host-libs/dockerplugin/provider"
	"github.com/hpe-storage/common-host-libs/model"
	"github.com/hpe-storage/common-host-libs/util"
)

var (
	defaultCreationTimeout   = time.Duration(300) * time.Second
	listOfCreateKeysToRemove = []string{"logLevel", volumeDirKey, plugin.DeleteConflictDelayKey, plugin.MountConflictDelayKey}
)

//@APIVersion 1.0.0
//@Title  implement the Nimble Volume Driver Create for docker
//@Description implement the /VolumeDriver.Create Docker end point
//@Accept json
//@Resource /VolumeDriver.Create
//@Success 200 CreateResponse
//@Router /VolumeDriver.Create [post]
//@BasePath http:/VolumeDriver.Create
// nolint : gocyclo exceeded
// VolumeDriverCreate implement the /VolumeDriver.Create Docker end point
func VolumeDriverCreate(w http.ResponseWriter, r *http.Request) {
	log.Debug("volumeCreate called")
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

	// populate defaut create options
	err = populateVolCreateOptions(pluginReq)
	if err != nil {
		log.Errorf("%s failed to add mount options from config file using defaults", err.Error())
	}

	// validate fsMode and fsOwner if specified in the request
	fsMode, fsOwner, err := getFileSystemModeAndOwnerFromRequest(pluginReq)
	if err != nil {
		dr := DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(dr)
		return
	}
	fsOpts := &model.FilesystemOpts{Mode: fsMode, Owner: fsOwner}

	// populate delayed create option to pluginReq except for import and clone workflows
	if !isValidDelayedCreateOpt(pluginReq) {
		log.Tracef("valid delayedCreate opts (%v) setting delayedCreate to true", pluginReq.Opts)
		pluginReq.Opts[delayedCreateOpt] = true
	}

	// remove global options from create request
	removeGlobalOptionsFromCreateRequest(pluginReq)

	// check if valid fileystem was present in the request
	if !isValidFilesystem(pluginReq) {
		dr := DriverResponse{Err: fmt.Sprintf("invalid filesystem type(%s), please enter one of the following options (%s)", pluginReq.Opts["filesystem"], strings.Join(plugin.SupportedFileSystems, " "))}
		json.NewEncoder(w).Encode(dr)
		return
	}

	mapMutex.Lock(pluginReq.Name)
	log.Debugf("taken lock on %s in create", pluginReq.Name)
	defer mapMutex.Unlock(pluginReq.Name)

	//1. container-provider /VolumeDriver.Create called
	var dr DriverResponse
	_, err = providerClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.CreateURI, Payload: &pluginReq, Response: &cr, ResponseError: &dr})
	if err != nil {
		if cr.Err != "" {
			dr := DriverResponse{Err: fmt.Errorf("unable to create the volume %s %s", pluginReq.Name, cr.Err).Error()}
			json.NewEncoder(w).Encode(dr)
			return
		}
		log.Tracef("err: %s", err.Error())
		dr := DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(dr)
		return
	}
	if cr.Err != "" {
		if strings.Contains(strings.ToLower(cr.Err), "unable to find any logged-in fc sessions") {
			//TODO issue rescan on the fc host
		} else if strings.Contains(strings.ToLower(cr.Err), "unrecognized field") {
			errMsg := fmt.Sprintf("unknown options for create volume %v", pluginReq.Opts)
			dr := DriverResponse{Err: errMsg}
			json.NewEncoder(w).Encode(dr)
			return
		}
		dr := DriverResponse{Err: cr.Err}
		json.NewEncoder(w).Encode(dr)
		return
	} else if cr.Help != "" {
		// create called with Help option
		log.Trace("Help Message")
		cr.Err = cr.Help
		json.NewEncoder(w).Encode(cr)
	} else if len(cr.Volumes) > 0 {
		// check if it is delayedCreate Response else continue with older create
		log.Tracef("response from volume create (%+v)", cr.Volumes[0])
		if val, ok := cr.Volumes[0].Status[delayedCreateOpt]; ok {
			log.Tracef("delayedCreate response %s", val)
			json.NewEncoder(w).Encode(cr)
			return
		}
		// obtain chapi client
		chapiClient, err := chapi.NewChapiClient()
		if err != nil {
			log.Trace("err: ", err.Error())
			cr = &CreateResponse{Err: "Unable to get chapi client" + err.Error()}
			json.NewEncoder(w).Encode(cr)
			return
		}
		// Creation of new volume
		log.Debug("Volume creation initiated for ", cr.Volumes[0].Name)
		discoveryIP := cr.Volumes[0].DiscoveryIP
		iqn := cr.Volumes[0].Iqn
		log.Tracef("SNO :%s DiscoveryIP :%s IQN %s", cr.Volumes[0].SerialNumber, discoveryIP, iqn)

		// change the connection mode to manual for docker
		cr.Volumes[0].ConnectionMode = manualMode
		//2. attach the device, create file system
		device, err := createFileSystemOnVolume(cr.Volumes, pluginReq, fsOpts)
		if err != nil {
			// since device creation failed. Cleanup the cache
			invalidateHostContextCache()
			// cleanup host side device if device exist
			if device != nil {
				// detach device and cleanup host side
				log.Debugf("initiating device detach for %+v", device)
				// set the target scope
				device.TargetScope = cr.Volumes[0].TargetScope

				// offline the device
				err = chapiClient.OfflineDevice(device)
				if err != nil {
					log.Debug("OfflineDevice err: ", err.Error())
				}
			}
			// call Nimble.Detach (remove acl's)
			err = nimbleDetach(cr.Volumes[0], pluginReq)
			if err != nil {
				log.Trace("err: ", err.Error())
			}
			if device != nil {
				// delete the device
				err = chapiClient.DeleteDevice(device)
				if err != nil {
					log.Trace("err: ", err.Error())
				}
			}
			var dr DriverResponse
			//force delete the volume on create failures else it will lie around in offline state
			pluginReq.Opts["destroyOnRm"] = true
			providerClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.RemoveURI, Payload: &pluginReq, Response: &dr, ResponseError: nil})
			dr = DriverResponse{Err: err.Error()}
			json.NewEncoder(w).Encode(dr)
			// final return after all the cleanup
			return
		}
		//3. Now offline the device
		log.Debugf("device %+v is unmounted for volume %+v, offline the device", device, cr.Volumes[0])
		// set the target scope
		device.TargetScope = cr.Volumes[0].TargetScope

		err = chapiClient.OfflineDevice(device)
		if err != nil {
			dr := DriverResponse{Err: "unable to detach volume from host " + err.Error()}
			json.NewEncoder(w).Encode(dr)
			return
		}

		//4. invoke Nimble.Detach (remove acl's)
		err = nimbleDetach(cr.Volumes[0], pluginReq)
		if err != nil {
			dr := DriverResponse{Err: "unable to detach volume from array " + err.Error()}
			json.NewEncoder(w).Encode(dr)
			return
		}

		//5. Finally delete the device
		err = chapiClient.DeleteDevice(device)
		if err != nil {
			dr := DriverResponse{Err: "unable to detach volume from host " + err.Error()}
			json.NewEncoder(w).Encode(dr)
			return
		}
	}
	log.Infof("%s: request=(%+v) response=(%+v)", provider.CreateURI, pluginReq, cr.Volumes)
	json.NewEncoder(w).Encode(cr)
	return
}

// Attach the device, Create filesystem on the device
// nolint : gocyclo
func createFileSystemOnVolume(vols []*model.Volume, pluginReq *PluginRequest, fsOpts *model.FilesystemOpts) (*model.Device, error) {
	log.Tracef("createFileSystemOnVolume called for %+v", vols)
	log.Traceln("Vol :", vols, "Host :", pluginReq.Host)

	// obtain chapi client with large timeout of 5 minutes max for creation
	chapiClient, err := chapi.NewChapiClientWithTimeout(defaultCreationTimeout)
	if err != nil {
		return nil, err
	}

	//1. Create and attach the device
	log.Tracef("calling attach device with vols %+v", vols)
	devices, err := chapiClient.AttachDevice(vols)
	if err != nil {
		if devices != nil {
			return devices[0], err
		}
		return nil, err
	}

	if len(devices) == 0 || len(vols) == 0 {
		return nil, errors.New("unable to retrieve volume / device ")
	}

	log.Trace("Device found ", devices[0].Pathname)
	device := devices[0]
	vol := vols[0]

	//2. Make a put request to put a partition / filesystem on the device
	fileSystemType := getFileSystemTypeFromRequest(pluginReq)
	//make sure volume.Mountpoint is populated
	vol.MountPoint = plugin.MountDir + vol.Name
	err = chapiClient.SetupFilesystemAndPermissions(device, vol, fileSystemType)
	if err != nil {
		return nil, fmt.Errorf("unable to setup filesystem for device %s, err(%s)", device.AltFullPathName, err.Error())
	}
	err = chapiClient.UnmountDevice(vol)
	if err == nil {
		// delete the mountPoint
		os.RemoveAll(vol.MountPoint)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}
	return device, err
}

func populateVolCreateOptions(req *PluginRequest) (err error) {
	log.Trace(">>>>> populateVolCreateOptions")
	defer log.Trace("<<<<< populateVolCreateOptions")

	// populate options based on correct priority order

	// check for short form or size and add complete key
	if _, present := req.Opts["size"]; present {
		req.Opts["sizeInGiB"] = req.Opts["size"]
	}

	// check if config file exist and load config
	volumeDriverConfFile := plugin.PluginConfigDir + plugin.DriverConfigFile
	// check if volumeDriverConfig is initialized or not
	if plugin.VolumeDriverConfig == nil {
		err = plugin.LoadHPEVolConfig()
		if err != nil {
			return err
		}
		_, err = plugin.VolumeDriverConfig.GetCache().GetMap(plugin.Section.String(plugin.Global))
		if err != nil {
			// volumeDriverConfig is not initialized yet.
			log.Tracef("error %s to retrieve data from existing config, load %s", err.Error(), volumeDriverConfFile)
			exists, _, _ := util.FileExists(volumeDriverConfFile)
			if !exists {
				return fmt.Errorf("unable to populate volume create options, driver config file %s doesn't exist", volumeDriverConfFile)
			}
			log.Tracef("volumeDriver Config file exists at %s", volumeDriverConfFile)
			err = plugin.LoadHPEVolConfig()
			if err != nil {
				return err
			}
		}
	}
	// validate if the volumeDriverConfFile in the cache is current with respect to modification time of the config file. Check if the cache is dirty
	plugin.UpdateVolumeDriverConfigCache(volumeDriverConfFile)

	var updatedOpts map[string]interface{}
	updatedOpts, err = plugin.GetUpdatedOptsFromConfig(req.Opts)
	if err != nil {
		return err
	}
	// update original options in the request
	req.Opts = updatedOpts

	log.Tracef("updated opts %+v", req.Opts)
	return nil
}

func isValidDelayedCreateOpt(pluginReq *PluginRequest) bool {
	log.Trace(">>>> isValidDelayedCreateOpt called")
	defer log.Tracef("<<<< isValidDelayedCreateOpt")
	_, ok := pluginReq.Opts["cloneOf"]
	if ok {
		return true
	}
	_, ok = pluginReq.Opts["importVolAsClone"]
	if ok {
		return true
	}
	_, ok = pluginReq.Opts["importVol"]
	if ok {
		return true
	}
	return false
}

func removeGlobalOptionsFromCreateRequest(pluginReq *PluginRequest) error {
	log.Trace(">>>>> removeGlobalOptionsFromCreateRequest called")
	defer log.Trace("<<<<< removeGlobalOptionsFromCreateRequest")
	for _, keyToRemove := range listOfCreateKeysToRemove {
		if _, ok := pluginReq.Opts[keyToRemove]; ok {
			delete(pluginReq.Opts, keyToRemove)
		}
	}
	return nil
}
