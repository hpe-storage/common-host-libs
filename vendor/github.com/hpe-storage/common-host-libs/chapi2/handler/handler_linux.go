// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package handler

import (
	"encoding/json"
	"net/http"

	"github.com/hpe-storage/common-host-libs/linux"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/tunelinux"
)

//@APIVersion 1.0.0
//@Title GetHostRecommendations
//@Description get Recommendations for=host id=id
//@Accept json
//@Resource /api/v1/recommendations
//@Success 200 linux.Recommendation
//@Router /api/v1/recommendations [get]
func GetHostRecommendations(w http.ResponseWriter, r *http.Request) {
	var chapiResp Response
	var settings []*tunelinux.Recommendation

	settings, err := tunelinux.GetRecommendations()
	if err != nil {
		handleError(w, chapiResp, err, http.StatusInternalServerError)
		return
	}
	chapiResp.Data = settings
	json.NewEncoder(w).Encode(chapiResp)
}

//@APIVersion 1.0.0
//@Title GetDeletingDevices
//@Description get devices in deletion state
//@Accept json
//@Resource /api/v1/deletingdevices
//@Success 200 linux.Recommendation
//@Router /api/v1/deletingdevices [get]
func GetDeletingDevices(w http.ResponseWriter, r *http.Request) {
	var chapiResp Response

	devices := linux.GetDeletingDevices()
	if devices == nil {
		// this is not an error condition, but just means no deletions are pending
		log.Info("no device deletions pending")
	}

	chapiResp.Data = devices
	json.NewEncoder(w).Encode(chapiResp)
}

//@APIVersion 1.0.0
//@Title GetChapInfo
//@Description get iSCSI CHAP info configured on host
//@Accept json
//@Resource /api/v1/chap
//@Success 200 chapi2.ChapInfo
//@Router /api/v1/chap [get]
func GetChapInfo(w http.ResponseWriter, r *http.Request) {
	function := func() (interface{}, error) {
		return linux.GetChapInfo()
	}
	handleRequest(function, "getChapInfo", w, r)
}

// CHAPI for Linux does not need to validate the request header.  See handler_windows.go for the
// checks CHAPI for Windows needs to perform.
func validateRequestHeader(w http.ResponseWriter, r *http.Request) bool {
	return true
}
