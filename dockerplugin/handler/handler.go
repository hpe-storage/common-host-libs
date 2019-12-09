// Copyright 2019 Hewlett Packard Enterprise Development LP

package handler

import (
	"github.com/hpe-storage/common-host-libs/concurrent"
	"github.com/hpe-storage/common-host-libs/dockerplugin/plugin"
	"github.com/hpe-storage/common-host-libs/dockerplugin/provider"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/model"
	"net/http"
	"regexp"
)

const (
	donotUnmount           = "donotunmount"
	destroyOnDetach        = "destroyondetach"
	manualMode             = "manual"
	busyMount              = "busymount"
	defaultChannelCapacity = 30
	// options
	delayedCreateOpt = "delayedCreate"
	volumeDirKey     = "volumeDir"
	inUseKey         = "inUse"
)

var (
	mapMutex      = concurrent.NewMapMutex()
	fsModeRegexp  = regexp.MustCompile(fsModePattern)
	fsOwnerRegexp = regexp.MustCompile(fsOwnerPattern)
)

//PluginRequest : Request routed for the plugin
type PluginRequest struct {
	Name        string                 `json:"name,omitempty"`
	Opts        map[string]interface{} `json:"opts,omitempty"`
	ID          string                 `json:"id,omitempty"`
	Host        *Host                  `json:"host,omitempty"`
	Preferences map[string]interface{} `json:"preference,omitempty"`
	Scope       bool                   `json:"scope,omitempty"`
	User        *provider.User         `json:"user,omitempty"`
	ReqID       string                 `json:"req_id,omitempty"`
}

//NimbleDetachRequest : Request to call detach on container provider
type NimbleDetachRequest struct {
	Volume *model.Volume  `json:"volume,omitempty"`
	Host   *Host          `json:"host,omitempty"`
	User   *provider.User `json:"user,omitempty"`
	ReqID  string         `json:"req_id,omitempty"`
}

//NimbleAttachRequest : Request to call attach on container provider
type NimbleAttachRequest struct {
	Volume *model.Volume  `json:"volume,omitempty"`
	Host   *Host          `json:"host,omitempty"`
	User   *provider.User `json:"user,omitempty"`
	ReqID  string         `json:"req_id,omitempty"`
}

// PluginActivate : stuct to parse plugin response from Array
type PluginActivate struct {
	Activate []string `json:"implements,omitempty"`
	Err      string   `json:"Err"`
}

// PluginCapability : get capabilities of the plugin
type PluginCapability struct {
	Capability *Scope `json:"capabilities,omitempty"`
	Err        string `json:"Err"`
}

// Scope : scope of the driver (local/global)
type Scope struct {
	Scope string `json:"scope,omitempty"`
}

//DriverResponse : Driver response struct
type DriverResponse struct {
	Err string `json:"Err"`
}

// HPEVolumeConfigResponse : Nimble Config response
type HPEVolumeConfigResponse struct {
	Options *HPEVolumeOptions `json:"hpevolumeConfig,omitempty"`
}

// HPEVolumeOptions :
type HPEVolumeOptions struct {
	GlobalOptions   map[string]string `json:"global"`
	DefaultOptions  map[string]string `json:"defaults"`
	OverrideOptions map[string]string `json:"overrides"`
}

//ListResponse : Volume response struct
type ListResponse struct {
	Volumes []*model.Volume `json:"volumes,omitempty"`
	Err     string          `json:"Err"`
}

//CreateResponse : Volume create response struct
type CreateResponse struct {
	Err     string          `json:"Err"`
	Volumes []*model.Volume `json:"volumes,omitempty"`
	Help    string          `json:"help"`
}

//VolumeResponse : Volume response struct
type VolumeResponse struct {
	Volume *model.Volume `json:"volume,omitempty"`
	Err    string        `json:"Err"`
}

//VolumeUnmountResponse : Volume unmount response
type VolumeUnmountResponse struct {
	VolumeResponse
	Message string `json:"message"`
}

//MountResponse : mount response for docker
type MountResponse struct {
	MountPoint string `json:"Mountpoint,omitempty"`
	Err        string `json:"Err"`
}

// ErrorResponse struct
type ErrorResponse struct {
	Info string
}

// VersionResponse struct
type VersionResponse struct {
	Version string `json:"Version,omitempty"`
	Err     string `json:"Err,omitempty"`
}

// Host is a duplicate of model.Host which references a Network defined in this package
type Host struct {
	UUID              string                    `json:"id,omitempty"`
	Name              string                    `json:"name,omitempty"`
	Domain            string                    `json:"domain,omitempty"`
	NodeID            string                    `json:"node_id,omitempty"`
	AccessProtocol    string                    `json:"access_protocol,omitempty"`
	NetworkInterfaces []*model.NetworkInterface `json:"networks,omitempty"`
	Initiators        []*model.Initiator        `json:"initiators,omitempty"`
	Version           string                    `json:"version,omitempty"`
}

// populate hostcontext into request and fetch user info if running in cloud vm
func preparePluginRequest(request *http.Request) (pluginReq *PluginRequest, err error) {
	pluginReq, err = populateHostContextAndScope(request)
	if err != nil {
		return nil, err
	}

	// Add user credentials for cloud/simplivity container-provider
	user, err := provider.GetProviderAccessKeys()
	if err != nil {
		return nil, err
	}
	if user != nil {
		pluginReq.User = user
	}

	return pluginReq, err
}

// read off the channel to consume the work when the specific handler is done with work allocated to it
func unblockChannelHandler(handler, volume string, channel chan string) {
	log.Tracef("unblockChannelHandler called for %s : %s", handler, volume)
	select {
	case <-channel:
		log.Debugf("returning from %s for %s channel length :%d, channel capacity :%d", handler, volume, len(channel), cap(channel))
	default:
		log.Debugf("default handler for %s for %s channel length :%d channel capacity :%d", handler, volume, len(channel), cap(channel))
	}
}

func populateHostContextAndScope(r *http.Request) (*PluginRequest, error) {
	log.Trace("populateHostContextAndScope called")
	scope := plugin.IsLocalScopeDriver()
	//Populate Host Context to the Plugin Request
	pluginReq, err := getHostContext(r.Body)
	if err != nil {
		return nil, err
	}
	pluginReq.Scope = scope
	// add host nlt version version
	pluginReq.Host.Version = plugin.Version
	log.Trace("host context in Plugin Req: ", pluginReq.Host, " Scope :", pluginReq.Scope)
	return pluginReq, nil
}
