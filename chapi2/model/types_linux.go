// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package model

const (
	// FsCreateOpt filesystem create type
	FsCreateOpt = "filesystem"
	// FsModeOpt filesystem mode option
	FsModeOpt = "fsMode"
	// FsOwnerOpt filesystem owner option
	FsOwnerOpt = "fsOwner"
)

// Path struct
type Path struct {
	Name  string `json:"-"`
	Major string `json:"-"`
	Minor string `json:"-"`
	Hcils string `json:"-"`
	State string `json:"-"`
}

// NetworkPrivate provides model.Network platform specific private data
type NetworkPrivate struct {
}

// TargetPortalPrivate provides model.TargetPortal platform specific private data
type TargetPortalPrivate struct {
}

// DevicePrivate provides model.Device platform specific private data
type DevicePrivate struct {
	Paths []Path `json:"-"` // Physical path details (used internally by CHAPI server)
}

// MountPrivate provides model.Mount platform specific private data
type MountPrivate struct {
}
