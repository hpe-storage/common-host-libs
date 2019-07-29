// (c) Copyright 2019 Hewlett Packard Enterprise Development LP
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build windows

package windows

import (
	"os"
	"path/filepath"
	"syscall"
)

// Define additional error codes not defined by the src/syscall/types_windows.go module
// (or not defined in older Go versions)
const (
	ERROR_INVALID_PARAMETER syscall.Errno = 87 // // The parameter is incorrect.
)

// List of constant for Windows platform
const (
	Platform               = "windows"
	Proto                  = "tcp"
	PluginListenPort       = "8080"
	GlobalPluginListenPort = "8081"
	DockerListenPort       = "2375"
	Hostname               = "localhost"
)

// Windows paths
var (
	LogPath          string
	PluginHome       string
	PluginCertHome   string
	PluginConfigHome string
	DockerPath       string
)

// Initialize windows package paths
func init() {

	// Get the ProgramData location
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}

	// Initialize the package paths
	LogPath = filepath.Join(programData, `\hpe-storage\log`) + `\` // e.g. C:\ProgramData\hpe-storage\log\
	PluginHome = filepath.Join(programData, `hpe-storage`) + `\`   // e.g. C:\ProgramData\hpe-storage\
	PluginCertHome = filepath.Join(PluginHome, `certs`) + `\`      // e.g. C:\ProgramData\hpe-storage\certs\
	PluginConfigHome = filepath.Join(PluginHome, `conf`) + `\`     // e.g. C:\ProgramData\hpe-storage\conf\
	DockerPath = filepath.Join(programData, `\Docker`) + `\`       // e.g. C:\ProgramData\Docker\
}
