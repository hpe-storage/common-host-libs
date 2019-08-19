// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// Copyright 2012 The Go Authors. All rights reserved.
// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"path/filepath"

	"github.com/hpe-storage/common-host-libs/chapi2/chapiclient"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/windows"
)

func cleanup() {
}

func main() {

	// Create our log file
	chapiLogFile := filepath.Join(windows.LogPath, "chapi_client_tester.log")
	log.InitLogging(chapiLogFile, &log.LogParams{Level: "trace"}, false)
	defer cleanup()

	chapiFolder := `C:\Users\berhanm\go\src\github.com\hpe-storage\common-host-utils\cmd\chapid`

	chapiClient, err := chapiclient.NewChapiClient(chapiFolder)
	if err != nil {
		fmt.Printf("Failed NewChapiClient, err=%v\n\n", err)
	}

	chapi2, err := chapiClient.GetDevices("")
	fmt.Printf("err = %v\n", err)
	fmt.Printf("Objects:\n%v\n\n", chapi2)
}
