// (c) Copyright 2021 Hewlett Packard Enterprise Development LP

// In this package, we provide new functions to replace the equivalent ones located in the following
// base package:
//
// github.com/hectane/go-acl
//
// Unfortunately, that package does not handle errors correctly.  Instead, we rolled our own advapi32
// wrapper and rely on "golang.org/x/sys/windows" to submit the requests.

// +build windows

package advapi32

import (
	"syscall"
	"unsafe"

	"github.com/hectane/go-acl/api"
	"golang.org/x/sys/windows"
)

// Lazy load our advapi32.dll APIs
var (
	advapi32                  = windows.NewLazySystemDLL("advapi32.dll")
	procSetEntriesInAclW      = advapi32.NewProc("SetEntriesInAclW")
	procSetNamedSecurityInfoW = advapi32.NewProc("SetNamedSecurityInfoW")
	procSetSecurityInfo       = advapi32.NewProc("SetSecurityInfo")
)

// SetEntriesInAcl -- Creates a new access control list (ACL) by merging new access control or audit
// control information into an existing ACL structure.
// https://docs.microsoft.com/en-us/windows/desktop/api/aclapi/nf-aclapi-setentriesinacla
func SetEntriesInAcl(ea []api.ExplicitAccess, OldAcl windows.Handle, NewAcl *windows.Handle) error {
	ret, _, _ := procSetEntriesInAclW.Call(
		uintptr(len(ea)),
		uintptr(unsafe.Pointer(&ea[0])),
		uintptr(OldAcl),
		uintptr(unsafe.Pointer(NewAcl)),
	)
	if ret != 0 {
		return syscall.Errno(ret)
	}
	return nil
}

// SetNamedSecurityInfo -- Sets specified security information in the security descriptor of a
// specified object. The caller identifies the object by name.
// https://docs.microsoft.com/en-us/windows/desktop/api/aclapi/nf-aclapi-setnamedsecurityinfow
func SetNamedSecurityInfo(objectName string, objectType int32, secInfo uint32, owner, group *windows.SID, dacl, sacl windows.Handle) error {
	ret, _, _ := procSetNamedSecurityInfoW.Call(
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(objectName))),
		uintptr(objectType),
		uintptr(secInfo),
		uintptr(unsafe.Pointer(owner)),
		uintptr(unsafe.Pointer(group)),
		uintptr(dacl),
		uintptr(sacl),
	)
	if ret != 0 {
		return syscall.Errno(ret)
	}
	return nil
}

// SetSecurityInfo -- The SetSecurityInfo function sets specified security information in the
// security descriptor of a specified object. The caller identifies the object by a handle.
// https://docs.microsoft.com/en-us/windows/win32/api/aclapi/nf-aclapi-setsecurityinfo
func SetSecurityInfo(handle syscall.Handle, objectType int32, secInfo uint32, owner, group *windows.SID, dacl, sacl windows.Handle) error {
	ret, _, _ := procSetSecurityInfo.Call(
		uintptr(handle),
		uintptr(objectType),
		uintptr(secInfo),
		uintptr(unsafe.Pointer(owner)),
		uintptr(unsafe.Pointer(group)),
		uintptr(dacl),
		uintptr(sacl),
	)
	if ret != 0 {
		return syscall.Errno(ret)
	}
	return nil
}
