// Copyright 2019 Hewlett Packard Enterprise Development LP

package storageprovider

import (
	"reflect"
	"testing"
)

func TestCreateCredentials(t *testing.T) {
	type args struct {
		secrets map[string]string
	}

	// Valid On-Array params
	map1 := map[string]string{
		usernameKey:    "admin",
		passwordKey:    "admin",
		arrayIPKey:     "1.1.1.1",
		serviceNameKey: "csp-service",
		portKey:        "8080",
	}
	// Valid On-Array credential
	cred1 := &Credentials{
		Username:    "admin",
		Password:    "admin",
		ArrayIP:     "1.1.1.1",
		ServiceName: "csp-service",
		Port:        8080,
	}

	// Valid Off-Array params
	map2 := map[string]string{
		usernameKey:    "admin",
		passwordKey:    "admin",
		arrayIPKey:     "1.1.1.1",
		contextPathKey: "/csp",
		portKey:        "443",
	}
	// Valid Off-Array credential
	cred2 := &Credentials{
		Username:    "admin",
		Password:    "admin",
		ArrayIP:     "1.1.1.1",
		ContextPath: "/csp",
		Port:        443,
	}

	// Invalid params (Missing Port)
	map3 := map[string]string{
		usernameKey:    "admin",
		passwordKey:    "admin",
		arrayIPKey:     "1.1.1.1",
		serviceNameKey: "csp-service",
	}

	// Invalid params (Missing serviceName/contextPath)
	map4 := map[string]string{
		usernameKey: "admin",
		passwordKey: "admin",
		arrayIPKey:  "1.1.1.1",
		portKey:     "443",
	}

	tests := []struct {
		name    string
		args    args
		want    *Credentials
		wantErr bool
	}{
		{"Test valid on-array args", args{map1}, cred1, false},
		{"Test valid off-array args", args{map2}, cred2, false},
		{"Test missing/invalid port", args{map3}, nil, true},
		{"Test missing serviceName/contextPath", args{map4}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateCredentials(tt.args.secrets)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateCredentials() = %v, want %v", got, tt.want)
			}
		})
	}
}
