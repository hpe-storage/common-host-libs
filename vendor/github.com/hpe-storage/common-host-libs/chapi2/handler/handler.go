// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hpe-storage/common-host-libs/chapi2/cerrors"
	chapiDriver "github.com/hpe-storage/common-host-libs/chapi2/driver"
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	log "github.com/hpe-storage/common-host-libs/logger"
)

var (
	driver chapiDriver.Driver
)

const (
	// Shared error messages
	errorMessageEmptyFileSystem       = "empty filesystem type passed in the request"
	errorMessageEmptyMountID          = "empty mount id passed in the request"
	errorMessageEmptySerialNumber     = "empty serial number passed in the request"
	errorMessageHTTPHeaderNotProvided = "http.Header not provided for authorization"
	errorMessageInvalidToken          = "invalid token: "
	errorMessageTokenNotSupplied      = "local access token not supplied"
)

//Response :
type Response struct {
	Data interface{} `json:"data,omitempty"`
	Err  interface{} `json:"errors,omitempty"`
}

func init() {
	driver = &chapiDriver.ChapiServer{}
}

//@APIVersion 1.0.0
//@Title GetHostInfo
//@Description retrieves specific host information
//@Accept json
//@Resource /api/v1/hosts
//@Success 200 Host
//@Router /api/v1/hosts [get]
func GetHostInfo(w http.ResponseWriter, r *http.Request) {
	if !validateRequestHeader(w, r) {
		return
	}
	var chapiResp Response
	host, err := driver.GetHostInfo()
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}
	chapiResp.Data = host
	json.NewEncoder(w).Encode(chapiResp)
}

//@APIVersion 1.0.0
//@Title GetHostNetworks
//@Description get host networks
//@Accept json
//@Resource /api/v1/networks
//@Success 200 Network
//@Router /api/v1/networks [get]
func GetHostNetworks(w http.ResponseWriter, r *http.Request) {
	if !validateRequestHeader(w, r) {
		return
	}
	var chapiResp Response
	var nics []*model.Network

	nics, err := driver.GetHostNetworks()
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}
	chapiResp.Data = nics
	json.NewEncoder(w).Encode(chapiResp)
}

//@APIVersion 1.0.0
//@Title GetHostInitiators
//@Description get Initiators
//@Accept json
//@Resource /api/v1/initiators
//@Success 200 Initiators
//@Router /api/v1/initiators [get]
func GetHostInitiators(w http.ResponseWriter, r *http.Request) {
	if !validateRequestHeader(w, r) {
		return
	}
	var chapiResp Response
	var inits []*model.Initiator

	inits, err := driver.GetHostInitiators()
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}
	chapiResp.Data = inits
	json.NewEncoder(w).Encode(chapiResp)
}

//@APIVersion 1.0.0
//@Title GetDevices
//@Description retrieves all devices on host, optionally with serial filter
//@Accept json
//@Resource /api/v1/devices
//@Success 200 {array} Devices
//@Router /api/v1/devices [get]
func GetDevices(w http.ResponseWriter, r *http.Request) {
	if !validateRequestHeader(w, r) {
		return
	}
	var chapiResp Response
	serialNumber := ""
	keys, ok := r.URL.Query()["serial"]

	if ok && len(keys[0]) > 0 {
		serialNumber = keys[0]
	}
	devices, err := driver.GetDevices(serialNumber)
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}
	chapiResp.Data = devices
	json.NewEncoder(w).Encode(chapiResp)
}

//@APIVersion 1.0.0
//@Title GetAllDeviceDetails
//@Description retrieves all devices details on host, optionally with serial filter
//@Accept json
//@Resource /api/v1/devices
//@Success 200 {array} Devices
//@Router /api/v1/devices/details [get]
func GetAllDeviceDetails(w http.ResponseWriter, r *http.Request) {
	if !validateRequestHeader(w, r) {
		return
	}
	var chapiResp Response
	serialNumber := ""
	keys, ok := r.URL.Query()["serial"]

	if ok && len(keys[0]) > 0 {
		serialNumber = keys[0]
	}
	devices, err := driver.GetAllDeviceDetails(serialNumber)
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}
	chapiResp.Data = devices
	json.NewEncoder(w).Encode(chapiResp)
}

//@APIVersion 1.0.0
//@Title GetPartitionsForDevice
//@Description get all partitions for a Nimble Device fpr host id=id and device serialnumber=serialnumber
//@Accept json
//@Resource /api/v1/devices/{serialNumber}/partitions
//@Success 200 {array} DevicePartitions
//@Router /api/v1/devices/{serialNumber}/partitions [get]
func GetPartitionsForDevice(w http.ResponseWriter, r *http.Request) {
	if !validateRequestHeader(w, r) {
		return
	}
	var chapiResp Response
	vars := mux.Vars(r)
	serialNumber := vars["serialNumber"]

	if serialNumber == "" {
		handleError(w, chapiResp, errors.New(errorMessageEmptySerialNumber), http.StatusBadRequest)
		return
	}

	// Located the device. Now find all partitions
	partitions, err := driver.GetPartitionInfo(serialNumber)
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}
	chapiResp.Data = partitions
	json.NewEncoder(w).Encode(chapiResp)
}

// Create host device with attributes passed in the body of the http request
//@APIVersion 1.0.0
//@Title CreateDevice
//@Description attach nimble device for the PublishInfo passed
//@Accept json
//@Resource /api/v1/devices
//@Success 200 {array} Device
//@Router /api/v1/devices [post]
func CreateDevice(w http.ResponseWriter, r *http.Request) {
	if !validateRequestHeader(w, r) {
		return
	}
	var chapiResp Response

	var publishInfo *model.PublishInfo
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&publishInfo)
	defer r.Body.Close()

	if err != nil {
		handleError(w, chapiResp, err, http.StatusBadRequest)
		return
	}

	devices, err := driver.CreateDevice(*publishInfo)
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}
	chapiResp.Data = devices
	json.NewEncoder(w).Encode(chapiResp)
}

// DeleteDevice : disconnect and delete the device from the host
//@APIVersion 1.0.0
//@Title DeleteDevice
//@Description delete device for device serialnumber=serialnumber
//@Accept json
//@Resource /api/v1/devices/{serialNumber}
//@Success 200
//@Router /api/v1/devices/{serialNumber} [delete]
func DeleteDevice(w http.ResponseWriter, r *http.Request) {
	if !validateRequestHeader(w, r) {
		return
	}
	var chapiResp Response
	vars := mux.Vars(r)
	serialNumber := vars["serialNumber"]

	if serialNumber == "" {
		handleError(w, chapiResp, errors.New(errorMessageEmptySerialNumber), http.StatusBadRequest)
		return
	}

	err := driver.DeleteDevice(serialNumber)
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}

	chapiResp.Data = &model.Device{}
	json.NewEncoder(w).Encode(chapiResp)
}

//@APIVersion 1.0.0
//@Title OfflineDevice
//@Description offline the device on host with specific serialNumber
//@Accept json
//@Resource /api/v1/devices/{serialNumber}
//@Success 200
//@Router /api/v1/devices/{serialNumber}/actions/offline [put]
func OfflineDevice(w http.ResponseWriter, r *http.Request) {
	if !validateRequestHeader(w, r) {
		return
	}
	var chapiResp Response
	vars := mux.Vars(r)
	serialNumber := vars["serialNumber"]

	if serialNumber == "" {
		handleError(w, chapiResp, errors.New(errorMessageEmptySerialNumber), http.StatusBadRequest)
		return
	}

	err := driver.OfflineDevice(serialNumber)
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}

	chapiResp.Data = &model.Device{}
	json.NewEncoder(w).Encode(chapiResp)
	return
}

//@APIVersion 1.0.0
//@Title CreateFileSystem on device
//@Description create a filesysten on the device serialnumber=serialnumber
//@Accept json
//@Resource /api/v1/devices/{serialNumber}/filesystem/{fileSystem}
//@Success 200 {array}
//@Router /api/v1/devices/{serialNumber}/filesystem/{fileSystem} [put]
func CreateFileSystem(w http.ResponseWriter, r *http.Request) {
	if !validateRequestHeader(w, r) {
		return
	}
	var chapiResp Response
	vars := mux.Vars(r)
	serialNumber := vars["serialNumber"]
	fileSystem := vars["fileSystem"]

	if serialNumber == "" {
		handleError(w, chapiResp, errors.New(errorMessageEmptySerialNumber), http.StatusBadRequest)
		return
	}

	if fileSystem == "" {
		handleError(w, chapiResp, errors.New(errorMessageEmptyFileSystem), http.StatusBadRequest)
		return
	}

	err := driver.CreateFileSystem(serialNumber, fileSystem)
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(chapiResp)
}

//@APIVersion 1.0.0
//@Title GetMounts
//@Description retrieves all mounts on host, optionally with serial filter
//@Accept json
//@Resource /api/v1/mounts
//@Success 200 {array} Mounts
//@Router /api/v1/mounts [get]
func GetMounts(w http.ResponseWriter, r *http.Request) {
	if !validateRequestHeader(w, r) {
		return
	}
	var chapiResp Response
	serialNumber := ""
	keys, ok := r.URL.Query()["serial"]

	if ok && len(keys[0]) > 0 {
		serialNumber = keys[0]
	}
	mounts, err := driver.GetMounts(serialNumber)
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}
	chapiResp.Data = mounts
	json.NewEncoder(w).Encode(chapiResp)
}

//@APIVersion 1.0.0
//@Title GetAllMountDetails
//@Description retrieves all mount details on host, optionally with serial filter
//@Accept json
//@Resource /api/v1/mounts
//@Success 200 {array} Mounts
//@Router /api/v1/mounts/details [get]
func GetAllMountDetails(w http.ResponseWriter, r *http.Request) {
	if !validateRequestHeader(w, r) {
		return
	}
	var chapiResp Response
	serialNumber := ""
	mountId := ""
	keys, ok := r.URL.Query()["serial"]

	if ok && len(keys[0]) > 0 {
		serialNumber = keys[0]
	}

	keys, ok = r.URL.Query()["mountId"]
	if ok && len(keys[0]) > 0 {
		mountId = keys[0]
	}
	mounts, err := driver.GetAllMountDetails(serialNumber, mountId)
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}
	chapiResp.Data = mounts
	json.NewEncoder(w).Encode(chapiResp)
}

//@APIVersion 1.0.0
//@Title  CreateMount
//@Description Mount an attached device with a details passed in the request
//@Accept json
//@Resource /api/v1/mounts
//@Success 200 {array} Mount
//@Router /api/v1/mounts [post]
func CreateMount(w http.ResponseWriter, r *http.Request) {
	if !validateRequestHeader(w, r) {
		return
	}
	var chapiResp Response
	var mount *model.Mount

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&mount)
	defer r.Body.Close()

	if err != nil {
		handleError(w, chapiResp, err, http.StatusBadRequest)
		return
	}

	if mount.SerialNumber == "" {
		handleError(w, chapiResp, errors.New(errorMessageEmptySerialNumber), http.StatusBadRequest)
		return
	}

	mnt, err := driver.CreateMount(mount.SerialNumber, mount.MountPoint, mount.FsOpts)
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}
	chapiResp.Data = mnt
	json.NewEncoder(w).Encode(chapiResp)
}

//@APIVersion 1.0.0
//@Title  DeleteMount
//@Description Unmount specified mount point on the host
//@Accept json
//@Resource /mounts
//@Success 200 {array} Mount
//@Router /api/v1/mounts/{mountId} [delete]
func DeleteMount(w http.ResponseWriter, r *http.Request) {
	if !validateRequestHeader(w, r) {
		return
	}
	var chapiResp Response
	var serialNumber string
	vars := mux.Vars(r)
	mountId := vars["mountId"]
	if mountId == "" {
		handleError(w, chapiResp, errors.New(errorMessageEmptyMountID), http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&serialNumber)
	defer r.Body.Close()
	if err != nil {
		handleError(w, chapiResp, err, http.StatusBadRequest)
		return
	}

	err = driver.DeleteMount(serialNumber, mountId)
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}

	chapiResp.Data = &model.Mount{}
	json.NewEncoder(w).Encode(chapiResp)
}

// standard method for handling requests
func handleRequest(function func() (interface{}, error), functionName string, w http.ResponseWriter, r *http.Request) {
	var chapiResp Response

	data, err := function()
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}

	chapiResp.Data = data
	json.NewEncoder(w).Encode(chapiResp)
}

func handleError(w http.ResponseWriter, chapiResp Response, err error, statusCode int) {
	log.Error("Err :", err.Error())
	w.WriteHeader(statusCode)
	chapiResp.Err = cerrors.NewChapiError(err)
	json.NewEncoder(w).Encode(chapiResp)
}
