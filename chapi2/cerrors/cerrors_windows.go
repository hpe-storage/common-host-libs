// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package cerrors

import (
	"syscall"

	"github.com/hpe-storage/common-host-libs/windows/iscsidsc"
)

// IscsiErrToCerrors takes an iscsidsc API error and, where applicable, converts it to a
// cerrors object.  This routine is only designed to map common errors.
func IscsiErrToCerrors(err error) error {
	switch err {
	case syscall.Errno(iscsidsc.ISDSC_TARGET_NOT_FOUND):
		err = NewChapiError(NotFound)
	case syscall.Errno(iscsidsc.ISDSC_CONNECTION_FAILED):
		err = NewChapiError(ConnectionFailed)
	}
	return err
}
