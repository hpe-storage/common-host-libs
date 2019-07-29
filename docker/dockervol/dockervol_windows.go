/*
(c) Copyright 2017 Hewlett Packard Enterprise Development LP

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
package dockervol

import (
	"fmt"
	"github.com/hpe-storage/common-host-libs/connectivity"
)

var (
	hostName        = "http://localhost"
	port     uint16 = 8081
)

// GetHostURL to consruct HTTP URL of the form "hostname:port"
func GetHostURL(hostName string, port uint16) string {
	return fmt.Sprintf("%v:%v", hostName, port)
}

// NewDockerVolumePlugin creates a DockerVolumePlugin which can be used to communicate with
// a Docker Volume Plugin. The communication happens over http on Windows.
func NewDockerVolumePlugin(options *Options) (*DockerVolumePlugin, error) {
	hostUrl := GetHostURL(hostName, port)
	var err error

	dvp := &DockerVolumePlugin{
		stripK8sOpts:                 options.StripK8sFromOptions,
		client:                       connectivity.NewHTTPClientWithTimeout(hostUrl, dvpSocketTimeout),
		ListOfStorageResourceOptions: options.ListOfStorageResourceOptions,
		FactorForConversion:          options.FactorForConversion,
	}

	if options.SupportsCapabilities {
		// test connectivity
		_, err = dvp.Capabilities()
		if err != nil {
			return dvp, err
		}
	}

	return dvp, nil

}
