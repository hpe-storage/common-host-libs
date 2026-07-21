// Copyright 2019 Hewlett Packard Enterprise Development LP

package linux

import (
	"fmt"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/model"
	"github.com/hpe-storage/common-host-libs/util"
	"io/ioutil"
	"os"
	"strings"
)

const fcHostBasePath = "/sys/class/fc_host"
const fcHostPortNameFormat = "/sys/class/fc_host/host%s/port_name"
const fcHostNodeNameFormat = "/sys/class/fc_host/host%s/node_name"
const fcHostScanPathFormat = "/sys/class/scsi_host/host%s/scan"

// FcHostLIPNameFormat :
const FcHostLIPNameFormat = "/sys/class/fc_host/host%s/issue_lip"

const fcRemotePortBasePath = "/sys/class/fc_remote_ports"
const fcRemotePortNameFormat = "/sys/class/fc_remote_ports/%s/port_name"

// GetHostPort get the host port details for given host number from H:C:T:L of device
func GetHostPort(hostNumber string) (hostPort *model.FcHostPort, err error) {
	hostPath := fmt.Sprintf(fcHostPortNameFormat, hostNumber)
	portName, err := util.FileReadFirstLine(hostPath)
	if err != nil {
		log.Warnf("unable to get port WWN for host %s, error %s", hostNumber, err.Error())
		return nil, err
	}
	log.Tracef("got port WWN %s for host %s", portName, hostNumber)
	hostPath = fmt.Sprintf(fcHostNodeNameFormat, hostNumber)
	nodeName, err := util.FileReadFirstLine(hostPath)
	if err != nil {
		log.Warnf("unable to get node WWN for host %s, error %s", hostNumber, err.Error())
		return nil, err
	}
	log.Tracef("got node WWN %s for host %s", nodeName, hostNumber)
	hostPort = &model.FcHostPort{HostNumber: hostNumber, NodeWwn: strings.TrimPrefix(nodeName, "0x"), PortWwn: strings.TrimPrefix(portName, "0x")}
	return hostPort, nil
}

// GetAllFcHostPorts get all the FC host port details on the host
func GetAllFcHostPorts() (hostPorts []*model.FcHostPort, err error) {
	log.Tracef("GetAllFcHostPorts called")
	var hostNumbers []string
	args := []string{"-1", fcHostBasePath}
	exists, _, err := util.FileExists(fcHostBasePath)
	if !exists {
		log.Warn("no fc adapters found on the host")
		return nil, nil
	}

	out, _, err := util.ExecCommandOutput("ls", args)
	if err != nil {
		log.Warnf("unable to get list of host fc ports, error %s", err.Error())
		return nil, err
	}

	hostNumbers = strings.Split(out, "\n")
	if len(hostNumbers) == 0 {
		log.Errorf("no fc adapters found on the host")
		return nil, nil
	}

	for _, host := range hostNumbers {
		if host != "" {
			hostPort, err := GetHostPort(strings.TrimPrefix(host, "host"))
			if err != nil {
				log.Warnf("unable to get details of fc host port %s, error %s", host, err.Error())
				continue
			}
			hostPorts = append(hostPorts, hostPort)
		}
	}
	return hostPorts, nil
}

// GetAllFcHostPortWWN get all FC host port WWN's on the host
func GetAllFcHostPortWWN() (portWWNs []string, err error) {
	log.Tracef("GetAllFcHostPortWWN called")
	hostPorts, err := GetAllFcHostPorts()
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
//nolint: dupl
func RescanFcTarget(lunID string) (err error) {
	log.Tracef(">>> RescanFcTarget called on lunID %s", lunID)
	defer log.Traceln("<<< RescanFcTarget")

	// Get the list of FC hosts to rescan
	fcHosts, err := GetAllFcHostPorts()
	if err != nil {
		return err
	}
	for _, fcHost := range fcHosts {
		// perform rescan for all devices
		fcHostScanPath := fmt.Sprintf(fcHostScanPathFormat, fcHost.HostNumber)
		isFCHostScanPathExists, _, _ := util.FileExists(fcHostScanPath)
		if !isFCHostScanPathExists {
			log.Tracef("fc host scan path %s does not exist", fcHostScanPath)
			continue
		}
		var err error
		if lunID == "" {
			// fallback to the generic host rescan
			err = ioutil.WriteFile(fcHostScanPath, []byte("- - -"), 0644)
			if err != nil {
				log.Debugf("error writing to file %s : %s", fcHostScanPath, err.Error())
			}
		} else {
			log.Tracef("\n SCANNING fc lun id %s", lunID)
			err = ioutil.WriteFile(fcHostScanPath, []byte("- - "+lunID), 0644)
			if err != nil {
				log.Debugf("error writing to file %s : %s", fcHostScanPath, err.Error())
			}
		}
		if err != nil {
			log.Errorf("unable to rescan for fc devices on host port :%s lun: %s err %s", fcHost.HostNumber, lunID, err.Error())
			return err
		}
	}
	return nil
}

// GetFcHostNumbersForTargetWwpns returns FC host numbers that have remote port
// connections to the specified target WWPNs. This allows scoping SCSI rescans
// to only the hosts connected to a specific storage array.
func GetFcHostNumbersForTargetWwpns(targetWwpns []string) ([]string, error) {
	log.Tracef(">>> GetFcHostNumbersForTargetWwpns called with targets %v", targetWwpns)
	defer log.Trace("<<< GetFcHostNumbersForTargetWwpns")

	if len(targetWwpns) == 0 {
		return nil, fmt.Errorf("no target WWPNs provided")
	}

	// Build a set of target WWPNs for fast lookup (normalize to lowercase, strip 0x)
	targetSet := make(map[string]bool)
	for _, t := range targetWwpns {
		normalized := strings.ToLower(strings.TrimPrefix(t, "0x"))
		targetSet[normalized] = true
	}

	// List all rport directories under /sys/class/fc_remote_ports/
	entries, err := os.ReadDir(fcRemotePortBasePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read %s: %s", fcRemotePortBasePath, err.Error())
	}

	hostSet := make(map[string]bool)
	for _, entry := range entries {
		name := entry.Name()
		// rport directories are named rport-H:B-T (e.g., rport-5:0-0)
		if !strings.HasPrefix(name, "rport-") {
			continue
		}

		// Read the remote port's port_name (target WWPN)
		portNamePath := fmt.Sprintf(fcRemotePortNameFormat, name)
		portName, err := util.FileReadFirstLine(portNamePath)
		if err != nil {
			log.Debugf("unable to read port_name for %s: %s", name, err.Error())
			continue
		}
		normalizedPort := strings.ToLower(strings.TrimPrefix(portName, "0x"))

		if !targetSet[normalizedPort] {
			continue
		}

		// Extract host number from rport-H:B-T
		// The H part is after "rport-" and before the first ":"
		parts := strings.TrimPrefix(name, "rport-")
		colonIdx := strings.Index(parts, ":")
		if colonIdx < 0 {
			continue
		}
		hostNum := parts[:colonIdx]
		hostSet[hostNum] = true
		log.Infof("FC remote port %s (wwpn %s) maps to host %s", name, normalizedPort, hostNum)
	}

	if len(hostSet) == 0 {
		return nil, fmt.Errorf("no FC hosts found for target WWPNs %v", targetWwpns)
	}

	var hostNumbers []string
	for h := range hostSet {
		hostNumbers = append(hostNumbers, h)
	}
	log.Infof("FC hosts for target WWPNs: %v", hostNumbers)
	return hostNumbers, nil
}

// RescanFcHostsForLun rescans only the specified FC host adapters for the given LUN ID.
// This avoids disturbing unrelated volumes on other arrays sharing the same LUN ID.
func RescanFcHostsForLun(hostNumbers []string, lunID string) error {
	log.Tracef(">>> RescanFcHostsForLun called with hosts %v lun %s", hostNumbers, lunID)
	defer log.Trace("<<< RescanFcHostsForLun")

	for _, hostNum := range hostNumbers {
		fcHostScanPath := fmt.Sprintf(fcHostScanPathFormat, hostNum)
		isFCHostScanPathExists, _, _ := util.FileExists(fcHostScanPath)
		if !isFCHostScanPathExists {
			log.Tracef("fc host scan path %s does not exist", fcHostScanPath)
			continue
		}
		var scanCmd string
		if lunID == "" {
			scanCmd = "- - -"
		} else {
			scanCmd = "- - " + lunID
		}
		log.Infof("scanning FC host %s for lun %s (path %s)", hostNum, lunID, fcHostScanPath)
		err := ioutil.WriteFile(fcHostScanPath, []byte(scanCmd), 0644)
		if err != nil {
			log.Errorf("unable to rescan FC host %s for lun %s: %s", hostNum, lunID, err.Error())
			return err
		}
	}
	return nil
}

// verifies if the scsi slaves are fc devices are not
func isFibreChannelDevice(slaves []string) bool {
	log.Tracef("isFibreChannelDevice called")
	// time.Sleep(time.Duration(1) * time.Second)
	for _, slave := range slaves {
		log.Tracef("handling path %s", slave)
		args := []string{"-l", "/sys/block/" + slave}
		out, _, _ := util.ExecCommandOutput("ls", args)
		if strings.Contains(out, "rport") {
			log.Tracef("%s is a FC device", slave)
			return true
		}
	}
	return false
}
