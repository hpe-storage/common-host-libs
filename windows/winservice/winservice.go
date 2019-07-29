// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

//-------------------------------------------------------------------------------------------------
//
// This winservice package is a slightly modified version of the example service the Go team
// provided here:
//
// https://godoc.org/golang.org/x/sys/windows/svc/example
// https://github.com/golang/sys/tree/master/windows/svc/example
//
// The install.go, manage.go, and service.go modules have been moved into this winservice package
// so that the service routines can be shared among our Windows clients.  The winservice client
// starts by initializing a WinService object.  For example:
//
//		var (
//			myService winservice.WinService
//		)
//
//		func main() {
//      	...
//			myService.UseEventLog = false
//			myService.Start = serviceStart
//			myService.Stop = serviceStop
//      	...
//			return
//		}
//
// 		func serviceStart() {
//			// This routine will be called when the winservice framework starts the service
// 		}
//
// 		func serviceStop() {
//			// This routine will be called when the winservice framework stops the service
// 		}
//
//-------------------------------------------------------------------------------------------------
//
// When comparing the windows/svc/example sample code, with this package, you will see a number of
// minor differences including:
//
//	main.go
//		o	The module main.go is *not* part of the winservice package.  Your Windows service would
//			replace the functionality found in this module.  See the following module for an
//			example on how to properly utilize the winservice package:
//			github.com/hpe-storage/common-host-utils/cmd/chapid/chapid_windows.go
//	beep.go
//		o	File removed as it is not needed.  Only used by golang's sample service.
//	install.go
//		o	Made InstallService and RemoveService public member functions.
//		o	The InstallService routine now requires a service description where one was not
//			supported in the sample code.  This allows users, viewing the services, to get a
//			meaningful service description.
//		o	Event log usage is now optional.  If the caller wants the framework to record activity
//			to the application event log, the UseEventLog boolean must be set to true.  If support
//			for additional events is needed, the framework can be adjusted at a later date to
//			provide the necessary method.
//	manage.go
//		o	Made StartService and ControlService public member functions.
//	service.go
//		o	Made RunService a public member function.
//		o	Made Execute function a member of WinService instead of myservice.  This gives the
//			Execute function access to the WinService.Start/WinService.Stop callback routines.
//		o	Removed the ability to pause/resume the service for simplicity.  It's not anticipated
//			that this capability will be needed (since stop/start is sufficient).  Support can be
//			integrated at a later date should it be needed.
//
//-------------------------------------------------------------------------------------------------

package winservice

// WinService is a structure that the Windows client service must initialize prior to calling our
// winservice member functions.
type WinService struct {
	UseEventLog bool   // Does the Windows service want events recorded to the application event log?
	Start       func() // Pointer to function that framework will call to start the service
	Stop        func() // Pointer to function that framework will call to stop the service
}
