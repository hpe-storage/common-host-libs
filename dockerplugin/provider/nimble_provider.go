// Copyright 2019 Hewlett Packard Enterprise Development LP

package provider

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/hpe-storage/common-host-libs/cert"
	"github.com/hpe-storage/common-host-libs/connectivity"
	"github.com/hpe-storage/common-host-libs/linux"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/util"
)

// struct to map all blocked keys per option
type optionLimit struct {
	option      string
	blockedKeys []string
}

const (
	nimbleProviderPort = "8443"
	// DefaultContainerProviderVersion indicates default provider api version
	DefaultContainerProviderVersion = "0.0"
)

// LoginRequest : container provider login Request
type LoginRequest struct {
	Username string `json:"UserName,omitempty"`
	Password string `json:"Password,omitempty"`
	Cert     string `json:"Cert,omitempty"`
}

// LoginResponse : container provider login response
type LoginResponse struct {
	Err string `json:"Err,omitempty"`
}

// IsNimblePlugin returns true if plugin type is for nimble platform
func IsNimblePlugin() bool {
	if os.Getenv("PLUGIN_TYPE") == "nimble" {
		return true
	}
	return false
}

func getBasicContainerProviderClient(ipAddress string) *connectivity.Client {
	containerproviderLoginURI := fmt.Sprintf("https://%s:8443/container-provider", ipAddress)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	containerProviderLoginClient := connectivity.NewHTTPSClient(containerproviderLoginURI, transport)
	return containerProviderLoginClient
}

//AddRemoveCertContainerProvider :
func AddRemoveCertContainerProvider(containerProviderURI string, ipAddress string, hostCert string, username string, password string) error {
	log.Tracef(">>>>> AddRemoveCertContainerProvider called with %s %s", containerProviderURI, ipAddress)
	defer log.Trace("<<<<< AddRemoveCertContainerProvider")

	containerProviderLoginClient := getBasicContainerProviderClient(ipAddress)
	request := &LoginRequest{Username: username, Password: password, Cert: hostCert}
	response := &LoginResponse{}

	_, err := containerProviderLoginClient.DoJSON(&connectivity.Request{Action: "POST", Path: containerProviderURI, Payload: &request, Response: &response, ResponseError: nil})
	if err != nil {
		log.Error(err.Error())
		return err
	}

	if response.Err != "" {
		log.Errorf("Failure while attempting %s to container provider: %s", containerProviderURI, response.Err)
		return errors.New("failed to configure certificate on container provider: " + response.Err)
	}

	return nil
}

// get client for nimble container provider
func getNimbleContainerProviderClient() (*connectivity.Client, error) {
	log.Trace(">>>>> getNimbleContainerProviderClient")
	defer log.Trace("<<<<< getNimbleContainerProviderClient")

	providerURI, err := GetProviderURI("", nimbleProviderPort, "/container-provider")
	if err != nil {
		return nil, err
	}

	is, _, err := util.FileExists(HostCertFile)
	if !is {
		log.Errorf("host cert file %s doesnt exist", HostCertFile)
		return nil, err
	}

	is, _, err = util.FileExists(HostKeyFile)
	if !is {
		log.Errorf("host key file %s doesnt exist", HostKeyFile)
		return nil, err
	}

	is, _, err = util.FileExists(ServerCertFile)
	if !is {
		log.Errorf("host key file %s doesnt exist", ServerCertFile)
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(HostCertFile, HostKeyFile)
	if err != nil {
		log.Error("unable to load client cert", err.Error())
		return nil, err
	}

	//Load CA cert which is the server cert in our case
	caCert, err := ioutil.ReadFile(ServerCertFile)
	if err != nil {
		log.Trace("unable to load server cert", err.Error())
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()

	// Setup HTTPS client
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	containerProviderClient := connectivity.NewHTTPSClientWithTimeout(providerURI, transport,
		providerClientTimeout)

	return containerProviderClient, nil
}

// LoginAndCreateCerts :
func LoginAndCreateCerts(ipAddress string, username string, password string, isv2 bool) error {
	log.Tracef(">>>>> LoginAndCreateCerts called with ipAddress(%s) username(%s)", ipAddress, username)
	defer log.Trace("<<<<< LoginAndCreateCerts")

	// get array certificate
	groupCert, err := cert.GetCertFromGroup(ipAddress, nimbleProviderPort)
	if err != nil {
		log.Error("LoginAndCreateCerts err", err.Error())
		return err
	}
	serverCertPem, err := cert.ConvertCertToPem(groupCert)
	if err != nil {
		return err
	}
	//write array cert to  dockerplugin.ServerCertFile
	err = cert.WriteCertPemToFile(serverCertPem, ServerCertFile)
	if err != nil {
		return err
	}
	isHostCertsPresent := checkIfHostCertsExist()

	// reuse the host certs if they are present
	if !isHostCertsPresent {
		err := createAndAddHostCerts(ipAddress, username, password, HostKeyFile, HostCertFile)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkIfHostCertsExist() bool {
	log.Trace(">>>>> checkIfHostCertsExist")
	defer log.Trace("<<<<< checkIfHostCertsExist")
	is, _, _ := util.FileExists(HostKeyFile)
	if !is {
		log.Debugf("host key file %s doesnt exist", HostKeyFile)
		return is
	}

	is, _, _ = util.FileExists(HostCertFile)
	if !is {
		log.Debugf("host key file %s doesnt exist", HostCertFile)
		return is
	}
	log.Debugf("hostCertificate exists at %s. Reusing them to connect to the array", HostCertFile)
	return true
}

func createAndAddHostCerts(ipAddress, username, password, hostKeyFile, hostCertFile string) error {
	// get common name as hostname
	cn, err := linux.GetHostNameAndDomain()
	if err != nil {
		return err
	}
	// generate host key and certificate
	_, hostKey, hostCert, err := cert.GenerateCert(cn[0])
	if err != nil {
		return err
	}

	// write the hostKey and hostCert in PEM format
	err = cert.WriteCertPemToFile(hostKey, hostKeyFile)
	if err != nil {
		return err
	}
	err = cert.WriteCertPemToFile(hostCert, hostCertFile)
	if err != nil {
		return err
	}
	// invoke dockerplugin.NimbleLoginURI end point of container provider to add certificate
	err = AddRemoveCertContainerProvider(NimbleLoginURI, ipAddress, hostCert, username, password)
	if err != nil {
		return err
	}
	return nil
}
