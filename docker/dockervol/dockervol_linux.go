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
	"strings"

	"github.com/hpe-storage/common-host-libs/connectivity"
)

// NewDockerVolumePlugin creates a DockerVolumePlugin which can be used to communicate with
// a Docker Volume Plugin.  options.socketPath can be the full path to the socket file or
// the name of a Docker V2 plugin.  In the case of the V2 plugin, the name of th plugin
// is used to look up the full path to the socketfile.
func NewDockerVolumePlugin(options *Options) (*DockerVolumePlugin, error) {
	var err error
	if !strings.HasPrefix(options.SocketPath, "/") {
		// this is a v2 plugin, so we need to find its socket file
		options.SocketPath, err = getV2PluginSocket(options.SocketPath, "")
	}
	if err != nil {
		return nil, err
	}

	if options.SocketPath == "" {
		options.SocketPath = defaultSocketPath
	}
	dvp := &DockerVolumePlugin{
		stripK8sOpts:                 options.StripK8sFromOptions,
		client:                       connectivity.NewSocketClientWithTimeout(options.SocketPath, dvpSocketTimeout),
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
