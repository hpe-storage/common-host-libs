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

	// Valid Off-Array params
	map1 := map[string]string{
		usernameKey:    "admin",
		passwordKey:    "admin",
		backendKey:     "1.1.1.1",
		serviceNameKey: "csp-service",
		servicePortKey: "8080",
	}
	// Valid Off-Array credential
	cred1 := &Credentials{
		Username:    "admin",
		Password:    "admin",
		Backend:     "1.1.1.1",
		ServiceName: "csp-service",
		ServicePort: 8080,
	}

	// Valid On-Array params
	map2 := map[string]string{
		usernameKey:    "admin",
		passwordKey:    "admin",
		backendKey:     "1.1.1.1",
		contextPathKey: "/csp",
		servicePortKey: "443",
	}
	// Valid On-Array credential
	cred2 := &Credentials{
		Username:    "admin",
		Password:    "admin",
		Backend:     "1.1.1.1",
		ContextPath: "/csp",
		ServicePort: 443,
	}

	// Invalid params (Missing Port/Off-Array)
	map3 := map[string]string{
		usernameKey:    "admin",
		passwordKey:    "admin",
		backendKey:     "1.1.1.1",
		serviceNameKey: "csp-service",
	}

	// Invalid params (Missing backend)
	map4 := map[string]string{
		usernameKey:    "admin",
		passwordKey:    "admin",
		contextPathKey: "/csp",
		servicePortKey: "443",
	}

	// Invalid params (Missing username)
	map5 := map[string]string{
		passwordKey:    "admin",
		backendKey:     "1.1.1.1",
		contextPathKey: "/csp",
		servicePortKey: "443",
	}

	// Invalid params (Missing password)
	map6 := map[string]string{
		usernameKey:    "admin",
		backendKey:     "1.1.1.1",
		contextPathKey: "/csp",
		servicePortKey: "443",
	}

	// Empty params map
	map7 := map[string]string{}

	tests := []struct {
		name    string
		args    args
		want    *Credentials
		wantErr bool
	}{
		{"Test valid on-array args", args{map1}, cred1, false},
		{"Test valid off-array args", args{map2}, cred2, false},
		{"Test missing/invalid port", args{map3}, nil, true},
		{"Test missing backend", args{map4}, nil, true},
		{"Test missing username", args{map5}, nil, true},
		{"Test missing password", args{map6}, nil, true},
		{"Test empty credentials", args{map7}, nil, true},
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
