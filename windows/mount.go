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
	"github.com/hpe-storage/common-host-libs/util"
	log "github.com/hpe-storage/common-host-libs/logger"
)

//AddPartitionAccessPath - mount the volume to specified flexvolpath.
func AddPartitionAccessPath(flexvolPath, dockerPath string) error {
	// Get volume object for a disk mounted by plugin.
	args := []string{"-command", "$vol = Get-Volume -FilePath", dockerPath, ";",
		"Get-Partition -Volume $vol", "| Add-PartitionAccessPath -AccessPath", flexvolPath}
	_, rc, err := util.ExecCommandOutput("powershell", args)
	if rc != 0 || err != nil {
		log.Errorf("Failed to add partition access of volume %s on the host, error %v", dockerPath, err.Error())
		return err
	}

	return nil
}

//RemovePartitionAccessPath - mount the volume to specified flexvolpath.
func RemovePartitionAccessPath(flexvolPath string) error {
	args := []string{"-command", "$vol = Get-Volume -FilePath", flexvolPath, ";",
		"Get-Partition -Volume $vol", "| Remove-PartitionAccessPath -AccessPath", flexvolPath}
	_, rc, err := util.ExecCommandOutput("powershell", args)
	if rc != 0 || err != nil {
		log.Errorf("Failed to remove partition access path, flexpath %s  error %v", flexvolPath, err.Error())
		return err
	}

	return nil
}

//GetDockerVolAccessPath - Get the docker volume access path of the partition
func GetDockerVolAccessPath(flexvolPath string) (string, error) {
	args := []string{"-command", "$vol = Get-Volume -FilePath", flexvolPath,
		";", "(Get-Partition -Volume $vol).AccessPaths|where {$_ -match \".*hpe-storage-mounts.*\"}"}
	dockerPath, rc, err := util.ExecCommandOutput("powershell", args)
	if rc != 0 || err != nil {
		log.Errorf("Failed to get hpe-storage volume %s access path on the host, error %v", flexvolPath, err.Error())
		return "", err
	}

	return dockerPath, nil
}
