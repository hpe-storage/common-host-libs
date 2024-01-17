// Copyright 2019 Hewlett Packard Enterprise Development LP

package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/hpe-storage/common-host-libs/chapi"
	"github.com/hpe-storage/common-host-libs/connectivity"
	"github.com/hpe-storage/common-host-libs/dockerplugin/plugin"
	"github.com/hpe-storage/common-host-libs/dockerplugin/provider"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/model"
)

const (
	fsOwnerPattern = "^[\\d]+:[\\d]+$"
	fsModePattern  = "^[0-7]{1,4}$"
)

var (
	mountRequestsChan = make(chan string, defaultChannelCapacity)
)

//@APIVersion 1.0.0
//@Title  implement the Nimble Volume Driver Mount for docker
//@Description implement the /VolumeDriver.Mount Docker end point
//@Accept json
//@Resource /VolumeDriver.Mount
//@Success 200 MountResponse
//@Router /VolumeDriver.Mount [post]
//@BasePath http:/VolumeDriver.Mount
// nolint : gocyclo exceeded
// VolumeDriverMount implement the /VolumeDriver.Mount Docker end point
func VolumeDriverMount(w http.ResponseWriter, r *http.Request) {
	log.Debug("volumeDriverMount called")
	var mr MountResponse
	pluginReq, err := populateHostContextAndScope(r)
	if err != nil {
		mr = MountResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(mr)
		return
	}

	// obtain chapi client
	chapiClient, err := chapi.NewChapiClientWithTimeout(defaultCreationTimeout)
	if err != nil {
		mr = MountResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(mr)
		return
	}
	var respMount []*model.Mount
	volResp := &VolumeResponse{}

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

	if provider.IsHPECloudVolumesPlugin() {
		// initialize options
		pluginReq.Opts = make(map[string]interface{})
		// filter host networks to only include internal IP addresses specified by user
		err = filterHostNetworks(pluginReq)
		if err != nil {
			mr = MountResponse{Err: err.Error()}
			json.NewEncoder(w).Encode(mr)
			return
		}
	}

	//get containerProviderClient
	providerClient, err := provider.GetProviderClient()
	if err != nil {
		err = errors.New("unable to setup the container-provider client " + err.Error())
		resp := &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	//1. cleanup stale mounts which may existing if proper cleanup was not done
	err = cleanupStaleMounts(providerClient, chapiClient, pluginReq)
	if err != nil {
		resp := &DriverResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	//2. this method does poll to container provider to check if other hosts are attached until mountConflictDelay
	processMountConflictDelay(pluginReq.Name, providerClient, pluginReq, plugin.MountConflictDelay)

	mapMutex.Lock(pluginReq.Name)
	log.Debugf("taken lock for volume %s in Mount", pluginReq.Name)
	defer mapMutex.Unlock(pluginReq.Name)

	mountRequestsChan <- pluginReq.Name
	log.Tracef("taken channel for volume %s in mount with channel length :%d channel capacity :%d", pluginReq.Name, len(mountRequestsChan), cap(mountRequestsChan))
	defer unblockChannelHandler("mount", pluginReq.Name, mountRequestsChan)

	//3. container-provider /VolumeDriver.Mount called
	log.Debugf("/VolumeDriver.Mount for volume %s request=%+v", pluginReq.Name, pluginReq)
	_, err = providerClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.MountURI, Payload: &pluginReq, Response: &volResp, ResponseError: &volResp})
	log.Debugf("/VolumeDriver.Mount for volume %s response=%+v", pluginReq.Name, volResp)
	if volResp.Err != "" {
		if strings.Contains(volResp.Err, busyMount) {
			mr = MountResponse{Err: "another mount request creating filesystem on the volume, failing request."}
			json.NewEncoder(w).Encode(mr)
			return
		}
		mr = MountResponse{Err: volResp.Err}
		json.NewEncoder(w).Encode(mr)
		return
	}
	if err != nil {
		mr = MountResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(mr)
		return
	}
	volume := volResp.Volume
	log.Tracef("retrieved volume response from container provider for volume: %+v", volume)

	//4.  Get mounts from host
	err = chapiClient.GetMounts(&respMount, volume.SerialNumber)
	if err != nil && !(strings.Contains(err.Error(), "object was not found")) {
		mr = MountResponse{Err: err.Error()}
		json.NewEncoder(w).Encode(mr)
		return
	}
	mountPoint := plugin.MountDir + volume.Name
	// change the connection mode to manual for docker
	volume.ConnectionMode = manualMode
	//5. Attach and Mount the volume
	mr = mountVolumeOnHost(chapiClient, respMount, pluginReq.Host.UUID, volume, mountPoint)
	if mr.Err != "" {
		// if mount failed don't cleanup yet as this could be delayedCreate and we need to create filesystem on it
		// check for filesystem create options specified in the VolumeInfo
		if _, ok := volume.Status[delayedCreateOpt]; ok {
			mr = handleDelayedCreateAndMountFilesystem(chapiClient, volume, mountPoint)
		}
		// if there is an error in delayedCreateMount or existing mount, handle cleanup on failure
		if mr.Err != "" {
			// cleanup failed mount workflow
			log.Errorf("mount response error %s", mr.Err)
			err = cleanupMountFailure(chapiClient, volume, mountPoint, pluginReq)
			if err != nil {
				log.Errorf("unable to cleanup device for volume %v and mounpoint %s. err :(%s)", volume, mountPoint, err.Error())
			}
		}
	}
	//always try to cleanup the filesystem metadata on the volume when there is no error on mount
	if mr.Err == "" {
		if _, ok := volume.Status[delayedCreateOpt]; ok {
			err := removeDelayedCreateMetadata(pluginReq, volume)
			// if the metadata update failed don't treat this as an error as next node will take care of it
			if err != nil {
				log.Tracef(err.Error())
			}
		}
	}

	log.Infof("%s: request=(%+v) response=(%+v)", provider.MountURI, pluginReq, mr)
	json.NewEncoder(w).Encode(mr)
	return
}

func getStringSliceParam(paramName string, opts map[string]interface{}) (strSlice []string, err error) {
	prefix := "getStringParam"
	if _, ok := opts[paramName]; !ok {
		return nil, fmt.Errorf("%s: param name %s key not found in request", prefix, paramName)
	}

	switch value := opts[paramName].(type) {
	case []interface{}:
		for _, d := range value {
			strSlice = append(strSlice, strings.TrimSpace(fmt.Sprintf("%v", d)))
		}
		return strSlice, nil
	case string:
		return []string{strings.TrimSpace(value)}, nil
	default:
		return nil, fmt.Errorf("param name:%v is not a slice.  value:%v kind:%s type:%s", paramName, opts[paramName], reflect.TypeOf(opts[paramName]).Kind(), reflect.TypeOf(opts[paramName]))
	}
}

func isIPAddress(host string) bool {
	parts := strings.Split(host, ".")

	if len(parts) < 4 {
		return false
	}

	for _, x := range parts {
		if i, err := strconv.Atoi(x); err == nil {
			if i < 0 || i > 255 {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

// filter only required networks from current node from user input(cloud volumes)
func filterHostNetworks(pluginReq *PluginRequest) error {
	log.Tracef(">>>>>> filterHostNetworks")
	defer log.Tracef("<<<<< filterHostNetworks")

	// fetch entries from volume-driver.json
	err := populateVolCreateOptions(pluginReq)
	if err != nil {
		log.Errorf("%s failed to add create options from config file using defaults", err.Error())
	}

	// verify if interfaces are specified by user
	if _, ok := pluginReq.Opts["initiators"]; !ok {
		return errors.New("initiators are not specified in the volume-driver.json file for mount request")
	}

	initiators, err := getStringSliceParam("initiators", pluginReq.Opts)
	if err != nil {
		return err
	}

	index := 0
	for _, initiator := range initiators {
		// ignore unwanted networks based on user input
		for _, network := range pluginReq.Host.NetworkInterfaces {
			// check if initiator ip addresses or interfaces are provided to match with current node
			if (isIPAddress(initiator) && network.AddressV4 == initiator) || network.Name == initiator {
				pluginReq.Host.NetworkInterfaces[index] = network
				log.Debugf("matched filtered network %s, ip %s", network.Name, network.AddressV4)
				index++
			}
		}
	}
	// trim unwanted network interfaces
	pluginReq.Host.NetworkInterfaces = pluginReq.Host.NetworkInterfaces[:index]

	return nil
}

// handleDelayedCreateAndMountFilesystem the exception workflow on a failed mount to create a filesystem and mount it if the create fs metadata is present
func handleDelayedCreateAndMountFilesystem(chapiClient *chapi.Client, volume *model.Volume, mountPoint string) (mr MountResponse) {
	log.Tracef(">>>>> handleDelayedCreateAndMountFilesystem called with mountPoint %s and volume (%+v) ", mountPoint, volume)
	defer log.Tracef("<<<<< handleDelayedCreateAndMountFilesystem for volume %+v", volume)
	fsType, ok := volume.Status[model.FsCreateOpt]
	if !ok {
		// fail safe. the filesystem should always be present
		return MountResponse{Err: "no filesystem or filesystem options present"}
	}

	// if there is an error create filesystem and mount it from fsType retrieved
	log.Tracef("initiating create filesystem %s for volume (%s)", fsType, volume.Name)
	volume.MountPoint = mountPoint
	//1. attach device
	devices, err := chapiClient.AttachDevice([]*model.Volume{volume})
	if err != nil {
		log.Tracef(err.Error())
		return MountResponse{Err: err.Error()}
	}
	if len(devices) == 0 {
		return MountResponse{Err: fmt.Errorf("no device found for volume %s to create filesystem %s", volume.Name, fsType).Error()}
	}
	//2. create filesystem
	//make sure volume.Mountpoint is populated
	volume.MountPoint = plugin.MountDir + volume.Name
	err = chapiClient.SetupFilesystemAndPermissions(devices[0], volume, fsType.(string))
	if err != nil {
		log.Tracef(err.Error())
		return MountResponse{Err: err.Error()}
	}
	// the filesystem is already mounted so just return from here
	return MountResponse{MountPoint: mountPoint, Err: ""}
}

func removeMountConflictMetadata(containerProviderClient *connectivity.Client, pluginReq *PluginRequest, volName string) error {
	log.Tracef(">>> removeMountConflictMetadata called for %s", volName)
	defer log.Tracef("<<<< removeMountConflictMetadata")
	pluginReq.Opts = make(map[string]interface{})
	//reset mountConflict to 0
	pluginReq.Opts["mountConflictDelay"] = "0"
	var cr *CreateResponse
	_, err := containerProviderClient.DoJSON(&connectivity.Request{Action: "POST", Path: provider.UpdateURI, Payload: &pluginReq, Response: &cr, ResponseError: &cr})
	if err != nil {
		err = fmt.Errorf("unable to remove mountConflictDelay from volume metadata (%s)", err.Error())
		return err
	}
	return nil
}

func removeDelayedCreateMetadata(pluginReq *PluginRequest, volume *model.Volume) error {
	log.Tracef("removeDelayedCreateMetadata to remove delayedCreate opt for %s", volume.Name)
	pluginReq.Opts = make(map[string]interface{})
	pluginReq.Opts[delayedCreateOpt] = false
	// remove update of fileystem as it would use the default ones which we don't want
	if _, ok := pluginReq.Opts["filesystem"]; ok {
		delete(pluginReq.Opts, "filesystem")
	}
	var cr *CreateResponse

	client, err := provider.GetProviderClient()
	if err != nil {
		return err
	}
	_, err = client.DoJSON(&connectivity.Request{Action: "POST", Path: provider.UpdateURI, Payload: &pluginReq, Response: &cr, ResponseError: nil})
	if err != nil {
		err = fmt.Errorf("unable to remove delayedCreate from volume metadata (%s)", err.Error())
		return err
	}
	return nil
}

func mountVolumeOnHost(chapiClient *chapi.Client, respMount []*model.Mount, hostID string, volume *model.Volume, mountPoint string) (mr MountResponse) {
	log.Tracef(">>>>>> mountVolumeOnHost called with mountPoint %s and respMount %+v", mountPoint, respMount)
	defer log.Tracef("<<<<< mountVolumeOnHost")
	for _, mts := range respMount {
		if mts.Mountpoint == mountPoint && mts.Device != nil && mts.Device.State != model.FailedState.String() && mts.Device.State != model.LunIDConflict.String() {
			log.Trace("Mount of volume found")
			return MountResponse{MountPoint: mts.Mountpoint, Err: ""}
		}
	}
	log.Tracef("No mounts found for volume %s on host side, perform mount on the host", volume.Name)

	err := chapiClient.AttachAndMountDevice(volume, plugin.MountDir+volume.Name)
	if err != nil {
		return MountResponse{Err: err.Error()}
	}
	return MountResponse{MountPoint: mountPoint, Err: ""}
}

// perform mount cleanup
//nolint: gocyclo
func cleanupMountFailure(chapiClient *chapi.Client, volume *model.Volume, mountPoint string, pluginReq *PluginRequest) error {
	log.Tracef("cleanupMountFailure called for serialNumber %s and mountPoint %s", volume.SerialNumber, mountPoint)
	//1. retrieve the device from volume
	device, err := chapiClient.GetDeviceFromVolume(volume)
	if err != nil {
		log.Errorf("unable to get device to cleanup after mount failure, err %s", err.Error())
		// continue as we still need to remove mount-id's added to volume metadata etc
	}

	//2. delete the mount point
	log.Tracef("removing the mount point %s", mountPoint)
	err = os.RemoveAll(mountPoint)
	if err != nil {
		return err
	}

	//3. peform cleanup on the container provider
	//get containerProviderClient
	client, err := provider.GetProviderClient()
	if err != nil {
		return err
	}
	// container-provider /VolumeDriver.Unmount called
	volResp := &VolumeUnmountResponse{}
	_, err = client.DoJSON(&connectivity.Request{Action: "POST", Path: provider.UnmountURI, Payload: &pluginReq, Response: &volResp, ResponseError: nil})
	if err != nil {
		return err
	}
	if volResp.Message != donotUnmount {
		if device != nil {
			err := chapiClient.OfflineDevice(device)
			if err != nil {
				// ignore errors and proceed with cleaning mount entries and offline device
				err = fmt.Errorf("unable to offline device %s after mount failure %s ", device.MpathName, err.Error())
				log.Errorf(err.Error())
			}
		}
		//call Nimble.Detach
		err = nimbleDetach(volume, pluginReq)
		if err != nil {
			log.Errorf("unable to detach nimble volume %s", err.Error())
		}
		if device != nil {
			// detach only if donoUnmount is false
			err = chapiClient.DeleteDevice(device)
			if err != nil {
				// ignore errors and proceed with cleaning mount entries and delete device nevertheless
				err = fmt.Errorf("unable to delete device %s after mount failure %s ", device.MpathName, err.Error())
				log.Errorf(err.Error())
			}
		}
	} else {
		log.Tracef("volume mounted by other containers. skippping volume remove and cleanup.")
	}
	return nil
}

func getFileSystemTypeFromRequest(pluginReq *PluginRequest) string {
	log.Trace("retrieving filesystemType from request", pluginReq.Opts)
	fsType, found := pluginReq.Opts[model.FsCreateOpt].(string)
	if !found || strings.TrimSpace(fsType) == "" {
		fsType = "xfs"
	}
	return fsType
}

func getFileSystemModeAndOwnerFromRequest(pluginReq *PluginRequest) (string, string, error) {
	log.Trace("retrieving filesystemType from request", pluginReq.Opts)
	fsMode, found := pluginReq.Opts[model.FsModeOpt].(string)
	if !found || strings.TrimSpace(fsMode) == "" {
		fsMode = ""
	}
	if fsMode != "" && !fsModeRegexp.MatchString(fsMode) {
		// invalid fsMode
		return "", "", fmt.Errorf("invalid fsMode (%s) specified for filesystem", fsMode)
	}
	fsOwner, found := pluginReq.Opts[model.FsOwnerOpt].(string)
	if !found || strings.TrimSpace(fsOwner) == "" {
		fsOwner = ""
	}
	if fsOwner != "" && !fsOwnerRegexp.MatchString(fsOwner) {
		// invalid fsOwner
		return "", "", fmt.Errorf("invalid fsOwner (%s) specified for filesystem", fsOwner)
	}
	log.Tracef("fsMode (%s) fsOwner (%s)", fsMode, fsOwner)
	return fsMode, fsOwner, nil
}

func isValidFilesystem(pluginReq *PluginRequest) bool {
	log.Tracef("isValidFilesystem called")
	val, ok := pluginReq.Opts["filesystem"]
	if !ok {
		// no filesystem passed in the cli return true as it will use the defaults
		return true
	}
	for _, v := range plugin.SupportedFileSystems {
		if v == strings.ToLower(val.(string)) {
			return true
		}
	}
	return false
}

// Unmount stale mounts if all the below conditions are met
// 1. mount point is found for the volume
// 2. device is still attached.
// 3. all the scsi paths are in failed state
// 4. LUN Unit Not Supported error received on inquiry on any of the failed paths
// nolint: gocyclo
func cleanupStaleMounts(containerProviderClient *connectivity.Client, chapiClient *chapi.Client, pluginReq *PluginRequest) (err error) {
	log.Debugf(">>>>>> cleanupStaleMounts called for %s", pluginReq.Name)
	defer log.Debugf("<<<<<< cleanupStaleMounts")
	// retrieve the volumeInfo from container provider
	var respMount []*model.Mount
	volumeInfo, _ := getVolumeInfo(containerProviderClient, pluginReq)
	if volumeInfo == nil {
		return fmt.Errorf("unable to find volume %s, failing request", pluginReq.Name)
	}
	log.Tracef("volumeInfo is %+v", volumeInfo)
	// get the mounts of the volumes's serial number
	_ = chapiClient.GetMounts(&respMount, volumeInfo.SerialNumber)
	if respMount == nil || len(respMount) == 0 {
		log.Tracef("no existing stale mounts found for volume %s, continue with mount", volumeInfo.Name)
		return
	}
	// check if all paths of the device are failed else just return
	log.Tracef("previous mount exists at %s. verifying if it is stale mount", respMount[0].Mountpoint)
	device, _ := chapiClient.GetDeviceFromVolume(volumeInfo)
	if device == nil {
		return fmt.Errorf("unable to retrieve device for existing mounts %+v for volume %s", respMount, volumeInfo.Name)
	}
	log.Tracef("device obtained is %+v", device)
	// safe condition for unmount
	// 1. the volume is not in use
	// 2. the volume is not connected to this iscsi/fc host
	if !volumeInfo.InUse || (!isCurrentHostAttachedIscsi(volumeInfo, pluginReq) && !isCurrentHostAttachedFC(volumeInfo, pluginReq)) {
		log.Tracef("device state is %s, execute unmount on existing mounts for device %v %s", device.State, volumeInfo.InUse, device.MpathName)
		// check if volume is not in use or in use by a different host
		// iterate through all the mounts and unmount
		for _, mount := range respMount {
			// safe to unmount (best effort)
			log.Debugf("performing an unmount for %s on %+v", volumeInfo.Name, mount)
			var rspMount *model.Mount
			errMsg := chapiClient.Unmount(mount, rspMount)
			if errMsg != nil && (!strings.Contains(strings.ToLower(errMsg.Error()), "not mounted") ||
				!strings.Contains(strings.ToLower(errMsg.Error()), "no such file or directory")) {
				log.Errorf("unable to cleanly unmount %s error=:%s", mount.Mountpoint, err.Error())
				return errMsg
			}
			// best effort to remove the device only if target scope is group (i.e GST or FC device)
			if volumeInfo.AccessProtocol == "fc" || volumeInfo.TargetScope == model.GroupScope.String() {
				chapiClient.DeleteDevice(device)
			}
		}
	}
	return
}

/* processMountConflictDelay
   The method checks the volume info to check if it is currently inUse. Also fetches the iscsi / fc sessions
   If the volume is inUse, we poll every tick (5 secs) to check if the volume has iscsi/fc sessions for the current host.
   Eventually after timeout (mountConflictDelay) we return
*/
//nolint: gocyclo
func processMountConflictDelay(volName string, containerProviderClient *connectivity.Client, pluginReq *PluginRequest, mountConflictDelay int) {
	log.Tracef(">>>>> processMountConflictDelay called for %s with a timeout of %d seconds", volName, mountConflictDelay)
	defer log.Tracef("<<<<<< processMountConflictDelay")
	tick := time.Tick(5 * time.Second)
	timeout := time.After(time.Duration(mountConflictDelay) * time.Second)
	var isCurrentHostAttached bool

	volume, err := nimbleGetVolumeInfo(containerProviderClient, pluginReq)
	// Error from nimbleGetVolumeInfo(), we should bail
	if err != nil {
		log.Tracef("unable to get volume information for %s. err=%s", volName, err.Error())
		return
	}
	if !volume.InUse {
		log.Infof("volume is not inUse %s. Returning.", volName)
		return
	}

	// Keep trying until we're timed out or got a result or got an error
	try := 0
	for {
		select {
		// Got a timeout! return
		case <-timeout:
			log.Infof("mountConflictDelay timeout occurred after %d seconds for %s. Returning", mountConflictDelay, volName)
			// best effort to reset the mountConflictDelay on the array to 0 so that we don't process mountconflict delay there
			removeMountConflictMetadata(containerProviderClient, pluginReq, volName)

			return
		// Got a tick, we should check on nimbleGetVolumeInfo()
		case <-tick:
			try++
			trySeconds := try * 5 // try times the tick
			var volume *model.Volume
			var err error

			volume, err = nimbleGetVolumeInfo(containerProviderClient, pluginReq)
			// Error from nimbleGetVolumeInfo(), we should bail
			if err != nil {
				log.Tracef("%d / %d seconds: unable to get volume information for %s, err=%s Continuing.", trySeconds, mountConflictDelay, volName, err.Error())
				continue
			}

			if !volume.InUse {
				log.Infof("%d / %d seconds: volume is not inUse %s. Returning.", trySeconds, mountConflictDelay, volName)
				return
			}

			// reset the values of other hosts attached to false on each tick
			isCurrentHostAttached = false
			// if the volume is inUse and has other initiators connected to it then continue with mountConflictDelay
			if len(volume.FcSessions) != 0 {
				isCurrentHostAttached = isCurrentHostAttachedFC(volume, pluginReq)
			} else if len(volume.IscsiSessions) != 0 {
				isCurrentHostAttached = isCurrentHostAttachedIscsi(volume, pluginReq)
			}

			// ideally we should not reach this condition but if we do, we will continue with mount
			if isCurrentHostAttached {
				log.Tracef("%d / %d seconds: current host is attached to the volume %s. Returning.", trySeconds, mountConflictDelay, volume.Name)
				return
			}

			log.Infof("%d / %d seconds: volume %s is attached to other hosts. Continuing.", trySeconds, mountConflictDelay, volName)

		}
	}
}

func isCurrentHostAttachedIscsi(volume *model.Volume, pluginReq *PluginRequest) bool {
	log.Tracef(">>>>> isCurrentHostAttachedIscsi called for %s", volume.Name)
	defer log.Trace("<<<<< isCurrentHostAttachedIscsi")

	if pluginReq.Host == nil || pluginReq.Host.Initiators == nil {
		log.Infof("no host initiators found to validate the Iscsi sessions for %s", volume.Name)
		return false
	}
	//initialize host iscsi initiators
	var iscsiInits []string
	for _, initiator := range pluginReq.Host.Initiators {
		if initiator.Type == "iscsi" {
			for _, iscsiInitiator := range initiator.Init {
				iscsiInits = append(iscsiInits, iscsiInitiator)
			}
		}
	}

	for _, iscsiSession := range volume.IscsiSessions {
		for _, iscsiInit := range iscsiInits {
			if strings.TrimSpace(iscsiSession.InitiatorNameStr()) == strings.TrimSpace(iscsiInit) {
				log.Debugf("host iscsi initiator %s matched volume iscsi session", iscsiInit)
				return true
			}
		}
		if iscsiSession.InitiatorIP != "" {
			for _, network := range pluginReq.Host.NetworkInterfaces {
				if strings.TrimSpace(iscsiSession.InitiatorIP) == strings.TrimSpace(network.AddressV4) {
					log.Debugf("host iscsi initiator %s matched volume iscsi connection", network.AddressV4)
					return true
				}
			}
		}
	}
	return false
}

func isCurrentHostAttachedFC(volume *model.Volume, pluginReq *PluginRequest) bool {
	log.Tracef(">>>>> isCurrentHostAttachedFC called for %s", volume.Name)
	defer log.Trace("<<<<< isCurrentHostAttachedFC")

	if pluginReq.Host == nil || pluginReq.Host.Initiators == nil {
		log.Infof("no host initiators found to validate the Fibre Channel sessions for %s", volume.Name)
		return false
	}
	//initialize host fc initiators
	var fcInits []string
	for _, initiator := range pluginReq.Host.Initiators {
		if initiator.Type == "fc" {
			for _, fcInitiator := range initiator.Init {
				fcInits = append(fcInits, fcInitiator)
			}
		}
	}

	for _, fcSession := range volume.FcSessions {
		for _, fcInit := range fcInits {
			if strings.TrimSpace(strings.Replace(fcSession.InitiatorWwpnStr(), ":", "", -1)) == strings.TrimSpace(fcInit) {
				log.Infof("host initiator %s matched volume FC sessions %s", fcInit, fcSession)
				return true
			}
		}
	}
	return false
}
