package linux

import (
	"fmt"
	"strings"
	"testing"
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
