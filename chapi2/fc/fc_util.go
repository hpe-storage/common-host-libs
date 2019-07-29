// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package fc

import (
	log "github.com/hpe-storage/common-host-libs/logger"
)

// getAllFcHostPortWwn get all FC host port WWN's on the host
func getAllFcHostPortWwn() ([]string, error) {
	log.Infof(">>>>> getAllFcHostPortWwn called")
	defer log.Info("<<<<< getAllFcHostPortWwn")
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
