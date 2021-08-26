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

package dockervol

import (
	"os"
	"testing"
)

const (
	volumeName = "testingisfun"
	mountID    = "123"
)

var (
	options = &Options{
		SocketPath: "/var/run/docker/plugins/nimble.sock",
		Debug:      true,
	}
)

func TestDockerVolumeLife(t *testing.T) {
	_, err := os.Stat(options.SocketPath)
	if err != nil {
		t.Skip("Skipping TestDockerVolumeLife can't find ", options.SocketPath)
	}

	c, _ := NewDockerVolumePlugin(options)
	cap, _ := c.Capabilities()
	t.Logf("capabilities=%v", cap)

	info, err := c.Get(volumeName)
	if err != nil {
		cleanup(c)
	}

	vol, err := c.Create(volumeName, map[string]interface{}{"size": "123"})
	if err != nil {
		t.Fatalf("unable to create volume")
	}

	if vol == "" {
		t.Logf("create returned %v, setting to %s", vol, volumeName)
		vol = volumeName
	}

	mnt, err := c.Mount(vol, mountID)
	if err != nil {
		cleanup(c)
		t.Fatalf("unable to mount volume")
	}

	info, err = c.Get(vol)
	if err != nil {
		cleanup(c)
	}
	t.Logf("mountPoint=%v", info.Volume.Mountpoint)

	if info.Volume.Mountpoint != mnt {
		cleanup(c)
		t.Fatalf("Mountpoint should return %v; got %v", mnt, info.Volume.Mountpoint)
	}

	cleanup(c)
}

func cleanup(c *DockerVolumePlugin) {
	c.Unmount(volumeName, mountID)
	c.Delete(volumeName, "")
}
