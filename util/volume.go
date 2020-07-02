package util

import (
	"encoding/json"
	"github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/model"
)

func GetVolumeObject(serialNumber, lunID string) *model.Volume {

	var volObj *model.Volume
	volObj.SerialNumber = serialNumber
	volObj.LunID = lunID

	return volObj
}

func GetSecondaryArrayLUNIds(details string) []int32 {
	var secondaryArrayDetails model.SecondaryBackendDetails
	err := json.Unmarshal([]byte(details), &secondaryArrayDetails)
	if err != nil {
		logger.Tracef("\n Error in GetSecondaryArrayLUNIds %s", err.Error())
	}
	numberOfSecondaryBackends := len(secondaryArrayDetails.PeerArrayDetails)
	var secondaryLunIds []int32 = make([]int32, numberOfSecondaryBackends)
	for i := 0; i < numberOfSecondaryBackends; i++ {
		secondaryLunIds[i] = secondaryArrayDetails.PeerArrayDetails[i].LunID
	}
	return secondaryLunIds
}

func GetSecondaryArrayTargetNames(details string) []string {
	var secondaryArrayDetails model.SecondaryBackendDetails
	err := json.Unmarshal([]byte(details), &secondaryArrayDetails)
	if err != nil {
		logger.Tracef("\n Error in GetSecondaryArrayTargetNames %s", err.Error())
	}
	numberOfSecondaryBackends := len(secondaryArrayDetails.PeerArrayDetails)
	var secondaryTargetNames []string
	for i := 0; i < numberOfSecondaryBackends; i++ {
		for _, targetNameRetrieved := range secondaryArrayDetails.PeerArrayDetails[i].TargetNames {
			secondaryTargetNames = append(secondaryTargetNames, targetNameRetrieved)
		}
	}
	return secondaryTargetNames
}

func GetSecondaryArrayDiscoveryIps(details string) []string {
	var secondaryArrayDetails model.SecondaryBackendDetails
	err := json.Unmarshal([]byte(details), &secondaryArrayDetails)
	if err != nil {
		logger.Tracef("\n Error in GetSecondaryArrayDiscoveryIps %s", err.Error())
	}
	numberOfSecondaryBackends := len(secondaryArrayDetails.PeerArrayDetails)
	var secondaryDiscoverIps []string
	for i := 0; i < numberOfSecondaryBackends; i++ {
		for _, discoveryIpRetrieved := range secondaryArrayDetails.PeerArrayDetails[i].DiscoveryIPs {
			secondaryDiscoverIps = append(secondaryDiscoverIps, discoveryIpRetrieved)
		}
	}
	return secondaryDiscoverIps
}
