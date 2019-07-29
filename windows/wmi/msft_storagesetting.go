// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package wmi handles WMI queries
package wmi

import (
	log "github.com/hpe-storage/common-host-libs/logger"
)

// RescanDisks calls the UpdateHostStorageCache method of the MSFT_StorageSetting WMI class.
// It's equivalent to performing a rescan within diskpart.exe.
func RescanDisks() error {
	log.Info(">>>>> RescanDisks")
	defer log.Info("<<<<< RescanDisks")

	results, err := ExecWmiMethod("MSFT_StorageSetting", "UpdateHostStorageCache", rootMicrosoftWindowsStorage)

	if results != nil {
		// Log the RescanDisks result
		log.Infof("RescanDisks status = %v", results.Value())

		// Release the VARIANT
		results.Clear()
	}

	return err
}
