// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package powershell wraps Windows Powershell cmdlets
package powershell

import (
	"fmt"

	log "github.com/hpe-storage/common-host-libs/logger"
)

const (
	// MinimumGPTSize represents the minimum GPT size (i.e. 128 MiB)
	MinimumGPTSize = 128 * 1024 * 1024

	// PartitionStyleMBR is used to initialize a disk with the MBR partitioning style
	PartitionStyleMBR = "MBR"

	// PartitionStyleGPT is used to initialize a disk with the GPT partitioning style
	PartitionStyleGPT = "GPT"

	// TimeoutPartitionAndFormatVolume specifies the partition and format cmdlet timeout (in seconds)
	TimeoutPartitionAndFormatVolume = 5 * 60
)

// AddPartitionAccessPath wraps the Add-PartitionAccessPath cmdlet
func AddPartitionAccessPath(accessPath string, diskNumber uint32, partitionNumber uint32) (string, int, error) {
	log.Tracef(">>>>> AddPartitionAccessPath, accessPath=%v, diskNumber=%v, partitionNumber=%v", accessPath, diskNumber, partitionNumber)
	defer log.Trace("<<<<< AddPartitionAccessPath")

	arg := fmt.Sprintf("Add-PartitionAccessPath -AccessPath %v -DiskNumber %v -PartitionNumber %v", accessPath, diskNumber, partitionNumber)
	return execCommandOutput(arg)
}

// ClearDisk wraps the Clear-Disk cmdlet
func ClearDisk(diskNumber int, removeData bool) (string, int, error) {
	log.Tracef(">>>>> ClearDisk, diskNumber=%v, removeData=%v", diskNumber, removeData)
	defer log.Trace("<<<<< ClearDisk")

	arg := fmt.Sprintf("Clear-Disk -Number %v -RemoveData:%v -Confirm:$False", diskNumber, psBoolToText(removeData))
	return execCommandOutput(arg)
}

// InitializeDisk wraps the Initialize-Disk cmdlet.  We only need the ability to create GPT partitions
// which is why this routine doesn't expose the option to select the partition style.
func InitializeDisk(path string, partitionStyle string) (string, int, error) {
	log.Tracef(">>>>> InitializeDisk, path=%v, partitionStyle=%v", path, partitionStyle)
	defer log.Trace("<<<<< InitializeDisk")

	// If no partition style provided, default to GPT
	if partitionStyle == "" {
		partitionStyle = PartitionStyleGPT
	}

	arg := fmt.Sprintf(`Initialize-Disk -Path "%v" -PartitionStyle %v`, path, partitionStyle)
	return execCommandOutput(arg)
}

// PartitionAndFormatVolume wraps the New-Partition and Format-Volume cmdlets to allow the caller to
// create and format a volume with the specified file system.  If no file system is passed in, we
// default to NTFS.
func PartitionAndFormatVolume(diskPath string, fileSystem string) (string, int, error) {
	log.Tracef(">>>>> PartitionAndFormatVolume, diskPath=%v, fileSystem=%v", diskPath, fileSystem)
	defer log.Trace("<<<<< PartitionAndFormatVolume")

	// Default to NTFS if file system not provided
	if len(fileSystem) == 0 {
		fileSystem = "NTFS"
	}

	arg := fmt.Sprintf(`New-Partition -DiskPath "%v" -UseMaximumSize:$True | Format-Volume -FileSystem %v`, diskPath, fileSystem)
	return execCommandOutputWithTimeout(arg, TimeoutPartitionAndFormatVolume)
}

// RemovePartitionAccessPath wraps the Remove-PartitionAccessPath cmdlet
func RemovePartitionAccessPath(accessPath string, diskNumber uint32, partitionNumber uint32) (string, int, error) {
	log.Tracef(">>>>> RemovePartitionAccessPath, accessPath=%v, diskNumber=%v, partitionNumber=%v", accessPath, diskNumber, partitionNumber)
	defer log.Trace("<<<<< RemovePartitionAccessPath")

	arg := fmt.Sprintf("Remove-PartitionAccessPath -AccessPath %v -DiskNumber %v -PartitionNumber %v", accessPath, diskNumber, partitionNumber)
	return execCommandOutput(arg)
}

// SetDiskOffline wraps the Set-Disk cmdlet with the -Path and -IsOffline options
func SetDiskOffline(path string, isOffline bool) (string, int, error) {
	log.Tracef(">>>>> SetDiskIsOffline, path=%v, isOffline=%v", path, isOffline)
	defer log.Trace("<<<<< SetDiskIsOffline")

	return execCommandOutput(fmt.Sprintf(`Set-Disk -Path "%v" -IsOffline %v`, path, psBoolToText(isOffline)))
}

// SetDiskReadOnly wraps the Set-Disk cmdlet with the -Path and -IsReadonly options
func SetDiskReadOnly(path string, isReadonly bool) (string, int, error) {
	log.Tracef(">>>>> SetDiskReadOnly, path=%v, isReadonly=%v", path, isReadonly)
	defer log.Trace("<<<<< SetDiskReadOnly")

	return execCommandOutput(fmt.Sprintf(`Set-Disk -Path "%v" -IsReadonly %v`, path, psBoolToText(isReadonly)))
}

// UpdateDisk wraps the Update-Disk cmdlet with the -Path option
func UpdateDisk(path string) (string, int, error) {
	log.Tracef(">>>>> UpdateDisk, path=%v", path)
	defer log.Trace("<<<<< UpdateDisk")

	return execCommandOutput(fmt.Sprintf(`Update-Disk -Path "%v"`, path))
}
