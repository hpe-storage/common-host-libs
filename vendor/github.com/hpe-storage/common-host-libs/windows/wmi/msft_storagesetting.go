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
	log.Trace(">>>>> RescanDisks")
	defer log.Trace("<<<<< RescanDisks")

	results, err := ExecWmiMethod("MSFT_StorageSetting", "UpdateHostStorageCache", rootMicrosoftWindowsStorage)

	if results != nil {
		// Log the RescanDisks result
		log.Tracef("RescanDisks status = %v", results.Value())

		// Release the VARIANT
		results.Clear()
	}

	return err
}
