// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package chapi2

import (
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hpe-storage/common-host-libs/chapi2/handler"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/util"
)

var (
	chapidLock     sync.Mutex   // CHAPI lock
	chapidListener net.Listener // CHAPI TCP listener
	chapiRunning   int32        // 1 if CHAPI server is active, else 0
)

// Additional endpoints only supported by CHAPI for Windows
var platformSpecificEndpoints = []util.Route{
	util.Route{
		Name:        "Keyfile",
		Method:      "GET",
		Pattern:     "/api/v1/keyfile",
		HandlerFunc: handler.GetKeyfile,
	},
}

// Run will invoke a new chapid listener
func Run() (err error) {
	// acquire lock to avoid multiple chapid servers
	chapidLock.Lock()
	defer chapidLock.Unlock()

	// check if chapid is already running
	swapped := atomic.CompareAndSwapInt32(&chapiRunning, 0, 1)
	if !swapped {
		// return err as nil to indicate chapid is already running for current process
		return nil
	}

	chapidResult := make(chan error)
	// start chapid server
	go startChapid(chapidResult)
	// wait for the response on channel
	err = <-chapidResult

	return err
}

// This function will invoke a new chapid listener with socket filename containing current process ID
// NOTE: invoke this function as go routine as it will block on socket listener
func startChapid(result chan error) {
	log.Trace(">>>>> startChapid")
	defer log.Trace("<<<<< startChapid")

	var err error
	// create Listener object
	chapidListener, err = net.Listen("tcp", ":0")
	if err != nil {
		log.Error("listen error, Unable to create ChapidServer ", err.Error())
		result <- err
	} else {
		// Get the allocated TCP Port, and initialize the CHAPI instance data
		port := chapidListener.Addr().(*net.TCPAddr).Port
		err = handler.InitChapiInstanceData(port)
		if err != nil {
			log.Error("initChapiInstanceData error, Unable to create ChapidServer ", err.Error())
			result <- err
		} else {
			// Allocate our mux.Router object
			router := NewRouter()

			// indicate on channel before we block on listener
			result <- nil
			err = http.Serve(chapidListener, router)
			if err != nil {
				log.Tracef("exiting chapid server, err=%v", err.Error())
			}

			// Remove our CHAPI port and key files now that CHAPI has exited
			handler.RemoveChapiInstanceData()
		}
	}

	// Before exiting thread, clear CHAPI running flag
	atomic.StoreInt32(&chapiRunning, 0)
}

// StopChapid will stop the given http listener
func StopChapid() error {
	log.Trace(">>>>> StopChapid")
	defer log.Trace("<<<<< StopChapid")

	// Block any new CHAPI creation request while we're in the middle of trying to close
	// out any existing CHAPI server.
	chapidLock.Lock()
	defer chapidLock.Unlock()

	// stop the listener
	if chapidListener != nil {
		err := chapidListener.Close()
		if err != nil {
			log.Error("Unable to close chapid listener " + chapidListener.Addr().String())
		} else {
			// Wait up to 2 seconds for the CHAPI thread to exit
			for i := 0; (i < 2*10) && (atomic.LoadInt32(&chapiRunning) == 1); i++ {
				time.Sleep(100 * time.Millisecond)
			}
		}
		chapidListener = nil
	}
	return nil
}
