// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/hectane/go-acl/api"
	"github.com/hpe-storage/common-host-libs/chapi2/cerrors"
	"github.com/hpe-storage/common-host-libs/chapi2/model"
	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/windows/advapi32"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/sys/windows"
)

const (
	defaultFileSystem  = "ntfs"                 // Default file system to use
	configRelativePath = `Nimble Storage\CHAPI` // Path appended to %ProgramData% where we store CHAPI data
	ChapiPortFileName  = "CHAPIPort.txt"        // Client reads in this file to enumerate CHAPI port
	chapiKeyFileName   = "keyfile.txt"          // CHAPI for Windows authentication file
)

var (
	configDir         string // Path to CHAPI configuration files
	chapiPortFilePath string // Path to "CHAPIPort.txt"
	chapiKeyFilePath  string // Path to "keyfile.txt"
	chapiKeyGUID      string // CHAPI authentication key GUID
)

func init() {

	// Enumerate the system's %ProgramData% folder.  If it's unavailable (extremely unlikely),
	// revert to the default value.
	programDataPath := os.Getenv("ProgramData")
	if programDataPath == "" {
		programDataPath = `C:\ProgramData`
		log.Tracef("Unable to enumerate ProgramData folder, using default folder %v", programDataPath)
	}

	// Set configDir path
	configDir = filepath.Join(programDataPath, configRelativePath)

	// Set the CHAPI key access GUID
	chapiKeyGUID = uuid.NewV4().String()

	// Set the CHAPIPort.txt and keyfile.txt paths
	exePath, _ := os.Executable()
	exePath = filepath.Dir(exePath)
	chapiPortFilePath = filepath.Join(exePath, ChapiPortFileName)
	chapiKeyFilePath = filepath.Join(exePath, chapiKeyFileName)
}

//@APIVersion 1.0.0
//@Title GetKeyfile
//@Description Retrieves authentication keyfile location for CHAPI for Windows
//@Accept json
//@Resource /hosts
//@Success 200 {array} Hosts
//@Router /keyfile/ [get]
func GetKeyfile(w http.ResponseWriter, r *http.Request) {
	log.Tracef(">>>>> getKeyfile called, chapiKeyFilePath=%v", chapiKeyFilePath)
	defer log.Trace("<<<<< getKeyfile")

	log.Infof("CHAPI key path - %v", chapiKeyFilePath)
	var chapiResp Response
	chapiResp.Data = model.KeyFileInfo{Path: chapiKeyFilePath}
	json.NewEncoder(w).Encode(chapiResp)
}

// CHAPI for Windows clients send an authorization key in the request header for every CHAPI endpoint
// except for the "keyfile" endpoint.  The "keyfile" endpoint is used to retrieve the location of the
// key file.  The key is then retrieved from that file.  Only processes with administrator access can
// access the key file.  This private helper function validates that the authorization key is set
// correctly in the header.  If the authorization key is not provided, or doesn't match, this routine
// takes care of returning the error to the CHAPI client as well as returning false from this function.
// True is returned if the header is valid.
func validateRequestHeader(w http.ResponseWriter, r *http.Request) bool {

	status := false
	var err error
	if (r == nil) || (r.Header == nil) {
		// If no header was passed in, fail the request
		err = cerrors.NewChapiError(cerrors.Unauthenticated, errorMessageHTTPHeaderNotProvided)
	} else {

		// Check each key/value pair available in the header
		for key, val := range r.Header {

			// Skip all entries except for the CHAPILocalAccessKey key
			if !strings.EqualFold(key, "CHAPILocalAccessKey") || (len(val) != 1) {
				continue
			}

			// If the authorization key matches, return no error
			if (val[0] != "") && (val[0] == chapiKeyGUID) {
				// Valid token provided!
				status = true
			} else {
				// Invalid token provided
				err = cerrors.NewChapiError(cerrors.Unauthenticated, errorMessageInvalidToken+val[0])
			}

			// Break out of loop
			break
		}

		// If token not provided, set error
		if (status == false) && (err == nil) {
			err = cerrors.NewChapiError(cerrors.Unauthenticated, errorMessageTokenNotSupplied)
		}
	}

	// If token authentication failed, return error to caller
	if !status {
		var chapiResp Response
		handleError(w, chapiResp, err, http.StatusUnauthorized)
	}

	// Return true if header is valid, else false
	return status
}

// InitChapiInstanceData is called to initialize our CHAPI instance data.  The CHAPI TCP port is
// provided as input.
func InitChapiInstanceData(port int) (err error) {
	log.Tracef(">>>>> initChapiInstanceData, port=%v", port)
	defer log.Trace("<<<<< initChapiInstanceData")

	// Create the CHAPI port text file
	log.Tracef("CHAPI port file location %v", chapiPortFilePath)
	err = initChapiFileData(chapiPortFilePath, strconv.Itoa(port)+"\r\n")
	if err != nil {
		return err
	}

	// Create the CHAPI key file
	log.Tracef("CHAPI key file location %v", chapiKeyFilePath)
	err = initChapiFileData(chapiKeyFilePath, chapiKeyGUID+"\r\n")
	if err != nil {
		return err
	}

	// Ensure that only processes with Administrator access can read our key file
	err = setAdministratorOnlyAccess(chapiKeyFilePath)
	if err != nil {
		log.Errorf("Unable to write CHAPI key file, err=%v", err)
		return err
	}

	// CHAPI instance data successfully created
	return nil
}

// RemoveChapiInstanceData removes any clea
func RemoveChapiInstanceData() {
	// Remove our CHAPI port and key files during cleanup
	os.Remove(chapiPortFilePath)
	os.Remove(chapiKeyFilePath)
}

// initChapiFileData is a general support routine that will write out the given text string to the
// given file path.
func initChapiFileData(filePath string, fileText string) error {

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		log.Errorf("Unable to create CHAPI file, filePath=%v, err=%v", filePath, err)
		return err
	}
	defer file.Close()

	// Write the file contents
	_, err = file.WriteString(fileText)
	if err != nil {
		log.Errorf("Unable to write CHAPI file, filePath=%v, err=%v", filePath, err)
		return err
	}

	// Success!
	return nil
}

// setAdministratorOnlyAccess sets the ACLs for the given file such that only processes with Administrator
// privileges can access it.
func setAdministratorOnlyAccess(filepath string) error {
	log.Tracef(">>>>> setAdministratorOnlyAccess, filepath=%v", filepath)
	defer log.Trace("<<<<< setAdministratorOnlyAccess")

	// Allocate and initialize our security identifier (SID)
	identAuth := windows.SECURITY_NT_AUTHORITY
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(&identAuth, 2, windows.SECURITY_BUILTIN_DOMAIN_RID, windows.DOMAIN_ALIAS_RID_ADMINS, 0, 0, 0, 0, 0, 0, &sid)

	if err != nil {
		log.Errorf("Unexpected AllocateAndInitializeSid failure, %v", err)
	} else {

		// Initialize an ExplicitAccess structure to allow Administrator full control
		ea := make([]api.ExplicitAccess, 1)
		ea[0].AccessPermissions = syscall.GENERIC_ALL
		ea[0].AccessMode = api.SET_ACCESS
		ea[0].Inheritance = api.NO_INHERITANCE
		ea[0].Trustee.TrusteeForm = api.TRUSTEE_IS_SID
		ea[0].Trustee.TrusteeType = api.TRUSTEE_IS_GROUP
		ea[0].Trustee.Name = (*uint16)(unsafe.Pointer(sid))

		// Create a new ACL for Administrator full control
		var acl windows.Handle
		err = advapi32.SetEntriesInAcl(ea, 0, &acl)

		if err != nil {
			log.Errorf("Unexpected SetEntriesInAcl failure, %v", err)
		} else {
			// Set the Administrator only ACL to our file
			var secInfo uint32 = api.DACL_SECURITY_INFORMATION + api.PROTECTED_DACL_SECURITY_INFORMATION
			err = advapi32.SetNamedSecurityInfo(filepath, api.SE_FILE_OBJECT, secInfo, nil, nil, acl, 0)

			if err != nil {
				log.Errorf("Unexpected SetNamedSecurityInfo failure, %v", err)
			}

			// Free the ACL
			windows.LocalFree(acl)
		}

		// Free the SID
		windows.FreeSid(sid)
	}

	return err
}
