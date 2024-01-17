// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

/*
This package allows you to easily enumerate WMI classes and have them unmarshalled automatically
into Go objects.  When enumerating a WMI class that only returns a single class, you pass in a
pointer to the Go struct.  When enumerating a WMI class that returns one or more classes, you
pass in a pointer to a slice of pointers.  Here are two examples:

Example #1 - WMI class returns a single object

	var operatingSystem *wmi.Win32_OperatingSystem
	err := ExecQuery("SELECT * FROM Win32_OperatingSystem", `ROOT\CIMV2`, &operatingSystem)

Example #2 - WMI class returns one or more objects

	var volumes []*wmi.Win32_Volume
	err := ExecQuery("SELECT * FROM Win32_Volume", `ROOT\CIMV2`, &volumes)

Go Struct Definition

	When defining your Go structure, you should mirror the WMI class definition.  For example,
	here is a Win32_Volume WMI class definition:

	class Win32_Volume
	{
		uint16   Access;
		boolean  Automount;
		uint16   Availability;
		uint64   BlockSize;
		boolean  BootVolume;
		uint64   Capacity;
		...
	}

	Here is the Go struct definition

	type Win32_Volume struct {
		Access        uint16
		Automount     bool
		Availability  uint16
		BlockSize     uint64
		BootVolume    bool
		Capacity      uint64
		...
	}

	Note above how the property names and types align.

Go Struct Field Tags

	You can attach tags to any Go struct field.  These tags provide guidance when unmarhsalling a
	WMI class field into a Go field.

	Example #1

		type Win32_Volume struct {
			BlockSizeInBytes uint16 `wmi:"BlockSize"`

		We know that the Win32_Volume WMI class has a "BlockSize" field.  The above example shows
		how you can give the Go field a different name than the WMI class.  We do not recommend
		you use this capability, as it's always best to align with the WMI field name, but it's
		available should you have a desire to rename the Go field name.

	Example #2

		type Win32_Volume struct {
			MyPrivateData uint64 `wmi:"-"`

		When you specify the "-" field name, this instructs the unmarshalling engine to ignore this
		field.  This can be useful if you attach your own vendor unique fields to the Go struct

	Example #3

		type Win32_Volume struct {
			ConfigManagerErrorCode uint32 `wmi:",nil=0xFFFFFFFF"`

		Every WMI class field is nullable.  That means that a uint32 value can be null.  This
		complicates the unmarshalling because how do you convert that null into a Go uint32?
		There are three techniques you can use in defining your Go object to deal with a value
		that might be null.  Using our ConfirManagerErrorCode as an example, here are three
		options:

		Option #1 - ConfigManagerErrorCode uint32

			This is the recommended option for most WMI class fields.  If WMI returns a null value,
			we will leave the value as its default value (i.e. 0 for a uint32).  In almost all
			cases, the default value meets our needs.

		Option #2 - ConfigManagerErrorCode uint32 `wmi:",nil=0xFFFFFFFF"`

			This is the recommended option for those WMI class fields where the default value may
			not be optimal if null is returned from WMI.  For example, the Win32_Volume definition
			of ConfigManagerErrorCode has 0 defined as "This device is working properly".  Clearly
			a default value of 0 is not optimal should the WMI class return null for this field.
			By attaching the `wmi:",nil=0xFFFFFFFF"` tag, we are informing the WMI unmarshalling
			engine to return 0xFFFFFFFF should a null value be returned by WMI.  A value of
			0xFFFFFFFF is undefined and can be used to represent an invalid or unavailable value.

		Option #3 - ConfigManagerErrorCode *uint32

			In theory, you could define every single WMI class as being a pointer.  Then every field
			is nullable.  In practice, this is usually not a recommended option.  You would have too
			many small allocations when such allocations were not really required.  However, this
			option is available to you if desired.
*/

package wmi

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	log "github.com/hpe-storage/common-host-libs/logger"
	"golang.org/x/sys/windows"
)

// Package variables
var (
	// WMI locks
	lock sync.Mutex

	// Lazy load the ole32.dll APIs
	ole32                      = windows.NewLazySystemDLL("ole32.dll")
	procCoInitializeSecurity   = ole32.NewProc("CoInitializeSecurity")
	modoleaut32, _             = syscall.LoadDLL("oleaut32.dll")
	procSafeArrayGetElement, _ = modoleaut32.FindProc("SafeArrayGetElement")

	// WMI Class and Interface GUIDs
	CLSID_WbemLocator    = ole.NewGUID("4590f811-1d3a-11d0-891f-00aa004b2e24")
	IID_IWbemLocator     = ole.NewGUID("dc12a687-737f-11cf-884d-00aa004b2e24")
	IID_IWbemClassObject = ole.NewGUID("dc12a681-737f-11cf-884d-00aa004b2e24")

	// Map to convert CIM type to reflect.Kind
	cimTypeToGoType map[CIMTYPE_ENUMERATION]reflect.Type

	comInitialized bool          // Did COM successfully initialize?
	wmiWbemLocator *ole.IUnknown // Enumerated WMI locator object
)

// Namespaces we use for WMI queries
const (
	rootCIMV2                   = `ROOT\CIMV2`
	rootMicrosoftWindowsStorage = `ROOT\Microsoft\Windows\Storage`
	rootMSCluster               = `ROOT\MSCluster`
	rootWMI                     = `ROOT\WMI`
)

// HRESULT values
const (
	S_OK                     = 0
	S_FALSE                  = 1
	WBEM_S_NO_ERROR          = 0
	WBEM_S_FALSE             = 1
	WBEM_E_CRITICAL_ERROR    = 0x8004100A
	WBEM_E_NOT_SUPPORTED     = 0x8004100C
	WBEM_E_INVALID_NAMESPACE = 0x8004100E
	WBEM_E_INVALID_CLASS     = 0x80041010
)

// The CIMTYPE_ENUMERATION enumeration defines values that specify different CIM data types
type CIMTYPE_ENUMERATION uint32

const (
	CIM_ILLEGAL    CIMTYPE_ENUMERATION = 0xFFF
	CIM_EMPTY      CIMTYPE_ENUMERATION = 0
	CIM_SINT8      CIMTYPE_ENUMERATION = 16
	CIM_UINT8      CIMTYPE_ENUMERATION = 17
	CIM_SINT16     CIMTYPE_ENUMERATION = 2
	CIM_UINT16     CIMTYPE_ENUMERATION = 18
	CIM_SINT32     CIMTYPE_ENUMERATION = 3
	CIM_UINT32     CIMTYPE_ENUMERATION = 19
	CIM_SINT64     CIMTYPE_ENUMERATION = 20
	CIM_UINT64     CIMTYPE_ENUMERATION = 21
	CIM_REAL32     CIMTYPE_ENUMERATION = 4
	CIM_REAL64     CIMTYPE_ENUMERATION = 5
	CIM_BOOLEAN    CIMTYPE_ENUMERATION = 11
	CIM_STRING     CIMTYPE_ENUMERATION = 8
	CIM_DATETIME   CIMTYPE_ENUMERATION = 101
	CIM_REFERENCE  CIMTYPE_ENUMERATION = 102
	CIM_CHAR16     CIMTYPE_ENUMERATION = 103
	CIM_OBJECT     CIMTYPE_ENUMERATION = 13
	CIM_FLAG_ARRAY CIMTYPE_ENUMERATION = 0x2000
)

// EOLE_AUTHENTICATION_CAPABILITIES specifies various capabilities in CoInitializeSecurity
// and IClientSecurity::SetBlanket (or its helper function CoSetProxyBlanket).
type EOLE_AUTHENTICATION_CAPABILITIES uint32

const (
	EOAC_NONE              EOLE_AUTHENTICATION_CAPABILITIES = 0
	EOAC_MUTUAL_AUTH       EOLE_AUTHENTICATION_CAPABILITIES = 0x1
	EOAC_STATIC_CLOAKING   EOLE_AUTHENTICATION_CAPABILITIES = 0x20
	EOAC_DYNAMIC_CLOAKING  EOLE_AUTHENTICATION_CAPABILITIES = 0x40
	EOAC_ANY_AUTHORITY     EOLE_AUTHENTICATION_CAPABILITIES = 0x80
	EOAC_MAKE_FULLSIC      EOLE_AUTHENTICATION_CAPABILITIES = 0x100
	EOAC_DEFAULT           EOLE_AUTHENTICATION_CAPABILITIES = 0x800
	EOAC_SECURE_REFS       EOLE_AUTHENTICATION_CAPABILITIES = 0x2
	EOAC_ACCESS_CONTROL    EOLE_AUTHENTICATION_CAPABILITIES = 0x4
	EOAC_APPID             EOLE_AUTHENTICATION_CAPABILITIES = 0x8
	EOAC_DYNAMIC           EOLE_AUTHENTICATION_CAPABILITIES = 0x10
	EOAC_REQUIRE_FULLSIC   EOLE_AUTHENTICATION_CAPABILITIES = 0x200
	EOAC_AUTO_IMPERSONATE  EOLE_AUTHENTICATION_CAPABILITIES = 0x400
	EOAC_DISABLE_AAA       EOLE_AUTHENTICATION_CAPABILITIES = 0x1000
	EOAC_NO_CUSTOM_MARSHAL EOLE_AUTHENTICATION_CAPABILITIES = 0x2000
	EOAC_RESERVED1         EOLE_AUTHENTICATION_CAPABILITIES = 0x4000
)

// Authentication Level Constants
const (
	RPC_C_AUTHN_LEVEL_DEFAULT       = 0
	RPC_C_AUTHN_LEVEL_NONE          = 1
	RPC_C_AUTHN_LEVEL_CONNECT       = 2
	RPC_C_AUTHN_LEVEL_CALL          = 3
	RPC_C_AUTHN_LEVEL_PKT           = 4
	RPC_C_AUTHN_LEVEL_PKT_INTEGRITY = 5
	RPC_C_AUTHN_LEVEL_PKT_PRIVACY   = 6
)

// Impersonation Level Constants
const (
	RPC_C_IMP_LEVEL_DEFAULT     = 0
	RPC_C_IMP_LEVEL_ANONYMOUS   = 1
	RPC_C_IMP_LEVEL_IDENTIFY    = 2
	RPC_C_IMP_LEVEL_IMPERSONATE = 3
	RPC_C_IMP_LEVEL_DELEGATE    = 4
)

// WBEM_GENERIC_FLAG_TYPE enumeration is used to indicate and update the type of the flag
type WBEM_GENERIC_FLAG_TYPE uint32

const (
	WBEM_FLAG_RETURN_WBEM_COMPLETE   WBEM_GENERIC_FLAG_TYPE = 0x0
	WBEM_FLAG_RETURN_IMMEDIATELY     WBEM_GENERIC_FLAG_TYPE = 0x10
	WBEM_FLAG_FORWARD_ONLY           WBEM_GENERIC_FLAG_TYPE = 0x20
	WBEM_FLAG_NO_ERROR_OBJECT        WBEM_GENERIC_FLAG_TYPE = 0x40
	WBEM_FLAG_SEND_STATUS            WBEM_GENERIC_FLAG_TYPE = 0x80
	WBEM_FLAG_ENSURE_LOCATABLE       WBEM_GENERIC_FLAG_TYPE = 0x100
	WBEM_FLAG_DIRECT_READ            WBEM_GENERIC_FLAG_TYPE = 0x200
	WBEM_MASK_RESERVED_FLAGS         WBEM_GENERIC_FLAG_TYPE = 0x1F000
	WBEM_FLAG_USE_AMENDED_QUALIFIERS WBEM_GENERIC_FLAG_TYPE = 0x20000
	WBEM_FLAG_STRONG_VALIDATION      WBEM_GENERIC_FLAG_TYPE = 0x100000
)

// WBEM_TIMEOUT_TYPE contains values used to specify the timeout for the IEnumWbemClassObject::Next method
type WBEM_TIMEOUT_TYPE uint32

const (
	WBEM_NO_WAIT  WBEM_TIMEOUT_TYPE = 0
	WBEM_INFINITE WBEM_TIMEOUT_TYPE = 0xFFFFFFFF
)

// WBEM_CONDITION_FLAG_TYPE contains flags used with the IWbemClassObject::GetNames method.
type WBEM_CONDITION_FLAG_TYPE uint32

const (
	WBEM_FLAG_ALWAYS                    WBEM_CONDITION_FLAG_TYPE = 0
	WBEM_FLAG_ONLY_IF_TRUE              WBEM_CONDITION_FLAG_TYPE = 0x1
	WBEM_FLAG_ONLY_IF_FALSE             WBEM_CONDITION_FLAG_TYPE = 0x2
	WBEM_FLAG_ONLY_IF_IDENTICAL         WBEM_CONDITION_FLAG_TYPE = 0x3
	WBEM_MASK_PRIMARY_CONDITION         WBEM_CONDITION_FLAG_TYPE = 0x3
	WBEM_FLAG_KEYS_ONLY                 WBEM_CONDITION_FLAG_TYPE = 0x4
	WBEM_FLAG_REFS_ONLY                 WBEM_CONDITION_FLAG_TYPE = 0x8
	WBEM_FLAG_LOCAL_ONLY                WBEM_CONDITION_FLAG_TYPE = 0x10
	WBEM_FLAG_PROPAGATED_ONLY           WBEM_CONDITION_FLAG_TYPE = 0x20
	WBEM_FLAG_SYSTEM_ONLY               WBEM_CONDITION_FLAG_TYPE = 0x30
	WBEM_FLAG_NONSYSTEM_ONLY            WBEM_CONDITION_FLAG_TYPE = 0x40
	WBEM_MASK_CONDITION_ORIGIN          WBEM_CONDITION_FLAG_TYPE = 0x70
	WBEM_FLAG_CLASS_OVERRIDES_ONLY      WBEM_CONDITION_FLAG_TYPE = 0x100
	WBEM_FLAG_CLASS_LOCAL_AND_OVERRIDES WBEM_CONDITION_FLAG_TYPE = 0x200
	WBEM_MASK_CLASS_CONDITION           WBEM_CONDITION_FLAG_TYPE = 0x300
)

// WBEM_FLAVOR_TYPE lists qualifier flavors
type WBEM_FLAVOR_TYPE uint32

const (
	WBEM_FLAVOR_DONT_PROPAGATE                  WBEM_FLAVOR_TYPE = 0
	WBEM_FLAVOR_FLAG_PROPAGATE_TO_INSTANCE      WBEM_FLAVOR_TYPE = 0x1
	WBEM_FLAVOR_FLAG_PROPAGATE_TO_DERIVED_CLASS WBEM_FLAVOR_TYPE = 0x2
	WBEM_FLAVOR_MASK_PROPAGATION                WBEM_FLAVOR_TYPE = 0xf
	WBEM_FLAVOR_OVERRIDABLE                     WBEM_FLAVOR_TYPE = 0
	WBEM_FLAVOR_NOT_OVERRIDABLE                 WBEM_FLAVOR_TYPE = 0x10
	WBEM_FLAVOR_MASK_PERMISSIONS                WBEM_FLAVOR_TYPE = 0x10
	WBEM_FLAVOR_ORIGIN_LOCAL                    WBEM_FLAVOR_TYPE = 0
	WBEM_FLAVOR_ORIGIN_PROPAGATED               WBEM_FLAVOR_TYPE = 0x20
	WBEM_FLAVOR_ORIGIN_SYSTEM                   WBEM_FLAVOR_TYPE = 0x40
	WBEM_FLAVOR_MASK_ORIGIN                     WBEM_FLAVOR_TYPE = 0x60
	WBEM_FLAVOR_NOT_AMENDED                     WBEM_FLAVOR_TYPE = 0
	WBEM_FLAVOR_AMENDED                         WBEM_FLAVOR_TYPE = 0x80
	WBEM_FLAVOR_MASK_AMENDED                    WBEM_FLAVOR_TYPE = 0x80
)

// IWbemLocatorVtbl is the IWbemLocator COM virtual table
type IWbemLocatorVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	ConnectServer  uintptr
}

// IWbemServicesVtbl is the IWbemServices COM virtual table
type IWbemServicesVtbl struct {
	QueryInterface             uintptr
	AddRef                     uintptr
	Release                    uintptr
	OpenNamespace              uintptr
	CancelAsyncCall            uintptr
	QueryObjectSink            uintptr
	GetObject                  uintptr
	GetObjectAsync             uintptr
	PutClass                   uintptr
	PutClassAsync              uintptr
	DeleteClass                uintptr
	DeleteClassAsync           uintptr
	CreateClassEnum            uintptr
	CreateClassEnumAsync       uintptr
	PutInstance                uintptr
	PutInstanceAsync           uintptr
	DeleteInstance             uintptr
	DeleteInstanceAsync        uintptr
	CreateInstanceEnum         uintptr
	CreateInstanceEnumAsync    uintptr
	ExecQuery                  uintptr
	ExecQueryAsync             uintptr
	ExecNotificationQuery      uintptr
	ExecNotificationQueryAsync uintptr
	ExecMethod                 uintptr
	ExecMethodAsync            uintptr
}

// IEnumWbemClassObjectVtbl is the IEnumWbemClassObject COM virtual table
type IEnumWbemClassObjectVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Reset          uintptr
	Next           uintptr
	NextAsync      uintptr
	Clone          uintptr
	Skip           uintptr
}

// IWbemClassObjectVtbl is the IWbemClassObject COM virtual table
type IWbemClassObjectVtbl struct {
	QueryInterface          uintptr
	AddRef                  uintptr
	Release                 uintptr
	GetQualifierSet         uintptr
	Get                     uintptr
	Put                     uintptr
	Delete                  uintptr
	GetNames                uintptr
	BeginEnumeration        uintptr
	Next                    uintptr
	EndEnumeration          uintptr
	GetPropertyQualifierSet uintptr
	Clone                   uintptr
	GetObjectText           uintptr
	SpawnDerivedClass       uintptr
	SpawnInstance           uintptr
	CompareTo               uintptr
	GetPropertyOrigin       uintptr
	InheritsFrom            uintptr
	GetMethod               uintptr
	PutMethod               uintptr
	DeleteMethod            uintptr
	BeginMethodEnumeration  uintptr
	NextMethod              uintptr
	EndMethodEnumeration    uintptr
	GetMethodQualifierSet   uintptr
	GetMethodOrigin         uintptr
}

// interfaceFieldInfo is used to store details about each field in a Go struct
type interfaceFieldInfo struct {
	index     int          // Index into the Go structure
	fieldType reflect.Type // Field type
	fieldKind reflect.Kind // Field kind
	nilValue  interface{}  // "nil" tag attribute (nil value if not provided)
}

// Initialize
func init() {
	// Create a map to convert a CIMTYPE_ENUMERATION into reflect.Kind
	cimTypeToGoType = map[CIMTYPE_ENUMERATION]reflect.Type{
		CIM_SINT8:    reflect.TypeOf(int8(0)),
		CIM_UINT8:    reflect.TypeOf(uint8(0)),
		CIM_SINT16:   reflect.TypeOf(int16(0)),
		CIM_UINT16:   reflect.TypeOf(uint16(0)),
		CIM_SINT32:   reflect.TypeOf(int32(0)),
		CIM_UINT32:   reflect.TypeOf(uint32(0)),
		CIM_SINT64:   reflect.TypeOf(int64(0)),
		CIM_UINT64:   reflect.TypeOf(uint64(0)),
		CIM_REAL32:   reflect.TypeOf(float32(0)),
		CIM_REAL64:   reflect.TypeOf(float64(0)),
		CIM_BOOLEAN:  reflect.TypeOf(false),
		CIM_STRING:   reflect.TypeOf(""),
		CIM_DATETIME: reflect.TypeOf(""),
		CIM_CHAR16:   reflect.TypeOf(uint16(0)),
		// We do not support conversion for the following CIM types
		CIM_ILLEGAL:   nil,
		CIM_EMPTY:     nil,
		CIM_REFERENCE: nil,
		CIM_OBJECT:    nil,
	}

	// Initialize the COM library for use by our calling thread (all init functions are run on the
	// startup thread).  Handle case where COM library is already initialized on this thread.
	comInitialized = true
	err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)
	if err != nil {
		// If an ole.OleError error is returned, and S_OK or S_FALSE is returned, then we
		// ignore the error and continue COM initialization.
		comInitialized = false
		if oleCode, ok := err.(*ole.OleError); ok == true {
			switch oleCode.Code() {
			case S_OK, S_FALSE:
				comInitialized = true
			}
		}
		// Log error if unexpected failure occurs
		if !comInitialized {
			log.Errorf("Unable to initialize COM, err=%v", err)
		}
	}

	// Set general COM security levels
	if comInitialized {
		hres, _, _ := procCoInitializeSecurity.Call(
			uintptr(0),
			uintptr(0xFFFFFFFF),                  // COM authentication
			uintptr(0),                           // Authentication services
			uintptr(0),                           // Reserved
			uintptr(RPC_C_AUTHN_LEVEL_DEFAULT),   // Default authentication
			uintptr(RPC_C_IMP_LEVEL_IMPERSONATE), // Default Impersonation
			uintptr(0),                           // Authentication info
			uintptr(EOAC_NONE),                   // Additional capabilities
			uintptr(0))                           // Reserved
		if FAILED(hres) {
			log.Errorf("Unable to initialize COM security, err=%v", ole.NewError(hres))
		} else {
			// Obtain the initial locator to WMI
			wmiWbemLocator, err = ole.CreateInstance(CLSID_WbemLocator, IID_IWbemLocator)
			if err != nil {
				log.Errorf("Unable to obtain the initial locator to WMI, err=%v", err)
				wmiWbemLocator = nil
			}
		}
	}
}

// Cleanup is an optional routine that should only be called when the process using the WMI package
// is exiting.
func Cleanup() {
	lock.Lock()
	defer lock.Unlock()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if wmiWbemLocator != nil {
		wmiWbemLocator.Release()
		wmiWbemLocator = nil
	}
	if comInitialized {
		ole.CoUninitialize()
	}
}

// ExecQuery executes the given WMI query, in the given namespace, and returns JSON objects
func ExecQuery(wqlQuery string, namespace string, dst interface{}) (err error) {

	log.Tracef(">>>>> ExecQuery, wqlQuery=%v, namespace=%v", wqlQuery, namespace)
	defer log.Trace("<<<<< ExecQuery")

	// If our package init routine was unable to initialize COM, immediately fail the request
	if wmiWbemLocator == nil {
		log.Error("COM initialization was not successful during init(), failing WMI query")
		return ole.NewError(WBEM_E_CRITICAL_ERROR)
	}

	// Get the destination object path and type
	dstPath, dstType := getInterfaceType(dst)

	// There are only two type of destination types this routine supports.
	//
	// dstPath		ptr.ptr.struct
	// Description	We were passed in a pointer to a pointer to a struct
	// Example		var operatingSystem *Win32_OperatingSystem
	//				err := ExecQuery("SELECT * FROM Win32_OperatingSystem", `ROOT\CIMV2`, &operatingSystem)
	//
	// dstPath		ptr.slice.ptr.struct
	// Description	We were passed in a pointer to an array of pointers to structs
	// Example		var volumes []*Win32_Volume
	//				err := ExecQuery("SELECT * FROM Win32_Volume", `ROOT\CIMV2`, &volumes)
	//
	// For any other destination object, we log an error and fail the request.
	var isSlicePtr bool
	switch dstPath {
	case "ptr.ptr.struct":
		isSlicePtr = false
	case "ptr.slice.ptr.struct":
		isSlicePtr = true
	default:
		// Log and fail request if an unsupported destination object was passed in
		log.Errorf("Unsupported destination object, dstType=%v, dstPath=%v", dstType.Name(), dstPath)
		return windows.ERROR_INVALID_PARAMETER
	}

	// Log details about the destination object
	log.Tracef("Destination object, dstType=%v, dstPath=%v, isSlicePtr=%v", dstType.Name(), dstPath, isSlicePtr)

	// How we execute WMI queries from Go is modeled from Microsoft's sample C++ code
	// https://docs.microsoft.com/en-us/windows/desktop/wmisdk/example--getting-wmi-data-from-the-local-computer

	// Variable used to store COM HRESULT
	var hres uintptr

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

	// Connect to WMI through the IWbemLocator::ConnectServer method
	var pSvc *ole.IUnknown
	namespaceUTF16 := syscall.StringToUTF16(namespace)
	myVTable := (*IWbemLocatorVtbl)(unsafe.Pointer(wmiWbemLocator.RawVTable))
	hres, _, _ = syscall.Syscall9(myVTable.ConnectServer, 9, // Call the IWbemLocator::ConnectServer method
		uintptr(unsafe.Pointer(wmiWbemLocator)),
		uintptr(unsafe.Pointer(&namespaceUTF16[0])),
		uintptr(0),
		uintptr(0),
		uintptr(0),
		uintptr(0),
		uintptr(0),
		uintptr(0),
		uintptr(unsafe.Pointer(&pSvc)))
	if FAILED(hres) {
		err = ole.NewError(hres)
		msg := fmt.Sprintf("Failed IWbemLocator::ConnectServer method, err=%v", err)
		// If WMI namespace isn't present on this host, we don't consider that an error and log the
		// result as informational, else it's logged as an error.
		if hres == WBEM_E_INVALID_NAMESPACE {
			log.Trace(msg)
		} else {
			log.Error(msg)
		}
		return err
	}
	defer pSvc.Release()

	// Use the IWbemServices pointer to send the WMI query
	var pEnumerator *ole.IUnknown
	wqlUTF16 := syscall.StringToUTF16(`WQL`)
	queryUTF16 := syscall.StringToUTF16(wqlQuery)
	pSvcVTable := (*IWbemServicesVtbl)(unsafe.Pointer(pSvc.RawVTable))
	hres, _, _ = syscall.Syscall6(pSvcVTable.ExecQuery, 6, // Call the IWbemServices::ExecQuery method
		uintptr(unsafe.Pointer(pSvc)),
		uintptr(unsafe.Pointer(&wqlUTF16[0])),
		uintptr(unsafe.Pointer(&queryUTF16[0])),
		uintptr(WBEM_FLAG_FORWARD_ONLY|WBEM_FLAG_RETURN_IMMEDIATELY),
		uintptr(0),
		uintptr(unsafe.Pointer(&pEnumerator)))
	if FAILED(hres) {
		err = ole.NewError(hres)
		log.Errorf("Failed IWbemServices::ExecQuery method, err=%v", err)
		return err
	}
	defer pEnumerator.Release()

	// Delcare our return object as reflect.Value.  If we're returning an array of pointers to
	// structs (i.e. isSlicePtr==true), then we'll set returnObject to be a slice of structs.
	var returnObject reflect.Value
	if isSlicePtr {
		returnObject = reflect.MakeSlice(reflect.TypeOf(dst).Elem(), 0, 0)
	}

	// Enumerate each WMI object
	for itemCount := 0; ; itemCount++ {

		var pclsObj *ole.IUnknown
		var uReturn uint32

		// Enumerate the next WMI object
		pEnumeratorVTable := (*IEnumWbemClassObjectVtbl)(unsafe.Pointer(pEnumerator.RawVTable))
		hres, _, _ = syscall.Syscall6(pEnumeratorVTable.Next, 5,
			uintptr(unsafe.Pointer(pEnumerator)), // Call the IEnumWbemClassObject::Next method
			uintptr(WBEM_INFINITE),
			uintptr(1),
			uintptr(unsafe.Pointer(&pclsObj)),
			uintptr(unsafe.Pointer(&uReturn)),
			uintptr(0))

		// Break out of while loop when no more objects returned
		if uReturn == 0 {
			// If no objects enumerated, and WMI query is not supported, log event and fail request
			// with ERROR_NOT_SUPPORTED
			if (itemCount == 0) && ((hres == WBEM_E_NOT_SUPPORTED) || (hres == WBEM_E_INVALID_CLASS)) {
				log.Tracef("WMI query not supported, hres=%08Xh, wqlQuery=%v", hres, wqlQuery)
				return windows.ERROR_NOT_SUPPORTED
			}
			if (hres != WBEM_S_NO_ERROR) && (hres != WBEM_S_FALSE) {
				log.Errorf("Failed IEnumWbemClassObject::Next method, itemCount=%v, hres=%xh", itemCount, hres)
			}
			break
		}

		// Log the number of WMI classes enumerated thus far
		log.Tracef("Enumerating WMI class object %v", itemCount)

		// Allocate a new Go object for the WMI class and then unmarshall the WMI class into the Go object
		dstObject := reflect.New(dstType)
		err = wmiClassToGoObject(pclsObj, dstObject.Interface(), "")

		// Release COM object before analyzing error; we're done with the object now
		pclsObj.Release()
		pclsObj = nil

		// Error out if wmiClassToGoObject() failed
		if err != nil {
			log.Errorf("Unable to unmarshal WMI class into Go object, err=%v", err)
			return err
		}

		// Now that the WMI class has been successfully unmarshalled into our Go object, adjust the
		// return object accordingly.
		if !isSlicePtr {
			// If the passed in destination object is only for a single WMI class, but we enumerated
			// multiple WMI classes, fail the request.
			if itemCount != 0 {
				log.Error("Multiple WMI classes enumerated when destination object can only handle a single object")
				return windows.ERROR_INVALID_PARAMETER
			}
			// Return our unmarshalled WMI class
			returnObject = dstObject
		} else {
			// Append our unmarshalled WMI class to our return slice
			returnObject = reflect.Append(returnObject, dstObject)
		}
	}

	// Fail request if return object was not enumerated
	if returnObject == reflect.ValueOf(nil) {
		log.Errorf("Unexpected nil return object, failing request")
		return ole.NewError(WBEM_E_CRITICAL_ERROR)
	}

	// Return our enumerated WMI class, unmarshalled into a Go object, to the caller
	dv := reflect.ValueOf(dst).Elem()
	dv.Set(returnObject)
	return nil
}

// SUCCEEDED function returns true if HRESULT succeeds, else false
func SUCCEEDED(hresult uintptr) bool {
	return int32(hresult) >= 0
}

// FAILED function returns true if HRESULT fails, else false
func FAILED(hresult uintptr) bool {
	return int32(hresult) < 0
}

// getInterfaceType traverses the passed in interface and returns a string representing the type
// of object it is.  For example, "ptr.ptr.struct" indicates that the dst interface is a pointer
// to a pointer to a struct.  The dstPath string returns this value.  The dstType value returns
// the type of end destination structure (e.g. wmi.MSFT_Disk).
func getInterfaceType(dst interface{}) (dstPath string, dstType reflect.Type) {
	for dstType = reflect.TypeOf(dst); dstType != nil; dstType, dstPath = dstType.Elem(), dstPath+"." {
		dstKind := dstType.Kind()
		dstPath += dstKind.String()
		if (dstKind != reflect.Ptr) && (dstKind != reflect.Slice) {
			break
		}
	}
	return dstPath, dstType
}

// wmiClassToGoObject unmarshals the IUnknown WMI class into the Go object.  It's the responsibility
// of the caller to pass in a pointer to the Go object in order for this routine to populate
// the object accordingly.
func wmiClassToGoObject(wmiClass *ole.IUnknown, goObject interface{}, classProperty string) (err error) {

	// Traverse the Go object to build up a key/value map of its fields
	fieldMap, err := interfaceToFieldMap(goObject)
	if err != nil {
		log.Errorf("Unexpected failure enumerating field map, err=%v", err)
		return err
	}

	// Get the IWbemClassObject VTable
	pClassVTable := (*IWbemClassObjectVtbl)(unsafe.Pointer(wmiClass.RawVTable))

	// // Get the WMI class name
	var vtProp ole.VARIANT
	classUTF16 := syscall.StringToUTF16(`__CLASS`)
	hres, _, _ := syscall.Syscall6(pClassVTable.Get, 6,
		uintptr(unsafe.Pointer(wmiClass)),
		uintptr(unsafe.Pointer(&classUTF16[0])),
		uintptr(0),
		uintptr(unsafe.Pointer(&vtProp)),
		uintptr(0),
		uintptr(0))
	if FAILED(hres) {
		err = ole.NewError(hres)
		log.Errorf("Unable to query WMI class name, %v", err)
		return err
	}

	// Convert the WMI class name to text and log results
	className := syscall.UTF16ToString((*[1024]uint16)(unsafe.Pointer(uintptr(vtProp.Val)))[:])
	log.Tracef("Enumerated WMI class name, __CLASS=%v", className)
	ole.VariantClear(&vtProp)

	// Query the WMI class property names
	var classPropertyNames *ole.SafeArray
	hres, _, _ = syscall.Syscall6(pClassVTable.GetNames, 5, // Call the IWbemClassObject::GetNames method
		uintptr(unsafe.Pointer(wmiClass)),
		uintptr(0),
		uintptr(WBEM_FLAG_ALWAYS|WBEM_FLAG_NONSYSTEM_ONLY),
		uintptr(0),
		uintptr(unsafe.Pointer(&classPropertyNames)),
		uintptr(0))
	if FAILED(hres) {
		err = ole.NewError(hres)
		log.Errorf("Unable to query WMI class property names, %v", err)
		return err
	}

	// Convert the class property names into an ole.SafeArrayConversion object and enumerate
	// the class property names into a string array.
	safeClassPropertyNames := ole.SafeArrayConversion{Array: classPropertyNames}
	defer safeClassPropertyNames.Release()
	classNames := safeClassPropertyNames.ToStringArray()

	// We're passed in a pointer to the go object we need to fill out.  Get its reflect.Value
	// so that we can fill in the struct fields.
	goObjectValue := reflect.ValueOf(goObject).Elem()

	// At this point we know each of the WMI class field names and each of the Go struct field
	// names.  Check each Go field to see if a WMI field was found.  If not, fill in the field's
	// default value (if provided) before we enumerate each WMI class field.
	for k, v := range fieldMap {
		// Search each WMI class field for a match to our Go field "k"
		match := false
		for _, className := range classNames {
			if className == k {
				match = true
				break
			}
		}

		// If no match found, log the fact that the WMI class didn't return a property specified
		// in the Go object.  We also set the default return value (if provided in WMI tags).
		if !match {
			if v.nilValue != nil {
				f := goObjectValue.Field(v.index)
				f.Set(reflect.ValueOf(v.nilValue))
			}
			log.Tracef(`Field "%v" defined in Go object but not supported by WMI on this host, nilValue=%v`, k, v.nilValue)
		}
	}

	// Enumerate each class property
	for _, classProperty := range classNames {

		// Knowing the WMI class property, get the Go field details
		fieldInfo, ok := fieldMap[classProperty]
		if !ok {
			// If there is no Go field definition, for the enumerated WMI property, log as informational
			// so that we can add the property to the Go definition.
			log.Tracef(`Property "%v" returned by WMI but not defined in Go object`, classProperty)
			continue
		}

		// Get the next class property
		var cimType CIMTYPE_ENUMERATION
		var flavor uint32
		propertyUTF16 := syscall.StringToUTF16(classProperty)
		hres, _, _ := syscall.Syscall6(pClassVTable.Get, 6, // Call the IWbemClassObject::Get method
			uintptr(unsafe.Pointer(wmiClass)),
			uintptr(unsafe.Pointer(&propertyUTF16[0])), // LPCWSTR wszName - Name of the desired property.
			uintptr(0),                        // long    lFlags   - Reserved. This parameter must be 0 (zero).
			uintptr(unsafe.Pointer(&vtProp)),  // VARIANT *pVal    - Returned WMI class property (as variant)
			uintptr(unsafe.Pointer(&cimType)), // CIMTYPE *pType   - CIM type (i.e. CIMTYPE_ENUMERATION)
			uintptr(unsafe.Pointer(&flavor)))  // long    *plFlavor - Property origin (i.e. WBEM_FLAVOR_TYPE)
		if FAILED(hres) {
			err = ole.NewError(hres)
			log.Errorf("Unable to query WMI class property value, classProperty=%v, err=%v", classProperty, err)
			return err
		}

		// Convert the WMI variant into a Go object
		propertyValue, err := wmiVariantToGoObject(classProperty, &vtProp, cimType, fieldInfo)
		ole.VariantClear(&vtProp)
		if err != nil {
			log.Errorf("Failed unmarshalling WMI variant into Go object, classProperty=%v, VT=%v, cimType=%v, fieldInfo=%v, err=%v", classProperty, vtProp.VT, cimType, fieldInfo, err)
			return err
		}

		// If a nil value was returned from wmiVariantToGoObject, and a default value was specified
		// in the WMI tags, use the tag property for the field's value.
		if (propertyValue == nil) && (fieldInfo.nilValue != nil) {
			propertyValue = fieldInfo.nilValue
		}

		// Set the Go field's value
		if propertyValue != nil {
			f := goObjectValue.Field(fieldInfo.index)
			kindWMI := reflect.TypeOf(propertyValue).Kind()
			if kindWMI != fieldInfo.fieldKind {
				// If the field kinds do not match, log an error!!!  The Go object must not have been
				// defined properly.
				log.Errorf("WMI/Go kind mismatch, classProperty=%v, WMI=%v, Go=%v", classProperty, kindWMI, fieldInfo.fieldKind)
			} else {
				f.Set(reflect.ValueOf(propertyValue))
			}
		}
	}

	return nil
}

// interfaceToFieldMap takes a pointer to a struct, traverse the struct, and populates a map with
// details about each field.  The map key is field name while the map value contains details about
// that struct field.
func interfaceToFieldMap(ptrStruct interface{}) (mapStruct map[string]interfaceFieldInfo, err error) {

	// If we were not given a pointer to a struct, fail the request
	dstPath, _ := getInterfaceType(ptrStruct)
	if dstPath != "ptr.struct" {
		log.Errorf("Invalid interface object, dstPath=%v", dstPath)
		return nil, windows.ERROR_INVALID_PARAMETER
	}

	// Allocate an empty map to start
	mapStruct = make(map[string]interfaceFieldInfo)

	// Get the structure type and enumerate each structure field
	t := reflect.TypeOf(ptrStruct).Elem()
	for i := 0; i < t.NumField(); i++ {

		// Get the structure field (i.e. reflect.StructField) and its field name
		f := t.Field(i)
		name := f.Name

		// Initialize a interfaceFieldInfo struct for the current field
		var fieldData interfaceFieldInfo
		fieldData.index = i
		fieldData.fieldType = f.Type
		fieldData.fieldKind = f.Type.Kind()

		// See if are wmi tags attached to this field
		wmiTag := f.Tag.Get("wmi")
		if wmiTag != "" {

			// Split the comma separated tags
			wmiTags := strings.Split(wmiTag, ",")

			// Field name value set?
			if len(wmiTags) >= 1 {
				if wmiTags[0] == "-" {
					// Ignore this field
					continue
				} else if wmiTags[0] != "" {
					// Go field name doesn't align with WMI field name.  Get WMI field name from tag.
					name = wmiTags[0]
				}
			}

			// Field null default value set?
			if len(wmiTags) >= 2 {
				// Split the default value
				overrides := strings.Split(wmiTags[1], "=")
				switch overrides[0] {
				case "nil":
					if len(overrides) > 1 {
						// Override the WMI null value with the specified value?
						fieldData.nilValue, err = stringToObject(overrides[1], f.Type.Kind())
					}
				default:
					// Invalid wmi attribute
					log.Errorf("Invalid WMI value setting, wmiTag=%v", wmiTag)
					return nil, windows.ERROR_INVALID_PARAMETER
				}
			}
		}

		// Add the field data structure to the key/value map
		mapStruct[name] = fieldData
	}

	// Return the fully enumerated key/value map to the caller
	return mapStruct, nil
}

// stringToObject takes the given string, of the given kind, and converts it into an interface
// which is then returned to the caller.  An error is returned if an unsuppoprted string and/or
// kind was passed in.
func stringToObject(valueText string, valueKind reflect.Kind) (v interface{}, err error) {

	var intValue int64
	var uintValue uint64
	var floatValue float64

	switch valueKind {

	// Already a string, no conversion required
	case reflect.String:
		v, err = valueText, nil

	// Convert boolean text
	case reflect.Bool:
		v, err = strconv.ParseBool(valueText)

	// Convert signed integer text
	case reflect.Int:
		intValue, err = strconv.ParseInt(valueText, 0, 64)
		v = int(intValue)
	case reflect.Int8:
		intValue, err = strconv.ParseInt(valueText, 0, 8)
		v = int8(intValue)
	case reflect.Int16:
		intValue, err = strconv.ParseInt(valueText, 0, 16)
		v = int16(intValue)
	case reflect.Int32:
		intValue, err = strconv.ParseInt(valueText, 0, 32)
		v = int32(intValue)
	case reflect.Int64:
		intValue, err = strconv.ParseInt(valueText, 0, 64)
		v = int64(intValue)

	// Convert unsigned integer text
	case reflect.Uint:
		uintValue, err = strconv.ParseUint(valueText, 0, 64)
		v = uint(uintValue)
	case reflect.Uint8:
		uintValue, err = strconv.ParseUint(valueText, 0, 8)
		v = uint8(uintValue)
	case reflect.Uint16:
		uintValue, err = strconv.ParseUint(valueText, 0, 16)
		v = uint16(uintValue)
	case reflect.Uint32:
		uintValue, err = strconv.ParseUint(valueText, 0, 32)
		v = uint32(uintValue)
	case reflect.Uint64:
		uintValue, err = strconv.ParseUint(valueText, 0, 64)
		v = uint64(uintValue)

	// Convert float text
	case reflect.Float32:
		floatValue, err = strconv.ParseFloat(valueText, 32)
		v = float32(floatValue)
	case reflect.Float64:
		floatValue, err = strconv.ParseFloat(valueText, 32)
		v = float64(floatValue)

	// Unsupported data type
	default:
		err = windows.ERROR_INVALID_PARAMETER
	}

	// Log an error if an unsupported input type was passed in
	if err != nil {
		log.Errorf("Unsupported stringToObject input values, valueText=%v, valueKind=%v", valueText, valueKind)
	}

	return v, err
}

// numberToObject takes the given number, of the given kind, and converts it into an interface
// which is then returned to the caller.  An error is returned if an unsuppoprted kind was
// passed in.  An "int64" is used as input because that is the VARIANT.Val type.
func numberToObject(valueNumber int64, valueKind reflect.Kind) (v interface{}, err error) {

	switch valueKind {

	// Convert boolean
	case reflect.Bool:
		v = (valueNumber != 0)

	// Signed integers
	case reflect.Int8:
		v = int8(valueNumber)
	case reflect.Int16:
		v = int16(valueNumber)
	case reflect.Int32:
		v = int32(valueNumber)
	case reflect.Int64:
		v = int64(valueNumber)

	// Unsigned integers
	case reflect.Uint8:
		v = uint8(valueNumber)
	case reflect.Uint16:
		v = uint16(valueNumber)
	case reflect.Uint32:
		v = uint32(valueNumber)
	case reflect.Uint64:
		v = uint64(valueNumber)

	// Unsupported data type
	default:
		err = windows.ERROR_INVALID_PARAMETER
	}

	// Log an error if an unsupported input type was passed in
	if err != nil {
		log.Errorf("Unsupported numberToObject input values, valueNumber=%v, valueKind=%v", valueNumber, valueKind)
	}

	return v, err
}

// wmiVariantToGoObject takes an enumerated WMI VARIANT and converts it into the Go object type
// specified by the fieldData parameter.
func wmiVariantToGoObject(classProperty string, vtProp *ole.VARIANT, cimType CIMTYPE_ENUMERATION, fieldData interfaceFieldInfo) (v interface{}, err error) {

	switch vtProp.VT {
	case ole.VT_NULL:
		// If it's a NULL variant, return nil
		v = nil

	case ole.VT_BSTR, ole.VT_BOOL, ole.VT_UI1, ole.VT_UI2, ole.VT_UI4, ole.VT_UI8, ole.VT_I1, ole.VT_I2, ole.VT_I4, ole.VT_I8:
		// Convert variant string, boolean, or integer value to interface object
		cimGoType, ok := cimTypeToGoType[cimType]
		if !ok || (cimGoType == nil) {
			log.Errorf("Unsupported CIM type conversion, classProperty=%v, cimType=%v, cimGoType=%v, ok=%v", classProperty, cimType, cimGoType, ok)
			return nil, windows.ERROR_INVALID_PARAMETER
		}
		if vtProp.VT == ole.VT_BSTR {
			v, err = stringToObject(vtProp.ToString(), cimGoType.Kind())
		} else {
			v, err = numberToObject(vtProp.Val, cimGoType.Kind())
		}

	case ole.VT_ARRAY + ole.VT_BSTR:
		// Convert variant string array to interface object
		safeArrayConversion := vtProp.ToArray()
		if safeArrayConversion == nil {
			log.Errorf("Invalid variant string array, classProperty=%v, VT=%v, Val=%v", classProperty, vtProp.VT, vtProp.Val)
			return nil, windows.ERROR_INVALID_PARAMETER
		}
		v = safeArrayConversion.ToStringArray()

	case ole.VT_ARRAY + ole.VT_UI1, ole.VT_ARRAY + ole.VT_UI2, ole.VT_ARRAY + ole.VT_UI4, ole.VT_ARRAY + ole.VT_UI8:
		fallthrough
	case ole.VT_ARRAY + ole.VT_I1, ole.VT_ARRAY + ole.VT_I2, ole.VT_ARRAY + ole.VT_I4, ole.VT_ARRAY + ole.VT_I8:
		// Convert variant number array to interface object

		// Invalid data if the CIM type isn't an array, or the Go field isn't a slice or array
		if (cimType&CIM_FLAG_ARRAY) == 0 || ((fieldData.fieldKind != reflect.Slice) && (fieldData.fieldKind != reflect.Array)) {
			log.Errorf("Invalid array details, classProperty=%v, cimType=%v, fieldKind=%v", classProperty, cimType, fieldData.fieldKind)
			return nil, windows.ERROR_INVALID_PARAMETER
		}

		// Convert the CIM type to its equivalent Go type
		cimGoType, ok := cimTypeToGoType[cimType&^CIM_FLAG_ARRAY]
		if !ok || (cimGoType == nil) {
			log.Errorf("Unsupported CIM type array conversion, classProperty=%v, cimType=%v, cimGoType=%v, ok=%v", classProperty, cimType, cimGoType, ok)
			return nil, windows.ERROR_INVALID_PARAMETER
		}

		// If the slice/array kind doesn't match the enumerated CIM data, fail the request
		goKind := fieldData.fieldType.Elem().Kind()
		if goKind != cimGoType.Kind() {
			log.Errorf("Mismatched array types, classProperty=%v, goKind=%v, cimGoType=%v", classProperty, goKind, cimGoType)
			return nil, windows.ERROR_INVALID_PARAMETER
		}

		// Convert the safe array to an interface array
		valueArray := vtProp.ToArray().ToValueArray()

		// Make a new slice matching the destination slice type
		valueSlice := reflect.MakeSlice(reflect.SliceOf(cimGoType), 0, 0)

		// Convert the interface array into our properly typed slice
		for _, valueArr := range valueArray {
			var result interface{}
			result, err = stringToObject(fmt.Sprintf("%v", valueArr), cimGoType.Kind())
			if err != nil {
				return nil, err
			}
			valueSlice = reflect.Append(valueSlice, reflect.ValueOf(result))
		}

		if fieldData.fieldKind == reflect.Slice {
			// If the destination is a slice, we simply return our enumerated slice
			v = valueSlice.Interface()
		} else if fieldData.fieldKind == reflect.Array {
			valueArray := reflect.New(fieldData.fieldType)
			valueArrayElem := valueArray.Elem()

			// If the destination array length doesn't match the source array length, we don't
			// return an error to the caller but we'll log an error.
			if valueArrayElem.Len() != valueSlice.Len() {
				log.Errorf("WMI array length mismatch, classProperty=%v, srcLen=%v, dstlen=%v", classProperty, valueSlice.Len(), valueArrayElem.Len())
			}

			// Copy from slice into our array
			copied := reflect.Copy(valueArrayElem, valueSlice)

			// If destination array wasn't filled, we don't return an error to the caller but we'll
			// log an error.
			if copied != valueArrayElem.Len() {
				log.Errorf("WMI array copy mismatch, classProperty=%v, copied=%v, dstLen=%v", classProperty, copied, valueArrayElem.Len())
			}

			// Return enumerated array
			v = valueArrayElem.Interface()
		} else {
			// Should never get here
			err = windows.ERROR_INVALID_PARAMETER
		}

	case ole.VT_UNKNOWN:
		// Convert WMI object to a single Go object

		// Start by getting the WMI class object
		wmiObject := (*ole.IUnknown)(unsafe.Pointer(uintptr(vtProp.Val)))

		// We only support a destination struct pointer
		if fieldData.fieldKind != reflect.Ptr {
			log.Errorf("Unsupported destination object, reflect.Ptr expected, classProperty=%v, dstKind=%v", classProperty, fieldData.fieldKind)
			return nil, windows.ERROR_INVALID_PARAMETER
		}

		// Allocate a new structure for this WMI class
		goObject := reflect.New(fieldData.fieldType.Elem())

		// Convert the WMI class to its equivalent Go object and return it to the caller
		err = wmiClassToGoObject(wmiObject, goObject.Interface(), classProperty)
		if err != nil {
			return nil, err
		}
		v = goObject.Interface()

	case ole.VT_ARRAY + ole.VT_UNKNOWN:
		// Convert the array of WMI objects to a slice of Go objects

		// We only support a destination slice
		if fieldData.fieldKind != reflect.Slice {
			log.Errorf("Unsupported destination object for array of WMI objects, classProperty=%v, dstKind=%v", classProperty, fieldData.fieldKind)
			return nil, windows.ERROR_INVALID_PARAMETER
		}

		// Make a slice to store our Go objects converted from WMI objects
		valueSlice := reflect.MakeSlice(fieldData.fieldType, 0, 0)

		// Retrieve the array of IUnknown COM objects
		safeArrayConversion := vtProp.ToArray()
		iUnknownArray := toIUnknownArray(safeArrayConversion)

		// Enumerate each WMI object
		for _, iUnknownObject := range iUnknownArray {

			// Allocate a new object and unmarshall WMI object into Go object
			goObject := reflect.New(fieldData.fieldType.Elem().Elem())
			wmiClassToGoObject(iUnknownObject, goObject.Interface(), classProperty)

			// Release the current COM object as it is no longer needed
			iUnknownObject.Release()

			// Append Go oject to return slice
			valueSlice = reflect.Append(valueSlice, goObject)
		}

		// Return slice to the caller
		v = valueSlice.Interface()

	default:
		// Log an error for any unsupported variants
		log.Errorf("Unsupported Variant/CIM type, classProperty=%v, VT=%v, cimType=%v", classProperty, vtProp.VT, cimType)
		return nil, windows.ERROR_INVALID_PARAMETER
	}

	return v, err
}

// toIUnknownArray is similar to the "(sac *SafeArrayConversion) ToValueArray()" function except this
//  one is designed to specifically extract an array of IUnknown COM objects.
func toIUnknownArray(sac *ole.SafeArrayConversion) (iUnknownArray []*ole.IUnknown) {
	totalElements, _ := sac.TotalElements(0)
	iUnknownArray = make([]*ole.IUnknown, totalElements)

	for i := int32(0); i < totalElements; i++ {
		var pv *ole.IUnknown
		procSafeArrayGetElement.Call(
			uintptr(unsafe.Pointer(sac.Array)),
			uintptr(unsafe.Pointer(&i)),
			uintptr(unsafe.Pointer(&pv)))

		iUnknownArray[i] = pv
	}

	valueArray := sac.ToValueArray()
	_ = valueArray

	return iUnknownArray
}
