// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

package wmi

import (
	"runtime"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	log "github.com/hpe-storage/common-host-libs/logger"
)

// ExecWmiMethod is used to execute a WMI method.  A VARIANT is returned from the WMI method as well
// as an error should the WMI method be executed.
func ExecWmiMethod(className, methodName, namespace string, params ...interface{}) (result *ole.VARIANT, err error) {
	log.Tracef(">>>>> ExecWmiMethod, className=%v, methodName=%v, namespace=%v", className, methodName, namespace)
	defer log.Trace("<<<<< ExecWmiMethod")

	// Only support one WMI query at a time
	lock.Lock()
	defer lock.Unlock()

	// LockOSThread wires the calling goroutine to its current operating system thread. The calling
	// goroutine will always execute in that thread, and no other goroutine will execute in it,
	// until the calling goroutine has made as many calls to UnlockOSThread as to LockOSThread. If
	// the calling goroutine exits without unlocking the thread, the thread will be terminated.
	//
	// A goroutine should call LockOSThread before calling OS services or non-Go library functions
	// that depend on per-thread state.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Get WMI interface
	unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		log.Panic(err)
	}
	defer unknown.Release()

	// Get WMI IDispatch interface
	wmi, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		log.Panic(err)
	}
	defer wmi.Release()

	// Connect to WMI
	connectServerRaw, err := oleutil.CallMethod(wmi, "ConnectServer", nil, namespace)
	if err != nil {
		log.Panic(err)
	}
	connectServer := connectServerRaw.ToIDispatch()
	defer connectServerRaw.Clear()

	// Get the WMI class
	wmiClassRaw, err := oleutil.CallMethod(connectServer, "Get", className)
	if err != nil {
		log.Panic(err)
	}
	wmiClass := wmiClassRaw.ToIDispatch()
	defer wmiClassRaw.Clear()

	// Execute the WMI method method
	results, err := oleutil.CallMethod(wmiClass, methodName, params...)
	return results, err
}
