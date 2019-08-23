// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package chapiclient

import (
	"os"
	"strconv"
	"time"

	"github.com/hpe-storage/common-host-libs/chapi2"
	"github.com/hpe-storage/common-host-libs/connectivity"
	log "github.com/hpe-storage/common-host-libs/logger"
)

// Client contains the Linux specific Client properties
type Client struct {
	ClientBase        // Embedded platform independent struct
	socket     string // Socket name on which chapid is listening
}

// newChapiClient is the platform specific handler for the NewChapiClient method
func newChapiClient() (*Client, error) {
	return NewChapiSocketClientWithTimeout(nil)
}

// newChapiClientWithTimeout is the platform specific handler for the NewChapiClientWithTimeout method
func newChapiClientWithTimeout(timeout time.Duration) (*Client, error) {
	return NewChapiSocketClientWithTimeout(&timeout)
}

// NewChapiClientWithTimeout returns the CHAPI client object, using Linux sockets, that is used
// to communicate with the  CHAPI server over HTTP.  A custom timeout value is supported.
func NewChapiSocketClientWithTimeout(timeout *time.Duration) (chapiClient *Client, err error) {

	// Get the socket name
	socketName := GetSocketName()

	// Setup the CHAPI client object with a timeout value (if provided)
	var socketClient *connectivity.Client
	if timeout == nil {
		log.Traceln("Setting up CHAPI client with socket ", socketName)
		socketClient = connectivity.NewSocketClient(socketName)
	} else {
		log.Traceln("Setting up CHAPI client with socket ", socketName, " and timeout ", timeout)
		socketClient = connectivity.NewSocketClientWithTimeout(socketName, *timeout)
	}

	// Allocate and initialize a new Client object
	chapiClient = &Client{
		socket:     socketName,
		ClientBase: ClientBase{client: socketClient},
	}

	// Success, return CHAPI client object
	return chapiClient, nil
}

// GetSocketName returns unix socket name (per process)
func GetSocketName() string {
	if chapi2.IsChapidRunning(chapi2.ChapidSocketPath + chapi2.ChapidSocketName) {
		return chapi2.ChapidSocketPath + chapi2.ChapidSocketName
	}
	return chapi2.ChapidSocketPath + chapi2.ChapidSocketName + strconv.Itoa(os.Getpid())
}

// print is the platform specific routine to dump the CHAPI client struct
func (chapiClient *Client) print() {
	log.Traceln("Socket : ", chapiClient.socket)
	log.Traceln("Header : ", chapiClient.header)
}
