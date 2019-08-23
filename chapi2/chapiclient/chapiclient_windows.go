// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package chapiclient

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hpe-storage/common-host-libs/chapi2/handler"
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	"github.com/hpe-storage/common-host-libs/connectivity"
	log "github.com/hpe-storage/common-host-libs/logger"
)

const (
	// Host Endpoints (Windows specific)
	keyFileURI = apiVersion + "/keyfile"
)

const (
	// CHAPI will wait up to 2 minutes for a request to complete (if default timeout used)
	defaultChapiTimeout = 2 * time.Minute
)

// Client contains the Windows specific Client properties
type Client struct {
	ClientBase        // Embedded platform independent struct
	hostname   string // URL where server is running
	port       uint64 // Port number where the server is listening
}

// newChapiClient is the platform specific handler for the NewChapiClient method
func newChapiClient() (*Client, error) {
	return NewChapiWindowsClient("", nil)
}

// newChapiClientWithTimeout is the platform specific handler for the NewChapiClientWithTimeout method
func newChapiClientWithTimeout(timeout time.Duration) (*Client, error) {
	return NewChapiWindowsClient("", &timeout)
}

// NewChapiWindowsClient returns the a CHAPI Client object that this client uses to communicate with
// the CHAPI server.  The following input parameters are available:
//
//		chapiFolder		Folder where CHAPI binary is located.  CHAPI for Windows stores its TCP/IP
//						port value in a text file in this folder.  If an empty string is passed in
//						as input, this routine uses the currently running executable's folder.  If
//						your binary is installed alongside the CHAPI for Windows server, you can
//						simply pass in an empty string.
//
//		timeout			This parameter specifies how long CHAPI will wait for the REST endpoint to
//						complete a request.  If 'nil' is passed in, an internal default value will
//						be used.
func NewChapiWindowsClient(chapiFolder string, timeout *time.Duration) (chapiClient *Client, err error) {

	// If no CHAPI executable folder path is provided, use the running executable's path
	if chapiFolder == "" {
		if chapiFolder, err = os.Executable(); err != nil {
			return nil, err
		}
		chapiFolder = filepath.Dir(chapiFolder)
	}

	// Get the full path to the CHAPI port text file
	chapiPortFilePath := filepath.Join(chapiFolder, handler.ChapiPortFileName)
	log.Tracef("CHAPI port filepath = %v", chapiPortFilePath)

	// Read in the CHAPI port text file
	buf, err := ioutil.ReadFile(chapiPortFilePath)
	if err != nil {
		log.Errorf("Failed to read CHAPI port file, err=%v", err)
		return nil, err
	}

	// Convert CHAPI port text a buffer to a string
	chapiPort := strings.TrimSuffix(string(buf), "\r\n")
	log.Tracef("CHAPI port obtained is '%v' from file '%v'", chapiPort, chapiPortFilePath)

	// Convert CHAPI port text to a uint64
	port, err := strconv.ParseUint(chapiPort, 10, 64)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Allocate a CHAPI HTTP Client object so we can send endpoint requests
	chapiClient, err = newChapiHTTPClientWithTimeout("", port, timeout)
	if err != nil {
		return nil, err
	}

	// Obtain access/secret key and insert as HTTP headers
	accessKey, err := chapiClient.GetAccessKey()
	if err != nil {
		log.Errorf("Failed to get host access-key, err=%v", err)
		return nil, err
	}

	// Add the HTTP header
	header := map[string]string{"CHAPILocalAccessKey": accessKey}
	chapiClient.addHeader(header)

	// Return the initialize CHAPI Client object
	return chapiClient, nil
}

// newChapiHTTPClientWithTimeout creates a CHAPI http client using a specified timeout
func newChapiHTTPClientWithTimeout(hostName string, port uint64, timeout *time.Duration) (*Client, error) {

	// If no hostName provided, default to the local host (e.g. http://127.0.0.1)
	if hostName == "" {
		hostName = "http://127.0.0.1"
	}

	// If no timeout specified, use the default timeout
	chapiTimeout := defaultChapiTimeout
	if timeout != nil {
		chapiTimeout = *timeout
	}

	// Format the URL as "hostname:port"
	hostURL := fmt.Sprintf("%v:%v", hostName, port)
	log.Tracef("Setting up CHAPI client, hostURL=%v, timeout=%v", hostURL, chapiTimeout)

	// Initialize an HTTP client with the specified timeout
	httpClient := connectivity.NewHTTPClientWithTimeout(hostURL, chapiTimeout)

	// Initialize a CHAPI client object with the initialized HTTP client, host name, and port
	chapiClient := &Client{
		ClientBase: ClientBase{client: httpClient},
		hostname:   hostName,
		port:       port,
	}

	// Return the successfully initialized CHAPI client object
	return chapiClient, nil
}

// print is the platform specific routine to dump the CHAPI client struct
func (chapiClient *Client) print() {
	log.Traceln("Hostname : ", chapiClient.hostname)
	log.Traceln("Port     : ", chapiClient.port)
	log.Traceln("Header   : ", chapiClient.header)
}

// GetAccessKey will retrieve the access key from the host
func (chapiClient *Client) GetAccessKey() (accessKey string, err error) {
	log.Trace(">>>>> GetAccessKey called")
	defer log.Trace("<<<<< GetAccessKey")

	// Initialize CHAPI response object, submit request to specified endpoint, return status
	var accessKeyPath *model.KeyFileInfo
	chapiResp := Response{Data: &accessKeyPath, Err: nil}
	if _, err = chapiClient.chapiClientDoJSON(&connectivity.Request{Action: "GET", Path: keyFileURI, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp}); err != nil {
		return "", err
	}

	// If for some reason the accessKeyPath is nil (should not occur), fail the request
	if accessKeyPath == nil {
		err = fmt.Errorf("invalid accessKeyPath object")
		log.Error(err)
		return "", err
	}

	// Log the access key file path
	log.Tracef("accessKeyPath = %v", accessKeyPath.Path)

	// Read the file and extract the accessKey
	buf, err := ioutil.ReadFile(accessKeyPath.Path)
	if err != nil {
		return "", err
	}
	accessKey = strings.TrimSuffix(string(buf), "\r\n")
	// For security, accessKey is not logged

	// Success!  Return enumerated access key.
	return accessKey, nil
}
