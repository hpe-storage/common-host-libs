// Copyright 2026 Hewlett Packard Enterprise Development LP
package linux

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ESC-18251: enablement must be detectable from the [Install] symlinks alone,
// without any D-Bus call to a possibly busy PID 1.
func TestIsSystemdUnitEnabled(t *testing.T) {
	root := t.TempDir()
	hostRoot := t.TempDir()

	// iscsid.service enabled via multi-user.target.wants on the host root
	wants := filepath.Join(hostRoot, "multi-user.target.wants")
	if err := os.MkdirAll(wants, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wants, "iscsid.service"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	// multipathd.service enabled via sysinit.target.requires in the container root
	requires := filepath.Join(root, "sysinit.target.requires")
	if err := os.MkdirAll(requires, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(requires, "multipathd.service"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	dirs := []string{root, hostRoot}

	tests := []struct {
		unit string
		want bool
	}{
		{"iscsid.service", true},
		{"multipathd.service", true},
		{"nfs-utils.service", false},
	}
	for _, tt := range tests {
		if got := isSystemdUnitEnabled(tt.unit, dirs); got != tt.want {
			t.Errorf("isSystemdUnitEnabled(%q) = %v, want %v", tt.unit, got, tt.want)
		}
	}

	if isSystemdUnitEnabled("iscsid.service", []string{filepath.Join(root, "does-not-exist")}) {
		t.Errorf("expected false for a missing unit directory")
	}
}

// ESC-18251 review R1: on SLES/RHEL iscsid is socket-activated, so an enabled
// iscsid.socket (typically under /usr/lib/systemd/system/sockets.target.wants via
// presets) must count as "already enabled" and suppress the systemctl call.
func TestSystemdEnabledUnitsCoverSocketActivation(t *testing.T) {
	vendorDir := t.TempDir()
	wants := filepath.Join(vendorDir, "sockets.target.wants")
	if err := os.MkdirAll(wants, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wants, "iscsid.socket"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	found := false
	for _, unit := range systemdEnabledUnits[iscsi] {
		if isSystemdUnitEnabled(unit, []string{vendorDir}) {
			found = true
		}
	}
	if !found {
		t.Errorf("iscsid.socket enabled via sockets.target.wants should satisfy the iscsi enablement check")
	}

	if _, ok := systemdEnabledUnits[multipath]; !ok {
		t.Errorf("multipath must have an enablement unit list")
	}
}

// ESC-18251: the SUSE multipathd workaround must not trigger a daemon-reload on
// hosts where it has already been applied.
func TestSuseWorkaroundNeeded(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"pristine unit", "[Unit]\nConditionKernelCommandLine=!multipath=off\nConditionKernelCommandLine=!nompath\n", true},
		{"only nompath active", "[Unit]\n#ConditionKernelCommandLine=!multipath=off\nConditionKernelCommandLine=!nompath\n", true},
		{"already patched", "[Unit]\n#ConditionKernelCommandLine=!multipath=off\n#ConditionKernelCommandLine=!nompath\n", false},
		{"no condition at all", "[Unit]\nDescription=Device-Mapper Multipath Device Controller\n", false},
	}
	for _, tt := range tests {
		if got := suseWorkaroundNeeded(tt.content); got != tt.want {
			t.Errorf("%s: suseWorkaroundNeeded = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// ESC-18251 fix verification F4/N3: the warning must name the real failure.
func TestClassifySystemctlError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want SystemctlErrorKind
	}{
		{"nil", nil, ""},
		{"connect refused from pod pid ns", errors.New("command systemctl failed with rc=1 err=Failed to connect to bus: No data available"), SystemctlErrConnect},
		{"client timeout", errors.New("command systemctl failed with rc=1 err=Failed to execute operation: Connection timed out"), SystemctlErrTimeout},
		{"exec wrapper timeout", errors.New("command systemctl with pid: 42 killed as timeout of 180 seconds reached"), SystemctlErrTimeout},
		{"chroot no-op", errors.New("systemctl start iscsid: Running in chroot, ignoring command 'start'"), SystemctlErrChroot},
		{"unit failure", errors.New("command systemctl failed with rc=5 err=Unit foo.service not found"), SystemctlErrUnit},
	}
	for _, tt := range tests {
		if got := ClassifySystemctlError(tt.err); got != tt.want {
			t.Errorf("%s: ClassifySystemctlError = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// ESC-18251 fix verification F3/N2: with hostPID=true and no SYSTEMD_IGNORE_CHROOT,
// systemctl prints "Running in chroot, ignoring command" and exits 0 without
// touching PID 1. That must surface as an error, never as success.
func TestRunSystemctlDetectsChrootNoop(t *testing.T) {
	orig := systemctlExec
	t.Cleanup(func() { systemctlExec = orig })

	var gotArgs []string
	systemctlExec = func(cmd string, args []string) (string, int, error) {
		gotArgs = append([]string{cmd}, args...)
		return "Running in chroot, ignoring command 'daemon-reload'\n", 0, nil
	}
	_, _, err := runSystemctl("daemon-reload")
	if err == nil {
		t.Fatalf("expected an error for the chroot no-op, got nil")
	}
	if ClassifySystemctlError(err) != SystemctlErrChroot {
		t.Errorf("expected chroot classification, got %q (%v)", ClassifySystemctlError(err), err)
	}
	if strings.Join(gotArgs, " ") != "systemctl daemon-reload" {
		t.Errorf("unexpected invocation %v", gotArgs)
	}

	systemctlExec = func(string, []string) (string, int, error) { return "", 0, nil }
	if _, _, err := runSystemctl("start", "iscsid"); err != nil {
		t.Errorf("clean success must not be an error: %v", err)
	}

	systemctlExec = func(string, []string) (string, int, error) {
		return "", 1, errors.New("command systemctl failed with rc=1 err=Failed to connect to bus: No data available")
	}
	_, _, err = runSystemctl("start", "iscsid")
	if ClassifySystemctlError(err) != SystemctlErrConnect {
		t.Errorf("expected connect classification, got %q (%v)", ClassifySystemctlError(err), err)
	}
}
