// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package fc

import (
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	log "github.com/hpe-storage/common-host-libs/logger"
)

type FcPlugin struct {
}

func NewFcPlugin() *FcPlugin {
	return &FcPlugin{}
}

func (plugin *FcPlugin) DiscoverDevice(serial string, lunId string) error {
	return nil
}

func (plugin *FcPlugin) RescanAdapter(adapter string) error {
	return nil
}

// GetInitiators get all host fc initiators (port WWNs)
func (plugin *FcPlugin) GetFcInitiators() (*model.Initiator, error) {
	log.Trace(">>>>> GetFcInitiators")
	defer log.Trace("<<<<< GetFcInitiators")

	inits, err := getAllFcHostPortWwn()
	if err != nil {
		return nil, err
	}
	if len(inits) == 0 {
		// not a fc host
		return nil, nil
	}
	fcInit := &model.Initiator{
		AccessProtocol: "fc",
		Init:           inits,
	}
	return fcInit, nil
}

// GetAllFcHostPorts get all the FC host port details on the host
func (plugin *FcPlugin) GetAllFcHostPorts() ([]*model.FcHostPort, error) {
	log.Trace(">>>>> GetAllFcHostPorts")
	defer log.Trace("<<<<< GetAllFcHostPorts")
	return getAllFcHostPorts()
}

// GetAllFcHostPortWWN get all FC host port WWN's on the host
func (plugin *FcPlugin) GetAllFcHostPortWwn() ([]string, error) {
	log.Trace(">>>>> GetAllFcHostPortWwn called")
	defer log.Trace("<<<<< GetAllFcHostPortWwn")
	hostPorts, err := getAllFcHostPorts()
	if err != nil {
		return nil, err
	}
	if len(hostPorts) == 0 {
		return nil, nil
	}
	var inits []string
	for _, hostPort := range hostPorts {
		inits = append(inits, hostPort.PortWwn)
	}
	return inits, nil
}

// RescanFcTarget rescans host ports for new Fibre Channel devices
func (plugin *FcPlugin) RescanFcTarget(lunID string) error {
	log.Tracef(">>>>> RescanFcTarget called with lun id %s", lunID)
	defer log.Trace("<<<<< RescanFcTarget")
	return rescanFcTarget(lunID)
}
