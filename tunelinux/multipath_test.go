package tunelinux

import (
	"errors"
	"testing"
)

func TestParseMultipathDevices(t *testing.T) {
	tests := []struct {
		name    string
		out     string
		wantLen int
		wantErr error
	}{
		{
			name: "empty output",
			out:  "",
		},
		{
			name: "whitespace output",
			out:  " \n\t ",
		},
		{
			name:    "timeout output",
			out:     "timeout",
			wantErr: ErrMultipathTimeout,
		},
		{
			name:    "receiving packet output",
			out:     "multipathd: receiving packet failed",
			wantErr: ErrMultipathTimeout,
		},
		{
			name:    "invalid JSON",
			out:     "{not-json}",
			wantErr: errAny,
		},
		{
			name:    "trailing JSON garbage",
			out:     `{"maps":[]} trailing`,
			wantErr: errAny,
		},
		{
			name: "valid no maps",
			out:  `{"maps":[]}`,
		},
		{
			name: "valid devices",
			out: `{
				"maps": [
					{"name":"healthy","vend":"Nimble","paths":2},
					{"name":"unhealthy","vend":"Nimble","paths":0,"path_faults":1},
					{"name":"orphan","queueing":"off","features":"0"},
					{"name":"unsupported","vend":"Other","paths":2}
				]
			}`,
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devices, err := parseMultipathDevices(tt.out)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("parseMultipathDevices() error = %v, want nil", err)
			}
			if tt.wantErr == errAny && err == nil {
				t.Fatalf("parseMultipathDevices() error = nil, want error")
			}
			if tt.wantErr != nil && tt.wantErr != errAny && !errors.Is(err, tt.wantErr) {
				t.Fatalf("parseMultipathDevices() error = %v, want %v", err, tt.wantErr)
			}
			if len(devices) != tt.wantLen {
				t.Fatalf("parseMultipathDevices() returned %d devices, want %d", len(devices), tt.wantLen)
			}
		})
	}
}

func TestParseMultipathDevicesClassifiesUnhealthyAndOrphan(t *testing.T) {
	devices, err := parseMultipathDevices(`{
		"maps": [
			{"name":"healthy","vend":"Nimble","paths":2},
			{"name":"unhealthy","vend":"Nimble","paths":0,"path_faults":1},
			{"name":"orphan","queueing":"off","features":"0"}
		]
	}`)
	if err != nil {
		t.Fatalf("parseMultipathDevices() error = %v, want nil", err)
	}
	if len(devices) != 3 {
		t.Fatalf("parseMultipathDevices() returned %d devices, want 3", len(devices))
	}
	if devices[0].IsUnhealthy {
		t.Fatalf("device %q IsUnhealthy = true, want false", devices[0].Name)
	}
	if !devices[1].IsUnhealthy {
		t.Fatalf("device %q IsUnhealthy = false, want true", devices[1].Name)
	}
	if !devices[2].IsUnhealthy {
		t.Fatalf("device %q IsUnhealthy = false, want true", devices[2].Name)
	}
}

var errAny = errors.New("any error")
