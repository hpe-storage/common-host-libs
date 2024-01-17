// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package ioctl provides Windows IOCTL support
package ioctl

import "fmt"

const (
	INVALID_HANDLE_VALUE = ^uintptr(0)
)

const (
	METHOD_NEITHER      = 3
	METHOD_BUFFERED     = 0
	FILE_ANY_ACCESS     = 0
	FILE_SPECIAL_ACCESS = 0
	FILE_READ_ACCESS    = 1
	FILE_WRITE_ACCESS   = 2
)

const (
	IOCTL_SCSI_BASE   = 0x00000004
	IOCTL_DISK_BASE   = 0x00000007
	IOCTL_VOLUME_BASE = 0x00000056
)

const (
	IOCTL_DISK_GET_DRIVE_GEOMETRY_EX     = (IOCTL_DISK_BASE << 16) | (FILE_ANY_ACCESS << 14) | (0x0028 << 2) | METHOD_BUFFERED
	IOCTL_SCSI_GET_ADDRESS               = (IOCTL_SCSI_BASE << 16) | (FILE_ANY_ACCESS << 14) | (0x0406 << 2) | METHOD_BUFFERED
	IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS = (IOCTL_VOLUME_BASE << 16) | (FILE_ANY_ACCESS << 14) | (0x0000 << 2) | METHOD_BUFFERED
)

// Helper function to convert a disk number to a disk path
func diskPathFromNumber(diskNumber uint32) string {
	return fmt.Sprintf(`\\.\PHYSICALDRIVE%v`, diskNumber)
}
