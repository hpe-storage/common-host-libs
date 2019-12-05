package linux

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/hpe-storage/common-host-libs/model"
	"github.com/hpe-storage/common-host-libs/util"
)

var (
	input                           = []string{"PRUNEPATHS=\"/tmp /var/spool /media /home/.ecryptfs /var/lib/schroot\"", "PRUNE_BIND_MOUNTS=\"yes\""}
	inputWithSpace                  = []string{"PRUNEPATHS = \"/tmp /var/spool /media /home/.ecryptfs /var/lib/schroot\"", "PRUNE_BIND_MOUNTS=\"yes\""}
	inputWithNoPathEntry            = []string{"PRUNEPATHS=\"\"", "PRUNE_BIND_MOUNTS=\"yes\""}
	inputWithMountDirAlreadyPresent = []string{"PRUNEPATHS=\"/tmp /var/spool /media /home/.ecryptfs /var/lib/schroot /opt/nimble/\"", "PRUNE_BIND_MOUNTS=\"yes\""}
	mountDir                        = "/opt/nimble/"
)

func TestMountDevice(t *testing.T) {
	_, err := MountDevice(nil, "", nil)
	if err == nil {
		t.Error("empty device or mountpoint should not be allowed to be mounted")
	}
}

// TestCreateFilesystemWithOptions take 18 seconds on a OSX laptop running in a vm and in a container
// This test wont fire unless the GOOS is linux and mkfs.xfs is installed (it isn't in golang:1.x)
func TestCreateFilesystemWithOptions(t *testing.T) {
	// This test requires linux
	if runtime.GOOS != "linux" {
		return
	}

	// This test requires xfs
	// apt-get update && apt-get install xfsprogs
	_, rc, _ := util.ExecCommandOutputWithTimeout("which", []string{"mkfs.xfs"}, 5)
	if rc != 0 {
		return
	}

	device := &model.Device{
		AltFullPathName: "/tmp/disk",
	}

	var optionTests = []struct {
		name      string
		options   []string
		shouldErr bool
	}{
		{"xfs with wrong options", []string{"-m", "XXXscrc=1", "-K", "-i", "maxpct=0"}, true},
		{"xfs with no options", nil, false},
		{"xfs with options", []string{"-m", "crc=1", "-K", "-i", "maxpct=0"}, false},
	}

	for _, tc := range optionTests {
		t.Run(tc.name, func(t *testing.T) {
			// create a small device
			_, _, err := util.ExecCommandOutputWithTimeout("rm", []string{"-f", device.AltFullPathName}, 5)
			if err != nil {
				t.Error(tc.name, "Unable to remove device", device.AltFullPathName, err.Error())
			}
			_, _, err = util.ExecCommandOutputWithTimeout("dd", []string{"if=/dev/zero", "of=" + device.AltFullPathName, "count=4096", "bs=4096"}, 30)
			if err != nil {
				t.Error(tc.name, "Unable to create device ", device.AltFullPathName, err.Error())
			}

			// try to treat it as a real device
			err = SetupFilesystemWithOptions(device, "xfs", tc.options)
			if err == nil {
				t.Error(tc.name, "Expected SetupFilesystemWithOptions(...) to fail")
			}

			// test filesystem creation
			err = RetryCreateFileSystemWithOptions(device.AltFullPathName, "xfs", tc.options)
			if err != nil && tc.shouldErr != true {
				t.Error(tc.name, "Expected RetryCreateFileSystemWithOptions(...) should not fail!", err.Error())
			}
		})
	}

}

func TestUnmount(t *testing.T) {
	_, err := UnmountFileSystem("")
	if err == nil {
		t.Error("empty filesystem cannot be unmounted")
	}
}

func TestExcludeMountDirFromUpdateDb(t *testing.T) {

	changedEntries, isChanged := updateDbConfiguration(input, mountDir)
	if !isChanged || !strings.Contains(strings.Join(changedEntries, "\n"), mountDir) {
		t.Errorf("unable to include mount directory in PRUNEPATHS")
	}

	changedEntries, isChanged = updateDbConfiguration(inputWithSpace, mountDir)
	if !isChanged || !strings.Contains(strings.Join(changedEntries, "\n"), mountDir) {
		t.Errorf("unable to include mount directory in PRUNEPATHS with space")
	}

	changedEntries, isChanged = updateDbConfiguration(inputWithNoPathEntry, mountDir)
	if !isChanged || !strings.Contains(strings.Join(changedEntries, "\n"), mountDir) {
		t.Errorf("unable to include mount directory in empty PRUNEPATHS")
	}

	changedEntries, isChanged = updateDbConfiguration(inputWithMountDirAlreadyPresent, mountDir)
	if isChanged {
		t.Errorf("ducplicate entries were added in PRUNEPATHS")
	}

	// make sure duplicate entries are not added
	checkForDuplicates(changedEntries, t)
}

func checkForDuplicates(changedEntries []string, t *testing.T) {
	found := false
	for _, entry := range changedEntries {
		fmt.Println(entry)
		if !strings.Contains(entry, "PRUNEPATHS") {
			continue
		}
		entry = strings.Replace(entry, "PRUNEPATHS=", "", 1)
		entry = strings.Trim(entry, "\"")
		paths := strings.Fields(entry)
		for _, path := range paths {
			if !strings.EqualFold(path, mountDir) {
				continue
			}
			if found {
				t.Error("duplicate entries were found in PRUNEPATHS ")
				break
			}
			found = true
		}
		break
	}
}
