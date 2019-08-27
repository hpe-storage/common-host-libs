// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package chapi2

import (
	"github.com/gorilla/mux"
	"github.com/hpe-storage/common-host-libs/chapi2/handler"
	"github.com/hpe-storage/common-host-libs/util"
)

// NewRouter creates a new mux.Router
func NewRouter() *mux.Router {
	routes := []util.Route{
		///////////////////////////////////////////////////////////////////////////////////////////
		// Endpoint:  		GET /hosts
		// Description: 	This endpoint returns host information.
		// Input Object:	None
		// Output Object:	chapi2.Host object
		// Sample Output:
		// LINUX                                                    WINDOWS
		// {                                                        {
		//     "data": {                                                "data":  {
		//         "id": "827c1723-5742-4661-be56-121edb7e263c",            "id":  "4ba7f223-f4ce-4aca-99ff-a150b6df50be",
		//         "name": "localhost.localdomain",                         "name":  "HITDEV-WIN011",
		//         "domain":  "americas.domain.net",                       "domain":  "americas.domain.net",
		//     }                                                        }
		// }                                                        }
		///////////////////////////////////////////////////////////////////////////////////////////
		util.Route{
			Name:        "Hosts",
			Method:      "GET",
			Pattern:     "/api/v1/hosts",
			HandlerFunc: handler.GetHostInfo,
		},

		///////////////////////////////////////////////////////////////////////////////////////////
		// Endpoint:  		GET /api/v1/networks
		// Description: 	This endpoint returns NIC information.
		// Input Object:	None
		// Output Object:	Array of chapi2.Network objects
		// Sample Output:
		// LINUX                                          WINDOWS
		// {                                              {
		//     "data": [                                      "data": [
		//         {                                              {
		//             "Mac": "00:15:5d:d6:32:59",                    "Mac":  "00:05:9a:3c:7a:00",
		//             "Mtu": 1500,                                   "Mtu":  1406,
		//             "Up": true,                                    "Up":  true,
		//             "address_v4": "xxx.xxx.xxx.xxx",                "address_v4":  "xxx.xxx.xxx.xxx",
		//             "mask_v4": "xxx.xxx.xxx.xxx",                    "mask_v4":  "xxx.xxx.xxx.xxx",
		//             "name": "eth1"                                 "name":  "Ethernet 2"
		//         },                                             },
		//         {                                              {
		//             "Mac": "00:15:5d:d6:32:5a",                    "Mac":  "f4:8c:50:8e:ca:7e",
		//             "Mtu": 1500,                                   "Mtu":  1500,
		//             "Up": true,                                    "Up":  true,
		//             "address_v4": "xxx.xxx.xxx.xxx",                "address_v4":  "xxx.xxx.xxx.xxx",
		//             "mask_v4": "xxx.xxx.xxx.xxx",                    "mask_v4":  "xxx.xxx.xxx.xxx",
		//             "name": "eth2"                                 "name":  "Wi-Fi"
		//         }                                              }
		//     ]                                              ]
		// }                                              }
		///////////////////////////////////////////////////////////////////////////////////////////
		util.Route{
			Name:        "HostNetworks",
			Method:      "GET",
			Pattern:     "/api/v1/networks",
			HandlerFunc: handler.GetHostNetworks,
		},

		///////////////////////////////////////////////////////////////////////////////////////////
		// Endpoint:  		GET /api/v1/initiators
		// Description: 	This endpoint returns initiator information.
		// Input Object:	None
		// Output Object:	Array of chapi2.Initiator objects
		// Sample Output:
		// LINUX                                                  WINDOWS
		// {                                                      {
		//     "data":  [                                             "data":  [
		//         {                                                      {
		//             "access_protocol":  "fc",                              "access_protocol":  "fc",
		//             "initiator":  [                                        "initiator":  [
		//                 "10:00:00:90:FA:73:6E:CA",                             "10:00:00:90:FA:73:6E:CA",
		//                 "10:00:00:90:FA:73:6E:CB"                              "10:00:00:90:FA:73:6E:CB"
		//             ]                                                      ]
		//         },                                                     },
		//         {                                                      {
		//              "access_protocol":  "iscsi",                           "access_protocol":  "iscsi",
		//              "initiator":  [                                        "initiator":  [
		//                  "iqn.1994-05.com.redhat:32b8e991e082"                  "iqn.1991-05.com.microsoft:hitdev-win011.local"
		//             ]                                                      ]
		//         }                                                      }
		//     ]                                                      ]
		// }                                                      }
		///////////////////////////////////////////////////////////////////////////////////////////
		util.Route{
			Name:        "HostInitiators",
			Method:      "GET",
			Pattern:     "/api/v1/initiators",
			HandlerFunc: handler.GetHostInitiators,
		},

		///////////////////////////////////////////////////////////////////////////////////////////
		// Endpoint:  		GET /api/v1/devices
		// Description: 	This endpoint returns all the Nimble volumes attached to the host
		// Input Object:	None
		// Output Object:	Array of chapi2.Device objects with basic details
		// Sample Output:
		// LINUX                                                            WINDOWS
		// {                                                                {
		//     "data": [                                                        "data": [
		//         {                                                                {
		//             "serial_number": "28174883c7719ac236c9ce900...",                "serial_number": "28174883c7719ac236c9ce900...",
		//         }                                                                }
		//     ]                                                                ]
		// }                                                                  }
		///////////////////////////////////////////////////////////////////////////////////////////
		util.Route{
			Name:        "Devices",
			Method:      "GET",
			Pattern:     "/api/v1/devices",
			HandlerFunc: handler.GetDevices,
		},

		///////////////////////////////////////////////////////////////////////////////////////////
		// Endpoint:  		GET /api/v1/devices/details
		// Description: 	This endpoint returns all the Nimble volumes attached to the host
		// Input Object:	None
		// Output Object:	Array of chapi2.Device objects with detailed information
		// Sample Output:
		// LINUX                                                            WINDOWS
		// {                                                                {
		//     "data": [                                                        "data": [
		//         {                                                                {
		//             "alt_full_path_name": "/dev/mapper/mpathg",                      "alt_full_path_name": "\\\\?\\mpio#disk&ven_nimble&...",
		//             "iscsi_target": {                                                "iscsi_target": {
		//                 "Address": "xxx.xxx.xxx.xxx",                                      "Address": "xxx.xxx.xxx.xxx",
		//                 "Name": "iqn.2007-11.com.nimblestorage:...",                     "Name": "iqn.2007-11.com.nimblestorage:...",
		//                 "Port": "3260",                                                  "Port": "3260",
		//                 "Scope": "",                                                     "Scope": "",
		//                 "Tag": "2460"                                                },
		//             },                                                               "path_name": "Disk3",
		//             "major": "253",                                                  "serial_number": "28174883c7719ac236c9ce900...",
		//             "minor": "3",                                                    "size": 20048,
		//             "mpath_device_name": "mpathg",                               }
		//             "path_name": "dm-3",                                     ]
		//             "serial_number": "28174883c7719ac236c9ce900...",     }
		//             "size": 20048,
		//             "slaves": [
		//                 "sdb",
		//                 "sdc"
		//             ],
		//             "state": "active"
		//         }
		//     ]
		// }
		///////////////////////////////////////////////////////////////////////////////////////////
		util.Route{
			Name:        "AllDeviceDetails",
			Method:      "GET",
			Pattern:     "/api/v1/devices/details",
			HandlerFunc: handler.GetAllDeviceDetails,
		},

		///////////////////////////////////////////////////////////////////////////////////////////
		// Endpoint:  		GET /api/v1/devices/{serialNumber}/partitions
		// Description: 	This endpoint returns partition information for the specified volume.
		// Input Object:	None
		// Output Object:	Array of chapi2.DevicePartition objects
		// Sample Output:
		// LINUX    WINDOWS
		// TBD      {
		//              "data":  [
		//                  {
		//                      "name":  "Disk #1, Partition #0",
		//                      "partition_type":  "GPT: Basic Data",
		//                      "size":  5333057536
		//                  }
		//              ]
		//          }
		///////////////////////////////////////////////////////////////////////////////////////////
		util.Route{
			Name:        "PartitionsForDevice",
			Method:      "GET",
			Pattern:     "/api/v1/devices/{serialNumber}/partitions",
			HandlerFunc: handler.GetPartitionsForDevice,
		},

		///////////////////////////////////////////////////////////////////////////////////////////
		// Endpoint:  		POST /api/v1/devices
		// Description: 	Connect to the specified Nimble volume.
		// Input Object:	Array of chapi2.Volume objects
		// Output Object:	Array of chapi2.Device objects
		// Sample Input:    [
		//                      {
		//                          "discovery_ip":  "xxx.xxx.xxx.xxx",
		//                          "iqn":  "iqn.2007-11.com.nimblestorage:group-c32-array3-g5a2cdea9cf0b91f1",
		//                          "access_protocol":  "iscsi",
		//                          "serial_number":  "28174883c7719ac236c9ce900584f2795"
		//                      }
		//                  ]
		// Sample Output:	See "GET /api/v1/devices/details" endpoint
		///////////////////////////////////////////////////////////////////////////////////////////
		util.Route{
			Name:        "CreateDevice",
			Method:      "POST",
			Pattern:     "/api/v1/devices",
			HandlerFunc: handler.CreateDevice,
		},

		///////////////////////////////////////////////////////////////////////////////////////////
		// Endpoint:  		DELETE /api/v1/devices/{serialNumber}
		// Description: 	Disconnects the specified Nimble serial number.  If it's an iSCSI GST
		//					or FC LUN, the volume remains, the volume is only offlined on the host.
		// Input Object:	None
		// Output Object:	None (only Error details if request fails)
		///////////////////////////////////////////////////////////////////////////////////////////
		util.Route{
			Name:        "DeleteDevice",
			Method:      "DELETE",
			Pattern:     "/api/v1/devices/{serialNumber}",
			HandlerFunc: handler.DeleteDevice,
		},

		///////////////////////////////////////////////////////////////////////////////////////////
		// Endpoint:  		PUT /api/v1/devices/{serialNumber}/actions/offline
		// Description: 	Offlines the device with specified serial number on the host.  This is not
		//					an offline at the array, but rather an offline only at the host.
		// Input Object:	None
		// Output Object:	None (only Error details if request fails)
		///////////////////////////////////////////////////////////////////////////////////////////
		util.Route{
			Name:        "OfflineDevice",
			Method:      "PUT",
			Pattern:     "/api/v1/devices/{serialNumber}/actions/offline",
			HandlerFunc: handler.OfflineDevice,
		},

		///////////////////////////////////////////////////////////////////////////////////////////
		// Endpoint:  		PUT /api/v1/devices/{serialNumber}/{fileSystem}
		// Description: 	Formats the specified volume with the specified file system.
		// Input Object:	None
		// Output Object:	None
		// Sample Output:	See "GET /hosts/{id}/devices" endpoint
		///////////////////////////////////////////////////////////////////////////////////////////
		util.Route{
			Name:        "CreateFileSystem",
			Method:      "PUT",
			Pattern:     "/api/v1/devices/{serialNumber}/{fileSystem}",
			HandlerFunc: handler.CreateFileSystem,
		},

		///////////////////////////////////////////////////////////////////////////////////////////
		// Endpoint:  		GET /api/v1/mounts
		// Description: 	Enumerates all mount points on the host, optionally with given serial number
		// Input Object:	None
		// Output Object:	Array of chapi2.Mount objects
		// Sample Output:
		// LINUX         WINDOWS
		// TODO          {
		//                   "data":  [
		//                       {
		//                           "id":  "227bab86e6c96b83-1-1"
		//                       },
		//                       {
		//                           "id":  "519fe8378f5ddae3-2-2"
		//                       }
		//                   ]
		//               }
		///////////////////////////////////////////////////////////////////////////////////////////
		util.Route{
			Name:        "GetMounts",
			Method:      "GET",
			Pattern:     "/api/v1/mounts",
			HandlerFunc: handler.GetMounts,
		},

		///////////////////////////////////////////////////////////////////////////////////////////
		// Endpoint:  		GET /api/v1/mounts/details
		// Description: 	Enumerates all mount points on the host with detailed information, optionally with given serial number
		// Input Object:	None
		// Output Object:	Array of chapi2.Mount objects
		// Sample Output:
		// LINUX         WINDOWS
		// TODO          {
		//                   "data":  [
		//                       {
		//                           "id":  "227bab86e6c96b83-1-1",
		//                           "mount_point":  "C:\\MyMount1",
		//                           "serial_number":  "f4c97c5c1cd391756c9ce900584f2795"
		//                       },
		//                       {
		//                           "id":  "519fe8378f5ddae3-2-2",
		//                           "mount_point":  "C:\\MyMount2",
		//                           "serial_number":  "c5a28c28a2487d3d6c9ce900584f2795"
		//                       }
		//                   ]
		//               }
		///////////////////////////////////////////////////////////////////////////////////////////
		util.Route{
			Name:        "GetAllMountDetails",
			Method:      "GET",
			Pattern:     "/api/v1/mounts/details",
			HandlerFunc: handler.GetAllMountDetails,
		},

		///////////////////////////////////////////////////////////////////////////////////////////
		// Endpoint:  		POST /api/v1/mounts
		// Description: 	Mount a Nimble volume to the specified mount point location.
		// Input Object:	chapi2.Mount object - utilized input parameters listed below
		//                          mount.SerialNumber (required)
		//                          mount.MountPoint (required)
		//                          mount.FsOpts (optional)
		// Output Object:	chapi2.Mount object
		// Sample Output:	See "GET /api/v1/mounts/details" endpoint
		///////////////////////////////////////////////////////////////////////////////////////////
		util.Route{
			Name:        "CreateMount",
			Method:      "POST",
			Pattern:     "/api/v1/mounts",
			HandlerFunc: handler.CreateMount,
		},

		///////////////////////////////////////////////////////////////////////////////////////////
		// Endpoint:  		Delete /api/v1/mounts/{mountId}
		// Description: 	Unmount a device from the specified mount point location.
		// Input Object:	Nimble volume serial number (string only)
		// Output Object:	None (only Error details if request fails)
		///////////////////////////////////////////////////////////////////////////////////////////
		util.Route{
			Name:        "DeleteMount",
			Method:      "DELETE",
			Pattern:     "/api/v1/mounts/{mountId}",
			HandlerFunc: handler.DeleteMount,
		},
	}

	routes = append(routes, platformSpecificEndpoints...)
	router := mux.NewRouter().StrictSlash(true)
	util.InitializeRouter(router, routes)
	return router
}
