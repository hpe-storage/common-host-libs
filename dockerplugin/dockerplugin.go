// Copyright 2019 Hewlett Packard Enterprise Development LP

package dockerplugin

import (
	"github.com/gorilla/mux"
	"github.com/hpe-storage/common-host-libs/dockerplugin/handler"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/util"
	"net"
	"net/http"
)

var (
	// LogLevel represents plugin logging level, set info as default
	LogLevel = "info"
)

// NewRouter creates a new mux.Router
func NewRouter() *mux.Router {
	routes := []util.Route{
		util.Route{
			Name:        "Activate Plugin",
			Method:      "POST",
			Pattern:     "/Plugin.Activate",
			HandlerFunc: handler.ActivatePlugin,
		},
		util.Route{
			Name:        "Volume  list",
			Method:      "POST",
			Pattern:     "/VolumeDriver.List",
			HandlerFunc: handler.VolumeDriverList,
		},
		util.Route{
			Name:        "Volume Create",
			Method:      "POST",
			Pattern:     "/VolumeDriver.Create",
			HandlerFunc: handler.VolumeDriverCreate,
		},
		util.Route{
			Name:        "Volume Mount",
			Method:      "POST",
			Pattern:     "/VolumeDriver.Mount",
			HandlerFunc: handler.VolumeDriverMount,
		},
		util.Route{
			Name:        "Volume Remove",
			Method:      "POST",
			Pattern:     "/VolumeDriver.Remove",
			HandlerFunc: handler.VolumeDriverRemove,
		},
		util.Route{
			Name:        "Volume Driver Capabilities",
			Method:      "POST",
			Pattern:     "/VolumeDriver.Capabilities",
			HandlerFunc: handler.VolumeDriverCapabilities,
		},
		util.Route{
			Name:        "Volume Driver Get",
			Method:      "POST",
			Pattern:     "/VolumeDriver.Get",
			HandlerFunc: handler.VolumeDriverGet,
		},
		util.Route{
			Name:        "Volume Driver Path",
			Method:      "POST",
			Pattern:     "/VolumeDriver.Path",
			HandlerFunc: handler.VolumeDriverPath,
		},
		util.Route{
			Name:        "Volume Driver Unmount",
			Method:      "POST",
			Pattern:     "/VolumeDriver.Unmount",
			HandlerFunc: handler.VolumeDriverUnmount,
		},
		util.Route{
			Name:        "Volume Driver Update",
			Method:      "PUT",
			Pattern:     "/VolumeDriver.Update",
			HandlerFunc: handler.VolumeDriverUpdate,
		},
	}
	router := mux.NewRouter().StrictSlash(true)
	util.InitializeRouter(router, routes)
	return router
}

func runNimbledockerd(l net.Listener, m *mux.Router, c chan error) {
	log.Trace("Serving socket :", l.Addr().String())
	c <- http.Serve(l, m)
	// close the socket
	log.Tracef("closing the socket %v", l.Addr().String())
	l.Close()
}
