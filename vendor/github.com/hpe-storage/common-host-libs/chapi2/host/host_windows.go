// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package host

import (
	"encoding/binary"
	"net"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/hpe-storage/common-host-libs/chapi2/model"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/windows/shlwapi"
	"github.com/hpe-storage/common-host-libs/windows/wmi"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/sys/windows"
)

const (
	registryHostIDKey     = `SOFTWARE\Microsoft\Cryptography`
	registryHostIDValue   = `MachineGuid`
	registryHostIDDefault = `6f67b7d2-2bf2-4662-88c8-26e7274384e7`
)

var (
	hostId     string     // Enumerated host ID
	hostIdLock sync.Mutex // Host ID lock
)

//getNetworkInterfaces : get the array of network interfaces
func getNetworkInterfaces() ([]*model.Network, error) {
	log.Trace(">>>>> getNetworkInterfaces")
	defer log.Trace("<<<<< getNetworkInterfaces")

	// Start with an empty array of NICs to return
	var nics []*model.Network

	// Enumerate the system's network interfaces
	netInterfaces, err := net.Interfaces()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Enumerate the system's network adapters
	adapters, err := getAdaptersInfo()
	if err != nil {
		if err == windows.ERROR_NO_DATA {
			// Windows returns ERROR_NO_DATA, for the GetAdaptersInfo Win32 API, if no NICs were
			// found on the local computer.  If we let that error propagate up, the error message
			// recorded is "The pipe is being closed."  Instead, we more accurately return an
			// empty array of NICs with no error to indicate no NICs were found on this host.
			return nics, nil
		}
		return nil, err
	}

	// Enumerate the iSCSI initiators on this host
	iscsiInitiators, err := wmi.GetMSiSCSIPortalInfoClass()
	if err != nil {
		// It's possible that the host has NICs but the iSCSI service has not been configured yet.
		// In this case, we simply log the event but allow NIC enumeration to continue.
		log.Tracef("Unable to enumerate iSCSI initiators, continuing with NIC enumeration, err=%v", err)
		err = nil
	}

	// Enumerate the cluster IPs on this host
	clusterIPs, _ := wmi.GetClusterIPs()

	// Loop through each network interface
	for _, netInterface := range netInterfaces {

		// Find the adapter for the network interface.  Skip any network interface that doesn't have
		// an applicable adapter (e.g. getAdaptersInfo doesn't support loopback adapters).
		adapter, ok := adapters[netInterface.Index]
		if !ok {
			continue
		}

		// Loop through the adapter's IpAddressList
		for ipl := &adapter.IpAddressList; ipl != nil; ipl = ipl.Next {

			// Get the IPv4 address and mask
			ipAddress := strings.TrimRight(string(ipl.IpAddress.String[:]), "\x00")
			ipMask := strings.TrimRight(string(ipl.IpMask.String[:]), "\x00")

			// Skip any 0.0.0.0 IP address
			if ipAddress == "0.0.0.0" {
				continue
			}

			// Is this a cluster IP?
			isClusterIP := false
			for _, clusterIP := range clusterIPs {
				if (clusterIP.PrivateProperties != nil) && (clusterIP.PrivateProperties.Address == ipAddress) {
					isClusterIP = true
					break
				}
			}

			// Skip any cluster IP
			if isClusterIP {
				log.Tracef("Ignoring cluster IP %v", ipAddress)
				continue
			}

			// Traverse the list of enumerated ISCSI_PortalInfo objects to find the one that mathces
			// the current network adapter.
			var matchingPortal *wmi.ISCSI_PortalInfo
			if iscsiInitiators != nil {
				for _, portal := range iscsiInitiators.PortalInformation {
					// Skip portal if not IPv4
					if (portal.IPAddr.Type != wmi.ISCSI_IP_ADDRESS_IPV4) || (portal.IPAddr.IpV4Address == 0) {
						continue
					}

					// Convert the IPv4 uint32 into an IP object
					a := make([]byte, 4)
					binary.LittleEndian.PutUint32(a, portal.IPAddr.IpV4Address)
					ipv4 := net.IPv4(a[0], a[1], a[2], a[3])

					// Exclude link local interfaces (IPANA AIPA addresses 169.254.0-255.0-255)
					// Link local addresses are also used by Microsoft Clusters as 'Microsoft Failover Cluster Virtual Adapter addresses'
					if (a[0] == 169) && (a[1] == 254) {
						continue
					}

					// Found ISCSI_PortalInfo match for the current network adapter?
					if ipv4.String() == ipAddress {
						matchingPortal = portal
						break
					}
				}
			}

			// Append this interface/adapter to the return list of NICs
			nic := &model.Network{
				Name:      netInterface.Name,
				AddressV4: ipAddress,
				MaskV4:    ipMask,
				Mac:       netInterface.HardwareAddr.String(),
				Mtu:       int64(netInterface.MTU),
				Up:        true,
			}

			// If we were able to enumerate the ISCSI_PortalInfo object, for the current network
			// adapter, populate the Windows specific NetworkPrivate object
			if matchingPortal != nil {
				nic.Private = &model.NetworkPrivate{
					InitiatorInstance:   iscsiInitiators.InstanceName,
					InitiatorPortNumber: matchingPortal.Index,
				}
			}

			// Append network object to array of enumerated network objects
			nics = append(nics, nic)
		}
	}

	return nics, err
}

// getAdaptersInfo is a wrapper around syscall.GetAdaptersInfo which is a wrapper around the Win32
// GetAdaptersInfo function.  This function enumerates all the IPv4 addresses on this host.  A
// map of syscall.IpAdapterInfo objects is returned to the caller with the index to the map being
// the NIC index.
func getAdaptersInfo() (adapters map[int]*syscall.IpAdapterInfo, err error) {
	log.Trace(">>>>> getAdaptersInfo")
	defer log.Trace("<<<<< getAdaptersInfo")

	// We don't know the size of the buffer we'll need to retrieve the information from the Win32
	// API.  We'll start at 0 and increase as we start enumerating adapters from the API.
	bufferSizeMax := uint32(0)
	var adaptersBuffer []byte

	// We'll loop up to 5 times.  The first time through the loop is to determine the buffer size.
	// Now that we know the buffer size, the second time through is to enumerate the adapters
	// using the known buffer size.  The extra loop entries are to handle the incredibly unlikely
	// scenario where adapters are being added while we're in the middle of this loop and the
	// adapter size continues to increase.
	for i := 0; i < 5; i++ {

		// Allocate the buffer to pass to the Win32 API.  Note that on first pass through this
		// loop, we use an empty buffer so that we can enumerate the required buffer size.
		var ipAdapterInfo *syscall.IpAdapterInfo
		bufferSize := bufferSizeMax
		if bufferSize > 0 {
			adaptersBuffer = make([]byte, bufferSize)
			ipAdapterInfo = (*syscall.IpAdapterInfo)(unsafe.Pointer(&adaptersBuffer[0]))
		}

		// Call the Win32 GetAdaptersInfo API
		err = syscall.GetAdaptersInfo(ipAdapterInfo, &bufferSize)

		// If the API was successful, we can populate the map and return it to the caller
		if err == nil {
			adapters = make(map[int]*syscall.IpAdapterInfo)
			for ai := ipAdapterInfo; ai != nil; ai = ai.Next {
				adapters[int(ai.Index)] = ai
			}
			return adapters, nil
		}

		// If we get here, it means our last query was not successful.  For anything other than an
		// ERROR_BUFFER_OVERFLOW, return the fatal error to the caller.
		if err != syscall.ERROR_BUFFER_OVERFLOW {
			return nil, err
		}

		// If the returned buffer size is larger than the maximum size we've queried, update our
		// maximum size accordingly.
		if bufferSize > bufferSizeMax {
			bufferSizeMax = bufferSize
		}

		// To minimize the potential loops, on the off chance the NIC count is increasing while
		// we're in this loop, we'll add 4K more with each loop.
		bufferSizeMax += 4096
	}

	// If we get here, it means all our loops were exhausted and all queries failed with
	// ERROR_BUFFER_OVERFLOW.
	return nil, err
}

// Retrieve the host ID
func getHostId() (string, error) {

	// Use lock for thread safety
	hostIdLock.Lock()
	defer hostIdLock.Unlock()

	// If hostId is not enumerated yet, we'll enumerate it here
	if hostId == "" {

		// Retrieve the unique machine UUID from the registry
		err := shlwapi.SHGetValue(syscall.HKEY_LOCAL_MACHINE, registryHostIDKey, registryHostIDValue, &hostId)

		if err == nil {
			// If the registry value was present, make sure it's of a UUID format
			_, err = uuid.FromString(hostId)
			if err != nil {
				// Should never get here but, if we do, log the error and reset the hostId
				log.Errorf("Unexpected machine ID, key=%v, value=%v, data=%v, err=%v", registryHostIDKey, registryHostIDValue, hostId, err)
				hostId = ""
			}
		} else {
			// Should never get here but, if we do, log the fact that this host doesn't have a machine UUID
			log.Errorf("Missing machine ID, key=%v, value=%v, err=%v", registryHostIDKey, registryHostIDValue, err)
		}

		// If our machine ID enumerations were not successful, we'll use our built-in default value.
		// Even if we get to this code, it would have no impact on CHAPI since CHAPI only supports a
		// single host at a time.
		if hostId == "" {
			log.Errorf("Utilizing default machine ID, hostId=%v", registryHostIDDefault)
			hostId = registryHostIDDefault
		}
	}

	// Return enumerated host ID
	return hostId, nil
}

// Retrieve the computer's domain name. Return an empty string if not part of a domain (or domain name not available)
func getDomainName() (string, error) {
	// Allocate an initial data buffer large enough to hold the domain name
	dataLen := uint32(256)
	dataBuffer := make([]uint16, dataLen)

	// Use the Win32 GetComputerNameEx() API to get the domain name
	err := windows.GetComputerNameEx(windows.ComputerNameDnsDomain, &dataBuffer[0], &dataLen)

	// If our buffer wasn't large enough, increase the size and try once more
	if (err == syscall.ERROR_MORE_DATA) && (dataLen > 0) {
		dataBuffer = make([]uint16, dataLen)
		err = windows.GetComputerNameEx(windows.ComputerNameDnsDomain, &dataBuffer[0], &dataLen)
	}

	// We don't expect the query to fail.  If it does, log an informational entry and return an
	// empty domain string with no error.
	if err != nil {
		log.Tracef("Domain query failure, computer probably not part of a domain, error=%v", err)
		return "", nil
	}

	// Convert domain name from UTF16 to a Go string and return to caller
	return syscall.UTF16ToString(dataBuffer[:]), nil
}
