// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package mount

import (
	"path/filepath"

	"github.com/hpe-storage/common-host-libs/chapi2/cerrors"
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	"github.com/hpe-storage/common-host-libs/chapi2/multipath"
	"github.com/hpe-storage/common-host-libs/chapi2/virtualdevice"
	log "github.com/hpe-storage/common-host-libs/logger"
)

const (
	// Shared error messages
	errorMessageInvalidInputParameter       = "invalid input parameter"
	errorMessageMissingMountPoint           = "missing mount point"
	errorMessageMissingMountPointID         = "missing mount point ID"
	errorMessageMissingSerialNumber         = "missing serial number"
	errorMessageMountPointInUse             = `mount point "%v" already in use`
	errorMessageMountPointNotEmpty          = `mount point "%v" is not empty`
	errorMessageMountPointNotFound          = "mount point not found"
	errorMessageMultipathPluginNotSet       = "multipathPlugin not set"
	errorMessageMultipleMountPointsDetected = "multiple mount points detected"
	errorMessageUnsupportedPartition        = "unsupported partition"
	errorMessageVolumeAlreadyMounted        = `volume already mounted at "%v"`
)

type Mounter struct {
	multipathPlugin  *multipath.MultipathPlugin
	virtualDevPlugin *virtualdevice.VirtualDevPlugin
}

func NewMounter() *Mounter {
	return &Mounter{
		multipathPlugin:  multipath.NewMultipathPlugin(),
		virtualDevPlugin: virtualdevice.NewVirtualDevPlugin(),
	}
}

// GetMounts reports all mounts on this host for the specified Nimble volume
func (mounter *Mounter) GetMounts(serialNumber string) ([]*model.Mount, error) {
	return mounter.getMounts(serialNumber, "", false, true)
}

// GetAllMountDetails enumerates the specified mount point ID
func (mounter *Mounter) GetAllMountDetails(serialNumber string, mountId string) ([]*model.Mount, error) {
	return mounter.getMounts(serialNumber, mountId, true, true)
}

// CreateMount is called to mount the given device to the given mount point
func (mounter *Mounter) CreateMount(serialNumber string, mountPoint string, fsOptions *model.FileSystemOptions) (*model.Mount, error) {
	log.Tracef(">>>>> CreateMount, serialNumber=%v, mountPoint=%v, fsOptions=%v", serialNumber, mountPoint, fsOptions)
	defer log.Trace("<<<<< CreateMount")

	// Validate and enumerate the mount object for the given serial number and mount point
	mount, alreadyMounted, err := mounter.getMountForCreate(serialNumber, mountPoint)

	// Fail request if unable to validate and enumerate the mount object
	if err != nil {
		return nil, err
	}

	// If the volume is already mounted, at the requested mount point, return mount object with success
	if alreadyMounted {
		return mount, nil
	}

	// Mount the volume at the specified mount point
	err = mounter.createMount(mount, mountPoint, fsOptions)
	if err != nil {
		return nil, err
	}

	// Now that the device has been mounted, adjust the mount point and return the mount object
	mount.MountPoint = mountPoint
	return mount, nil
}

// DeleteMount is called to unmount the given mount point ID
func (mounter *Mounter) DeleteMount(serialNumber string, mountId string) error {
	log.Tracef(">>>>> DeleteMount, serialNumber=%v, mountId=%v", serialNumber, mountId)
	defer log.Trace("<<<<< DeleteMount")

	// Validate and enumerate the mount object for the given serial number and mount point ID
	mount, err := mounter.getMountForDelete(serialNumber, mountId)

	// Fail request if unable to validate and enumerate the mount object
	if err != nil {
		return err
	}

	// Call the platform specific deleteMount routine to dismount the volume
	return mounter.deleteMount(mount)
}

// enumerateDevices enumerates the given serialNumber (or all devices if serialNumber is empty).
// The allDetails boolean lets us know if we just need to enumerate basic details (false) or if
// all details are required (true).  We can optimize our enumeration (e.g. reduce the amount of
// enumeration required) if we only need basic details.
func (mounter *Mounter) enumerateDevices(serialNumber string, allDetails bool) ([]*model.Device, error) {
	if !allDetails {
		return mounter.multipathPlugin.GetDevices(serialNumber)
	}
	return mounter.multipathPlugin.GetAllDeviceDetails(serialNumber)
}

// getMountForCreate takes the Nimble serial number, and mount point path, validates the input
// data, and enumerates the Mount object.  The following properties are returned:
//      mount           - Enumerated model.Mount object for the provided serialNumber/mountPoint
//      alreadyMounted  - If volume is already mounted, at the requested mount point, true is
//                        returned else false ("mount" object returned if alreadyMounted==true)
//      err             - If volume cannot be mounted, an error object is returned ("mount" and
//                        "alreadyMounted" are invalid)
func (mounter *Mounter) getMountForCreate(serialNumber string, mountPoint string) (mount *model.Mount, alreadyMounted bool, err error) {
	log.Tracef(">>>>> getMountForCreate, serialNumber=%v, mountPoint=%v", serialNumber, mountPoint)
	defer log.Trace("<<<<< getMountForCreate")

	// If the serialNumber is not provided, fail the request
	if serialNumber == "" {
		err = cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageMissingSerialNumber)
		log.Error(err)
		return nil, false, err
	}

	// If the mountPoint is not provided, fail the request
	if mountPoint == "" {
		err = cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageMissingMountPoint)
		log.Error(err)
		return nil, false, err
	}

	// Enumerate all the mount points, with all details, for the given serial number
	var mounts []*model.Mount
	mounts, err = mounter.getMounts(serialNumber, "", true, false)
	if err != nil {
		return nil, false, err
	}

	// Fail request if no mount points detected
	if len(mounts) == 0 {
		err = cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageMountPointNotFound)
		log.Error(err)
		return nil, false, err
	}

	// We only support a device with a single mount point (e.g. one created by CHAPI).
	// Fail request if multiple mount points detected.
	if len(mounts) > 1 {
		err = cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageMultipleMountPointsDetected)
		log.Error(err)
		return nil, false, err
	}

	// Get the mount point object we're going to try and mount
	mount = mounts[0]

	// Handle case where Nimble volume is already mounted
	if mount.MountPoint != "" {
		// Convert the current mount point to its absolute path
		var currentMountPoint string
		currentMountPoint, err = filepath.Abs(mount.MountPoint)
		if err != nil {
			log.Errorf("Invalid current mount point, MountPoint=%v, err=%v", mount.MountPoint, err)
			return nil, false, err
		}

		// Convert the target mount point to its absolute path
		var requestedMountPoint string
		requestedMountPoint, err = filepath.Abs(mountPoint)
		if err != nil {
			log.Errorf("Invalid requested mount point, MountPoint=%v, err=%v", mountPoint, err)
			return nil, false, err
		}

		// If the current mount point matches the target mount point, there is nothing to do as we
		// are already mounted at the requested location.
		if isSamePathName(currentMountPoint, requestedMountPoint) {
			log.Tracef(`Mount point ID=%v, SerialNumber=%v, currentMountPoint=%v, already mounted`, mount.ID, mount.SerialNumber, currentMountPoint)
			return mount, true, nil
		}

		// If here, the device is already mounted but to a different location.  Log the error and
		// fail the request.
		err = cerrors.NewChapiErrorf(cerrors.AlreadyExists, errorMessageVolumeAlreadyMounted, currentMountPoint)
		log.Error(err)
		return nil, false, err
	}

	// Routine passed all checks; safe to attempt to mount volume to the given mount point
	return mount, false, nil
}

// getMountForDelete takes the Nimble serial number, and mount point ID, validates the input
// data, and enumerates the Mount object.  The following properties are returned:
//      mount             - Enumerated model.Mount object for the provided serialNumber/mountPointId
//      err               - If volume cannot be dismounted, an error object is returned
func (mounter *Mounter) getMountForDelete(serialNumber string, mountId string) (mount *model.Mount, err error) {
	log.Tracef(">>>>> getMountForDelete, serialNumber=%v, mountId=%v", serialNumber, mountId)
	defer log.Trace("<<<<< getMountForDelete")

	// If the serialNumber is not provided, fail the request
	if serialNumber == "" {
		err = cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageMissingSerialNumber)
		log.Error(err)
		return nil, err
	}

	// If the mountId is not provided, fail the request
	if mountId == "" {
		err = cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageMissingMountPointID)
		log.Error(err)
		return nil, err
	}

	// Find the specified mount point ID with all details
	var mounts []*model.Mount
	mounts, err = mounter.getMounts(serialNumber, mountId, true, true)
	if err != nil {
		return nil, err
	}

	// There should only be a single mount point object
	if len(mounts) != 1 {
		err = cerrors.NewChapiError(cerrors.InvalidArgument, errorMessageMountPointNotFound)
		log.Error(err)
		return nil, err
	}

	// Routine passed all checks; safe to attempt to dismount volume at the given mount point
	return mounts[0], nil
}

// logMountPoints logs the mount points array to our log file
func logMountPoints(mountPoints []*model.Mount, allDetails bool) {
	log.Tracef(">>>>> logMountPoints, allDetails=%v", allDetails)
	defer log.Trace("<<<<< logMountPoints")

	for _, mountPoint := range mountPoints {
		logMessage := "Enumerated mount point, ID=" + mountPoint.ID
		if allDetails {
			logMessage += ", SerialNumber=" + mountPoint.SerialNumber
			logMessage += ", MountPoint=" + mountPoint.MountPoint
		}
		log.Trace(logMessage)
	}
}

// TODO, Remove or implement member functions below that are not utilized

// func (mounter *Mounter) Mount(mount *model.Mount) error {
// 	return nil
// }

// func (mounter *Mounter) Unmount(mountPoint string) error {
// 	return nil
// }

// func (mounter *Mounter) ReMount(mount *model.Mount) error {
// 	return nil
// }

// func (mounter *Mounter) BindMount(mount *model.Mount) error {
// 	return nil
// }

// func (mounter *Mounter) GetFsType(mountPoint string) (*string, error) {
// 	return nil, nil
// }

// func (mounter *Mounter) GetFsOptions(mountPoint string) (*model.FileSystemOptions, error) {
// 	return nil, nil
// }
