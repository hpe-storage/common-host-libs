package tunelinux

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hpe-storage/common-host-libs/linux"
)

func TestReconfigureMultipathdIfConfigNotApplied(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "multipath.conf")
	appliedHashPath := filepath.Join(tempDir, "run", "multipath-config.sha256")
	config := []byte("defaults {\n\tfind_multipaths no\n}\n")
	if err := os.WriteFile(configPath, config, 0600); err != nil {
		t.Fatalf("write multipath config: %v", err)
	}
	reconfigureCalls := 0
	reconfigure := func() (string, error) {
		reconfigureCalls++
		return "ok", nil
	}

	if err := reconfigureMultipathdIfConfigNotApplied(configPath, appliedHashPath, reconfigure); err != nil {
		t.Fatalf("initial reconfigureMultipathdIfConfigNotApplied() error = %v, want nil", err)
	}
	if reconfigureCalls != 1 {
		t.Fatalf("reconfigure called %d times without applied state, want 1", reconfigureCalls)
	}

	if err := reconfigureMultipathdIfConfigNotApplied(configPath, appliedHashPath, reconfigure); err != nil {
		t.Fatalf("repeat reconfigureMultipathdIfConfigNotApplied() error = %v, want nil", err)
	}
	if reconfigureCalls != 1 {
		t.Fatalf("reconfigure called %d times for applied content, want 1", reconfigureCalls)
	}

	if err := os.WriteFile(configPath, append(config, []byte("devices {}\n")...), 0600); err != nil {
		t.Fatalf("update multipath config: %v", err)
	}
	if err := reconfigureMultipathdIfConfigNotApplied(configPath, appliedHashPath, reconfigure); err != nil {
		t.Fatalf("changed reconfigureMultipathdIfConfigNotApplied() error = %v, want nil", err)
	}
	if reconfigureCalls != 2 {
		t.Fatalf("reconfigure called %d times for changed content, want 2", reconfigureCalls)
	}
}

func TestReconfigureMultipathdIfConfigNotAppliedRetriesAfterFailure(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "multipath.conf")
	appliedHashPath := filepath.Join(tempDir, "run", "multipath-config.sha256")
	if err := os.WriteFile(configPath, []byte("defaults {}\n"), 0600); err != nil {
		t.Fatalf("write multipath config: %v", err)
	}
	reconfigureCalls := 0
	reconfigure := func() (string, error) {
		reconfigureCalls++
		if reconfigureCalls == 1 {
			return "", errors.New("reconfigure failed")
		}
		return "ok", nil
	}

	err := reconfigureMultipathdIfConfigNotApplied(configPath, appliedHashPath, reconfigure)
	if err == nil {
		t.Fatal("first reconfigureMultipathdIfConfigNotApplied() error = nil, want error")
	}
	// ESC-18251 Q8: the failure must be identifiable so node-init can continue.
	if !errors.Is(err, ErrMultipathdReconfigure) {
		t.Fatalf("first error = %v, want it to wrap ErrMultipathdReconfigure", err)
	}
	if err := reconfigureMultipathdIfConfigNotApplied(configPath, appliedHashPath, reconfigure); err != nil {
		t.Fatalf("retry reconfigureMultipathdIfConfigNotApplied() error = %v, want nil", err)
	}
	if reconfigureCalls != 2 {
		t.Fatalf("reconfigure called %d times after initial failure, want 2", reconfigureCalls)
	}
}

func TestReconfigureMultipathdIfConfigNotAppliedReturnsReadError(t *testing.T) {
	tempDir := t.TempDir()
	err := reconfigureMultipathdIfConfigNotApplied(filepath.Join(tempDir, "missing.conf"), filepath.Join(tempDir, "state", "hash"), func() (string, error) {
		t.Fatal("reconfigure called when config could not be read")
		return "", nil
	})
	if err == nil {
		t.Fatal("reconfigureMultipathdIfConfigNotApplied() error = nil, want read error")
	}
	if errors.Is(err, ErrMultipathdReconfigure) {
		t.Fatalf("a read error must not be classified as a reconfigure failure: %v", err)
	}
}

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

func withStubbedExec(t *testing.T, fn func(cmd string, args []string, timeout int) (string, int, error)) {
	t.Helper()
	origExec := execCommandOutput
	execCommandOutput = fn
	t.Cleanup(func() {
		execCommandOutput = origExec
	})
}

var errExecTimeout = fmt.Errorf("command multipathd killed as timeout of 60 seconds reached")

func TestGetMultipathDevicesReturnsTimeoutError(t *testing.T) {
	calls := 0
	withStubbedExec(t, func(cmd string, args []string, timeout int) (string, int, error) {
		calls++
		if timeout != linux.MultipathdCommandTimeout() {
			t.Fatalf("timeout = %d, want %d", timeout, linux.MultipathdCommandTimeout())
		}
		return "", 888, errExecTimeout
	})

	devices, err := GetMultipathDevices()
	if !errors.Is(err, ErrMultipathTimeout) {
		t.Fatalf("GetMultipathDevices() error = %v, want ErrMultipathTimeout", err)
	}
	if devices != nil {
		t.Fatalf("GetMultipathDevices() devices = %v, want nil", devices)
	}
	if calls != 1 {
		t.Fatalf("execCommandOutput called %d times, want 1", calls)
	}
}

func TestGetMultipathDevicesReturnsNonTimeoutError(t *testing.T) {
	calls := 0
	withStubbedExec(t, func(cmd string, args []string, timeout int) (string, int, error) {
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
	withStubbedExec(t, func(cmd string, args []string, timeout int) (string, int, error) {
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
