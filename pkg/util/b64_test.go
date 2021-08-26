// Copyright 2019 Hewlett Packard Enterprise Development LP

package util

import (
	"testing"
)

func TestDecodeBase64Credential(t *testing.T) {
	type args struct {
		b64data string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"Test valid b64 credential", args{"YWRtaW4="}, "admin", false},
		{"Test valid plain credential", args{"admin"}, "admin", false},
		{"Test valid b64 credential", args{"TmltMTIzQm9saQ=="}, "Nim123Boli", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeBase64Credential(tt.args.b64data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeBase64Credential() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DecodeBase64Credential() = %v, want %v", got, tt.want)
			}
		})
	}
}
