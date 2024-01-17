// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package iscsi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/hpe-storage/common-host-libs/chapi2/model"
	log "github.com/hpe-storage/common-host-libs/logger"
	ping "github.com/sparrc/go-ping"
)

// ITNexusPingCheck takes an array of CHAPI2 initiator ports, and an array of target ports, and
// returns a map of IT nexus connections that can reach each other (e.g. ICMP ping test). Each IT
// nexus is pinged in parallel for maximum performance.  The returned map key is the initiator
// port while the map value is an array of target ports.
func ITNexusPingCheck(initiatorPorts []*model.Network, targetPorts []*model.TargetPortal, pingCount, pingInterval, pingTimeout int) (map[*model.Network][]*model.TargetPortal, error) {
	log.Tracef(">>>>> ITNexusPingCheck, pingCount=%v, pingInterval=%v, pingTimeout=%v", pingCount, pingInterval, pingTimeout)
	defer log.Traceln("<<<<< ITNexusPingCheck")

	// If the echo packet count is zero or negative, set the default ping count to 3
	if pingCount <= 0 {
		pingCount = 3
	}

	// If the echo ping interval is zero or negative, set the default interval as 10 msec
	if pingInterval <= 0 {
		pingInterval = 10
	}

	// If the echo ping timeout is zero or negative, set the default ping timeout to 250 msec
	if pingTimeout <= 0 {
		pingTimeout = 250
	}

	// Allocate an initial empty initiator/target nexus map
	itNexus := make(map[*model.Network][]*model.TargetPortal)

	// In order to avoid any duplicate IT nexus, we build a map of each checked IT nexus
	itChecked := make(map[string]bool)

	// In order to optimize the performance of this routine, we're going to ping each IT nexus in
	// parallel.  Here we allocate a mutex and a WaitGroup.  The WaitGroup is used to wait for our
	// collection of go routines to finish.
	var mux sync.Mutex
	var wg sync.WaitGroup

	// Randomly pick a 64-bit tracker value to ensure that each ICMP request can be uniquely
	// attached to an IT nexus
	icmpTracker := int64(rand.Uint64())

	// Loop through each initiator and target
	for _, initiatorPort := range initiatorPorts {
		for _, targetPort := range targetPorts {
			log.Tracef("Checking initiatorPort=%-15s, targetPort=%-15s", initiatorPort.AddressV4, targetPort.Address)

			// Skip IT nexus if it was already added to the return map
			key := initiatorPort.AddressV4 + "-" + targetPort.Address
			if itChecked[key] == true {
				continue
			}
			itChecked[key] = true

			// Increment the WaitGroup counter
			wg.Add(1)

			// Increment our ICMP tracker to ensure a unique value is used for each instance
			icmpTracker++

			// In a separate go routine, ping the initiator and target
			go func(initiatorPort *model.Network, targetPort *model.TargetPortal, tracker int64) {

				// Decrement the WaitGroup counter when the goroutine completes.
				defer wg.Done()

				// Allocate a new ICMP ping object
				pinger, err := ping.NewPinger(targetPort.Address)
				if err != nil {
					log.Errorf("NewPinger creation failure, err=%v", err)
					return
				}

				// Give this thread's ICMP ping object a unique tracker value
				pinger.Tracker = tracker

				// Send a "privileged" raw ICMP ping (required by Windows)
				pinger.SetPrivileged(true)

				// Ping target from initiatorPort
				pinger.Source = initiatorPort.AddressV4

				// Count tells pinger to stop after sending (and receiving) Count echo packets
				pinger.Count = pingCount

				// Interval is the wait time between each packet sent
				pinger.Interval = time.Duration(pingInterval) * time.Millisecond

				// Timeout specifies a timeout before ping exits, regardless of how many packets have been received
				pinger.Timeout = time.Duration((pingCount*pingTimeout)+((pingCount-1)*pingInterval)) * time.Millisecond

				// Perform the ping test; if we received any ICMP packet back, add the IT nexus to the return map
				pinger.Run()
				packetsRecv := pinger.Statistics().PacketsRecv
				if packetsRecv != 0 {
					log.Tracef("Matched IT nexus initiatorPort=%-15s, targetPort=%-15s, packetsRecv=%v", initiatorPort.AddressV4, targetPort.Address, packetsRecv)
					mux.Lock()
					itNexus[initiatorPort] = append(itNexus[initiatorPort], targetPort)
					mux.Unlock()
				}

			}(initiatorPort, targetPort, icmpTracker)
		}
	}

	// Wait for all the ping threads to exit
	wg.Wait()

	// Log and return IT nexus map
	logITNexusMap(model.ConnectTypePing, itNexus)
	return itNexus, nil
}

// ITNexusSubnetCheck takes an array of CHAPI2 initiator ports, and an array of target ports, and
// returns a map of IT nexus connections that could be made.  The returned map key is the initiator
// port while the map value is an array of target ports.
func ITNexusSubnetCheck(initiatorPorts []*model.Network, targetPorts []*model.TargetPortal) (map[*model.Network][]*model.TargetPortal, error) {
	log.Traceln(">>>>> ITNexusSubnetCheck")
	defer log.Traceln("<<<<< ITNexusSubnetCheck")

	// Allocate an initial empty initiator/target nexus map
	itNexus := make(map[*model.Network][]*model.TargetPortal)

	// In order to avoid any duplicate IT nexus, we build a map of each checked IT nexus
	itChecked := make(map[string]bool)

	// Loop through each initiator and target
	for _, initiatorPort := range initiatorPorts {
		for _, targetPort := range targetPorts {
			log.Tracef("Checking ipInitiator=%-15s, ipMask=%-15s, ipTarget=%-15s", initiatorPort.AddressV4, initiatorPort.MaskV4, targetPort.Address)

			// Convert the initiator, subnet mask, and target into 32-bit values.
			// NOTE:  We currently only support IPv4
			uint32Initiator, errInitiator := ipToUint32(initiatorPort.AddressV4)
			uint32SubnetMask, errSubnetMask := ipToUint32(initiatorPort.MaskV4)
			uint32Target, errTarget := ipToUint32(targetPort.Address)

			// If the IT nexus isn't supported, log results and skip
			if (errInitiator != nil) || (errSubnetMask != nil) || (errTarget != nil) {
				log.Tracef("Skipping IT nexus, initiator=%-15s, mask=%-15s, target=%-15s, errSrc=%v, errMask=%v, errDst=%v",
					initiatorPort.AddressV4, initiatorPort.MaskV4, targetPort.Address, errInitiator, errSubnetMask, errTarget)
				continue
			}

			// If the initiator and target are not in the same subnet, skip IT nexus
			if (uint32Initiator & uint32SubnetMask) != (uint32Target & uint32SubnetMask) {
				continue
			}

			// Skip IT nexus if it was already added to the return map
			key := initiatorPort.AddressV4 + "-" + targetPort.Address
			if itChecked[key] == true {
				continue
			}
			itChecked[key] = true

			// Add IT nexus to return map
			log.Tracef("Matched IT nexus, ipInitiator=%-15s, ipMask=%-15s, ipTarget=%-15s", initiatorPort.AddressV4, initiatorPort.MaskV4, targetPort.Address)
			itNexus[initiatorPort] = append(itNexus[initiatorPort], targetPort)
		}
	}

	// Log and return IT nexus map
	logITNexusMap(model.ConnectTypeSubnet, itNexus)
	return itNexus, nil
}

// logITNexusMap is used to dump the itNexus map to the log file
func logITNexusMap(connectType string, itNexus map[*model.Network][]*model.TargetPortal) {
	itNexusCount := 0
	for initiatorPort, targetPorts := range itNexus {
		for _, targetPort := range targetPorts {
			itNexusCount++
			log.Infof("itNexus match, connectType=%v, initiatorPort<->targetPort = %v<->%v", connectType, initiatorPort.AddressV4, targetPort.Address)
		}
	}
	if itNexusCount == 0 {
		log.Infof("No itNexus matches, connectType=%v", connectType)
	}
}

// ipToUint32 takes an IP string and returns an IPv4 32-bit address.  An error is returned if the
// IP string is not a valid IPv4 address.
func ipToUint32(ipString string) (uint32, error) {

	// Convert the IP string into an IP object
	ip := net.ParseIP(ipString)
	if ip == nil {
		return 0, fmt.Errorf("invalid IP address %v", ipString)
	}

	// Make sure it's an IPv4 address
	ip = ip.To4()
	if ip == nil {
		return 0, fmt.Errorf("invalid IPv4 address %v", ipString)
	}

	// Convert IPv4 address into a uint32 value
	var ipUint32 uint32
	if err := binary.Read(bytes.NewBuffer(ip), binary.BigEndian, &ipUint32); err != nil {
		return 0, err
	}

	// Return the IP address as a uint32
	return ipUint32, nil
}
