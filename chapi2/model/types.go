// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package model

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// This model package *only* defines the CHAPI objects and properties that are passed between CHAPI
// clients and servers.  It replaces the types.go file in the common\model package.  Below you will
// find important information about these CHAPI2 objects.
//
// CASE CONSIDERATIONS
//
//		CHAPI1 has most JSON object properties in lower case but some in upper case.  Where
//		applicable, JSON properties are now all lower case.  For example, Mac becomes mac
//		and Mtu becomes mtu in the Network object.
//
// ADDITIONAL CHANGES
//
// 		Changed IscsiTarget Address/Port/Tag from a single element to an array of target portals
//		so that we can return all iSCSI target ports and not just the first one.
//
///////////////////////////////////////////////////////////////////////////////////////////////////

const (
	// AccessProtocolIscsi - iSCSI volume
	AccessProtocolIscsi = "iscsi"

	// AccessProtocolFC - Fibre Channel volume
	AccessProtocolFC = "fc"
)

const (
	// TargetScopeGroup - Multi-LUN capable target, Group Scoped Target (GST)
	TargetScopeGroup = "group" // Group Scoped Target (GST)

	// TargetScopeVolume - Single LUN capable target, Volume Scoped Target (VST)
	TargetScopeVolume = "volume" // Volume Scoped Target (VST)
)

const (
	// ConnectTypeDefault - CHAPI2 will automatically detect and choose the optimal connection type.
	// This setting is also used if the connect type is not provided (e.g. empty string)
	ConnectTypeDefault = "default"

	// ConnectTypePing - Ping each I_T nexus to detect where connections are possible.
	ConnectTypePing = "ping"

	// ConnectTypeSubnet - Only make connections to initiator ports in same subnet as target ports.
	ConnectTypeSubnet = "subnet"

	// ConnectTypeAutoInitiator - Let the host's iSCSI initiator automatically select the initiator
	// to use to make a connection to the target ports.
	ConnectTypeAutoInitiator = "auto_initiator"
)

///////////////////////////////////////////////////////////////////////////////////////////////////
// CHAPI Host Object
///////////////////////////////////////////////////////////////////////////////////////////////////

// Host : Host information
type Host struct {
	UUID   string `json:"id,omitempty"`     // Unique host identifier
	Name   string `json:"name,omitempty"`   // Host name
	Domain string `json:"domain,omitempty"` // Host domain name
}

// Hosts returns an array of Host objects
type Hosts []*Host

///////////////////////////////////////////////////////////////////////////////////////////////////
// CHAPI Network Object
///////////////////////////////////////////////////////////////////////////////////////////////////

// Network : network interface info for host
type Network struct {
	Name      string          `json:"name,omitempty"`       // NIC name (e.g. "eth0" for Linux, "Ethernet 1" for Windows)
	AddressV4 string          `json:"address_v4,omitempty"` // NIC IPv4 address
	MaskV4    string          `json:"mask_v4,omitempty"`    // NIC subnet mask
	Mac       string          `json:"mac,omitempty"`        // NIC MAC address
	Mtu       int64           `json:"mtu,omitempty"`        // NIC Maximum Transmission Unit (MTU)
	Up        bool            `json:"up"`                   // NIC available?
	Private   *NetworkPrivate `json:"-"`                    // Private network properties used internally by CHAPI
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// CHAPI Initiator Object
///////////////////////////////////////////////////////////////////////////////////////////////////

// Initiator : Initiator details
type Initiator struct {
	AccessProtocol string   `json:"access_protocol,omitempty"` // Access protocol ("iscsi" or "fc")
	Init           []string `json:"initiator,omitempty"`       // Initiator iqn if AccessProtocol=="iscsi" else WWPNs if "fc"
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// CHAPI IscsiTarget Object
///////////////////////////////////////////////////////////////////////////////////////////////////

// IscsiTarget struct
type IscsiTarget struct {
	Name          string          `json:"name,omitempty"`           // Target iSCSI iqn
	TargetPortals []*TargetPortal `json:"target_portals,omitempty"` // Target portals
	TargetScope   string          `json:"target_scope,omitempty"`   // GST="group", VST="volume" or empty if unknown scope or FC
}

// TargetPortal provides information for a single iSCSI target portal (i.e. Data IP)
type TargetPortal struct {
	Address string               `json:"address,omitempty"` // Target port IP address
	Port    string               `json:"port,omitempty"`    // Target port socket
	Tag     string               `json:"tag,omitempty"`     // Target port tag
	Private *TargetPortalPrivate `json:"-"`                 // Private TargetPortal properties used internally by CHAPI
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// CHAPI Device Object
///////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: create fc and iscsi specific attributes
// Device struct
type Device struct {
	SerialNumber    string         `json:"serial_number,omitempty"`      // Nimble volume serial number
	Pathname        string         `json:"path_name,omitempty"`          // Path name (e.g. "dm-3" for Linux, "Disk3" for Windows)
	AltFullPathName string         `json:"alt_full_path_name,omitempty"` // Alternate path name (e.g. "/dev/mapper/mpathg" for Linux, "\\?\mpio#disk&ven_nimble&..." for Windows)
	Size            uint64         `json:"size,omitempty"`               // Volume capacity in total number of bytes //TODO ensure clients/servers change from MiB to byte count
	State           string         `json:"state,omitempty"`              // TODO, Shiva to define states
	IscsiTarget     *IscsiTarget   `json:"iscsi_target,omitempty"`       // Pointer to iSCSI target if device connected to an iSCSI target
	Private         *DevicePrivate `json:"-"`                            // Private device properties used internally by CHAPI
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// CHAPI Partition Object
///////////////////////////////////////////////////////////////////////////////////////////////////

// DevicePartition Partition Info for a Device
type DevicePartition struct {
	Name          string `json:"name,omitempty"`           // Partition name (e.g. "sda, mpathp1, mpathp2" for Linux, "Disk #1, Partition #0" for Windows)
	PartitionType string `json:"partition_type,omitempty"` // Partition type (e.g. "TODO" for Linux, "GPT: Basic Data" for Windows)
	Size          uint64 `json:"size,omitempty"`           // Partition size in total number of bytes
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// CHAPI PublishInfo Object
///////////////////////////////////////////////////////////////////////////////////////////////////

// PublishInfo is the node side data required to access a volume
type PublishInfo struct {
	SerialNumber string                   `json:"serial_number,omitempty"`
	BlockDev     *BlockDeviceAccessInfo   `json:"block_device,omitempty"`
	VirtualDev   *VirtualDeviceAccessInfo `json:"virtual_device,omitempty"`
}

// BlockDeviceAccessInfo contains the common fields for accessing a block device
type BlockDeviceAccessInfo struct {
	AccessProtocol  string           `json:"access_protocol,omitempty"` // Access protocol ("iscsi" or "fc")
	TargetName      string           `json:"target_name,omitempty"`     // Target name (iqn for iSCSI, empty for FC) - // TODO, clarify FC usage?
	TargetScope     string           `json:"target_scope,omitempty"`    // GST="group", VST="volume" or empty if unknown scope or FC
	LunID           string           `json:"lun_id,omitempty"`          // LunID is only used by Linux for rescan optimization and not used/required for Windows
	IscsiAccessInfo *IscsiAccessInfo `json:"iscsi_access_info,omitempty"`
}

// IscsiAccessInfo contains the fields necessary for iSCSI access
type IscsiAccessInfo struct {
	ConnectType  string `json:"connect_type,omitempty"`  // How connections should be enumerated/established
	DiscoveryIP  string `json:"discovery_ip,omitempty"`  // iSCSI Discovery IP (empty for FC volumes)
	ChapUser     string `json:"chap_user,omitempty"`     // CHAP username (empty if CHAP not used)
	ChapPassword string `json:"chap_password,omitempty"` // CHAP password (empty if CHAP not used)
}

// VirtualDeviceAccessInfo contains the required data to access a virtual device
type VirtualDeviceAccessInfo struct {
	PciSlotNumber  string `json:"pci_slot_number,omitempty"`
	ScsiController string `json:"scsi_controller,omitempty"`
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// CHAPI Mount Object
///////////////////////////////////////////////////////////////////////////////////////////////////

// Mount structure represents all information required to mount and setup filesystem
type Mount struct {
	ID           string             `json:"id,omitempty"`            // Unique mount point ID
	MountPoint   string             `json:"mount_point,omitempty"`   // Mount point location e.g. "/mnt" for Linux, "C:\MountFolder" for Windows
	SerialNumber string             `json:"serial_number,omitempty"` // Nimble volume serial number
	FsOpts       *FileSystemOptions `json:"fs_options,omitempty"`    // Filesystem options like fsType, mode, owner and mount options
	Private      *MountPrivate      `json:"-"`                       // Private mount properties used internally by CHAPI
}

// FileSystemOptions represent file system options to be configured during mount
type FileSystemOptions struct {
	FsType    string   `json:"fs_type,omitempty"`       // Filesystem type
	FsMode    string   `json:"fs_mode,omitempty"`       // Filesystem permissions
	FsOwner   string   `json:"fs_owner,omitempty"`      // Filesystem owner
	MountOpts []string `json:"mount_options,omitempty"` // Mount options rw,ro nodiscard etc
}

// FcHostPort FC host port
type FcHostPort struct {
	HostNumber string `json:"-"`
	PortWwn    string `json:"-"`
	NodeWwn    string `json:"-"`
}
