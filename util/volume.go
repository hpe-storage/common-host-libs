package util

import (
	"github.com/hpe-storage/common-host-libs/model"
)

func GetVolumeObject(serialNumber, lunID string) *model.Volume {

	var volObj *model.Volume
	volObj.SerialNumber = serialNumber
	volObj.LunID = lunID

	return volObj
}
