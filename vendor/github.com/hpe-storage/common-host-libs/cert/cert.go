package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/util"
	"math/big"
	"os"
	"time"
)

var (
	privateKeyConst     = "PRIVATE KEY"
	certificateConst    = "CERTIFICATE"
	organization        = "HPE Nimble Storage"
	certificateValidity = time.Duration(87600) * time.Hour // 10 years
	signatureAlgorithm  = x509.SHA256WithRSA
)

//CertTemplate : helper function to create a cert template with a serial number and other required fields
func certTemplate() (*x509.Certificate, error) {
	// generate a random serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.New("failed to generate serial number: " + err.Error())
	}

	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{organization}},
		SignatureAlgorithm:    signatureAlgorithm,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(certificateValidity), // valid for certificateValidity
		BasicConstraintsValid: true,
	}
	return &tmpl, nil
}

//WriteCertPemToFile : write the certPem to a file
func WriteCertPemToFile(certPem string, filePath string) error {
	err := util.FileWriteString(filePath, certPem)
	if err != nil {
		os.RemoveAll(filePath)
		return err
	}
	// change permissions for the file to allow only read-only access to root user
	os.Chmod(filePath, 0400)
	return nil
}

// GenerateCert :
func GenerateCert(cn string) (*x509.Certificate, string, string, error) {
	log.Trace("GenerateCert called")

	if cn == "" {
		return nil, "", "", errors.New("common name cannot be empty")
	}
	// generate a new key-pair
	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", "", errors.New("generating random key: " + err.Error())
	}
	// PEM encode the private key
	rootKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type: privateKeyConst, Bytes: x509.MarshalPKCS1PrivateKey(rootKey),
	})

	rootCertTmpl, err := certTemplate()
	if err != nil {
		return nil, "", "", errors.New("error creating cert template: %v" + err.Error())
	}
	// describe what the certificate will be used for
	rootCertTmpl.IsCA = true
	rootCertTmpl.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	rootCertTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	rootCertTmpl.Issuer = pkix.Name{CommonName: cn}
	rootCertTmpl.Subject.CommonName = cn

	rootCert, err := createCert(rootCertTmpl, rootCertTmpl, &rootKey.PublicKey, rootKey)
	if err != nil {
		return nil, "", "", errors.New("error creating cert " + err.Error())
	}
	rootCertPEM, err := ConvertCertToPem(rootCert)
	if err != nil {
		return nil, "", "", err
	}
	log.Tracef("%s\n", rootCertPEM)

	return rootCert, string(rootKeyPEM), rootCertPEM, nil
}

func createCert(template, parent *x509.Certificate, pub interface{}, parentPriv interface{}) (
	cert *x509.Certificate, err error) {
	log.Tracef("createCert called with \n%v", template)

	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, pub, parentPriv)
	if err != nil {
		return nil, err
	}
	// parse the resulting certificate so we can use it again
	cert, err = x509.ParseCertificate(certDER)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

// ConvertCertToPem :
func ConvertCertToPem(cert *x509.Certificate) (string, error) {
	if cert == nil {
		return "", errors.New("invalid certificate")
	}
	b := pem.EncodeToMemory(&pem.Block{Type: certificateConst, Bytes: cert.Raw})
	certPem := string(b)
	log.Tracef("Cert block \n%s", certPem)
	return certPem, nil
}

//GetCertFromGroup :
func GetCertFromGroup(ipAddress string, port string) (*x509.Certificate, error) {
	log.Tracef("GetCertFromGroup called with %s:%s ", ipAddress, port)
	addr := fmt.Sprintf("%s:%s", ipAddress, port)
	config := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	cstate := conn.ConnectionState()
	log.Tracef("Version: %x", cstate.Version)
	log.Tracef("HandshakeComplete: %t", cstate.HandshakeComplete)
	conn.Close()

	for k, v := range cstate.PeerCertificates {
		log.Tracef("\nCertificate[%d]\n", k)
		log.Tracef("Subject:\t%s\n", v.Subject.CommonName)
		log.Tracef("Issuer:\t\t%s\n", v.Issuer.CommonName)
		log.Tracef("Expires:\t%s\n", v.NotAfter.String())
		log.Tracef("IP Addresses:\t%+v\n", v.IPAddresses)
	}
	if cstate.PeerCertificates == nil {
		return nil, errors.New("could not obtain server certificate chain")
	}
	return cstate.PeerCertificates[0], nil
}
