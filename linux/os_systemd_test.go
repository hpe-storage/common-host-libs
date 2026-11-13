// (c) Copyright 2026 Hewlett Packard Enterprise Development LP
package linux

import (
	"errors"
	"testing"
)

// ESC-18251: `systemctl is-active` must gate the systemd interactions. Only the
// literal "active" answer counts; "inactive", "activating", "unknown" and a
// failed query all mean "go ahead and configure".
func TestIsSystemdUnitActive(t *testing.T) {
	orig := systemctlExec
	t.Cleanup(func() { systemctlExec = orig })

	tests := []struct {
		name string
		out  string
		rc   int
		err  error
		want bool
	}{
		{"active", "active\n", 0, nil, true},
		{"inactive", "inactive\n", 3, errors.New("command systemctl failed with rc=3 err=inactive"), false},
		{"activating", "activating\n", 0, nil, false},
		{"unknown unit", "inactive\n", 4, errors.New("command systemctl failed with rc=4 err=inactive"), false},
		{"bus unreachable", "", 1, errors.New("Failed to connect to bus: No data available"), false},
	}
	for _, tt := range tests {
		var gotArgs []string
		systemctlExec = func(cmd string, args []string) (string, int, error) {
			gotArgs = append([]string{cmd}, args...)
			return tt.out, tt.rc, tt.err
		}
		if got := IsSystemdUnitActive("iscsid.service"); got != tt.want {
			t.Errorf("%s: IsSystemdUnitActive = %v, want %v", tt.name, got, tt.want)
		}
		if len(gotArgs) != 3 || gotArgs[0] != "systemctl" || gotArgs[1] != "is-active" || gotArgs[2] != "iscsid.service" {
			t.Errorf("%s: unexpected invocation %v", tt.name, gotArgs)
		}
	}
}

func TestSystemdServiceUnit(t *testing.T) {
	if u, err := systemdServiceUnit(iscsi); err != nil || u != "iscsid.service" {
		t.Errorf("iscsi -> %q, %v", u, err)
	}
	if u, err := systemdServiceUnit(multipath); err != nil || u != "multipathd.service" {
		t.Errorf("multipath -> %q, %v", u, err)
	}
	if _, err := systemdServiceUnit("nfs"); err == nil {
		t.Errorf("unknown service type must be an error")
	}
}
