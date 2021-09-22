// Copyright 2019 Hewlett Packard Enterprise Development LP
package csp

import (
	"os"
	"testing"

	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/model"
	"github.com/hpe-storage/common-host-libs/storageprovider"
	"github.com/stretchr/testify/assert"
)

const (
	volumeName   = "testCspVol"
	volumeSize   = 1024 * 1024 * 1024
	snapshotName = "testCspSnapshot"
	cloneName    = "testCspVolClone"
	cloneSize    = volumeSize
)

var (
	backend = ""
)

func TestPluginSuite(t *testing.T) {
	// uncomment to run integration tests
	// _TestPluginSuite(t)
}

func getBackend(t *testing.T) string {
	ip, isSet := os.LookupEnv("BACKEND")
	if isSet {
		backend = ip
	}
	if backend == "" {
		t.Fatal("valid backend IP is required for integration tests. This can be set using BACKEND env")
	}
	return backend
}

// nolint: gocyclo
func _TestPluginSuite(t *testing.T) {
	log.InitLogging("container-storage-provider-test.log", nil, false)

	provider := realCsp(t)

	// create a parent volume
	config := make(map[string]interface{})
	config["test"] = "test"
	volume, err := provider.CreateVolume(volumeName, volumeName, volumeSize, config)
	if err != nil {
		t.Fatal("Failed to create volume " + volumeName)
	}

	// Clear the provider auth-token (empty).
	// This is to test if login attempt is made when cached authToken is empty.
	provider.AuthToken = ""

	volume, err = provider.GetVolume(volume.ID)
	if err != nil {
		t.Fatal("Error retrieving volume")
	}
	assert.Equal(t, volume.Name, volumeName)

	volumeByName, err := provider.GetVolumeByName(volume.Name)
	if err != nil {
		t.Fatal("Error retrieving volume by name")
	}
	assert.Equal(t, volume.ID, volumeByName.ID)

	volumes, err := provider.GetVolumes()
	if err != nil {
		t.Fatal("Failed to get volumes")
	}
	assert.True(t, len(volumes) != 0)

	// CloneVolume without a source snapshot ID will indirectly test snapshot creation
	clone := createClone(t, provider, volume.ID, "", cloneSize)

	// Get the auto created snapshot
	snapshot, err := provider.GetSnapshot(clone.BaseSnapID)
	if err != nil {
		t.Fatal("Error retrieving snapshot")
	}
	assert.Equal(t, snapshot.ID, clone.BaseSnapID)

	// Delete the clone
	deleteVolume(t, provider, clone)

	// Base snapshot gets deleted when clone is deleted

	// Create a new snapshot manually
	snapshot, err = provider.CreateSnapshot(snapshotName, snapshotName, volume.ID, nil)
	if err != nil {
		t.Fatal("Failed to create snapshot " + snapshotName)
	}
	assert.Equal(t, snapshot.VolumeID, volume.ID)

	// Clone from that snapshot
	clone = createClone(t, provider, "", snapshot.ID, cloneSize)

	// Delete the clone
	deleteVolume(t, provider, clone)

	// Delete the snapshot
	deleteSnapshot(t, provider, snapshot)

	// Delete the parent
	deleteVolume(t, provider, volume)
}

func realCsp(t *testing.T) *ContainerStorageProvider {
	provider, err := NewContainerStorageProvider(
		&storageprovider.Credentials{
			Backend:     getBackend(t),
			ServicePort: 443,
			ContextPath: "/csp",
			Username:    "admin",
			Password:    "admin",
		},
	)
	if err != nil {
		t.Fatalf("Error building CSP, Error: %s", err.Error())
	}

	return provider
}

func createClone(t *testing.T, provider *ContainerStorageProvider, sourceVolumeID, snapshotID string, size int64) *model.Volume {
	config := make(map[string]interface{})
	config["test"] = "test"
	clone, err := provider.CloneVolume(cloneName, cloneName, sourceVolumeID, snapshotID, size, config)
	if err != nil {
		t.Fatal("Failed to clone volume")
	}
	assert.Equal(t, clone.Name, cloneName)
	assert.Equal(t, clone.Size, int64(cloneSize))

	snapshot, err := provider.GetSnapshot(clone.BaseSnapID)
	if err != nil {
		t.Fatal("Error retrieving snapshot of clone")
	}
	assert.Equal(t, snapshot.ID, clone.BaseSnapID)

	return clone
}

// nolint: dupl
func deleteVolume(t *testing.T, provider *ContainerStorageProvider, volume *model.Volume) {
	err := provider.DeleteVolume(volume.ID, true)
	if err != nil {
		t.Fatal("Could not delete volume " + volume.Name + ".  Error: " + err.Error())
	}
	volume, err = provider.GetVolume(volume.ID)
	if err != nil {
		t.Fatal("Error retrieving volume. Error: " + err.Error())
	}
	assert.Nil(t, volume)
}

// nolint: dupl
func deleteSnapshot(t *testing.T, provider *ContainerStorageProvider, snapshot *model.Snapshot) {
	err := provider.DeleteSnapshot(snapshot.ID)
	if err != nil {
		t.Fatal("Could not delete snapshot " + snapshot.Name + ".  Error: " + err.Error())
	}
	snapshot, err = provider.GetSnapshot(snapshot.ID)
	if err != nil {
		t.Fatal("Error retrieving snapshot.  Error: " + err.Error())
	}
	assert.Nil(t, snapshot)
}
