// Copyright 2019 Hewlett Packard Enterprise Development LP

package handler

import (
	"encoding/json"
	"github.com/hpe-storage/common-host-libs/chapi"
	log "github.com/hpe-storage/common-host-libs/logger"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

var (
	// HostContextCache cache used to store Host information
	hostContextCache    *Host
	hostCacheExpiration = time.Duration(60) // cache expiration time for 60 minutes
	hostCacheLock       sync.Mutex
	maxTries            = 3
)

type dockerInfoResponse struct {
	ID string `json:"id,omitempty"`
}

// SetupChapiClient will set the chapiclient for the listener on given socket path
func SetupChapiClient(chapidSocket string) {
	hostContextCache = nil
}

//SetLogLevel :
func SetLogLevel() (err error) {
	return nil
}

//SetMountDir :
func SetMountDir() (err error) {
	return nil
}

// populate the hostContext to the plugin request
func getHostContext(body io.ReadCloser) (*PluginRequest, error) {
	log.Trace("getHostContext called")
	pref := make(map[string]interface{})
	var hostCxt *Host
	var pluginReq *PluginRequest

	//Unmarshal the request into PluginRequest
	reqBuf, err := ioutil.ReadAll(body)
	log.Trace("Body :", string(reqBuf))
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(reqBuf, &pluginReq)
	if err != nil {
		return nil, err
	}
	if hostContextCache != nil {
		log.Trace("HostContextCache present. Reading from it")
		hostCacheLock.Lock()
		defer hostCacheLock.Unlock()
		hostCxt = hostContextCache
		log.Trace("Successfully read from cache.")
		pluginReq.Host = hostCxt
		pluginReq.Preferences = pref
		return pluginReq, nil
	}
	// if error to read from cache. Build the context
	if hostContextCache == nil {
		log.Trace("Cache absent.Building the Host context cache")
	}
	hostCxt, err = buildHostContext(body)
	if err != nil {
		return nil, err
	}
	pluginReq.Host = hostCxt
	pluginReq.Preferences = pref

	// Try saving the context into cache
	err = saveHostContextInCache(hostCxt)
	if err != nil {
		log.Trace("Unable to save the hostContext in cache, but continue the operation")
		return pluginReq, nil
	}
	return pluginReq, nil
}

func saveHostContextInCache(hostCxt *Host) error {
	log.Trace("saveHostContextInCache called ")
	hostContextCache = hostCxt

	log.Trace("Starting cache expiration timer for ", time.Minute*hostCacheExpiration)
	hostContextCacheTimer := time.NewTimer(time.Minute * hostCacheExpiration)
	go func() {
		<-hostContextCacheTimer.C
		log.Trace("HostContextCacheTimer expired after ", time.Minute*hostCacheExpiration, ". Invalidate the cache")
		if hostContextCache != nil {
			//invalidate the host Context cache
			err := invalidateHostContextCache()
			if err != nil {
				log.Trace("err to delete the cache :", err.Error())
			}
		}
		return
	}()
	return nil
}

// invalidateHostContextCache clear host context cache
func invalidateHostContextCache() error {
	log.Trace("invalidateHostContextCache called")
	hostCacheLock.Lock()
	defer hostCacheLock.Unlock()
	hostContextCache = nil
	return nil
}

func buildHostContext(body io.ReadCloser) (*Host, error) {
	log.Trace("buildHostContext called")
	var hostContext *Host
	// obtain chapi client
	chapiClient, err := chapi.NewChapiClient()
	if err != nil {
		return nil, err
	}
	networks, err := chapiClient.GetNetworks()
	if err != nil {
		return nil, err
	}
	initiators, err := chapiClient.GetInitiators()
	if err != nil {
		return nil, err
	}
	host, err := chapiClient.GetHostName()
	if err != nil {
		return nil, err
	}
	hostID, err := chapiClient.GetHostID()
	if err != nil {
		return nil, err
	}
	// populate host context
	hostContext = &Host{
		Initiators:        initiators,
		NetworkInterfaces: networks,
	}
	hostContext.Domain = host.Domain
	hostContext.Name = host.Name
	hostContext.AccessProtocol = getHostProtocol()
	// populate both node ID and UUID as host ID
	// retaining both for legacy purposes as container-provider depends on them.
	// this will avoid getting nodeID through dockerd
	hostContext.UUID = hostID
	hostContext.NodeID = hostID
	log.Trace("Hostcontext :", hostContext)
	return hostContext, nil
}

func getHostProtocol() (protocol string) {
	protocol, ok := os.LookupEnv("PROTOCOL")
	if !ok || protocol == "" {
		// assume iscsi by default if not specified
		return "iscsi"
	}
	return protocol
}
