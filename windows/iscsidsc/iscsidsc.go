// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package iscsidsc wraps the Windows iSCSI Discovery Library A
package iscsidsc

import (
	"encoding/hex"
	"math"
	"strings"
	"syscall"
	"unsafe"

	log "github.com/hpe-storage/common-host-libs/logger"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

const (
	logIscsiFailure = "iSCSI API error: "
)

// iSCSI definitions
const (
	ISCSI_ALL_INITIATOR_PORTS    = uint32(math.MaxUint32)
	ISCSI_ANY_INITIATOR_PORT     = uint32(math.MaxUint32)
	MAX_ISCSI_ALIAS_LEN          = 255
	MAX_ISCSI_HBANAME_LEN        = 256
	MAX_ISCSI_NAME_LEN           = 223
	MAX_ISCSI_PORTAL_ADDRESS_LEN = MAX_ISCSI_TEXT_ADDRESS_LEN
	MAX_ISCSI_PORTAL_ALIAS_LEN   = 256
	MAX_ISCSI_PORTAL_NAME_LEN    = 256
	MAX_ISCSI_TEXT_ADDRESS_LEN   = 256
	MAX_PATH                     = 260
)

// SCSI target status
const (
	SCSISTAT_GOOD                  = 0x00
	SCSISTAT_CHECK_CONDITION       = 0x02
	SCSISTAT_CONDITION_MET         = 0x04
	SCSISTAT_BUSY                  = 0x08
	SCSISTAT_INTERMEDIATE          = 0x10
	SCSISTAT_INTERMEDIATE_COND_MET = 0x14
	SCSISTAT_RESERVATION_CONFLICT  = 0x18
	SCSISTAT_COMMAND_TERMINATED    = 0x22
	SCSISTAT_QUEUE_FULL            = 0x28
)

// Win32 error codes
const (
	ERROR_SUCCESS             = 0
	ERROR_FILE_NOT_FOUND      = 2
	ERROR_INVALID_PARAMETER   = 87
	ERROR_INSUFFICIENT_BUFFER = 122
	ERROR_MORE_DATA           = 234
	ERROR_TIMEOUT             = 1460
	ERROR_FAIL                = 0xEEEEEEEE // Nimble defined error code
)

// ISCSI_LOGIN_OPTIONS_INFO_SPECIFIED
const (
	ISCSI_LOGIN_OPTIONS_HEADER_DIGEST         = 0x00000001
	ISCSI_LOGIN_OPTIONS_DATA_DIGEST           = 0x00000002
	ISCSI_LOGIN_OPTIONS_MAXIMUM_CONNECTIONS   = 0x00000004
	ISCSI_LOGIN_OPTIONS_DEFAULT_TIME_2_WAIT   = 0x00000008
	ISCSI_LOGIN_OPTIONS_DEFAULT_TIME_2_RETAIN = 0x00000010
	ISCSI_LOGIN_OPTIONS_USERNAME              = 0x00000020
	ISCSI_LOGIN_OPTIONS_PASSWORD              = 0x00000040
	ISCSI_LOGIN_OPTIONS_AUTH_TYPE             = 0x00000080
)

type ISCSI_LOGIN_OPTIONS_INFO_SPECIFIED uint32

// ISCSI_LOGIN_FLAGS
const (
	ISCSI_LOGIN_FLAG_REQUIRE_IPSEC           = 0x00000001
	ISCSI_LOGIN_FLAG_MULTIPATH_ENABLED       = 0x00000002
	ISCSI_LOGIN_FLAG_RESERVED1               = 0x00000004
	ISCSI_LOGIN_FLAG_ALLOW_PORTAL_HOPPING    = 0x00000008
	ISCSI_LOGIN_FLAG_USE_RADIUS_RESPONSE     = 0x00000010
	ISCSI_LOGIN_FLAG_USE_RADIUS_VERIFICATION = 0x00000020
)

type ISCSI_LOGIN_FLAGS uint32

// ISCSI_AUTH_TYPES
const (
	ISCSI_NO_AUTH_TYPE          = 0
	ISCSI_CHAP_AUTH_TYPE        = 1
	ISCSI_MUTUAL_CHAP_AUTH_TYPE = 2
)

type ISCSI_AUTH_TYPES uint32

// ISCSI_DIGEST_TYPES
const (
	ISCSI_DIGEST_TYPE_NONE   = 0
	ISCSI_DIGEST_TYPE_CRC32C = 1
)

type ISCSI_DIGEST_TYPES uint32

const (
	ISCSI_LOGIN_OPTIONS_VERSION = 0
)

// iSCSI error codes
const (
	ISDSC_NON_SPECIFIC_ERROR                     = 0xEFFF0001
	ISDSC_LOGIN_FAILED                           = 0xEFFF0002
	ISDSC_CONNECTION_FAILED                      = 0xEFFF0003
	ISDSC_INITIATOR_NODE_ALREADY_EXISTS          = 0xEFFF0004
	ISDSC_INITIATOR_NODE_NOT_FOUND               = 0xEFFF0005
	ISDSC_TARGET_MOVED_TEMPORARILY               = 0xEFFF0006
	ISDSC_TARGET_MOVED_PERMANENTLY               = 0xEFFF0007
	ISDSC_INITIATOR_ERROR                        = 0xEFFF0008
	ISDSC_AUTHENTICATION_FAILURE                 = 0xEFFF0009
	ISDSC_AUTHORIZATION_FAILURE                  = 0xEFFF000A
	ISDSC_NOT_FOUND                              = 0xEFFF000B
	ISDSC_TARGET_REMOVED                         = 0xEFFF000C
	ISDSC_UNSUPPORTED_VERSION                    = 0xEFFF000D
	ISDSC_TOO_MANY_CONNECTIONS                   = 0xEFFF000E
	ISDSC_MISSING_PARAMETER                      = 0xEFFF000F
	ISDSC_CANT_INCLUDE_IN_SESSION                = 0xEFFF0010
	ISDSC_SESSION_TYPE_NOT_SUPPORTED             = 0xEFFF0011
	ISDSC_TARGET_ERROR                           = 0xEFFF0012
	ISDSC_SERVICE_UNAVAILABLE                    = 0xEFFF0013
	ISDSC_OUT_OF_RESOURCES                       = 0xEFFF0014
	ISDSC_CONNECTION_ALREADY_EXISTS              = 0xEFFF0015
	ISDSC_SESSION_ALREADY_EXISTS                 = 0xEFFF0016
	ISDSC_INITIATOR_INSTANCE_NOT_FOUND           = 0xEFFF0017
	ISDSC_TARGET_ALREADY_EXISTS                  = 0xEFFF0018
	ISDSC_DRIVER_BUG                             = 0xEFFF0019
	ISDSC_INVALID_TEXT_KEY                       = 0xEFFF001A
	ISDSC_INVALID_SENDTARGETS_TEXT               = 0xEFFF001B
	ISDSC_INVALID_SESSION_ID                     = 0xEFFF001C
	ISDSC_SCSI_REQUEST_FAILED                    = 0xEFFF001D
	ISDSC_TOO_MANY_SESSIONS                      = 0xEFFF001E
	ISDSC_SESSION_BUSY                           = 0xEFFF001F
	ISDSC_TARGET_MAPPING_UNAVAILABLE             = 0xEFFF0020
	ISDSC_ADDRESS_TYPE_NOT_SUPPORTED             = 0xEFFF0021
	ISDSC_LOGON_FAILED                           = 0xEFFF0022
	ISDSC_SEND_FAILED                            = 0xEFFF0023
	ISDSC_TRANSPORT_ERROR                        = 0xEFFF0024
	ISDSC_VERSION_MISMATCH                       = 0xEFFF0025
	ISDSC_TARGET_MAPPING_OUT_OF_RANGE            = 0xEFFF0026
	ISDSC_TARGET_PRESHAREDKEY_UNAVAILABLE        = 0xEFFF0027
	ISDSC_TARGET_AUTHINFO_UNAVAILABLE            = 0xEFFF0028
	ISDSC_TARGET_NOT_FOUND                       = 0xEFFF0029
	ISDSC_LOGIN_USER_INFO_BAD                    = 0xEFFF002A
	ISDSC_TARGET_MAPPING_EXISTS                  = 0xEFFF002B
	ISDSC_HBA_SECURITY_CACHE_FULL                = 0xEFFF002C
	ISDSC_INVALID_PORT_NUMBER                    = 0xEFFF002D
	ISDSC_OPERATION_NOT_ALL_SUCCESS              = 0xAFFF002E
	ISDSC_HBA_SECURITY_CACHE_NOT_SUPPORTED       = 0xEFFF002F
	ISDSC_IKE_ID_PAYLOAD_TYPE_NOT_SUPPORTED      = 0xEFFF0030
	ISDSC_IKE_ID_PAYLOAD_INCORRECT_SIZE          = 0xEFFF0031
	ISDSC_TARGET_PORTAL_ALREADY_EXISTS           = 0xEFFF0032
	ISDSC_TARGET_ADDRESS_ALREADY_EXISTS          = 0xEFFF0033
	ISDSC_NO_AUTH_INFO_AVAILABLE                 = 0xEFFF0034
	ISDSC_NO_TUNNEL_OUTER_MODE_ADDRESS           = 0xEFFF0035
	ISDSC_CACHE_CORRUPTED                        = 0xEFFF0036
	ISDSC_REQUEST_NOT_SUPPORTED                  = 0xEFFF0037
	ISDSC_TARGET_OUT_OF_RESORCES                 = 0xEFFF0038
	ISDSC_SERVICE_DID_NOT_RESPOND                = 0xEFFF0039
	ISDSC_ISNS_SERVER_NOT_FOUND                  = 0xEFFF003A
	ISDSC_OPERATION_REQUIRES_REBOOT              = 0xAFFF003B
	ISDSC_NO_PORTAL_SPECIFIED                    = 0xEFFF003C
	ISDSC_CANT_REMOVE_LAST_CONNECTION            = 0xEFFF003D
	ISDSC_SERVICE_NOT_RUNNING                    = 0xEFFF003E
	ISDSC_TARGET_ALREADY_LOGGED_IN               = 0xEFFF003F
	ISDSC_DEVICE_BUSY_ON_SESSION                 = 0xEFFF0040
	ISDSC_COULD_NOT_SAVE_PERSISTENT_LOGIN_DATA   = 0xEFFF0041
	ISDSC_COULD_NOT_REMOVE_PERSISTENT_LOGIN_DATA = 0xEFFF0042
	ISDSC_PORTAL_NOT_FOUND                       = 0xEFFF0043
	ISDSC_INITIATOR_NOT_FOUND                    = 0xEFFF0044
	ISDSC_DISCOVERY_MECHANISM_NOT_FOUND          = 0xEFFF0045
	ISDSC_IPSEC_NOT_SUPPORTED_ON_OS              = 0xEFFF0046
	ISDSC_PERSISTENT_LOGIN_TIMEOUT               = 0xEFFF0047
	ISDSC_SHORT_CHAP_SECRET                      = 0xAFFF0048
	ISDSC_EVALUATION_PEROID_EXPIRED              = 0xEFFF0049
	ISDSC_INVALID_CHAP_SECRET                    = 0xEFFF004A
	ISDSC_INVALID_TARGET_CHAP_SECRET             = 0xEFFF004B
	ISDSC_INVALID_INITIATOR_CHAP_SECRET          = 0xEFFF004C
	ISDSC_INVALID_CHAP_USER_NAME                 = 0xEFFF004D
	ISDSC_INVALID_LOGON_AUTH_TYPE                = 0xEFFF004E
	ISDSC_INVALID_TARGET_MAPPING                 = 0xEFFF004F
	ISDSC_INVALID_TARGET_ID                      = 0xEFFF0050
	ISDSC_INVALID_ISCSI_NAME                     = 0xEFFF0051
	ISDSC_INCOMPATIBLE_ISNS_VERSION              = 0xEFFF0052
	ISDSC_FAILED_TO_CONFIGURE_IPSEC              = 0xEFFF0053
	ISDSC_BUFFER_TOO_SMALL                       = 0xEFFF0054
	ISDSC_INVALID_LOAD_BALANCE_POLICY            = 0xEFFF0055
	ISDSC_INVALID_PARAMETER                      = 0xEFFF0056
	ISDSC_DUPLICATE_PATH_SPECIFIED               = 0xEFFF0057
	ISDSC_PATH_COUNT_MISMATCH                    = 0xEFFF0058
	ISDSC_INVALID_PATH_ID                        = 0xEFFF0059
	ISDSC_MULTIPLE_PRIMARY_PATHS_SPECIFIED       = 0xEFFF005A
	ISDSC_NO_PRIMARY_PATH_SPECIFIED              = 0xEFFF005B
	ISDSC_DEVICE_ALREADY_PERSISTENTLY_BOUND      = 0xEFFF005C
	ISDSC_DEVICE_NOT_FOUND                       = 0xEFFF005D
	ISDSC_DEVICE_NOT_ISCSI_OR_PERSISTENT         = 0xEFFF005E
	ISDSC_DNS_NAME_UNRESOLVED                    = 0xEFFF005F
	ISDSC_NO_CONNECTION_AVAILABLE                = 0xEFFF0060
	ISDSC_LB_POLICY_NOT_SUPPORTED                = 0xEFFF0061
	ISDSC_REMOVE_CONNECTION_IN_PROGRESS          = 0xEFFF0062
	ISDSC_INVALID_CONNECTION_ID                  = 0xEFFF0063
	ISDSC_CANNOT_REMOVE_LEADING_CONNECTION       = 0xEFFF0064
	ISDSC_RESTRICTED_BY_GROUP_POLICY             = 0xEFFF0065
	ISDSC_ISNS_FIREWALL_BLOCKED                  = 0xEFFF0066
	ISDSC_FAILURE_TO_PERSIST_LB_POLICY           = 0xEFFF0067
	ISDSC_INVALID_HOST                           = 0xEFFF0068
)

// Lazy load the iSCSI DLL APIs
var (
	iscsidsc                             = windows.NewLazySystemDLL("iscsidsc.dll")
	procAddIScsiSendTargetPortalW        = iscsidsc.NewProc("AddIScsiSendTargetPortalW")
	procGetDevicesForIScsiSessionW       = iscsidsc.NewProc("GetDevicesForIScsiSessionW")
	procGetIScsiInitiatorNodeNameW       = iscsidsc.NewProc("GetIScsiInitiatorNodeNameW")
	procGetIScsiSessionListW             = iscsidsc.NewProc("GetIScsiSessionListW")
	procGetIScsiVersionInformation       = iscsidsc.NewProc("GetIScsiVersionInformation")
	procLoginIScsiTargetW                = iscsidsc.NewProc("LoginIScsiTargetW")
	procLogoutIScsiTarget                = iscsidsc.NewProc("LogoutIScsiTarget")
	procRemoveIScsiPersistentTargetW     = iscsidsc.NewProc("RemoveIScsiPersistentTargetW")
	procReportActiveIScsiTargetMappingsW = iscsidsc.NewProc("ReportActiveIScsiTargetMappingsW")
	procReportIScsiPersistentLoginsW     = iscsidsc.NewProc("ReportIScsiPersistentLoginsW")
	procReportIScsiSendTargetPortalsExW  = iscsidsc.NewProc("ReportIScsiSendTargetPortalsExW")
	procReportIScsiSendTargetPortalsW    = iscsidsc.NewProc("ReportIScsiSendTargetPortalsW")
	procReportIScsiTargetPortalsW        = iscsidsc.NewProc("ReportIScsiTargetPortalsW")
	procReportIScsiTargetsW              = iscsidsc.NewProc("ReportIScsiTargetsW")
	procSendScsiInquiry                  = iscsidsc.NewProc("SendScsiInquiry")
)

// ISCSI_CONNECTION_INFO (Wrapped version)
type ISCSI_CONNECTION_INFO struct {
	ConnectionID     ISCSI_UNIQUE_CONNECTION_ID
	InitiatorAddress string
	TargetAddress    string
	InitiatorSocket  uint16
	TargetSocket     uint16
	CID              [2]uint8
}

// ISCSI_CONNECTION_INFO_RAW (Raw version)
type ISCSI_CONNECTION_INFO_RAW struct {
	ConnectionID     ISCSI_UNIQUE_CONNECTION_ID
	InitiatorAddress *uint16
	TargetAddress    *uint16
	InitiatorSocket  uint16
	TargetSocket     uint16
	CID              [2]uint8
}

// ISCSI_DEVICE_ON_SESSION (Wrapped version)
type ISCSI_DEVICE_ON_SESSION struct {
	InitiatorName       string
	TargetName          string
	ScsiAddress         SCSI_ADDRESS
	DeviceInterfaceType uuid.UUID
	DeviceInterfaceName string
	LegacyName          string
	StorageDeviceNumber STORAGE_DEVICE_NUMBER
	DeviceInstance      uint32
}

// ISCSI_DEVICE_ON_SESSION_RAW (Raw version)
type ISCSI_DEVICE_ON_SESSION_RAW struct {
	InitiatorName       [MAX_ISCSI_HBANAME_LEN]uint16
	TargetName          [MAX_ISCSI_NAME_LEN + 1]uint16
	ScsiAddress         SCSI_ADDRESS
	DeviceInterfaceType uuid.UUID
	DeviceInterfaceName [MAX_PATH]uint16
	LegacyName          [MAX_PATH]uint16
	StorageDeviceNumber STORAGE_DEVICE_NUMBER
	DeviceInstance      uint32
}

// ISCSI_LOGIN_OPTIONS
type ISCSI_LOGIN_OPTIONS struct {
	Version              uint32
	InformationSpecified ISCSI_LOGIN_OPTIONS_INFO_SPECIFIED
	LoginFlags           ISCSI_LOGIN_FLAGS
	AuthType             ISCSI_AUTH_TYPES
	HeaderDigest         ISCSI_DIGEST_TYPES
	DataDigest           ISCSI_DIGEST_TYPES
	MaximumConnections   uint32
	DefaultTime2Wait     uint32
	DefaultTime2Retain   uint32
	UsernameLength       uint32
	PasswordLength       uint32
	Username             uintptr
	Password             uintptr
}

// SCSI_LUN_LIST
type SCSI_LUN_LIST struct {
	OSLUN     uint32
	TargetLUN uint64
}

// ISCSI_SESSION_INFO
type ISCSI_SESSION_INFO struct {
	SessionID      ISCSI_UNIQUE_SESSION_ID
	InitiatorName  string
	TargetNodeName string
	TargetName     string
	ISID           [6]uint8
	TSID           [2]uint8
	Connections    []*ISCSI_CONNECTION_INFO
}

// ISCSI_SESSION_INFO_RAW
type ISCSI_SESSION_INFO_RAW struct {
	SessionID       ISCSI_UNIQUE_SESSION_ID
	InitiatorName   *uint16
	TargetNodeName  *uint16
	TargetName      *uint16
	ISID            [6]uint8
	TSID            [2]uint8
	ConnectionCount uint32
	Connections     uintptr
}

// ISCSI_TARGET_MAPPING
type ISCSI_TARGET_MAPPING struct {
	InitiatorName  string
	TargetName     string
	OSDeviceName   string
	SessionId      ISCSI_UNIQUE_SESSION_ID
	OSBusNumber    uint32
	OSTargetNumber uint32
	LUNList        []SCSI_LUN_LIST
}

// ISCSI_TARGET_MAPPING_RAW
type ISCSI_TARGET_MAPPING_RAW struct {
	InitiatorName  [MAX_ISCSI_HBANAME_LEN]uint16
	TargetName     [MAX_ISCSI_NAME_LEN + 1]uint16
	OSDeviceName   [MAX_PATH]uint16
	SessionId      ISCSI_UNIQUE_SESSION_ID
	OSBusNumber    uint32
	OSTargetNumber uint32
	LUNCount       uint32
	LUNList        uintptr
}

// ISCSI_TARGET_PORTAL
type ISCSI_TARGET_PORTAL struct {
	SymbolicName string
	Address      string
	Socket       uint16
}

// ISCSI_TARGET_PORTAL_RAW
type ISCSI_TARGET_PORTAL_RAW struct {
	SymbolicName [MAX_ISCSI_PORTAL_NAME_LEN]uint16
	Address      [MAX_ISCSI_PORTAL_ADDRESS_LEN]uint16
	Socket       uint16
}

// ISCSI_TARGET_PORTAL_INFO
type ISCSI_TARGET_PORTAL_INFO struct {
	InitiatorName       string
	InitiatorPortNumber uint32
	SymbolicName        string
	Address             string
	Socket              uint16
}

// ISCSI_TARGET_PORTAL_INFO_RAW
type ISCSI_TARGET_PORTAL_INFO_RAW struct {
	InitiatorName       [MAX_ISCSI_HBANAME_LEN]uint16
	InitiatorPortNumber uint32
	SymbolicName        [MAX_ISCSI_PORTAL_NAME_LEN]uint16
	Address             [MAX_ISCSI_PORTAL_ADDRESS_LEN]uint16
	Socket              uint16
}

// ISCSI_TARGET_PORTAL_INFO_EX
type ISCSI_TARGET_PORTAL_INFO_EX struct {
	InitiatorName       string
	InitiatorPortNumber uint32
	SymbolicName        string
	Address             string
	Socket              uint16
	SecurityFlags       uint64
	LoginOptions        ISCSI_LOGIN_OPTIONS
}

// ISCSI_TARGET_PORTAL_INFO_EX_RAW
type ISCSI_TARGET_PORTAL_INFO_EX_RAW struct {
	InitiatorName       [MAX_ISCSI_HBANAME_LEN]uint16
	InitiatorPortNumber uint32
	SymbolicName        [MAX_ISCSI_PORTAL_NAME_LEN]uint16
	Address             [MAX_ISCSI_PORTAL_ADDRESS_LEN]uint16
	Socket              uint16
	SecurityFlags       uint64
	LoginOptions        ISCSI_LOGIN_OPTIONS
}

// ISCSI_UNIQUE_CONNECTION_ID
type ISCSI_UNIQUE_CONNECTION_ID struct {
	AdapterUnique   uint64
	AdapterSpecific uint64
}

// ISCSI_UNIQUE_SESSION_ID
type ISCSI_UNIQUE_SESSION_ID struct {
	AdapterUnique   uint64
	AdapterSpecific uint64
}

// ISCSI_VERSION_INFO
type ISCSI_VERSION_INFO struct {
	MajorVersion uint32
	MinorVersion uint32
	BuildNumber  uint32
}

// PERSISTENT_ISCSI_LOGIN_INFO
type PERSISTENT_ISCSI_LOGIN_INFO struct {
	TargetName             string
	IsInformationalSession uint8
	InitiatorInstance      string
	InitiatorPortNumber    uint32
	TargetPortal           ISCSI_TARGET_PORTAL
	SecurityFlags          uint64
	Mappings               *ISCSI_TARGET_MAPPING
	LoginOptions           ISCSI_LOGIN_OPTIONS
}

// PERSISTENT_ISCSI_LOGIN_INFO_RAW
type PERSISTENT_ISCSI_LOGIN_INFO_RAW struct {
	TargetName             [MAX_ISCSI_NAME_LEN + 1]uint16
	IsInformationalSession uint8
	InitiatorInstance      [MAX_ISCSI_HBANAME_LEN]uint16
	InitiatorPortNumber    uint32
	TargetPortal           ISCSI_TARGET_PORTAL_RAW
	SecurityFlags          uint64
	Mappings               *ISCSI_TARGET_MAPPING_RAW
	LoginOptions           ISCSI_LOGIN_OPTIONS
}

// SCSI_ADDRESS
type SCSI_ADDRESS struct {
	Length     uint32
	PortNumber uint8
	PathID     uint8
	TargetID   uint8
	Lun        uint8
}

// STORAGE_DEVICE_NUMBER
type STORAGE_DEVICE_NUMBER struct {
	DeviceType      uint32
	DeviceNumber    uint32
	PartitionNumber uint32
}

// safeUTF16PtrToString takes the given NULL terminated pointer to a UTF16 string and returns it as a Go string
func safeUTF16PtrToString(ptr *uint16) (str string) {
	if ptr != nil {
		str = syscall.UTF16ToString((*[1024]uint16)(unsafe.Pointer(ptr))[:])
	}
	return str
}

// logTraceHexDump dumps the given hex buffer to the trace log (if enabled)
func logTraceHexDump(dataBuffer []uint8, prefix string) {
	if !log.IsLevelEnabled(logrus.TraceLevel) {
		return
	}
	if prefix != "" {
		prefix += " - "
	}
	hexStrings := strings.Split(strings.TrimRight(hex.Dump(dataBuffer), "\r\n"), "\n")
	for _, hexString := range hexStrings {
		log.Tracef("%v%v", prefix, hexString)
	}
}
