// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package mount

import (
	"github.com/hpe-storage/common-host-libs/chapi2/model"
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
	// TODO
	return nil, nil
}

// createMount is called to mount the given device to the given mount point
func (mounter *Mounter) createMount(mount *model.Mount, mountPoint string, fsOptions *model.FileSystemOptions) error {
	// TODO
	return nil
}

// deleteMount is called to unmount the given mount point ID
func (mounter *Mounter) deleteMount(mount *model.Mount) error {
	// TODO
	return nil
}

// isSamePathName returns true if the two provided directory paths are equal else false.  Under
// Linux we perform a case sensitive comparison.  Under Windows, it's case insensitive.  This
// routine assumes that the caller (likely platform independent caller) has already retrieved the
// absolute representation of each path (e.g. filepath.Abs).
func isSamePathName(path1, path2 string) bool {
	return path1 == path2
}
