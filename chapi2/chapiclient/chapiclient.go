// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package chapiclient

import (
	"fmt"
	"strings"
	"time"

	"github.com/hpe-storage/common-host-libs/chapi2/cerrors"
	chapiDriver "github.com/hpe-storage/common-host-libs/chapi2/driver"
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	"github.com/hpe-storage/common-host-libs/connectivity"
	log "github.com/hpe-storage/common-host-libs/logger"
)

const (
	// REST endpoint API version
	apiVersion = "api/v1"

	// Host Endpoints
	hostURI       = apiVersion + "/hosts"      // api/v1/hosts
	initiatorsURI = apiVersion + "/initiators" // api/v1/initiators
	networksURI   = apiVersion + "/networks"   // api/v1/networks

	// Device Endpoints
	devicesURI           = apiVersion + "/devices"            // api/v1/devices
	devicesDetailURI     = devicesURI + "/details"            // api/v1/devices/details
	devicesPartitionsURI = devicesURI + "/%v/partitions"      // api/v1/devices/{serialnumber}/partitions
	devicesOfflineURI    = devicesURI + "/%v/actions/offline" // api/v1/devices/{serialnumber}/actions/offline
	devicesFileSystemURI = devicesURI + "/%v/%v"              // api/v1/devices/{serialnumber}/filesystem/{filesystem}

	// Mount Endpoints
	mountsURI       = apiVersion + "/mounts" // api/v1/mounts
	mountsDetailURI = mountsURI + "/details" // api/v1/mounts/details
	mountsDeleteURI = mountsURI + "/%v"      // api/v1/mounts/{mountId}
)

const (
	// Query Parameters
	queryMountID      = "mountId" // e.g. api/v1/mounts/details?serial=1234&mountId=5678
	querySerialNumber = "serial"  // e.g. api/v1/devices/details?serial=1234
)

// ClientBase defines platform independent properties and is embedded within the Client object
type ClientBase struct {
	client *connectivity.Client // HTTP client for connectivity to chapid server
	header map[string]string    // HTTP headers
}

var (
	// The "dummy" object is declared so that the Client object is required to support all the
	// chapiDriver.Driver methods.  If any are missing, a compilation error will occur.  This
	// ensures that the CHAPI client methods stay aligned with the CHAPI server methods.
	dummy chapiDriver.Driver = &Client{}
)

// Response object defines the data and/or error that are returned by a CHAPI endpoint
type Response struct {
	Data interface{}         `json:"data,omitempty"`
	Err  *cerrors.ChapiError `json:"errors,omitempty"`
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// CHAPI Client Initialization
///////////////////////////////////////////////////////////////////////////////////////////////////

// NewChapiClient returns the CHAPI client object that is used to communicate with the CHAPI server
// over HTTP.
func NewChapiClient() (*Client, error) {
	return newChapiClient() // Reflect to platform specific handler
}

// NewChapiClientWithTimeout returns the CHAPI client object that is used to communicate with the
// CHAPI server over HTTP.  A custom timeout value is supported.
func NewChapiClientWithTimeout(timeout time.Duration) (*Client, error) {
	return newChapiClientWithTimeout(timeout) // Reflect to platform specific handler
}

// Print to dump CHAPI client struct
func (chapiClient *Client) Print() {
	chapiClient.Print() // Reflect to platform specific handler
}

// addHeader is used to insert HTTP headers with each CHAPI request
func (chapiClient *Client) addHeader(header map[string]string) {
	chapiClient.header = header
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Host Methods
///////////////////////////////////////////////////////////////////////////////////////////////////

// GetHostInfo returns host name, domain, and network interfaces
func (chapiClient *Client) GetHostInfo() (host *model.Host, err error) {
	log.Trace(">>>>> GetHostInfo called")
	defer log.Trace("<<<<< GetHostInfo")

	// Initialize CHAPI response object, submit request to specified endpoint, return status
	chapiResp := Response{Data: &host, Err: nil}
	if _, err = chapiClient.chapiClientDoJSON(&connectivity.Request{Action: "GET", Path: hostURI, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp}); err != nil {
		return nil, err
	}
	return host, nil
}

// GetHostInitiators reports the initiators on this host
func (chapiClient *Client) GetHostInitiators() (initiators []*model.Initiator, err error) {
	log.Trace(">>>>> GetHostInitiators called")
	defer log.Trace("<<<<< GetHostInitiators")

	// Initialize CHAPI response object, submit request to specified endpoint, return status
	chapiResp := Response{Data: &initiators, Err: nil}
	if _, err = chapiClient.chapiClientDoJSON(&connectivity.Request{Action: "GET", Path: initiatorsURI, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp}); err != nil {
		return nil, err
	}
	return initiators, nil
}

// GetHostNetworks reports the networks on this host
func (chapiClient *Client) GetHostNetworks() (networks []*model.Network, err error) {
	log.Trace(">>>>> GetHostNetworks called")
	defer log.Trace("<<<<< GetHostNetworks")

	// Initialize CHAPI response object, submit request to specified endpoint, return status
	chapiResp := Response{Data: &networks, Err: nil}
	if _, err = chapiClient.chapiClientDoJSON(&connectivity.Request{Action: "GET", Path: networksURI, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp}); err != nil {
		return nil, err
	}
	return networks, nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Device methods
///////////////////////////////////////////////////////////////////////////////////////////////////

// GetDevices enumerates all the Nimble volumes with basic details.
// If serialNumber is non-empty then only specified device is returned
func (chapiClient *Client) GetDevices(serialNumber string) (devices []*model.Device, err error) {
	log.Tracef(">>>>> GetDevices called, serialNumber=%v", serialNumber)
	defer log.Trace("<<<<< GetDevices")

	// Initialize CHAPI response object, submit request to specified endpoint, return status
	chapiResp := Response{Data: &devices, Err: nil}
	devicesURIOut := chapiClient.appendQuerySerialNumber(devicesURI, serialNumber)
	if _, err = chapiClient.chapiClientDoJSON(&connectivity.Request{Action: "GET", Path: devicesURIOut, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp}); err != nil {
		return nil, err
	}
	return devices, nil
}

// GetAllDeviceDetails enumerates all the Nimble volumes with detailed information.
// If serialNumber is non-empty then only specified device is returned
func (chapiClient *Client) GetAllDeviceDetails(serialNumber string) (devices []*model.Device, err error) {
	log.Tracef(">>>>> GetAllDeviceDetails called, serialNumber=%v", serialNumber)
	defer log.Trace("<<<<< GetAllDeviceDetails")

	// Initialize CHAPI response object, submit request to specified endpoint, return status
	chapiResp := Response{Data: &devices, Err: nil}
	devicesURIOut := chapiClient.appendQuerySerialNumber(devicesDetailURI, serialNumber)
	if _, err = chapiClient.chapiClientDoJSON(&connectivity.Request{Action: "GET", Path: devicesURIOut, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp}); err != nil {
		return nil, err
	}
	return devices, nil
}

// GetPartitionInfo reports the partitions on the provided device
func (chapiClient *Client) GetPartitionInfo(serialNumber string) (partitions []*model.DevicePartition, err error) {
	log.Tracef(">>>>> GetPartitionInfo called, serialNumber=%v", serialNumber)
	defer log.Trace("<<<<< GetPartitionInfo")

	// Initialize CHAPI response object, submit request to specified endpoint, return status
	chapiResp := Response{Data: &partitions, Err: nil}
	devicePartitionsURIOut := fmt.Sprintf(devicesPartitionsURI, serialNumber)
	if _, err = chapiClient.chapiClientDoJSON(&connectivity.Request{Action: "GET", Path: devicePartitionsURIOut, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp}); err != nil {
		return nil, err
	}
	return partitions, nil
}

// CreateDevice will attach device on this host based on the details provided
func (chapiClient *Client) CreateDevice(publishInfo model.PublishInfo) (device *model.Device, err error) {
	log.Tracef(">>>>> CreateDevice called, publishInfo=%v", publishInfo)
	defer log.Trace("<<<<< CreateDevice")

	// Initialize CHAPI response object, submit request to specified endpoint, return status
	chapiResp := Response{Data: &device, Err: nil}
	if _, err = chapiClient.chapiClientDoJSON(&connectivity.Request{Action: "POST", Path: devicesURI, Header: chapiClient.header, Payload: &publishInfo, Response: &chapiResp, ResponseError: &chapiResp}); err != nil {
		return nil, err
	}
	return device, nil
}

// DeleteDevice will delete the given device from the host
func (chapiClient *Client) DeleteDevice(serialNumber string) (err error) {
	log.Tracef(">>>>> DeleteDevice called, serialNumber=%v", serialNumber)
	defer log.Trace("<<<<< DeleteDevice")

	// Initialize CHAPI response object, submit request to specified endpoint, return status
	chapiResp := Response{Data: nil, Err: nil}
	devicesURIOut := devicesURI + "/" + serialNumber
	if _, err = chapiClient.chapiClientDoJSON(&connectivity.Request{Action: "DELETE", Path: devicesURIOut, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp}); err != nil {
		return err
	}
	return nil
}

// OfflineDevice will offline the given device from the host
func (chapiClient *Client) OfflineDevice(serialNumber string) (err error) {
	log.Tracef(">>>>> OfflineDevice called, serialNumber=%v", serialNumber)
	defer log.Trace("<<<<< OfflineDevice")

	// Initialize CHAPI response object, submit request to specified endpoint, return status
	chapiResp := Response{Data: nil, Err: nil}
	deviceOfflineURIOut := fmt.Sprintf(devicesOfflineURI, serialNumber)
	if _, err = chapiClient.chapiClientDoJSON(&connectivity.Request{Action: "PUT", Path: deviceOfflineURIOut, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp}); err != nil {
		return err
	}
	return nil
}

// CreateFileSystem writes the given file system to the device with the given serial number
func (chapiClient *Client) CreateFileSystem(serialNumber string, filesystem string) (err error) {
	log.Tracef(">>>>> CreateFileSystem called, serialNumber=%v, filesystem=%v", serialNumber, filesystem)
	defer log.Trace("<<<<< CreateFileSystem")

	// Initialize CHAPI response object, submit request to specified endpoint, return status
	chapiResp := Response{Data: nil, Err: nil}
	deviceFileSystemURIOut := fmt.Sprintf(devicesFileSystemURI, serialNumber, filesystem)
	if _, err = chapiClient.chapiClientDoJSON(&connectivity.Request{Action: "PUT", Path: deviceFileSystemURIOut, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp}); err != nil {
		return err
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Mount Methods
///////////////////////////////////////////////////////////////////////////////////////////////////

// GetMounts reports all mounts on this host for the specified Nimble volume
func (chapiClient *Client) GetMounts(serialNumber string) (mounts []*model.Mount, err error) {
	log.Tracef(">>>>> GetMounts called, serialNumber=%v", serialNumber)
	defer log.Trace("<<<<< GetMounts")

	// Initialize CHAPI response object, submit request to specified endpoint, return status
	chapiResp := Response{Data: &mounts, Err: nil}
	mountsURIOut := chapiClient.appendQuerySerialNumber(mountsURI, serialNumber)
	if _, err = chapiClient.chapiClientDoJSON(&connectivity.Request{Action: "GET", Path: mountsURIOut, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp}); err != nil {
		return nil, err
	}
	return mounts, nil
}

// GetAllMountDetails enumerates the specified mount point ID
func (chapiClient *Client) GetAllMountDetails(serialNumber, mountPointID string) (mounts []*model.Mount, err error) {
	log.Tracef(">>>>> GetAllMountDetails called, serialNumber=%v, mountPointID=%v", serialNumber, mountPointID)
	defer log.Trace("<<<<< GetAllMountDetails")

	// Initialize CHAPI response object, submit request to specified endpoint, return status
	chapiResp := Response{Data: &mounts, Err: nil}
	mountsURIOut := chapiClient.appendQuerySerialNumber(mountsDetailURI, serialNumber)
	mountsURIOut = chapiClient.appendQueryMountPointID(mountsURIOut, mountPointID)
	if _, err = chapiClient.chapiClientDoJSON(&connectivity.Request{Action: "GET", Path: mountsURIOut, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp}); err != nil {
		return nil, err
	}
	return mounts, nil
}

// CreateMount mounts the given device to the given mount point
func (chapiClient *Client) CreateMount(serialNumber string, mountPoint string, fsOptions *model.FileSystemOptions) (mount *model.Mount, err error) {
	log.Tracef(">>>>> CreateMount called, serialNumber=%v, mountPoint=%v, fsOptions=%v", serialNumber, mountPoint, fsOptions)
	defer log.Trace("<<<<< CreateMount")

	// Initialize model.Mount submission object
	mountSubmission := model.Mount{
		SerialNumber: serialNumber,
		MountPoint:   mountPoint,
		FsOpts:       fsOptions,
	}

	// Initialize CHAPI response object, submit request to specified endpoint, return status
	chapiResp := Response{Data: &mount, Err: nil}
	if _, err = chapiClient.chapiClientDoJSON(&connectivity.Request{Action: "POST", Path: mountsURI, Header: chapiClient.header, Payload: &mountSubmission, Response: &chapiResp, ResponseError: &chapiResp}); err != nil {
		return nil, err
	}
	return mount, nil
}

// DeleteMount unmounts the given mount point, serialNumber can be optional in the body
func (chapiClient *Client) DeleteMount(serialNumber, mountPointID string) (err error) {
	log.Tracef(">>>>> DeleteMount called, serialNumber=%v, mountPointID=%v", serialNumber, mountPointID)
	defer log.Trace("<<<<< DeleteMount")

	// Initialize CHAPI response object, submit request to specified endpoint, return status
	chapiResp := Response{Data: nil, Err: nil}
	mountsDeleteURIOut := fmt.Sprintf(mountsDeleteURI, mountPointID)
	if _, err = chapiClient.chapiClientDoJSON(&connectivity.Request{Action: "DELETE", Path: mountsDeleteURIOut, Header: chapiClient.header, Payload: serialNumber, Response: &chapiResp, ResponseError: &chapiResp}); err != nil {
		return err
	}
	return nil
}

// CreateBindMount creates the given bind mount
func (chapiClient *Client) CreateBindMount(sourceMount string, targetMount string, bindType string) (mount *model.Mount, err error) {
	log.Tracef(">>>>> CreateBindMount called, sourceMount=%s, targetMount=%s bindType=%s", sourceMount, targetMount, bindType)
	defer log.Trace("<<<<< CreateBindMount")

	// TODO
	return nil, cerrors.NewChapiError(cerrors.Unimplemented)
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Internal Support Methods
///////////////////////////////////////////////////////////////////////////////////////////////////

// chapiClientDoJSON wraps the call to chapiClient.client.DoJSON().  If the request fails, and an
// error was returned by the CHAPI server, that error is returned instead.
func (chapiClient *Client) chapiClientDoJSON(r *connectivity.Request) (int, error) {

	// Start by calling submitting the request to the CHAPI endpoint
	statusCode, err := chapiClient.client.DoJSON(r)

	if err != nil {
		//  If we received an error from the CHAPI server, use that error object
		if cerror, ok := r.ResponseError.(*Response); ok && (cerror.Err != nil) {
			log.Error("CHAPI Error : ", cerror.Err)
			return 0, cerror.Err
		}

		// For all other errors, return the connectivity error
		log.Error("Connectivity Error : ", err)
		return 0, err
	}

	// CHAPI request was successful; return the HTTP status code with no error
	return statusCode, nil
}

// appendQuerySerialNumber appends a serial number query to the given URI
func (chapiClient *Client) appendQuerySerialNumber(uri string, serialNumber string) string {
	return chapiClient.appendQuery(uri, querySerialNumber, serialNumber)
}

// appendQueryMountPointID appends a mount ID query to the given URI
func (chapiClient *Client) appendQueryMountPointID(uri string, mountPointID string) string {
	return chapiClient.appendQuery(uri, queryMountID, mountPointID)
}

// appendQuery appends the key key/value query to the given URI
func (chapiClient *Client) appendQuery(uri string, key string, value string) string {
	// Don't append query if value is empty
	if value == "" {
		return uri
	}

	// First query appended or adding to existing query?
	if strings.Contains(uri, "?") {
		uri += "&"
	} else {
		uri += "?"
	}

	// Append and return the query
	uri += (key + "=" + value)
	return uri
}
