// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package cerrors

import (
	"errors"
	"testing"
)

func TestNewChapiError(t *testing.T) {

	var err *ChapiError
	errorMessage := "this is a simple test error message"
	errorTemplate := `Invalid ChapiError, received %v:"%v", expected %v:"%v"`

	err = NewChapiError(DataLoss, errorMessage)
	if (err.Code != DataLoss) || (err.Text != errorMessage) {
		t.Errorf(errorTemplate, err.Code, err.Text, DataLoss, errorMessage)
	}

	err = NewChapiError(DataLoss)
	if (err.Code != DataLoss) || (err.Text != err.Code.String()) {
		t.Errorf(errorTemplate, err.Code, err.Text, DataLoss, err.Code.String())
	}

	err = NewChapiError(errorMessage)
	if (err.Code != Unknown) || (err.Text != errorMessage) {
		t.Errorf(errorTemplate, err.Code, err.Text, Unknown, errorMessage)
	}

	err = NewChapiError(errors.New(errorMessage))
	if (err.Code != Unknown) || (err.Text != errorMessage) {
		t.Errorf(errorTemplate, err.Code, err.Text, Unknown, errorMessage)
	}

	err = NewChapiError(Unauthenticated, errors.New(errorMessage))
	if (err.Code != Unauthenticated) || (err.Text != errorMessage) {
		t.Errorf(errorTemplate, err.Code, err.Text, Unauthenticated, errorMessage)
	}

	err = NewChapiError(NewChapiError(errorMessage))
	if (err.Code != Unknown) || (err.Text != errorMessage) {
		t.Errorf(errorTemplate, err.Code, err.Text, Unknown, errorMessage)
	}

	err = NewChapiError(NewChapiError(errorMessage), ResourceExhausted)
	if (err.Code != ResourceExhausted) || (err.Text != errorMessage) {
		t.Errorf(errorTemplate, err.Code, err.Text, ResourceExhausted, errorMessage)
	}

	err = NewChapiError()
	if (err.Code != Internal) || (err.Text != errorMessageInvalidInputParameters) {
		t.Errorf(errorTemplate, err.Code, err.Text, Internal, errorMessageInvalidInputParameters)
	}
}
