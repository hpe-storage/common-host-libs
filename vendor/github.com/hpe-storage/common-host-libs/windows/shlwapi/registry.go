// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// This package wraps the Windows shell APIs that deal with the registry

// +build windows

package shlwapi

import (
	"encoding/binary"
	"reflect"
	"syscall"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
	"golang.org/x/sys/windows"
)

// Lazy load our advapi32.dll APIs
var (
	advapi32        = windows.NewLazySystemDLL("shlwapi.dll")
	procSHGetValueW = advapi32.NewProc("SHGetValueW")
	procSHSetValueW = advapi32.NewProc("SHSetValueW")
)

// SHSetValue wraps the SHSetValueW API.  The input parameters are used to set the value of a registry
// key.  The input src object can be either a string (REG_SZ), uint32 (REG_DWORD), or uint64 (REG_QWORD).
func SHSetValue(hkey uintptr, keyName string, valueName string, src interface{}) error {

	// Get a copy of the source value, its type, and kind
	srcValue := reflect.ValueOf(src)
	srcType := srcValue.Type()
	srcKind := srcType.Kind()

	// If we were passed in a pointer to an object, get contained type
	if srcKind == reflect.Ptr {
		srcValue = srcValue.Elem()
		srcType = srcValue.Type()
		srcKind = srcType.Kind()
	}

	// Reflect our enumerated value back to its type
	src = srcValue.Interface()

	// Data we used to call the Win32 SHSetValueW API
	var cbData uint32
	var typeData uint32
	var apiData []byte

	switch srcKind {
	case reflect.String: // We were passed in a string (store as REG_SZ)
		srcString := src.(string)
		cbData = uint32(len(srcString) * 2)
		typeData = syscall.REG_SZ
		apiData = make([]byte, cbData)
		for iSrc, iDst := 0, 0; iSrc < len(srcString); iSrc, iDst = iSrc+1, iDst+2 {
			apiData[iDst] = srcString[iSrc]
		}

	case reflect.Uint32: // We were passed in a uint32 (store as REG_DWORD)
		cbData = 4
		typeData = syscall.REG_DWORD
		apiData = make([]byte, cbData)
		binary.LittleEndian.PutUint32(apiData, src.(uint32))

	case reflect.Uint64: // We were passed in a uint64 (store as REG_QWORD)
		cbData = 8
		typeData = syscall.REG_QWORD
		apiData = make([]byte, cbData)
		binary.LittleEndian.PutUint64(apiData, src.(uint64))

	default: // Log any unsupported requests
		log.Errorf("Unsupported SHSetValue input, hkey=%v, registry=%v:%v, srcKind=%v", hkey, keyName, valueName, srcKind)
		return windows.ERROR_INVALID_PARAMETER
	}

	// Convert input strings to UTF16
	keyNameUTF16 := syscall.StringToUTF16(keyName)
	valueNameUTF16 := syscall.StringToUTF16(valueName)

	// Call the Win32 shell API
	ret, _, _ := procSHSetValueW.Call(
		hkey,
		uintptr(unsafe.Pointer(&keyNameUTF16[0])),
		uintptr(unsafe.Pointer(&valueNameUTF16[0])),
		uintptr(typeData),
		uintptr(unsafe.Pointer(&apiData[0])),
		uintptr(cbData),
	)

	// Return an error if the request failed (log failure)
	if ret != 0 {
		log.Errorf("SHSetValue, hkey=%v, registry=%v:%v, err=%v", hkey, keyName, valueName, syscall.Errno(ret))
		return syscall.Errno(ret)
	}

	// Return nil on success
	return nil
}

// SHGetValue wraps the SHGetValueW API.  The input parameters are used to get the value of a registry
// key.  The input dst object can be either a string (REG_SZ), uint32 (REG_DWORD), or uint64 (REG_QWORD)
// and it must be a pointer to that object.
func SHGetValue(hkey uintptr, keyName string, valueName string, dst interface{}) error {

	// We only support receiving a pointer to an object otherwise we'll immediately fail the request
	dstType := reflect.TypeOf(dst)
	dstKind := dstType.Kind()
	if dstKind != reflect.Ptr {
		return windows.ERROR_INVALID_PARAMETER
	}

	// Determine the destination kind
	dstKind = dstType.Elem().Kind()

	// Convert input strings to UTF16
	keyNameUTF16 := syscall.StringToUTF16(keyName)
	valueNameUTF16 := syscall.StringToUTF16(valueName)

	// We don't know how large the registry value is so we'll start by allocating a 256 byte buffer
	// which will be plenty large for the vast majority of requests we receive.
	cbData := 256

	// Enumerated registry data
	var apiData []uint8
	var dataType uint32
	var ret uintptr

	// Keep enumerating until ERROR_MORE_DATA is no longer returned (up to 5 times)
	i := 0
	for ret = uintptr(syscall.ERROR_MORE_DATA); (i < 5) && (cbData != 0) && (ret == uintptr(syscall.ERROR_MORE_DATA)); i++ {
		apiData = make([]uint8, cbData)
		ret, _, _ = procSHGetValueW.Call(
			hkey,
			uintptr(unsafe.Pointer(&keyNameUTF16[0])),
			uintptr(unsafe.Pointer(&valueNameUTF16[0])),
			uintptr(unsafe.Pointer(&dataType)),
			uintptr(unsafe.Pointer(&apiData[0])),
			uintptr(unsafe.Pointer(&cbData)),
		)
	}

	// Return an error if the request failed (failure logged as informational only as may not be an error condition)
	if ret != 0 {
		log.Tracef("SHGetValue, hkey=%v, registry=%v:%v, err=%v", hkey, keyName, valueName, syscall.Errno(ret))
		return syscall.Errno(ret)
	}

	// Custom handle data based on data type retrieved from the registry
	switch dataType {
	case syscall.REG_SZ, syscall.REG_EXPAND_SZ:
		// Fail request if retrieved data isn't a string
		if dstKind != reflect.String {
			log.Errorf("Expected string registry type, hkey=%v, registry=%v:%v, dstKind=%v", hkey, keyName, valueName, dstKind)
			return windows.ERROR_INVALID_PARAMETER
		}

		// Convert from byte array to UTF16 array
		utf16 := make([]uint16, cbData/2)
		for iSrc, iDst := 0, 0; iSrc < cbData; iSrc, iDst = iSrc+2, iDst+1 {
			utf16[iDst] = uint16(apiData[iSrc+0]) | (uint16(apiData[iSrc+1]) << 8)
		}

		// Convert UTF16 to string and return to caller
		reflect.ValueOf(dst).Elem().SetString(syscall.UTF16ToString(utf16))

	case syscall.REG_DWORD:
		// Fail request if retrieved data isn't a uint32
		if dstKind != reflect.Uint32 {
			log.Errorf("Expected uint32 registry type, hkey=%v, registry=%v:%v, dstKind=%v", hkey, keyName, valueName, dstKind)
			return windows.ERROR_INVALID_PARAMETER
		}

		// Convert byte array to uint32 and return to caller
		value := binary.LittleEndian.Uint32(apiData)
		dst = &value

	case syscall.REG_QWORD:
		// Fail request if retrieved data isn't a uint64
		if dstKind != reflect.Uint64 {
			log.Errorf("Expected uint64 registry type, hkey=%v, registry=%v:%v, dstKind=%v", hkey, keyName, valueName, dstKind)
			return windows.ERROR_INVALID_PARAMETER
		}

		// Convert byte array to uint64 and return to caller
		value := binary.LittleEndian.Uint64(apiData)
		dst = &value

	default:
		// Fail request if unsupported registry data type
		log.Errorf("Unexpected registry type, hkey=%v, registry=%v:%v, dataType=%v, dstKind=%v", hkey, keyName, valueName, dataType, dstKind)
		return windows.ERROR_INVALID_PARAMETER
	}

	return nil
}
