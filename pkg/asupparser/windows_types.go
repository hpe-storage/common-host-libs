/*
(c) Copyright 2018 Hewlett Packard Enterprise Development LP

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package asupparser

import (
	"encoding/xml"
)

///////////////////////////////////////////////////////////////////////////////
// Host Information Phase 1 - Windows XML Layout
///////////////////////////////////////////////////////////////////////////////

// XMLNodeHostInfo represent root element for host information
type XMLNodeHostInfo struct {
	// OS generic XML nodes
	XMLName    xml.Name `xml:"root"`
	Host       string   `xml:"host,attr"`
	Version    string   `xml:"version,attr"`
	SystemInfo string   `xml:"systeminfo"`
	OS         string   `xml:"os"`

	// Windows specific XML nodes
	GetWindowsFeature string `xml:"GET_WINDOWSFEATURE"`
	MpioRegisteredDsm string `xml:"MPIO_REGISTERED_DSM"`
}

// XMLArrayHeader represents header added by GDD for host info
type XMLArrayHeader struct {
	TimeStamp string `xml:"timestamp,attr"`
}

///////////////////////////////////////////////////////////////////////////////
// Host Information Phase 1 - Windows JSON Node Data Layout
///////////////////////////////////////////////////////////////////////////////

// JSONSystemInfo represents systeminfo element
type JSONSystemInfo struct {
	Name         string
	Manufacturer string
	Model        string
}

// JSONOS represents os element
type JSONOS struct {
	Name    string
	Version string
}

// JSONGetWindowsFeature represents windows feature element
type JSONGetWindowsFeature struct {
	Name        string
	Installed   bool
	FeatureType string
}

// JSONDsmParameter represents DSM information
type JSONDsmParameter struct {
	DsmName    string
	DsmVersion string
}

// JSONMpioRegisteredDsms represents registered windows DSM information
type JSONMpioRegisteredDsms struct {
	NumberDSMs    int
	DsmParameters []JSONDsmParameter
}

// HostInformation represents Windows host information
type HostInformation struct {
	SystemInfo         JSONSystemInfo
	OS                 JSONOS
	GetWindowsFeature  []JSONGetWindowsFeature
	MpioRegisteredDsms JSONMpioRegisteredDsms
}

// XMLDataEntry represents each host entry in the host info log
type XMLDataEntry struct {
	TimeStamp   string
	XMLHostInfo string
}
