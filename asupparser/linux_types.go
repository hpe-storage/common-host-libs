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

// Root represents root element of XML
type Root struct {
	ITNInfo      ITNInfo      `xml:"itn_info,omitempty"`
	HostInfoRoot HostInfoRoot `xml:"root,omitempty"`
}

// MultipathBlacklist represents blacklist section of multipath.conf
type MultipathBlacklist struct {
	MultipathDevice  MultipathDevice  `xml:"device,omitempty"`
	MultipathEntries MultipathEntries `xml:"entries,omitempty"`
}

// MultipathBlacklistExceptions represents blacklist_exceptions section of multipath.conf
type MultipathBlacklistExceptions struct {
	MultipathDevice  MultipathDevice  `xml:"device,omitempty"`
	MultipathEntries MultipathEntries `xml:"entries,omitempty"`
}

// MultipathConf represents multipath.conf settings
type MultipathConf struct {
	MultipathBlackList           MultipathBlacklist           `xml:"blacklist,omitempty"`
	MultipathBlacklistExceptions MultipathBlacklistExceptions `xml:"blacklist_exceptions,omitempty"`
	MultipathDefaults            MultipathDefaults            `xml:"defaults,omitempty"`
	MultipathDevices             MultipathDevices             `xml:"devices,omitempty"`
}

// MultipathDefaults represents defaults section of multipath.conf
type MultipathDefaults struct {
	MultipathProperties MultipathProperties `xml:"properties,omitempty"`
}

// MultipathDevice represents device section of multipath.conf
type MultipathDevice struct {
	MultipathProperties MultipathProperties `xml:"properties,omitempty"`
}

// MultipathDevices represents devices section of multipath.conf
type MultipathDevices struct {
	MultipathDevice MultipathDevice `xml:"device,omitempty"`
}

// Distro represents  OS distribution
type Distro struct {
	Text string `xml:",chardata"`
}

// Docker represents docker daemon of NLT
type Docker struct {
	Text string `xml:",chardata"`
}

// MultipathEntries represents parameter entries in multipath.conf
type MultipathEntries struct {
	MultipathProperties MultipathProperties `xml:"properties,omitempty"`
}

// Hostname represents hostname of  system
type Hostname struct {
	Text string `xml:",chardata"`
}

// ITNInfo represents ITN Info header element of host info collection
type ITNInfo struct {
	Attrtimestamp string `xml:"timestamp,attr"`
	Text          string `xml:",chardata"`
}

// Kernel represents kernel version of the  host
type Kernel struct {
	Text string `xml:",chardata"`
}

// Manufacturer represents system manufacturer info
type Manufacturer struct {
	Text string `xml:",chardata"`
}

// Multipath represents multipath version and multipath.conf settings
type Multipath struct {
	Attrversion   string        `xml:"version,attr"`
	MultipathConf MultipathConf `xml:"conf,omitempty"`
}

// NCM represents NCM info
type NCM struct {
	Text string `xml:",chardata"`
}

// NCMScaleout represent scale-out mode info
type NCMScaleout struct {
	Text string `xml:",chardata"`
}

// NLT represents NLT version and component info
type NLT struct {
	Attrversion string      `xml:"version,attr"`
	Docker      Docker      `xml:"docker,omitempty"`
	NCM         NCM         `xml:"ncm,omitempty"`
	NCMScaleout NCMScaleout `xml:"ncmscaleout,omitempty"`
	Oracle      Oracle      `xml:"oracle,omitempty"`
}

// Oracle represents oracle daemon info
type Oracle struct {
	Text string `xml:",chardata"`
}

// OS represents various information about OS
type OS struct {
	Distro  Distro  `xml:"distro,omitempty"`
	Kernel  Kernel  `xml:"kernel,omitempty"`
	Sanboot Sanboot `xml:"sanboot,omitempty"`
	Version Version `xml:"version,omitempty"`
}

// ProductName represent server information
type ProductName struct {
	Text string `xml:",chardata"`
}

// MultipathProperties represents various property key-value pairs in multipath.conf
type MultipathProperties struct {
	MultipathProperty []MultipathProperty `xml:"property,omitempty"`
}

// MultipathProperty represents key-value property in multipath.conf
type MultipathProperty struct {
	Attrname  string `xml:"name,attr"`
	Attrvalue string `xml:"value,attr"`
}

// HostInfoRoot represents root element of  xml data collected
type HostInfoRoot struct {
	AttrHost    string     `xml:"host,attr"`
	AttrVersion string     `xml:"version,attr"`
	Multipath   Multipath  `xml:"multipath,omitempty"`
	NLT         NLT        `xml:"nlt,omitempty"`
	OS          OS         `xml:"os,omitempty"`
	SystemInfo  SystemInfo `xml:"systeminfo,omitempty"`
}

// Sanboot represents is system is booted from SAN
type Sanboot struct {
	Text string `xml:",chardata"`
}

// SystemInfo represents system information
type SystemInfo struct {
	AttrTimestamp string       `xml:"timestamp,attr"`
	Hostname      Hostname     `xml:"hostname,omitempty"`
	Manufacturer  Manufacturer `xml:"manufacturer,omitempty"`
	ProductName   ProductName  `xml:"productname,omitempty"`
}

// Version represents OS version info
type Version struct {
	Text string `xml:",chardata"`
}
