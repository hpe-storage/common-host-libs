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

package linux

import (
	"testing"
)

var (
	devPath    = "proc"
	mountPoint = "/proc"
)

func TestGetDeviceFromMountPoint(t *testing.T) {
	dev, err := GetDeviceFromMountPoint(mountPoint)

	if err != nil {
		t.Error(
			"Unexpected error", err,
		)
	}
	if dev != devPath {
		t.Error(
			"For", "dev",
			"expected", devPath,
			"got", dev,
		)
	}
}
func TestGetMountPointFromDevice(t *testing.T) {
	m, err := GetMountPointFromDevice(devPath)

	if err != nil {
		t.Error(
			"Unexpected error", err,
		)
	}
	if m != mountPoint {
		t.Error(
			"For", "mountpoint",
			"expected", mountPoint,
			"got", m,
		)
	}
}

func TestGetSilly(t *testing.T) {
	m, err := GetMountPointFromDevice("")

	if err != nil {
		t.Error(
			"Unexpected error", err,
		)
	}
	if m != "" {
		t.Error(
			"For", "mountpoint",
			"expected", "",
			"got", m,
		)
	}
	dev, err := GetDeviceFromMountPoint("")

	if err != nil {
		t.Error(
			"Unexpected error", err,
		)
	}
	if dev != "" {
		t.Error(
			"For", "dev",
			"expected", "",
			"got", dev,
		)
	}
}
