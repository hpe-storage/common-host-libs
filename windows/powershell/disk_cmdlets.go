// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package powershell wraps Windows Powershell cmdlets
package powershell

import (
	"fmt"

	log "github.com/hpe-storage/common-host-libs/logger"
)

// AddPartitionAccessPath wraps the Add-PartitionAccessPath cmdlet
func AddPartitionAccessPath(accessPath string, diskNumber uint32, partitionNumber uint32) (string, int, error) {
	log.Infof(">>>>> AddPartitionAccessPath, accessPath=%v, diskNumber=%v, partitionNumber=%v", accessPath, diskNumber, partitionNumber)
	defer log.Info("<<<<< AddPartitionAccessPath")

	arg := fmt.Sprintf("Add-PartitionAccessPath -AccessPath %v -DiskNumber %v -PartitionNumber %v", accessPath, diskNumber, partitionNumber)
	return execCommandOutput(arg)
}

// ClearDisk wraps the Clear-Disk cmdlet
func ClearDisk(diskNumber int, removeData bool) (string, int, error) {
	log.Infof(">>>>> ClearDisk, diskNumber=%v, removeData=%v", diskNumber, removeData)
	defer log.Info("<<<<< ClearDisk")

	arg := fmt.Sprintf("Clear-Disk -Number %v -RemoveData:%v -Confirm:$False", diskNumber, psBoolToText(removeData))
	return execCommandOutput(arg)
}

// InitializeDisk wraps the Initialize-Disk cmdlet.  We only need the ability to create GPT partitions
// which is why this routine doesn't expose the option to select the partition style.
func InitializeDisk(path string) (string, int, error) {
	log.Infof(">>>>> InitializeDisk, path=%v", path)
	defer log.Info("<<<<< InitializeDisk")

	arg := fmt.Sprintf(`Initialize-Disk -Path "%v" -PartitionStyle GPT`, path)
	return execCommandOutput(arg)
}

// PartitionAndFormatVolume wraps the New-Partition and Format-Volume cmdlets to allow the caller to
// create and format a volume with the specified file system.  If no file system is passed in, we
// default to NTFS.
func PartitionAndFormatVolume(diskPath string, fileSystem string) (string, int, error) {
	log.Infof(">>>>> PartitionAndFormatVolume, diskPath=%v, fileSystem=%v", diskPath, fileSystem)
	defer log.Info("<<<<< PartitionAndFormatVolume")

	// Default to NTFS if file system not provided
	if len(fileSystem) == 0 {
		fileSystem = "NTFS"
	}

	arg := fmt.Sprintf(`New-Partition -DiskPath "%v" -UseMaximumSize:$True | Format-Volume -FileSystem %v`, diskPath, fileSystem)
	return execCommandOutput(arg)
}

// RemovePartitionAccessPath wraps the Remove-PartitionAccessPath cmdlet
func RemovePartitionAccessPath(accessPath string, diskNumber uint32, partitionNumber uint32) (string, int, error) {
	log.Infof(">>>>> RemovePartitionAccessPath, accessPath=%v, diskNumber=%v, partitionNumber=%v", accessPath, diskNumber, partitionNumber)
	defer log.Info("<<<<< RemovePartitionAccessPath")

	arg := fmt.Sprintf("Remove-PartitionAccessPath -AccessPath %v -DiskNumber %v -PartitionNumber %v", accessPath, diskNumber, partitionNumber)
	return execCommandOutput(arg)
}

// SetDiskOffline wraps the Set-Disk cmdlet with the -Path and -IsOffline options
func SetDiskOffline(path string, isOffline bool) (string, int, error) {
	log.Infof(">>>>> SetDiskIsOffline, path=%v, isOffline=%v", path, isOffline)
	defer log.Info("<<<<< SetDiskIsOffline")

	return execCommandOutput(fmt.Sprintf(`Set-Disk -Path "%v" -IsOffline %v`, path, psBoolToText(isOffline)))
}

// SetDiskReadOnly wraps the Set-Disk cmdlet with the -Path and -IsReadonly options
func SetDiskReadOnly(path string, isReadonly bool) (string, int, error) {
	log.Infof(">>>>> SetDiskReadOnly, path=%v, isReadonly=%v", path, isReadonly)
	defer log.Info("<<<<< SetDiskReadOnly")

	return execCommandOutput(fmt.Sprintf(`Set-Disk -Path "%v" -IsReadonly %v`, path, psBoolToText(isReadonly)))
}

// UpdateDisk wraps the Update-Disk cmdlet with the -Path option
func UpdateDisk(path string) (string, int, error) {
	log.Infof(">>>>> UpdateDisk, path=%v", path)
	defer log.Info("<<<<< UpdateDisk")

	return execCommandOutput(fmt.Sprintf(`Update-Disk -Path "%v"`, path))
}
