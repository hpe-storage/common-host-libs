// Copyright 2026 Hewlett Packard Enterprise Development LP

package linux

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// fakeFileInfo is a minimal os.FileInfo whose only meaningful field is Name(),
// which is all GetNvmeDeviceFromNamespace consumes when iterating /dev entries.
type fakeFileInfo struct{ name string }

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() any           { return nil }

// TestGetNvmeDeviceFromNamespace_SkipsUnreadableNguid is the ESC-17833 regression:
// the first NVMe namespace (e.g. a local boot drive like nvme0n1) has no readable
// nguid, and the target NVMe/TCP volume is on a later device (nvme1n1). Discovery
// must skip the unreadable device and still find the target instead of aborting.
func TestGetNvmeDeviceFromNamespace_SkipsUnreadableNguid(t *testing.T) {
	origReadDir := nvmeReadDir
	origReadNguid := readNvmeNamespaceNguid
	defer func() {
		nvmeReadDir = origReadDir
		readNvmeNamespaceNguid = origReadNguid
	}()

	const targetSerial = "60002ac00000003e0002ac940002b10c"

	nvmeReadDir = func(string) ([]os.FileInfo, error) {
		return []os.FileInfo{
			fakeFileInfo{name: "nvme0n1"}, // local boot drive: nguid read fails
			fakeFileInfo{name: "nvme1n1"}, // attached NVMe/TCP volume: matches
		}, nil
	}
	readNvmeNamespaceNguid = func(deviceName string) (string, error) {
		if deviceName == "nvme0n1" {
			return "", fmt.Errorf("open /sys/class/block/nvme0n1/subsystem/nvme0n1/nguid: no such file or directory")
		}
		// Return the (dashed) sysfs form; the function normalizes it.
		return "60002ac0-0000-003e-0002-ac940002b10c", nil
	}

	dev, err := GetNvmeDeviceFromNamespace(targetSerial)
	if err != nil {
		t.Fatalf("expected to find device on nvme1n1 despite nvme0n1 nguid failure, got error: %v", err)
	}
	if dev == nil {
		t.Fatalf("expected a device, got nil")
	}
	if dev.SerialNumber != targetSerial {
		t.Fatalf("expected serial %s, got %s", targetSerial, dev.SerialNumber)
	}
	if dev.Pathname != "/dev/nvme1n1" {
		t.Fatalf("expected /dev/nvme1n1, got %s", dev.Pathname)
	}
}

// TestGetNvmeDeviceFromNamespace_NotFound verifies a clean not-found error when no
// device matches (and one namespace has an unreadable nguid).
func TestGetNvmeDeviceFromNamespace_NotFound(t *testing.T) {
	origReadDir := nvmeReadDir
	origReadNguid := readNvmeNamespaceNguid
	defer func() {
		nvmeReadDir = origReadDir
		readNvmeNamespaceNguid = origReadNguid
	}()

	nvmeReadDir = func(string) ([]os.FileInfo, error) {
		return []os.FileInfo{
			fakeFileInfo{name: "nvme0n1"},
			fakeFileInfo{name: "nvme1n1"},
		}, nil
	}
	readNvmeNamespaceNguid = func(deviceName string) (string, error) {
		if deviceName == "nvme0n1" {
			return "", fmt.Errorf("no such file or directory")
		}
		return "aaaa-bbbb", nil
	}

	dev, err := GetNvmeDeviceFromNamespace("does-not-match")
	if err == nil {
		t.Fatalf("expected not-found error, got device %+v", dev)
	}
	if dev != nil {
		t.Fatalf("expected nil device, got %+v", dev)
	}
}
