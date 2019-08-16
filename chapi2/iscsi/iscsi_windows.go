// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package iscsi

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/hpe-storage/common-host-libs/chapi2/cerrors"
	"github.com/hpe-storage/common-host-libs/chapi2/host"
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/windows/iscsidsc"
	"github.com/hpe-storage/common-host-libs/windows/wmi"
	"golang.org/x/sys/windows/registry"
)

const (
	// Registry locations for minimum/maximum connections per target.  These are the locations used
	// by the Nimble Connection Service (NCS).  Future updates might move these to a more generic
	// location.
	regKeyNimbleStorageConnections     = `SOFTWARE\Nimble Storage\Connections`
	regValueMinConnectionsPerTarget    = "MinConnectionsPerTarget"
	regValueMaxConnectionsPerTargetVST = "MaxConnectionsPerTarget"
	regValueMaxConnectionsPerTargetGST = "MaxConnectionsPerTargetGST"

	// Minimum and maximum connections allowed per target
	absoluteMinIscsiConnections = 1
	absoluteMaxIscsiConnections = 32
	defaultMinIscsiConnections  = 4
	defaultMaxIscsiConnections  = 32
)

func getIscsiInitiators() (init *model.Initiator, err error) {
	log.Trace(">>>>> getIscsiInitiators")
	defer log.Trace("<<<<< getIscsiInitiators")

	var initiatorNodeName string
	initiatorNodeName, err = iscsidsc.GetIScsiInitiatorNodeName()
	if err != nil {
		err = cerrors.IscsiErrToCerrors(err)
		return nil, err
	}

	if initiatorNodeName == "" {
		err = cerrors.NewChapiError(cerrors.NotFound, errorMessageEmptyIqnFound)
		log.Error(err)
		return nil, err
	}
	log.Tracef("got iscsi initiator name as %s", initiatorNodeName)

	initiators := []string{initiatorNodeName}
	init = &model.Initiator{AccessProtocol: model.AccessProtocolIscsi, Init: initiators}
	return init, err
}

// getTargetScope enumerates the target scope for the given iSCSI target.  An empty string is
// returned if we were unable to determine the target scope.
func getTargetScope(targetName string) (string, error) {
	log.Tracef(">>>>> getTargetScope, targetName=%v", targetName)
	defer log.Trace("<<<<< getTargetScope")

	// Enumerate all the iSCSI sessions
	iscsiSessions, err := iscsidsc.GetIscsiSessionList()
	if err != nil {
		err = cerrors.IscsiErrToCerrors(err)
		log.Error(err.Error())
		return "", err
	}

	// Keep track of the last enumeration error.  If all attempted queries fail, we'll return
	// this error to the caller.
	var lastErr error

	// Loop through all the iSCSI sessions
	for _, iscsiSession := range iscsiSessions {

		// If the session isn't for our target, skip it
		if !strings.EqualFold(targetName, iscsiSession.TargetName) {
			continue
		}

		// If there are no session connections (e.g. reconnecting), skip this session
		if len(iscsiSession.Connections) == 0 {
			lastErr = cerrors.NewChapiErrorf(cerrors.NotFound, errorMessageNoActiveConnections, iscsiSession.SessionID.AdapterUnique, iscsiSession.SessionID.AdapterSpecific)
			log.Trace(lastErr.Error())
			continue
		}

		// Issue an Inquiry request on the current session
		scsiStatus, inquiryBuffer, _, inquiryErr := iscsidsc.SendScsiInquiry(iscsiSession.SessionID, 0, 0, 0)
		inquiryErr = cerrors.IscsiErrToCerrors(inquiryErr)
		if len(inquiryBuffer) >= nimbleTargetScopeOffset {

			// Convert the vendor/product ID into a string
			vendorProduct := string(inquiryBuffer[8:32])

			// If this isn't a Nimble target, log an error and fail request
			if vendorProduct != nimbleVendorProduct {
				lastErr = cerrors.NewChapiErrorf(cerrors.Internal, errorMessageNonNimbleTarget, vendorProduct)
				log.Error(lastErr.Error())
				return "", lastErr
			}

			// Get the target scope value from the Inquiry data
			var targetScope string
			targetScopeBits := inquiryBuffer[nimbleTargetScopeOffset] & 0x03
			switch targetScopeBits {
			case 0:
				targetScope = model.TargetScopeVolume
			case 1:
				targetScope = model.TargetScopeGroup
			default:
				// If an unexpected target scope is returned, log an error and fail request
				lastErr = cerrors.NewChapiErrorf(cerrors.Internal, errorMessageInvalidTargetScope, targetScopeBits)
				log.Error(lastErr.Error())
				return "", lastErr
			}

			// Successfully enumerated target scope on this session.  Log target scope and return to the caller
			log.Tracef("targetName=%v, targetScope=%v", targetName, targetScope)
			return targetScope, nil
		}

		// Inquiry request failed, update our lastErr
		lastErr = inquiryErr
		if lastErr == nil {
			lastErr = cerrors.NewChapiErrorf(cerrors.NotFound, errorMessageFailedInquiry, scsiStatus, len(inquiryBuffer))
		}
	}

	// We were unable to enumerate the target scope from any target session; return last error detected
	if lastErr == nil {
		// If we couldn't find any session for our target, we could end up here.  In that case,
		// we'll log a generic error.
		lastErr = cerrors.NewChapiErrorf(cerrors.NotFound, errorMessageNoTargetScope)
	}
	log.Error(lastErr.Error())
	return "", lastErr
}

// rescanIscsiTarget rescans host ports for iSCSI devices
func rescanIscsiTarget(lunID string) error {
	// Unlike Linux, Windows does not have Target/LUN specific rescan capabilities so a synchronous
	// disk rescan is initiated and the lunID is ignored.
	return wmi.RescanDisks()
}

// getTargetPortals enumerates the target portals for the given iSCSI target
func (plugin *IscsiPlugin) getTargetPortals(targetName string, ipv4Only bool) ([]*model.TargetPortal, error) {

	// Retrieve the target portals from the iSCSI initiator
	targetPortalsWindows, err := iscsidsc.ReportIScsiTargetPortals("", targetName, ipv4Only)
	if err != nil {
		err = cerrors.IscsiErrToCerrors(err)
		log.Error(err)
		return nil, err
	}

	// Convert the Win32 ISCSI_TARGET_PORTAL array to an array of model.TargetPortal objects
	var targetPortals []*model.TargetPortal
	for _, targetPortalWindows := range targetPortalsWindows {
		targetPortal := &model.TargetPortal{
			Address: targetPortalWindows.Address,
			Port:    strconv.Itoa(int(targetPortalWindows.Socket)),
			Private: &model.TargetPortalPrivate{
				WindowsTargetPortal: targetPortalWindows,
			},
		}
		targetPortals = append(targetPortals, targetPortal)
	}
	return targetPortals, nil
}

// loginTarget is called to connect to the given iSCSI target.  The parent LoginTarget() routine
// has already validated that the target iqn and blockDev.IscsiAccessInfo are provided.
func (plugin *IscsiPlugin) loginTarget(blockDev model.BlockDeviceAccessInfo) (err error) {
	log.Trace(">>>>> loginTarget")
	defer log.Trace("<<<<< loginTarget")

	log.Infof("Login iSCSI target %v", blockDev.TargetName)

	// Determine how we should try to connect to the iSCSI target
	var connectTypes []string
	if connectTypes, err = plugin.connectTypeToArray(blockDev.IscsiAccessInfo.ConnectType); err != nil {
		return err
	}

	// Add discovery IP to host if one was provided
	if blockDev.IscsiAccessInfo.DiscoveryIP != "" {
		if err = plugin.addDiscoveryPortal(blockDev.IscsiAccessInfo.DiscoveryIP); err != nil {
			return err
		}
	}

	// See if the requested iSCSI target is already connected on this host.  This mirrors the CHAPI1
	// behavior.  A future enhancement could be to compare the current connections, versus the
	// optimal connections, and add/replace connections as needed.
	if loggedIn, err := plugin.IsTargetLoggedIn(blockDev.TargetName); (loggedIn == true) || (err != nil) {

		// Failure querying logged in status
		if err != nil {
			return err
		}

		// iSCSI target is already connected!  If it is *not* a volume scoped target (e.g. it's
		// a group scoped target), perform a disk rescan before returning.
		if !strings.EqualFold(blockDev.TargetScope, model.TargetScopeVolume) {
			wmi.RescanDisks()
		}

		// Return no error.  Target is already connected.
		log.Infof("Target %v already connected", blockDev.TargetName)
		return nil
	}

	// Make sure the target was found through the discovery IP.  If not found on the first query,
	// perform a deep discovery and retry once more.
	if err = plugin.isTargetPresent(blockDev.TargetName); err != nil {
		return err
	}

	// Enumerate the host initiator ports
	initiatorPorts, err := host.NewHostPlugin().GetNetworks()
	if err != nil {
		return err
	}

	// Enumerate the target's data ports
	log.Infof("Get iSCSI target portals for %v", blockDev.TargetName)
	var targetPorts []*model.TargetPortal
	if targetPorts, err = plugin.GetTargetPortals(blockDev.TargetName, true); err != nil {
		return err
	}

	// Get the minimum and maximum connections allowed for the iSCSI target
	minConnectionCount, maxConnectionCount := getMinMaxConnectionsPerTarget(blockDev.TargetScope)
	log.Infof("Login connection type(s) = %v, minConnectionCount=%v, maxConnectionCount=%v", connectTypes, minConnectionCount, maxConnectionCount)

	// If all optimal connections are not established by this time, the login process will stop and
	// a timeout error will be returned to the caller.
	loginExpiration := time.Now().Add(time.Second * loginTimeout)

	// Keep track of the ITNexus connections made
	var connections []ITNexus

	// Loop through each type of connection type until one successfully connects with the target
	for _, connectType := range connectTypes {

		// Attempt to connect to the iSCSI target using the specified initiator ports and target ports
		log.Infof("Attempting login using connection type = %v", connectType)
		connections, err = plugin.loginTargetPorts(blockDev, initiatorPorts, targetPorts, connectType, loginExpiration, maxConnectionCount)

		// If no connections were established using the current connection type, move to next type
		if len(connections) == 0 {
			continue
		}

		// If we were only able to establish partial connections, we'll use those connections and
		// log/ignore any failed connections.
		if err != nil {
			log.Warnf("Partial connections established, ignoring error, connectType=%v, count=%v, err=%v", connectType, len(connections), err)
			err = nil
		}

		// Break out of loop; one or more connections were established
		log.Tracef("%v initial connection(s) established using connectType=%v", len(connections), connectType)
		break
	}

	// If no iSCSI connections could be established, we'll return the last error.  If
	// no connection attempts were made, an internal error is returned
	if len(connections) == 0 {
		if err == nil {
			err = cerrors.NewChapiError(cerrors.Internal, errorMessageNoAvailableConnections)
			log.Error(err)
		}
		return err
	}

	// We've established our initial connections.  To ensure we meet the minimum required connection
	// count, try creating additional connections using the same ITNexus as the already established
	// connections.
	if uint32(len(connections)) < minConnectionCount {
		log.Infof("Adding connections to reach minimum count, currentConnections=%v, minConnectionCount=%v", len(connections), minConnectionCount)
		for uint32(len(connections)) < minConnectionCount {
			var newConnections []ITNexus
			for _, connection := range connections {
				if err = plugin.loginTargetPort(blockDev, connection.initiatorPort, connection.targetPort, loginExpiration); err != nil {
					return err
				}
				newConnections = append(newConnections, connection)
			}
			connections = append(connections, newConnections...)
		}
	}

	// If it is *not* a volume scoped target (e.g. it's a group scoped target), perform a
	// disk rescan before returning.  It's possible a LUN has been added to a GST and we
	// need a rescan to ensure that the OS has detected all the target LUNs.
	if !strings.EqualFold(blockDev.TargetScope, model.TargetScopeVolume) {
		wmi.RescanDisks()
	}

	// Success!  iSCSI connections established!
	return nil
}

// logoutTarget is called to disconnect the given iSCSI target from this host.
func (plugin *IscsiPlugin) logoutTarget(targetName string) (err error) {
	log.Trace(">>>>> loginTarget")
	defer log.Trace("<<<<< loginTarget")

	log.Infof("Logout iSCSI target %v", targetName)

	// Logout all iSCSI target sessions and remove persistent settings
	return iscsidsc.LogoutIScsiTargetAll(targetName, true)
}

// connectTypeToArray takes the connectType string and returns an array of connection types that
// reflect the input type.
func (plugin *IscsiPlugin) connectTypeToArray(connectType string) (connectTypes []string, err error) {

	// Determine how we should try to connect to the iSCSI target using the provided iSCSI
	// ConnectType.  If property not provided, use the default value.
	switch connectType {
	case "", model.ConnectTypeDefault:
		// If the default option is selected, we try multiple connection techniques to try and log
		// into the iSCSI target.  We start with ConnectTypePing, then ConnectTypeSubnet and end
		// with ConnectTypeAutoInitiator.
		connectTypes = []string{model.ConnectTypePing, model.ConnectTypeSubnet, model.ConnectTypeAutoInitiator}
	case model.ConnectTypePing, model.ConnectTypeSubnet, model.ConnectTypeAutoInitiator:
		// Simple/singular connection type requested
		connectTypes = []string{connectType}
	default:
		// Invalid / Unsupported connection type
		err = cerrors.NewChapiErrorf(cerrors.InvalidArgument, errorMessageInvalidConnectionType, connectType)
		log.Error(err)
		return nil, err
	}

	return connectTypes, nil
}

// addDiscoveryPortal adds the given discovery IP to the system's discovery portals.
func (plugin *IscsiPlugin) addDiscoveryPortal(discoveryIP string) error {
	log.Tracef(">>>>> addDiscoveryPortal, discoveryIP=%v", discoveryIP)
	defer log.Traceln("<<<<< addDiscoveryPortal")

	// Enumerate the send target portals (e.g. discovery IPs)
	sendTargetPortals, err := iscsidsc.ReportIScsiSendTargetPortalsEx()
	if err != nil {
		err = cerrors.IscsiErrToCerrors(err)
		log.Error(err)
		return err
	}

	// Does this host already have an entry for the discovery IP?
	for _, sendTargetPortal := range sendTargetPortals {
		if sendTargetPortal.Address == discoveryIP {
			// If discovery IP is already registed on this host, return nil
			log.Infof("Use discovery IP %v", discoveryIP)
			return nil
		}
	}

	// Add discovery IP to host
	log.Infof("Add discovery IP %v", discoveryIP)
	if err = iscsidsc.AddIScsiSendTargetPortal("", iscsidsc.ISCSI_ANY_INITIATOR_PORT, discoveryIP); err != nil {
		err = cerrors.IscsiErrToCerrors(err)
		log.Error(err)
		return err
	}

	// Discovery IP added to host successfully!
	return nil
}

// isTargetLoggedIn checks to see if the given iSCSI target is already logged in.
func (plugin *IscsiPlugin) isTargetLoggedIn(targetName string) (bool, error) {
	log.Tracef(">>>>> isTargetLoggedIn, TargetName=%v", targetName)
	defer log.Traceln("<<<<< isTargetLoggedIn")

	// Get the current iSCSI session list
	iscsiSessions, err := iscsidsc.GetIscsiSessionList()
	if err != nil {
		err = cerrors.IscsiErrToCerrors(err)
		log.Error(err)
		return false, err
	}

	// See if the requested iSCSI target is already connected on this host
	for _, iscsiSession := range iscsiSessions {
		if strings.EqualFold(iscsiSession.TargetName, targetName) {
			return true, nil
		}
	}
	return false, nil
}

// isTargetPresent returns nil if the given iSCSI target can be detected by this host, else an
// applicable error is returned.
func (plugin *IscsiPlugin) isTargetPresent(targetName string) error {
	log.Tracef(">>>>> isTargetPresent, targetName=%v", targetName)
	defer log.Traceln("<<<<< isTargetPresent")

	// Check to see if target is available through a discovery query.  If not found on the first
	// query, perform a deep discovery and retry once more.
	for loop := 0; loop < 2; loop++ {
		if loop == 1 {
			// Post an informational log entry that we're now performing a deep discovery
			// since the default discovery did not detect the target.
			log.Infoln("Performing a deep discovery to discover iSCSI target")
		}
		targets, _ := iscsidsc.ReportIscsiTargets(loop == 1)
		for _, target := range targets {
			if strings.EqualFold(target, targetName) {
				// Return nil as soon as target is found
				return nil
			}
		}
	}

	// Fail query since target was not found
	err := cerrors.NewChapiError(cerrors.NotFound, errorMessageTargetNotFound)
	log.Error(err)
	return err
}

// loginTargetPorts is called to connect an iSCSI target
// Input Parameters
//		blockDev			Login details for the iSCSI target
//		initiatorPorts		Available initiator ports
//		targetPorts			Available target ports
//		connectType			Connection type
//      loginExpiration		Login attempts need to complete by this time
// Return Parameters
//		connectionCount		Number of successful login attempts
//		err					Error if unable to make any connection
func (plugin *IscsiPlugin) loginTargetPorts(
	blockDev model.BlockDeviceAccessInfo,
	initiatorPorts []*model.Network,
	targetPorts []*model.TargetPortal,
	connectType string,
	loginExpiration time.Time,
	maxConnectionCount uint32) (connections []ITNexus, err error) {

	log.Tracef(">>>>> loginTargetPorts, targetName=%v", blockDev.TargetName)
	defer log.Traceln("<<<<< loginTargetPorts")

	// Enumerate the IT_nexuses we should attempt to make connections with using the
	// specified connection type.
	var itNexus map[*model.Network][]*model.TargetPortal
	switch connectType {
	case model.ConnectTypePing:
		itNexus, _ = ITNexusPingCheck(initiatorPorts, targetPorts, 0, 0, 0)
	case model.ConnectTypeSubnet:
		itNexus, _ = ITNexusSubnetCheck(initiatorPorts, targetPorts)
	case model.ConnectTypeAutoInitiator:
		itNexus = make(map[*model.Network][]*model.TargetPortal)
		emptyInitiatorPort := &model.Network{AddressV4: "0.0.0.0"}
		for _, ipTarget := range targetPorts {
			itNexus[emptyInitiatorPort] = append(itNexus[emptyInitiatorPort], ipTarget)
		}
	default:
		err = cerrors.NewChapiErrorf(cerrors.Internal, errorMessageInvalidConnectionType, connectType)
		log.Error(err)
		return nil, err
	}

	// Keep track of the last login error that occurs (if any)
	var lastLoginError error

	// Loop through each initiator and the array of target ports to connect
	for initiatorPort, targetPorts := range itNexus {

		// Loop through each target port and attempt to make a connection to it
		for _, targetPort := range targetPorts {

			// Break out of ITNexus loop if maximum connection count reached
			if uint32(len(connections)) >= maxConnectionCount {
				log.Tracef("Maximum connection count reached, connections=%v, maxConnectionCount=%v", len(connections), maxConnectionCount)
				break
			}

			// Log into the given target port from the given initiator port.  If an error occurred,
			// move to the next IT nexus.
			if loginError := plugin.loginTargetPort(blockDev, initiatorPort, targetPort, loginExpiration); loginError != nil {
				lastLoginError = loginError
				continue
			}

			// Connection successful; append connection to connections array
			connections = append(connections, ITNexus{initiatorPort: initiatorPort, targetPort: targetPort})
		}
	}

	// If no connections were made, fail the request
	if len(connections) == 0 {
		err = lastLoginError
		if err == nil {
			err = cerrors.NewChapiError(cerrors.Internal, errorMessageNoAvailableConnections)
		}
		log.Error(err)
		return nil, err
	}

	// Success!  Return the connections established.
	log.Infof("%v connection(s) established", len(connections))
	return connections, nil
}

// loginTargetPort is called to log into a single target port from a single initiator port
func (plugin *IscsiPlugin) loginTargetPort(
	blockDev model.BlockDeviceAccessInfo,
	initiatorPort *model.Network,
	targetPort *model.TargetPortal,
	loginExpiration time.Time) error {

	log.Tracef(">>>>> loginTargetPort, targetName=%v", blockDev.TargetName)
	defer log.Traceln("<<<<< loginTargetPort")

	// If the amount of time given to login to an iSCSI target has expired, fail the
	// request.
	if time.Now().After(loginExpiration) {
		err := cerrors.NewChapiError(cerrors.Timeout, errorMessageLoginTimeout)
		log.Error(err)
		return err
	}

	// Determine the iSCSI initiator port number to use
	initiatorPortNumber := iscsidsc.ISCSI_ANY_INITIATOR_PORT
	if initiatorPort.Private != nil {
		// If here, the model.ConnectTypeAutoInitiator option is being used (i.e. let host initiator
		// decide which of its ports to use)
		initiatorPortNumber = initiatorPort.Private.InitiatorPortNumber
	}

	// Perform an iSCSI login
	_, _, err := iscsidsc.LoginIScsiTargetEx(
		blockDev.TargetName,                    // targetName string
		"",                                     // initiatorInstance string
		initiatorPortNumber,                    // initiatorPortNumber uint32
		targetPort.Private.WindowsTargetPortal, // targetPortal *ISCSI_TARGET_PORTAL
		iscsidsc.ISCSI_DIGEST_TYPE_NONE,        // headerDigest ISCSI_DIGEST_TYPES
		iscsidsc.ISCSI_DIGEST_TYPE_NONE,        // headerDigest ISCSI_DIGEST_TYPES
		blockDev.IscsiAccessInfo.ChapUser,      // chapUsername string
		blockDev.IscsiAccessInfo.ChapPassword,  // chapPassword string
		true) // isPersistent bool

	// Log error if failure connection not successful
	if err != nil {
		err = cerrors.IscsiErrToCerrors(err)
		log.Errorf("Connection failure, err=%v, iqn=%v, initiatorPort=%v, targetPort=%v", err, blockDev.TargetName, initiatorPort.AddressV4, targetPort.Address)
		return err
	}

	// Success!!!  Connection established.
	log.Infof("Connection established, iqn=%v, initiatorPort=%v, targetPort=%v", blockDev.TargetName, initiatorPort.AddressV4, targetPort.Address)
	return nil
}

// getMinMaxConnectionsPerTarget enumerates the minimum and maximum allowed iSCSI connections
// allowed per target.  Values are retrieved from the registry.
func getMinMaxConnectionsPerTarget(targetScope string) (minConnections, maxConnections uint32) {

	// Determine which registry value name to use to retrieve the maximum connection count
	var registryMaxConnections string
	if strings.EqualFold(targetScope, model.TargetScopeVolume) {
		registryMaxConnections = regValueMaxConnectionsPerTargetVST
	} else {
		registryMaxConnections = regValueMaxConnectionsPerTargetGST
	}

	// Determine the minimum connection count
	minConnections, errMin := getRegistryUint32(registry.LOCAL_MACHINE, regKeyNimbleStorageConnections, regValueMinConnectionsPerTarget)
	if (errMin != nil) || (minConnections < absoluteMinIscsiConnections) {
		// If registry value not present, or value less than absolute minimum, use default value
		minConnections = defaultMinIscsiConnections
	} else if minConnections > absoluteMaxIscsiConnections {
		// If registry value exceeds absolute maximum, limit to absolute maximum
		minConnections = absoluteMaxIscsiConnections
	}

	// Determine the maximum connection count
	maxConnections, errMax := getRegistryUint32(registry.LOCAL_MACHINE, regKeyNimbleStorageConnections, registryMaxConnections)
	if (errMax != nil) || (maxConnections < absoluteMinIscsiConnections) {
		// If registry value not present, or value less than absolute minimum, use default maximum
		maxConnections = defaultMaxIscsiConnections
	} else if maxConnections > absoluteMaxIscsiConnections {
		// If registry value exceeds absolute maximum, limit to absolute maximum
		maxConnections = absoluteMaxIscsiConnections
	}

	// Ensure maxConnections is always greater than or equal to minConnections
	if maxConnections < minConnections {
		maxConnections = minConnections
	}

	return minConnections, maxConnections
}

// getRegistryUint32 is a wrapper around the registry package.  Pass in the registry key, key path,
// and key value, and this routine returns the integer found there.  An error object is returned if
// the registry value could not be retrieved.
func getRegistryUint32(key registry.Key, path string, name string) (uint32, error) {

	// Start by opening the registry key
	k, err := registry.OpenKey(key, path, registry.QUERY_VALUE)
	if err != nil {
		return 0, err
	}
	defer k.Close()

	// Retrieve the integer value from the registry
	s, _, err := k.GetIntegerValue(name)
	if err != nil {
		return 0, err
	}

	// Fail request if retrieved value larger than 32-bits
	if s >= math.MaxUint32 {
		return 0, fmt.Errorf("registry value exceeds 32-bit limits; value=%v", s)
	}

	// Convert uint64 value to a uint32 and return to caller
	return uint32(s), nil
}
