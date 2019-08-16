// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package fc

import (
	"fmt"

	"github.com/hpe-storage/common-host-libs/chapi2/model"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/windows/wmi"
)

// getAllFcHostPorts get all the FC host port details on the host
func getAllFcHostPorts() (hostPorts []*model.FcHostPort, err error) {
	log.Trace(">>>>> GetAllFcHostPorts called")
	defer log.Trace("<<<<< GetAllFcHostPorts")

	// Enumerate the FC ports on this host
	var fcPorts []*wmi.MSFC_FibrePortHBAAttributes
	fcPorts, err = wmi.GetMSFC_FibrePortHBAAttributes()
	if err != nil {
		return nil, err
	}

	// Convert FC port array into array of FcHostPort
	for _, fcPort := range fcPorts {
		hostPort := new(model.FcHostPort)
		hostPort.PortWwn = wwnToString(fcPort.Attributes.PortWWN)
		hostPort.NodeWwn = wwnToString(fcPort.Attributes.NodeWWN)
		hostPorts = append(hostPorts, hostPort)
	}
	return hostPorts, nil
}

// rescanFcTarget rescans host ports for new Fibre Channel devices
func rescanFcTarget(lunID string) (err error) {
	// Unlike Linux, Windows does not have Target/LUN specific rescan capabilities so a synchronous
	// disk rescan is initiated and the lunID is ignored.
	return wmi.RescanDisks()
}

// wwnToString converts the given FC WWN into a string (e.g. "10:00:00:90:FA:73:6E:CA")
func wwnToString(wwn [8]uint8) string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X:%02X:%02X", wwn[0], wwn[1], wwn[2], wwn[3], wwn[4], wwn[5], wwn[6], wwn[7])
}
