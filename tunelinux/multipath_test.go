package tunelinux

import (
	"errors"
	"fmt"
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

func TestParseMultipathDevicesValidJSONWithQTimeoutsNotTimeout(t *testing.T) {
	out := `{
		"major_version": 0,
		"minor_version": 1,
		"maps": [{
			"name": "mpatha",
			"uuid": "360002ac00000000000005d8d0002b114",
			"sysfs": "dm-2",
			"queueing": "18 chk",
			"paths": 2,
			"features": "1 queue_if_no_path",
			"path_faults": 0,
			"vend": "3PARdata",
			"total_q_time": 0,
			"q_timeouts": 0,
			"path_groups": [{"paths": [{"dev": "sdb"}, {"dev": "sdd"}]}]
		}]
	}`
	devices, err := parseMultipathDevices(out)
	if errors.Is(err, ErrMultipathTimeout) {
		t.Fatalf("parseMultipathDevices() misclassified valid JSON as ErrMultipathTimeout")
	}
	if err != nil {
		t.Fatalf("parseMultipathDevices() error = %v, want nil", err)
	}
	if len(devices) != 1 {
		t.Fatalf("parseMultipathDevices() returned %d devices, want 1", len(devices))
	}
	if devices[0].Name != "mpatha" {
		t.Fatalf("parseMultipathDevices() device name = %q, want %q", devices[0].Name, "mpatha")
	}
}

func withStubbedExec(t *testing.T, fn func(cmd string, args []string) (string, int, error)) {
	t.Helper()
	origExec := execCommandOutput
	origSleep := multipathTimeoutRetrySleep
	execCommandOutput = fn
	multipathTimeoutRetrySleep = 0
	t.Cleanup(func() {
		execCommandOutput = origExec
		multipathTimeoutRetrySleep = origSleep
	})
}

var errExecTimeout = fmt.Errorf("command multipathd killed as timeout of 60 seconds reached")

func TestGetMultipathDevicesRetriesOnExecTimeout(t *testing.T) {
	calls := 0
	withStubbedExec(t, func(cmd string, args []string) (string, int, error) {
		calls++
		if calls < multipathTimeoutMaxTries {
			return "", 888, errExecTimeout
		}
		return `{"maps":[{"name":"healthy","vend":"Nimble","paths":2}]}`, 0, nil
	})

	devices, err := GetMultipathDevices()
	if err != nil {
		t.Fatalf("GetMultipathDevices() error = %v, want nil", err)
	}
	if calls != multipathTimeoutMaxTries {
		t.Fatalf("execCommandOutput called %d times, want %d", calls, multipathTimeoutMaxTries)
	}
	if len(devices) != 1 {
		t.Fatalf("GetMultipathDevices() returned %d devices, want 1", len(devices))
	}
}

func TestGetMultipathDevicesExhaustsRetriesOnExecTimeout(t *testing.T) {
	calls := 0
	withStubbedExec(t, func(cmd string, args []string) (string, int, error) {
		calls++
		return "", 888, errExecTimeout
	})

	devices, err := GetMultipathDevices()
	if !errors.Is(err, ErrMultipathTimeout) {
		t.Fatalf("GetMultipathDevices() error = %v, want ErrMultipathTimeout", err)
	}
	if devices != nil {
		t.Fatalf("GetMultipathDevices() devices = %v, want nil", devices)
	}
	if calls != multipathTimeoutMaxTries {
		t.Fatalf("execCommandOutput called %d times, want %d", calls, multipathTimeoutMaxTries)
	}
}

func TestGetMultipathDevicesDoesNotRetryOnNonTimeoutError(t *testing.T) {
	calls := 0
	withStubbedExec(t, func(cmd string, args []string) (string, int, error) {
		calls++
		return "", 1, fmt.Errorf("multipathd: command not found")
	})

	_, err := GetMultipathDevices()
	if err == nil {
		t.Fatal("GetMultipathDevices() error = nil, want error")
	}
	if errors.Is(err, ErrMultipathTimeout) {
		t.Fatalf("GetMultipathDevices() error = %v, want non-timeout error", err)
	}
	if calls != 1 {
		t.Fatalf("execCommandOutput called %d times, want 1", calls)
	}
}

func TestGetMultipathDevicesValidJSONWithQTimeoutsNoRetry(t *testing.T) {
	calls := 0
	withStubbedExec(t, func(cmd string, args []string) (string, int, error) {
		calls++
		return `{"maps":[{"name":"mpatha","vend":"3PARdata","paths":2,"q_timeouts":0,"total_q_time":0}]}`, 0, nil
	})

	devices, err := GetMultipathDevices()
	if err != nil {
		t.Fatalf("GetMultipathDevices() error = %v, want nil", err)
	}
	if calls != 1 {
		t.Fatalf("execCommandOutput called %d times, want 1", calls)
	}
	if len(devices) != 1 {
		t.Fatalf("GetMultipathDevices() returned %d devices, want 1", len(devices))
	}
}
