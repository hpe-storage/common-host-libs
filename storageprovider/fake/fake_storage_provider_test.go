// Copyright 2019 Hewlett Packard Enterprise Development LP
package fake

import (
	"testing"

	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/model"
	"github.com/stretchr/testify/assert"
)

const (
	volumeName        = "testCspVol"
	volumeSize        = 1024 * 1024 * 1024
	snapshotName      = "testCspSnapshot"
	cloneName         = "testCspVolClone"
	cloneSize         = 2 * 1024 * 1024 * 1024
	volumeGroupName   = "testCspVolumeGroup"
	snapshotGroupName = "testCspSnapshotGroup"
)

// nolint: gocyclo
func TestPluginSuite(t *testing.T) {
	log.InitLogging("fake-storage-provider-test.log", nil, false)

	provider := fakeCsp()

	// create a parent volume
	config := make(map[string]interface{})
	config["test"] = "test"
	volume, err := provider.CreateVolume(volumeName, volumeName, volumeSize, config)
	if err != nil {
		t.Fatal("Failed to create volume " + volumeName)
	}

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

	updatedVolume, err := provider.ExpandVolume(volume.ID, volume.Size*2)
	if err != nil {
		t.Fatal("Failed to expand volume")
	}
	assert.True(t, updatedVolume.Size == volume.Size*2)

	editConfig := make(map[string]interface{})
	editConfig["editme"] = "edited"
	editedVolume, err := provider.EditVolume(volume.ID, editConfig)
	if err != nil {
		t.Fatal("Failed to edit volume")
	}
	assert.True(t, editedVolume.Config["editme"] == "edited")

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

	// Delete the snapshot
	deleteSnapshot(t, provider, snapshot)

	// Create a new snapshot manually
	snapshotConfig := make(map[string]interface{})
	snapshotConfig["test"] = "test"

	snapshot, err = provider.CreateSnapshot(snapshotName, snapshotName, volume.ID, snapshotConfig)
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

	// Create Volume Group
	config["test"] = "test"

	volumeGroup, err := provider.CreateVolumeGroup(volumeGroupName, volumeGroupName, config)
	if err != nil {
		t.Fatal("Failed to create volume group" + volumeGroupName)
	}

	// Create Snapshot Group
	config["test"] = "test"

	snapshotGroup, err := provider.CreateSnapshotGroup(snapshotGroupName, volumeGroup.ID, config)
	if err != nil {
		t.Fatal("Failed to create snapshot group" + snapshotGroupName)
	}

	// Delete the Snapshot Group
	deleteSnapshotGroup(t, provider, snapshotGroup)

	// Delete the Volume Group
	deleteVolumeGroup(t, provider, volumeGroup)

}

func fakeCsp() *StorageProvider {
	provider := NewFakeStorageProvider()
	return provider
}

func createClone(t *testing.T, provider *StorageProvider, sourceVolumeID, snapshotID string, size int64) *model.Volume {
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
func deleteVolume(t *testing.T, provider *StorageProvider, volume *model.Volume) {
	err := provider.DeleteVolume(volume.ID, false)
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
func deleteSnapshotGroup(t *testing.T, provider *StorageProvider, snapshotGroup *model.SnapshotGroup) {
	err := provider.DeleteSnapshotGroup(snapshotGroup.ID)
	if err != nil {
		t.Fatal("Could not delete snapshot group" + snapshotGroup.Name + ".  Error: " + err.Error())
	}
}

// nolint: dupl
func deleteVolumeGroup(t *testing.T, provider *StorageProvider, volumeGroup *model.VolumeGroup) {
	err := provider.DeleteVolumeGroup(volumeGroup.ID)
	if err != nil {
		t.Fatal("Could not delete volume group" + volumeGroup.Name + ".  Error: " + err.Error())
	}
}

// nolint: dupl
func deleteSnapshot(t *testing.T, provider *StorageProvider, snapshot *model.Snapshot) {
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
