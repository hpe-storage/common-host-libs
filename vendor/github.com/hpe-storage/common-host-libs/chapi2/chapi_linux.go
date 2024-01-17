// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package chapi2

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"github.com/hpe-storage/common-host-libs/chapi2/cerrors"
	"github.com/hpe-storage/common-host-libs/chapi2/handler"
	"github.com/hpe-storage/common-host-libs/connectivity"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/util"
)

var (
	chapidLock     sync.Mutex
	chapidListener net.Listener
)

var (
	//ChapidSocketPath : the directory for chapid socket
	ChapidSocketPath = util.GetNltHome() + "etc/"
	//ChapidSocketName : chapid socket name
	ChapidSocketName = "chapid"
)

// Additional endpoints only supported by CHAPI for Linux
var platformSpecificEndpoints = []util.Route{
	util.Route{
		Name:        "Recommendations",
		Method:      "GET",
		Pattern:     "/api/v1/recommendations",
		HandlerFunc: handler.GetHostRecommendations,
	},
	util.Route{
		Name:        "DeletingDevices",
		Method:      "GET",
		Pattern:     "/api/v1/deletingdevices",
		HandlerFunc: handler.GetDeletingDevices,
	},
	util.Route{
		Name:        "ChapInfo",
		Method:      "GET",
		Pattern:     "/api/v1/chapinfo",
		HandlerFunc: handler.GetChapInfo,
	},
}

// Run will invoke a new chapid listener with socket filename containing current process ID
func Run() (err error) {
	// check if chapid is already running listening on standard socket or per process socket
	if IsChapidRunning(ChapidSocketPath+ChapidSocketName) ||
		IsChapidRunning(ChapidSocketPath+ChapidSocketName+strconv.Itoa(os.Getpid())) {
		// return err as nil to indicate chapid is already running for current process
		return nil
	}

	// acquire lock to avoid multiple chapid servers
	chapidLock.Lock()
	defer chapidLock.Unlock()

	//first check if the directory exists
	_, isdDir, _ := util.FileExists(ChapidSocketPath)
	if !isdDir {
		// create the directory for   chapidSocket
		err = os.MkdirAll(ChapidSocketPath, 0700)
		if err != nil {
			log.Error("Unable to create directory " + ChapidSocketPath + " for chapid routine to run")
			return err
		}
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
	log.Info(">>>>> startChapid")
	defer log.Info("<<<<< startChapid")

	var err error
	// create chapidSocket for listening
	chapidListener, err = net.Listen("unix", ChapidSocketPath+ChapidSocketName+strconv.Itoa(os.Getpid()))
	if err != nil {
		log.Error("listen error, Unable to create ChapidServer ", err.Error())
		result <- err
		return
	}
	router := NewRouter()
	// indicate on channel before we block on listener
	result <- nil
	err = http.Serve(chapidListener, router)
	if err != nil {
		log.Info("exiting chapid server", err.Error())
	}
}

// StopChapid will stop the given http listener
func StopChapid() error {
	log.Info(">>>>> StopChapid")
	defer log.Info("<<<<< StopChapid")

	chapidLock.Lock()
	defer chapidLock.Unlock()

	// stop the listener
	if chapidListener != nil {
		err := chapidListener.Close()
		if err != nil {
			log.Error("Unable to close chapid listener " + chapidListener.Addr().String())
		}
		chapidListener = nil
		os.RemoveAll(ChapidSocketPath + ChapidSocketName + strconv.Itoa(os.Getpid()))
	}
	return nil
}

// IsChapidRunning return true if chapid is running as part of service listening on given socket
func IsChapidRunning(chapidSocket string) bool {
	chapidLock.Lock()
	defer chapidLock.Unlock()

	//first check if the chapidSocket file exists
	isPresent, _, _ := util.FileExists(chapidSocket)
	if !isPresent {
		return false
	}
	var chapiResp handler.Response
	// generate chapid client for default daemon
	var errResp *cerrors.ChapiError
	chapiResp.Err = &errResp
	chapiClient := connectivity.NewSocketClient(chapidSocket)
	if chapiClient != nil {
		_, err := chapiClient.DoJSON(&connectivity.Request{Action: "GET", Path: "/hosts", Payload: nil, Response: &chapiResp, ResponseError: &chapiResp})
		if errResp != nil {
			log.Error(errResp.Error())
			return false
		}
		if err == nil {
			return true
		}
	}
	return false
}

// RunNimbled :
func RunNimbled(c chan error) {
	err := cleanupExistingSockets()
	if err != nil {
		log.Fatal("Unable to cleanup existing sockets")
	}
	//1. first check if the directory exists
	_, isdDir, _ := util.FileExists(ChapidSocketPath)
	if !isdDir {
		// create the directory for   chapidSocket
		err = os.MkdirAll(ChapidSocketPath, 0700)
		if err != nil {
			log.Fatal("Unable to create directory " + ChapidSocketPath + " for chapid server to run")
		}
	}

	//create chapidSocket for listening
	chapidSocket, err := net.Listen("unix", ChapidSocketPath+ChapidSocketName)
	if err != nil {
		log.Fatal("listen error, Unable to create ChapidServer ", err)
	}

	router := NewRouter()
	go runNimbled(chapidSocket, router, c)

}

func runNimbled(l net.Listener, m *mux.Router, c chan error) {
	log.Info("Serving socket :", l.Addr().String())
	c <- http.Serve(l, m)

	// close the socket
	log.Infof("closing the socket %v", l.Addr().String())
	defer l.Close()
	//cleanup the socket file
	defer os.RemoveAll(ChapidSocketPath + ChapidSocketName)
}

// cleanup the existing unix sockets before creating them again
func cleanupExistingSockets() (err error) {
	log.Info("Cleaning up existing socket")
	//clean up chapid socket
	isPresent, _, err := util.FileExists(ChapidSocketPath + ChapidSocketName)
	if err != nil {
		log.Info("err", err)
		return err
	}
	if isPresent {
		err = os.RemoveAll(ChapidSocketPath + ChapidSocketName)
		if err != nil {
			return err
		}
	}
	return nil
}
