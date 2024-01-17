// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package cerrors

import (
	"fmt"
	"strconv"

	log "github.com/hpe-storage/common-host-libs/logger"
)

type ChapiErrorCode uint32

const (
	OK                ChapiErrorCode = 0
	Canceled          ChapiErrorCode = 1
	Unknown           ChapiErrorCode = 2
	InvalidArgument   ChapiErrorCode = 3
	NotFound          ChapiErrorCode = 4
	AlreadyExists     ChapiErrorCode = 5
	PermissionDenied  ChapiErrorCode = 6
	ResourceExhausted ChapiErrorCode = 7
	Aborted           ChapiErrorCode = 8
	Unimplemented     ChapiErrorCode = 9
	Internal          ChapiErrorCode = 10
	DataLoss          ChapiErrorCode = 11
	Unauthenticated   ChapiErrorCode = 12
	Timeout           ChapiErrorCode = 13
	ConnectionFailed  ChapiErrorCode = 14
	_maxCode          ChapiErrorCode = 15
)

const (
	errorMessageInvalidInputParameters = "invalid input parameters"
)

type ChapiError struct {
	Code ChapiErrorCode `json:"code"`
	Text string         `json:"text,omitempty"`
}

// NewChapiError takes an array of objects and returns a pointer to a ChapiError object.  The
// following input parameters, in any order, are supported:
//     ChapiError     - ChapiError object
//     error          - All other error objects
//     ChapiErrorCode - CHAPI error code
//     string         - CHAPI error text
// This routine parses the input data to create and return a new ChapiError object
func NewChapiError(args ...interface{}) *ChapiError {

	// These are the optional parameters we support
	var chapiError *ChapiError
	var otherError *error
	errorCode := _maxCode
	errorMessage := ""

	// Parse the input parameters and populate local variables
	for _, arg := range args {
		switch arg.(type) {
		case ChapiErrorCode:
			errorCode = arg.(ChapiErrorCode)
		case string:
			errorMessage = arg.(string)
		case ChapiError:
			err := arg.(ChapiError)
			chapiError = &err
		case *ChapiError:
			chapiError = arg.(*ChapiError)
		case error:
			err := arg.(error)
			otherError = &err
		}
	}

	// Create a new initial ChapiError object
	err := &ChapiError{Code: _maxCode, Text: ""}

	// Populate the ChapiError Text property
	if chapiError != nil {
		err = chapiError
	} else if otherError != nil {
		err.Text = (*otherError).Error()
	} else if errorMessage != "" {
		err.Text = errorMessage
	}

	// Populate the ChapiError Code property
	if errorCode < _maxCode {
		err.Code = errorCode
	}

	// If neither an error message or an error code were provided, fail with generic error
	if (err.Code == _maxCode) && (err.Text == "") {
		return &ChapiError{Code: Internal, Text: errorMessageInvalidInputParameters}
	}

	// Handle condition where ChapiError Code property is still empty
	if err.Code == _maxCode {
		err.Code = Unknown
	}

	// Handle condition where ChapiError text property is still empty
	if err.Text == "" {
		err.Text = err.Code.String()
	}

	return err
}

func NewChapiErrorf(c ChapiErrorCode, format string, a ...interface{}) *ChapiError {
	return &ChapiError{Code: c, Text: fmt.Sprintf(format, a...)}
}

func (e *ChapiError) Error() string {
	return fmt.Sprintf("status: %d msg: %s", e.Code, e.Text)
}

func (e *ChapiError) LogAndError() ChapiError {
	log.Errorln(e.Error())
	return *e
}

// Code returns the status code contained in ChapiError
func (e *ChapiError) ErrorCode() ChapiErrorCode {
	if e == nil {
		return OK
	}
	return e.Code
}

// ErrorText returns the text contained in ChapiError
func (e *ChapiError) ErrorText() string {
	if e == nil {
		return ""
	}
	return e.Text
}

func (c ChapiErrorCode) String() string {
	switch c {
	case OK:
		return "OK"
	case Canceled:
		return "Canceled"
	case Unknown:
		return "Unknown"
	case InvalidArgument:
		return "InvalidArgument"
	case NotFound:
		return "NotFound"
	case AlreadyExists:
		return "AlreadyExists"
	case PermissionDenied:
		return "PermissionDenied"
	case ResourceExhausted:
		return "ResourceExhausted"
	case Aborted:
		return "Aborted"
	case Unimplemented:
		return "Unimplemented"
	case Internal:
		return "Internal"
	case DataLoss:
		return "DataLoss"
	case Unauthenticated:
		return "Unauthenticated"
	case Timeout:
		return "Timeout"
	case ConnectionFailed:
		return "ConnectionFailed"
	default:
		return "Code(" + strconv.FormatInt(int64(c), 10) + ")"
	}
}
