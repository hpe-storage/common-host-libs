package chapiclient

import (
	"fmt"
	"time"

	"github.com/hpe-storage/common-host-libs/chapi2/cerrors"
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	"github.com/hpe-storage/common-host-libs/connectivity"
	log "github.com/hpe-storage/common-host-libs/logger"
)

var (
	// // Hosts represents collection of hosts
	// Hosts model.Hosts
	// // DeviceURIfmt represents particular device endpoint format for GET  and POST requests
	// DeviceURIfmt = "%sdevices/%s"
	// HostURI represents hosts endpoint format for GET requests
	HostURI = "api/v1/hosts"
	// // MountURIfmt represents mounts endpoint format for GET and POST requests
	// MountURIfmt = "%smounts/%s"
	// DevicesURI represents devices endpoint format for GET requests
	DevicesURI = "api/v1/devices"
	// DevicesDetail represents devices endpoint format for GET requests
	DevicesDetailURI = "api/v1/devices/detail"
	// // UnmountURIfmt represents particular mount endpoint for POST requests
	// UnmountURIfmt = "%smounts/%s"
	// // CreateFSURIfmt represents devices endpoint format for POST requests
	// CreateFSURIfmt = "%sdevices/%s/%s"
	// NetworksURI represents networks endpoint format for GET requests
	NetworksURI = "api/v1/networks"
	// InitiatorsURI represents initiators endpoint for GET requests
	InitiatorsURI = "api/v1/initiators"
	// // HostnameURIfmt represents hostname endpoint for GET requests
	// HostnameURIfmt = "%shostname"
	// // ChapInfoURIfmt represents endpoint to obtain initiator CHAP credentials
	// ChapInfoURIfmt = "%schapinfo"
	// // LogLevel of the chapi client
	// LogLevel = "info"
)

// Client defines client for chapid and corresponding socket file or http:port info
type Client struct {
	// client is the http client for chapid server
	client *connectivity.Client
	// socket is the socket name on which chapid is listening
	socket string
	// hostname where server is running
	hostname string
	// port number where the server is listening on.
	port uint64
	// host ID where server is running
	hostID string
	// HTTP headers
	header map[string]string
}

// ErrorResponse struct
// type ErrorResponse struct {
// 	Code cerrors.ChapiErrorCode `json:"code"`
// 	Text string                 `json:"text,omitempty"`
// }

//Response :
type Response struct {
	Data interface{} `json:"data,omitempty"`
	Err  interface{} `json:"errors,omitempty"`
}

// AccessKeyPath struct
type AccessKeyPath struct {
	Path string `json:"path"`
}

// Print to dump chapi client struct
func (chapiClient *Client) Print() {
	log.Traceln("HostID   : ", chapiClient.hostID)
	log.Traceln("Socket   : ", chapiClient.socket)
	log.Traceln("Hostname : ", chapiClient.hostname)
	log.Traceln("Port     : ", chapiClient.port)
	log.Traceln("Header   : ", chapiClient.header)
}

// GetHostURL to consruct HTTP URL of the form "hostname:port"
func GetHostURL(hostName string, port uint64) string {
	return fmt.Sprintf("%v:%v", hostName, port)
}

// NewChapiHTTPClient to create chapi http client
func NewChapiHTTPClient(hostName string, port uint64) (*Client, error) {
	var chapiClient *Client

	hostURL := GetHostURL(hostName, port)
	log.Debugln("setting up chapi client with http", hostURL)
	// setup chapi client with timeout
	httpClient := connectivity.NewHTTPClient(hostURL)
	chapiClient = &Client{client: httpClient, hostname: hostName, port: port}
	return chapiClient, nil
}

// NewChapiHTTPClientWithTimeout to create chapi http client with timeout
func NewChapiHTTPClientWithTimeout(hostName string, port uint64, timeout time.Duration) (*Client, error) {
	var chapiClient *Client

	hostURL := GetHostURL(hostName, port)
	log.Debugln("setting up chapi client with http", hostURL, "and timeout", timeout)
	// setup chapi client with timeout
	httpClient := connectivity.NewHTTPClientWithTimeout(hostURL, timeout)
	chapiClient = &Client{client: httpClient, hostname: hostName, port: port}
	return chapiClient, nil
}

// NewChapiHTTPClientWithTimeoutAndHeader to create chapi http client with timeout and headers
func NewChapiHTTPClientWithTimeoutAndHeader(hostName string, port uint64, timeout time.Duration, header map[string]string) (*Client, error) {
	chapiClient, err := NewChapiHTTPClientWithTimeout(hostName, port, timeout)
	if err != nil {
		return nil, err
	}
	// Insert http headers
	chapiClient.AddHeader(header)
	return chapiClient, nil
}

// AddHeader to insert HTTP headers to chapi client
func (chapiClient *Client) AddHeader(header map[string]string) error {
	log.Traceln("Inserting http headers", header, "to chapi client")
	chapiClient.header = header
	return nil
}

// GetHostID will retrieve the host uuid from host
func (chapiClient *Client) GetHostID() (string, error) {
	log.Trace("GetHostID called")
	var host *model.Host
	var errResp *cerrors.ChapiError
	var chapiResp Response
	chapiResp.Data = &host
	chapiResp.Err = &errResp
	_, err := chapiClient.client.DoJSON(&connectivity.Request{Action: "GET", Path: HostURI, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp})
	if err != nil {
		if errResp != nil {
			log.Errorf(errResp.Text)
			return "", fmt.Errorf(errResp.Text)
		}
		log.Errorf("GetHostID Err :%s", err.Error())
		return "", err
	}

	if host == nil {
		err := fmt.Errorf("no valid host found")
		log.Errorf(err.Error())
		return "", err
	}
	log.Tracef("hostid obtained is :%s", host.UUID)

	return host.UUID, nil
}

// GetInitiators will return host initiator list (WWPNs in case of FC)
func (chapiClient *Client) GetInitiators() (initiators []*model.Initiator, err error) {
	log.Trace("GetInitiators called")

	var chapiResp Response
	chapiResp.Data = &initiators
	var errResp *cerrors.ChapiError
	chapiResp.Err = &errResp
	_, err = chapiClient.client.DoJSON(&connectivity.Request{Action: "GET", Path: InitiatorsURI, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp})
	if err != nil {
		if errResp != nil {
			log.Error(errResp)
			return nil, errResp
		}
		log.Errorf("GetInitiators Err :%s", err.Error())
		return nil, err
	}

	return initiators, nil
}

// GetNetworks return list of host network interface details
func (chapiClient *Client) GetNetworks() (networks []*model.Network, err error) {
	log.Trace("GetNetworks called")

	var errResp *cerrors.ChapiError
	var chapiResp Response
	chapiResp.Data = &networks
	chapiResp.Err = &errResp
	_, err = chapiClient.client.DoJSON(&connectivity.Request{Action: "GET", Path: NetworksURI, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp})
	if err != nil {
		if errResp != nil {
			log.Error(errResp)
			return nil, errResp
		}
		log.Errorf("GetNetworks Err :", err.Error())
		return nil, err
	}

	return networks, nil
}

// // AttachDevice will attach the given os device for given volume on the host
// func (chapiClient *Client) AttachDevice(volumes []*model.Volume) (devices []*model.Device, err error) {
// 	log.Tracef(">>>>> AttachDevice called with %#v", volumes)
// 	defer log.Trace("<<<<< AttachDevice")

// 	// check if volumes is not nil
// 	if len(volumes) == 0 {
// 		return nil, fmt.Errorf("no volume available to attach")
// 	}
// 	// fetch host ID
// 	err = chapiClient.cacheHostID()
// 	if err != nil {
// 		return nil, err
// 	}
// 	devicesURI := fmt.Sprintf(DevicesURIfmt, fmt.Sprintf(HostURIfmt, chapiClient.hostID))

// 	var errResp *cerrors.ChapiError
// 	var chapiResp Response
// 	chapiResp.Data = &devices
// 	chapiResp.Err = &errResp
// 	_, err = chapiClient.client.DoJSON(&connectivity.Request{Action: "POST", Path: devicesURI, Header: chapiClient.header, Payload: &volumes, Response: &chapiResp, ResponseError: &chapiResp})
// 	if err != nil {
// 		if errResp != nil {
// 			log.Errorf("AttachDevice: %s for volume(%s)", errResp.Text, volumes[0].Name)
// 			return nil, errResp
// 		}
// 		log.Errorf("AttachDevice: Err:%s for volume(%s)", err.Error(), volumes[0].Name)
// 		return nil, err
// 	}
// 	// Win CHAPI doesn't return Device[0].State, added OR condition to handele it.
// 	if len(devices) != 0 && ((devices[0].State == model.ActiveState.String()) || (devices[0].State == "")) {
// 		log.Debugf("Device found with active paths %+v", devices[0])
// 		return devices, nil
// 	}
// 	return nil, nil
// }

// // AttachAndMountDevice will attach the given os device for given volume and mounts the filesystem on the host
// func (chapiClient *Client) AttachAndMountDevice(volume *model.Volume, mountPath string) (err error) {
// 	log.Tracef(">>>>> AttachAndMountDevice on volume %#v to mount path %s", volume, mountPath)
// 	defer log.Trace("<<<<< AttachAndMountDevice")

// 	var vols []*model.Volume
// 	vols = append(vols, volume)

// 	// fetch host ID
// 	err = chapiClient.cacheHostID()
// 	if err != nil {
// 		return err
// 	}

// 	// first create and attach the device
// 	devices, err := chapiClient.AttachDevice(vols)
// 	if err != nil || len(devices) == 0 {
// 		err = fmt.Errorf("unable to attach device %s", err.Error())
// 		log.Tracef(err.Error())
// 		return err
// 	}
// 	// wait for a second to mount the device
// 	time.Sleep(time.Second)

// 	// perform the mount of the device created above
// 	err = chapiClient.retryMountFileSystem(volume, mountPath)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (chapiClient *Client) retryMountFileSystem(volume *model.Volume, mountPath string) (err error) {
// 	log.Tracef("retryMountFileSystem called with %#v and mountPoint %s", volume, mountPath)
// 	maxTries := 3
// 	try := 0
// 	for {
// 		err = chapiClient.MountFilesystem(volume, mountPath)
// 		if err == nil {
// 			return nil
// 		}
// 		log.Tracef("error trying to mount filesystem %v", err.Error())
// 		if strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "doesn't exist") {
// 			if try < maxTries {
// 				try++
// 				log.Debugf("try=%d for mountFileSystem for %s", try, volume.Name)
// 				time.Sleep(time.Duration(try) * time.Second)
// 				continue
// 			}
// 		}
// 		return err
// 	}
// }

// // CreateFilesystem calls chapi server to create filesystem
// func (chapiClient *Client) CreateFilesystem(device *model.Device, vol *model.Volume, filesystem string) (err error) {
// 	log.Tracef("CreateFilesystem called for %s and filesystem %s", device.MpathName, filesystem)
// 	// fetch host ID
// 	err = chapiClient.cacheHostID()
// 	if err != nil {
// 		return err
// 	}

// 	createFSURI := fmt.Sprintf(CreateFSURIfmt, fmt.Sprintf(HostURIfmt, chapiClient.hostID), device.SerialNumber, filesystem)

// 	var errResp *cerrors.ChapiError
// 	var dev *model.Device
// 	var chapiResp Response
// 	chapiResp.Data = &dev
// 	chapiResp.Err = &errResp
// 	_, err = chapiClient.client.DoJSON(&connectivity.Request{Action: "PUT", Path: createFSURI, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp})
// 	if err != nil {
// 		if errResp != nil {
// 			log.Error(errResp)
// 			return errors.New(errResp.Text)
// 		}
// 		log.Error("CreateFilesystem Err :", err.Error())
// 		return err
// 	}
// 	return nil
// }

// // Mount makes a POST request to chapi to mount filesystem on the host
// func (chapiClient *Client) Mount(reqMount *model.Mount, respMount *model.Mount) (err error) {
// 	log.Tracef("Mount called on for device %s serialNumber %s", reqMount.Device.MpathName, reqMount.Device.SerialNumber)
// 	// fetch host ID
// 	err = chapiClient.cacheHostID()
// 	if err != nil {
// 		return err
// 	}
// 	mountURI := fmt.Sprintf(MountURIfmt, fmt.Sprintf(HostURIfmt, chapiClient.hostID), reqMount.Device.SerialNumber)
// 	log.Tracef("mountURI=%s", mountURI)

// 	var errResp *cerrors.ChapiError
// 	var chapiResp Response
// 	chapiResp.Data = &respMount
// 	chapiResp.Err = &errResp
// 	_, err = chapiClient.client.DoJSON(&connectivity.Request{Action: "POST", Path: mountURI, Header: chapiClient.header, Payload: &reqMount, Response: &chapiResp, ResponseError: &chapiResp})
// 	if err != nil {
// 		if errResp != nil {
// 			log.Error(errResp)
// 			return errors.New(errResp.Text)
// 		}
// 		log.Error("Mount Err :", err.Error())
// 		return err
// 	}

// 	if respMount == nil {
// 		return errors.New("no valid Mount Point created")
// 	}
// 	return nil
// }

// GetDevices enumerates all the Nimble volumes with basic details.
// If serialNumber is non-empty then only specified device is returned
func (chapiClient *Client) GetDevices(serialNumber string) (devices []*model.Device, err error) {
	log.Trace("GetDevices called")

	chapiResp2 := Response{Data: &devices, Err: new(cerrors.ChapiError)}

	var errResp *cerrors.ChapiError
	var chapiResp Response
	chapiResp.Data = &devices
	chapiResp.Err = &errResp
	devicesURI := DevicesURI
	if serialNumber != "" {
		devicesURI += "?serial=" + serialNumber
	}
	_ = chapiResp
	_, err = chapiClient.client.DoJSON(&connectivity.Request{Action: "GET", Path: devicesURI, Header: chapiClient.header, Payload: nil, Response: &chapiResp2, ResponseError: &chapiResp2})
	if err != nil {
		if errResp != nil {
			log.Error(errResp)
			return nil, errResp
		}
		log.Debug("Err :", err.Error())
		return nil, err
	}
	return devices, nil
}

// // GetDeviceFromVolume will return os device for given storage volume
// func (chapiClient *Client) GetDeviceFromVolume(volume *model.Volume) (device *model.Device, err error) {
// 	log.Tracef(">>>>>GetDeviceFromVolume called for %s", volume.Name)
// 	defer log.Tracef("<<<<< GetDeviceFromVolume")

// 	// fetch host ID
// 	err = chapiClient.cacheHostID()
// 	if err != nil {
// 		return nil, err
// 	}
// 	serialNumber := GetSerialNumber(volume.SerialNumber)
// 	devicesURI := fmt.Sprintf(DeviceURIfmt, fmt.Sprintf(HostURIfmt, chapiClient.hostID), serialNumber)

// 	var errResp *cerrors.ChapiError
// 	var chapiResp Response
// 	chapiResp.Data = &device
// 	chapiResp.Err = &errResp
// 	_, err = chapiClient.client.DoJSON(&connectivity.Request{Action: "GET", Path: devicesURI, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp})
// 	if err != nil {
// 		if errResp != nil {
// 			log.Error(errResp)
// 			return nil, errResp
// 		}
// 		log.Error("GetDeviceFromVolume Err :", err.Error())
// 		return nil, err
// 	}

// 	if device == nil {
// 		return nil, fmt.Errorf("no matching device found for volume %s", volume.Name)
// 	}

// 	device.TargetScope = volume.TargetScope
// 	return device, nil
// }

// // GetMounts will return all mounted nimble volumes on the host
// func (chapiClient *Client) GetMounts(respMount *[]*model.Mount, serialNumber string) (err error) {
// 	log.Trace("GetMounts called")
// 	// fetch host ID
// 	err = chapiClient.cacheHostID()
// 	if err != nil {
// 		return err
// 	}
// 	log.Tracef("getting mounts for serial Number %s", serialNumber)

// 	mountsURI := fmt.Sprintf(MountURIfmt, fmt.Sprintf(HostURIfmt, chapiClient.hostID), serialNumber)

// 	var errResp *cerrors.ChapiError
// 	var chapiResp Response
// 	chapiResp.Data = &respMount
// 	chapiResp.Err = &errResp
// 	_, err = chapiClient.client.DoJSON(&connectivity.Request{Action: "GET", Path: mountsURI, Header: chapiClient.header, Payload: nil, Response: &chapiResp, ResponseError: &chapiResp})
// 	if err != nil {
// 		if errResp != nil {
// 			log.Error(errResp)
// 			return errors.New(errResp.Text)
// 		}
// 		log.Error("GetMounts Err :", err.Error())
// 		return err
// 	}
// 	return err
// }

// //UnmountDevice performs the host side workflow to unmount a volume
// func (chapiClient *Client) UnmountDevice(volume *model.Volume) error {
// 	log.Trace(">>>>> UnmountDevice for volume ", volume.Name)
// 	defer log.Trace("<<<<< UnmountDevice")

// 	var respMount []*model.Mount
// 	err := chapiClient.GetMounts(&respMount, GetSerialNumber(volume.SerialNumber))
// 	if err != nil && !(strings.Contains(err.Error(), "object was not found")) {
// 		return err
// 	}
// 	if respMount != nil {
// 		for _, mount := range respMount {
// 			log.Tracef("perform an unmount on the host with %#v", mount)
// 			if mount.Device.SerialNumber == GetSerialNumber(volume.SerialNumber) {
// 				log.Tracef("Device to ummount found :%s", mount.Mountpoint)
// 				var respMount model.Mount
// 				err = chapiClient.Unmount(mount, &respMount)
// 				if err != nil {
// 					return err
// 				}
// 				// TODO: This can be dangerous. Move this to docker plugin to be docker specific
// 				//3. delete the Mounpoint
// 				_, isDir, _ := util.FileExists(mount.Mountpoint)
// 				if isDir {
// 					os.RemoveAll(mount.Mountpoint)
// 					if err != nil {
// 						return err
// 					}
// 				}
// 				break
// 			}
// 		}
// 	}
// 	return nil
// }

// // Unmount make a DELETE request to chapi to unmount filesystem on the host
// func (chapiClient *Client) Unmount(reqMount *model.Mount, respMount *model.Mount) (err error) {
// 	log.Trace(">>>>> Unmount called on %s", reqMount.Mountpoint)
// 	defer log.Trace("<<<<< Unmount")
// 	// fetch host ID
// 	err = chapiClient.cacheHostID()
// 	if err != nil {
// 		return err
// 	}

// 	unMountURI := fmt.Sprintf(UnmountURIfmt, fmt.Sprintf(HostURIfmt, chapiClient.hostID), reqMount.ID)

// 	var errResp *cerrors.ChapiError
// 	var chapiResp Response
// 	chapiResp.Data = &respMount
// 	chapiResp.Err = &errResp
// 	_, err = chapiClient.client.DoJSON(
// 		&connectivity.Request{
// 			Action:        "DELETE",
// 			Path:          unMountURI,
// 			Header:        chapiClient.header,
// 			Payload:       &reqMount,
// 			Response:      &chapiResp,
// 			ResponseError: &chapiResp,
// 		},
// 	)

// 	if err != nil {
// 		if errResp != nil {
// 			log.Errorf("Unmount: err info %s", errResp.Text)
// 			return errors.New(errResp.Text)
// 		}
// 		log.Error("Unmount Err :", err.Error())
// 		return err
// 	}
// 	return err
// }

// //OfflineDevice : offline the device in preparation of removing ACL to avoid race-condition to be discovered again
// func (chapiClient *Client) OfflineDevice(device *model.Device) (err error) {
// 	log.Tracef(">>>>> OfflineDevice with %#v", device)
// 	defer log.Trace("<<<<< OfflineDevice")

// 	err = chapiClient.cacheHostID()
// 	if err != nil {
// 		return err
// 	}

// 	serialNumber := device.SerialNumber
// 	deviceOfflineURI := fmt.Sprintf("/hosts/%s/devices/%s/actions/offline", chapiClient.hostID, serialNumber)

// 	var deviceResp model.Device
// 	var errResp *cerrors.ChapiError
// 	var chapiResp Response
// 	chapiResp.Data = &deviceResp
// 	chapiResp.Err = &errResp
// 	_, err = chapiClient.client.DoJSON(&connectivity.Request{Action: "PUT", Path: deviceOfflineURI, Header: chapiClient.header, Payload: device, Response: &chapiResp, ResponseError: &chapiResp})
// 	if err != nil {
// 		if errResp != nil {
// 			log.Errorf("OfflineDevice Err info :%s", errResp.Text)
// 			return fmt.Errorf(errResp.Text)
// 		}
// 		log.Errorf("OfflineDevice Err :%s", err.Error())
// 		return err
// 	}
// 	return nil
// }

// //DeleteDevice : delete the os device on the host
// // nolint : Remove this once 'DetachDevice' from chapiclient_windows.go is removed
// func (chapiClient *Client) DeleteDevice(device *model.Device) (err error) {
// 	log.Tracef(">>>>> DeleteDevice with %#v", device)
// 	defer log.Trace("<<<<< DeleteDevice")

// 	// fetch host ID
// 	err = chapiClient.cacheHostID()
// 	if err != nil {
// 		return err
// 	}

// 	serialNumber := device.SerialNumber
// 	deviceURI := fmt.Sprintf(DeviceURIfmt, fmt.Sprintf(HostURIfmt, chapiClient.hostID), serialNumber)

// 	var deviceResp model.Device
// 	var errResp *cerrors.ChapiError
// 	var chapiResp Response
// 	chapiResp.Data = &deviceResp
// 	chapiResp.Err = &errResp
// 	_, err = chapiClient.client.DoJSON(&connectivity.Request{Action: "DELETE", Path: deviceURI, Header: chapiClient.header, Payload: device, Response: &chapiResp, ResponseError: &chapiResp})
// 	if err != nil {
// 		if errResp != nil {
// 			log.Errorf("DeleteDevice Err info %s", errResp.Text)
// 			return errors.New(errResp.Text)
// 		}
// 		log.Errorf("DeleteDevice Err :%s", err.Error())
// 		return err
// 	}
// 	return err
// }
