// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package mount

import (
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hpe-storage/common-host-libs/chapi2/cerrors"
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/windows/powershell"
	"github.com/hpe-storage/common-host-libs/windows/wmi"
)

const (
	PARTITION_BASIC_DATA_GUID = "{ebd0a0a2-b9e5-4433-87c0-68b6b72699c7}"
)

// getMounts enumerates the mountpoints for the given device / mount point.  The following input
// variables determine which mount points will get enumerated:
//
// #  serialNumber  mountId  Description
// 1.     No          No     Enumerate all Nimble Storage mount point objects
// 2.     No          Yes    Enumerate just the specified mount point
// 3.     Yes         No     Enumerate all mount point objects just for the specified serial number
// 4.     Yes         Yes    Enumerate just the specified mount point
//
// Option #2 and option #4 enumerate and return the same mount point.  Option #4, however, is
// recommended over option #2 because having the serial number reduces the amount of enumeration this
// routine needs to perform.
//
// If allDetails is false, this routine just enumerates and returns the mount point ID.  If true, the
// entire model.Mount object is enumerated and returned.
//
// If onlyMounted is false, this routine *only* returns mounted objects, else both mounted and
// dismounted objects are returned.  Being able to enumerate Mount objects that are not mounted is
// important because it provides details about the potential mount point.  For example, under
// Windows, this includes disk and partition details that are needed in order to mount a volume.
func (mounter *Mounter) getMounts(serialNumber string, mountId string, allDetails bool, onlyMounted bool) ([]*model.Mount, error) {
	log.Tracef(">>>>> getMounts, serialNumber=%v, mountId=%v, allDetails=%v, onlyMounted=%v", serialNumber, mountId, allDetails, onlyMounted)
	defer log.Trace("<<<<< getMounts")

	// Fail request if our Mounter object was not initialized properly
	if mounter.multipathPlugin == nil {
		err := cerrors.NewChapiError(cerrors.Internal, errorMessageMultipathPluginNotSet)
		log.Error(err)
		return nil, err
	}

	// If the caller passed in a mount point ID, with no serial number (Option #2), then log an error
	// and recommend the caller use Option #4 instead.  This will reduce the amount of enumeration
	// required by this routine.  The routine will, however, continue to function.
	if serialNumber == "" && mountId != "" {
		log.Errorf("No serial number provided with mountId=%v.  A serial number is recommended to reduce the amount of enumeration this routine requires.", mountId)
	}

	// Enumerate the Nimble device(s) on this host for the given serial number (or all Nimble
	// devices if serialNumber is empty)
	devices, err := mounter.enumerateDevices(serialNumber, allDetails)
	if err != nil {
		return nil, err
	}

	// Allocate an initial empty array of Mount objects to return to the caller
	var mountPoints []*model.Mount

	// Loop through each enumerated Nimble device
	for _, device := range devices {
		log.Tracef("Checking serial number %v, disk number %v, for mount points", device.SerialNumber, device.Private.WindowsDisk.Number)

		// Enumerate all the partitions on this Nimble device
		partitions, err := wmi.GetMSFTPartitionForDiskNumber(device.Private.WindowsDisk.Number)
		if err != nil {
			log.Errorf("Skipping device's partitions, err=%v", err)
			continue
		}

		// Loop through each enumerated partition
		for _, partition := range partitions {

			// Get the model.Mount for this partition, skip if it's an unsupported partition
			mountPoint, _ := getMountPointFromPartition(device, partition, allDetails, onlyMounted)
			if mountPoint == nil {
				continue
			}

			// If we were passed in a mount point ID as input, and the ID does not match, skip
			// this mount point ID.
			if (mountId != "") && (mountId != mountPoint.ID) {
				log.Tracef("Skipping mount point ID %v, does not match requested ID %v", mountPoint.ID, mountId)
				continue
			}

			// Append the model.Mount to our partition array
			mountPoints = append(mountPoints, mountPoint)

			// Return mountPoints array if we enumerated the one requested mount point
			if mountId != "" {
				logMountPoints(mountPoints, allDetails)
				return mountPoints, nil
			}
		}
	}

	// Log the enumerated mount points before exiting
	logMountPoints(mountPoints, allDetails)
	return mountPoints, nil
}

// getMountPointFromPartition takes the given device/partition, and the optional mount point ID,
// and returns a model.Mount object.  If an unsupported partition is detected, nil is returned.
func getMountPointFromPartition(device *model.Device, partition *wmi.MSFT_Partition, allDetails bool, onlyMounted bool) (*model.Mount, error) {
	log.Tracef(">>>>> getMountPointFromPartition, device=%v, partition=%v, allDetails=%v, onlyMounted=%v", device != nil, partition != nil, allDetails, onlyMounted)
	defer log.Trace("<<<<< getMountPointFromPartition")

	// Exit routine if called incorrectly
	if (device == nil) || (partition == nil) {
		err := cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageInvalidInputParameter)
		log.Error(err)
		return nil, err
	}

	// Ignore any unsupported partition.  Note that if a GPT partition is detected, only partition
	// type PARTITION_BASIC_DATA_GUID can be mounted.
	if partition.IsHidden || partition.IsShadowCopy || ((partition.GptType != "") && (partition.GptType != PARTITION_BASIC_DATA_GUID)) {
		log.Trace("Ignoring unsupported partition")
		logPartitionDetails(partition, 4)
		return nil, cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageUnsupportedPartition)
	}

	// Find the mount point paths (if assigned) for this volume
	mountPointPaths := getMountPointPaths(partition.AccessPaths)

	// If requested to only enumerate mounted volumes, and this volume isn't mounted, fail request
	if onlyMounted && (len(mountPointPaths) == 0) {
		return nil, cerrors.NewChapiError(cerrors.NotFound, errorMessageMountPointNotFound)
	}

	// CHAPI only supports a single mount point path for a volume.  If there happen to be multiple
	// mount points, we'll use the first one enumerated.
	var chapiMountPointPath string
	if pathCount := len(mountPointPaths); pathCount > 0 {
		chapiMountPointPath = mountPointPaths[0]
		if pathCount > 1 {
			log.Tracef(`Partition has multiple (%v) mount point paths, using first path "%v"`, pathCount, chapiMountPointPath)
		}
	}

	// Create the mount point ID from the device/partition details
	id := getMountPointID(device.SerialNumber, partition.DiskNumber, partition.PartitionNumber, partition.Offset)
	log.Tracef("Enumerated mount point ID %v for SerialNumber %v", id, device.SerialNumber)
	logPartitionDetails(partition, 4)

	// Create a model.Mount object with the mount point ID and Windows specific private data
	mountPoint := &model.Mount{
		ID: id,
		Private: &model.MountPrivate{
			WindowsDisk:      device.Private.WindowsDisk,
			WindowsPartition: partition,
		},
	}

	// If all details were requested, populate the rest of the Mount object
	if allDetails {
		mountPoint.MountPoint = chapiMountPointPath
		mountPoint.SerialNumber = device.SerialNumber
	}

	// Return the enumerated mount point
	return mountPoint, nil
}

// getMountPointPath takes the given MSFT_Partition AccessPaths array (i.e. An array of strings
// containing the various mount points for the partition) and returns back an array of mount
// point paths *if* the partition is currently mounted.
func getMountPointPaths(accessPaths []string) []string {
	log.Tracef(">>>>> getMountPointPath, accessPaths=%v", strings.Join(accessPaths, ","))
	defer log.Trace("<<<<< getMountPointPath")

	// Enumerate all the mount points
	var mountPointPaths []string
	for _, accessPath := range accessPaths {
		if accessPath != "" && !strings.HasPrefix(accessPath, `\\?\Volume`) {
			// Convert mount point path to absolute path
			if absPath, err := filepath.Abs(accessPath); (err == nil) && (absPath != accessPath) {
				log.Tracef(`Adjusting enumerated mount point path "%v" with absolute path "%v"`, accessPath, absPath)
				accessPath = absPath
			}
			mountPointPaths = append(mountPointPaths, accessPath)
		}
	}

	// Log enumerated mount point paths before returning
	log.Tracef("mountPointPaths=%v", strings.Join(mountPointPaths, ","))
	return mountPointPaths
}

// getMountPointID takes the device serial number, disk number, partition number, and partition
// offset to create a unique mount ID.
func getMountPointID(serialNumber string, diskNumber uint32, partitionNumber uint32, startingOffset uint64) string {

	// The serial number, partition number, and partition offset are used to uniquely identify
	// this mount point.  We're going to hash these three values into a 64-bit value.
	id := fmt.Sprintf("%v.%v.%v", serialNumber, partitionNumber, startingOffset)
	h := fnv.New64a()
	h.Write([]byte(id))

	// Even though there is almost zero chance of a hash collision, we're going to add an extra
	// layer of security by appending the disk and partition numbers to the 64-bit hash.  This
	// will guarantee that the mount point ID is unique across all devices / partitions.
	return fmt.Sprintf("%x-%x-%x", h.Sum64(), diskNumber, partitionNumber)
}

// logPartitionDetails logs the indented partition details
func logPartitionDetails(partition *wmi.MSFT_Partition, indent int) {
	spacer := strings.Repeat(" ", indent)
	log.Tracef("%vNumbers     : DiskNumber=%v, PartitionNumber=%v", spacer, partition.DiskNumber, partition.PartitionNumber)
	log.Tracef("%vAccessPaths : %v", spacer, strings.Join(partition.AccessPaths, ","))
	log.Tracef("%vStatus      : OperationalStatus=%v, TransitionState=%v", spacer, partition.OperationalStatus, partition.TransitionState)
	log.Tracef("%vOffset/Size : Offset=%v, Size=%v", spacer, partition.Offset, partition.Size)
	log.Tracef("%vType        : MbrType=%v, GptType=%v", spacer, partition.MbrType, partition.GptType)
	log.Tracef("%vFlags       : IsReadOnly=%v, IsOffline=%v, IsSystem=%v, IsActive=%v, IsHidden=%v, IsShadowCopy=%v",
		spacer, partition.IsReadOnly, partition.IsOffline, partition.IsSystem, partition.IsActive, partition.IsHidden, partition.IsShadowCopy)
}

// createMount is called to mount the given device to the given mount point
func (mounter *Mounter) createMount(mount *model.Mount, mountPoint string, fsOptions *model.FileSystemOptions) error {
	log.Tracef(`>>>>> createMount, mountPoint="%v", fsOptions=%v`, mountPoint, fsOptions)
	defer log.Trace("<<<<< createMount")

	// TODO - How is fsOptions going to be used under Windows?

	// Validate the Mount object
	if err := validateMount(mount); err != nil {
		return err
	}

	// Now that we validated the mount object, log details about the create mount request
	log.Tracef("SerialNumber=%v, PathName=%v, IsOffline=%v, IsReadOnly=%v",
		mount.SerialNumber, mount.Private.WindowsDisk.Path, mount.Private.WindowsDisk.IsOffline, mount.Private.WindowsDisk.IsReadOnly)

	// If the disk is offline, or read only, we first need to online the disk and/or make it writable
	if mount.Private.WindowsDisk.IsOffline || mount.Private.WindowsDisk.IsReadOnly {
		if err := mounter.multipathPlugin.MakeDiskOnlineAndWritable(mount.Private.WindowsDisk.Path, (mount.Private.WindowsDisk.IsOffline == true), (mount.Private.WindowsDisk.IsReadOnly == true)); err != nil {
			return err
		}

		// Now that the disk is online and writable, re-enumerate the device's mount point.  We need
		// to do this because the mount point data wasn't enumerable if the disk was offline.
		newMount, alreadyMounted, err := mounter.getMountForCreate(mount.SerialNumber, mountPoint)
		if err != nil {
			return err
		}

		// If the volume is already mounted, at the requested mount point, return success
		if alreadyMounted {
			return nil
		}

		// Update the mount object with the refreshed Mount object
		mount = newMount
	}

	// Determine details about the drive letter, or directory, we're going to mount to
	var isDriveLetterMount, isDirectoryExists, isDirectoryEmpty bool
	isDriveLetterMount = isWindowsDriveLetterPath(mountPoint)
	isDirectoryEmpty = true
	if _, err := os.Stat(mountPoint); !os.IsNotExist(err) {
		isDirectoryExists = true
		isDirectoryEmpty, _ = isEmptyDirectory(mountPoint)
	}
	log.Tracef("Mount point details, isDriveLetterMount=%v, isDirectoryExists=%v, isDirectoryEmpty=%v", isDriveLetterMount, isDirectoryExists, isDirectoryEmpty)

	// If it's a drive letter mount, and the drive letter already exists, fail the request
	if isDriveLetterMount && isDirectoryExists {
		err := cerrors.NewChapiErrorf(cerrors.AlreadyExists, errorMessageMountPointInUse, mountPoint)
		log.Error(err)
		return err
	}

	//  If it's a directory mount, and that directory exists, and that directory isn't empty, fail
	// the request.  You can only set a mountpoint to an empty directory.
	if !isDriveLetterMount && isDirectoryExists && !isDirectoryEmpty {
		err := cerrors.NewChapiErrorf(cerrors.AlreadyExists, errorMessageMountPointNotEmpty, mountPoint)
		log.Error(err)
		return err
	}

	// If we're mounting to a directory, adjust mount point path with absolute path if necessary
	if !isDriveLetterMount && (mountPoint != "") {
		if absPath, err := filepath.Abs(mountPoint); (err == nil) && (absPath != mountPoint) {
			log.Tracef(`Adjusting requested mount point path "%v" with absolute path "%v"`, mountPoint, absPath)
			mountPoint = absPath
		}
	}

	// If mounting to a directory, and if it doesn't exist, create it.  The Add-PartitionAccessPath
	// PowerShell cmdlet requires the directory to be present in order to add the mount point.
	createdMountDirectory := false
	if !isDriveLetterMount && !isDirectoryExists {
		if err := os.MkdirAll(mountPoint, os.ModePerm); err != nil {
			log.Error(err)
			return err
		}
		createdMountDirectory = true
		log.Tracef(`Created mount point directory "%v"`, mountPoint)
	}

	// Mount the device/partition to the specified mount point
	if _, _, err := powershell.AddPartitionAccessPath(mountPoint, mount.Private.WindowsPartition.DiskNumber, mount.Private.WindowsPartition.PartitionNumber); err != nil {
		if createdMountDirectory {
			// If we created an empty directory, to mount the Nimble volume, perform error cleanup
			// by removing the folder before returning
			if errRemove := os.Remove(mountPoint); errRemove != nil {
				log.Errorf(`Unable to remove created directory, directory="%v", err=%v`, mountPoint, errRemove)
			}
		}
		return err
	}

	// Success!
	return nil
}

// deleteMount is called to unmount the given mount point ID
func (mounter *Mounter) deleteMount(mount *model.Mount) error {
	log.Trace(">>>>> deleteMount")
	defer log.Trace("<<<<< deleteMount")

	// Validate the Mount object
	if err := validateMount(mount); err != nil {
		return err
	}

	// If the volume has more than one mount point, one or more mount points were not set by CHAPI
	// since CHAPI only supports a single mount point per device/partition.  Windows supports
	// multiple mount points per partition.  Since we cannot be certain which of the mount points
	// CHAPI might have created (if any), we fail the request.  We don't need to check for 0 mount
	// point paths since the mount.DeleteMount() routine has already taken care of that.
	mountPointPaths := getMountPointPaths(mount.Private.WindowsPartition.AccessPaths)
	if len(mountPointPaths) > 1 {
		err := cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageMultipleMountPointsDetected)
		log.Errorf("Multiple paths detected, paths=%v, err=%v", strings.Join(mountPointPaths, ","), err)
		return err
	}

	// Now that we validated the mount object, log details about the delete mount request
	log.Tracef("SerialNumber=%v, PathName=%v, IsOffline=%v, IsReadOnly=%v",
		mount.SerialNumber, mount.Private.WindowsDisk.Path, mount.Private.WindowsDisk.IsOffline, mount.Private.WindowsDisk.IsReadOnly)

	// Unmount the device/partition from the specified mount point
	_, _, err := powershell.RemovePartitionAccessPath(mount.MountPoint, mount.Private.WindowsPartition.DiskNumber, mount.Private.WindowsPartition.PartitionNumber)

	// If the mount point was removed, and we were mounted to an empty directory, we clean up after
	// ourselves by removing the empty directory.
	if (err == nil) && !isWindowsDriveLetterPath(mount.MountPoint) {
		log.Tracef(`Removing "%v" directory`, mount.MountPoint)
		if removeErr := os.Remove(mount.MountPoint); removeErr != nil {
			// If we were able to remove the mount point, but unable to remove the empty directory,
			// we'll simply log it as an error but not return the error to the caller.  From the
			// caller's perspective, we were able to delete the mount point.
			log.Errorf("Failed to remove mount point directory, err=%v", err)
		}
	}

	// Return nil for success, else error
	return err
}

// validateMount validates that the Mount object was initialized properly.  The Mount object has
// some private Windows properties that were populated during the getMounts() routine.  The Windows
// properties should *always* be available.  Adding a routine to validate that the properties were
// provided.
func validateMount(mount *model.Mount) error {
	if (mount == nil) || (mount.Private == nil) || (mount.Private.WindowsDisk == nil) || (mount.Private.WindowsPartition == nil) {
		err := cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageInvalidInputParameter)
		log.Error(err)
		return err
	}
	return nil
}

// isWindowsDriveLetterPath takes the given path and returns true if it's a path to a drive letter
// (e.g. c:\ or c:) else false is returned.
func isWindowsDriveLetterPath(accessPath string) bool {
	if lenAccessPath := len(accessPath); (lenAccessPath == 2) || (lenAccessPath == 3) {
		accessPath := strings.ToUpper(accessPath)
		if (accessPath[0] >= 'A') && (accessPath[0] <= 'Z') && (accessPath[1] == ':') {
			return (lenAccessPath == 2) || (accessPath[2] == '\\')
		}
	}
	return false
}

// isEmptyDirectory takes the given directory path and returns true if the directory is empty else
// false is returned.  If the path is invalid / inaccessible, an error is returned.
func isEmptyDirectory(accessPath string) (bool, error) {

	// Start by getting a handle to the directory path
	f, err := os.Open(accessPath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// Query the directory to see if there is at least one child file/subdirectory present
	_, err = f.Readdirnames(1)

	// If Readdirnames(1) fails with io.EOF, we know that the directory is empty
	if err == io.EOF {
		return true, nil
	}

	// Directory isn't empty
	return false, err
}

// isSamePathName returns true if the two provided directory paths are equal else false.  Under
// Linux we perform a case sensitive comparison.  Under Windows, it's case insensitive.  This
// routine assumes that the caller (likely platform independent caller) has already retrieved the
// absolute representation of each path (e.g. filepath.Abs).
func isSamePathName(path1, path2 string) bool {
	return strings.EqualFold(path1, path2)
}
