// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package chapiclient

// chapiclient.go is used *only* for CHAPI wrapped endpoints.  Any additional CHAPI client
// functionality, that extends the endpoint support, should be placed in this module.  For example,
// CHAPI1 has an AttachAndMountDevice() Client method that makes use of multiple CHAPI endpoints.
// This type of extended functionality, if needed, would be placed in this module.
