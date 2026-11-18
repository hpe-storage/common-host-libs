// (c) Copyright 2026 Hewlett Packard Enterprise Development LP
package linux

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ESC-18251: `systemctl is-active` must gate the systemd interactions. Only the
// literal "active" answer counts as active; "inactive", "activating" and a
// query that fails quickly are all "not active". A query that PID 1 did not
// answer is covered by TestSystemdNoAnswerLeavesUnitUntouched.
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

// ESC-18251: the socket check must only accept a *listening* socket with the
// exact name, and must never need systemctl when it finds one.
func TestIsServiceSocketListening(t *testing.T) {
	origPath, origExec := procNetUnixPath, systemctlExec
	t.Cleanup(func() { procNetUnixPath, systemctlExec = origPath, origExec })

	const header = "Num       RefCount Protocol Flags    Type St Inode Path\n"
	const iscsidListen = "0000000000000000: 00000002 00000000 00010000 0001 01 22495 @ISCSIADM_ABSTRACT_NAMESPACE\n"
	const mpathListen = "0000000000000000: 00000002 00000000 00010000 0001 01 18913 @/org/kernel/linux/storage/multipathd\n"
	// an accepted/connected socket carries the name but is not listening
	const iscsidConnected = "0000000000000000: 00000003 00000000 00000000 0001 03 22999 @ISCSIADM_ABSTRACT_NAMESPACE\n"
	const unrelated = "0000000000000000: 00000002 00000000 00010000 0001 01 1234 /run/systemd/private\n" +
		"0000000000000000: 00000002 00000000 00000000 0002 01 5678\n"

	tests := []struct {
		name    string
		content string
		unit    string
		want    bool
	}{
		{"iscsid listening", header + unrelated + iscsidListen, "iscsid.service", true},
		{"multipathd listening", header + mpathListen + unrelated, "multipathd.service", true},
		{"iscsid absent", header + unrelated + mpathListen, "iscsid.service", false},
		{"iscsid connected only", header + iscsidConnected, "iscsid.service", false},
		{"name is only a prefix", header + strings.TrimSuffix(iscsidListen, "\n") + "_X\n", "iscsid.service", false},
		{"unknown unit", header + iscsidListen, "nfs.service", false},
		{"empty file", "", "iscsid.service", false},
	}
	for _, tt := range tests {
		procNetUnixPath = filepath.Join(t.TempDir(), "unix")
		if err := os.WriteFile(procNetUnixPath, []byte(tt.content), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := IsServiceSocketListening(tt.unit); got != tt.want {
			t.Errorf("%s: IsServiceSocketListening = %v, want %v", tt.name, got, tt.want)
		}
	}

	procNetUnixPath = filepath.Join(t.TempDir(), "missing")
	if IsServiceSocketListening("iscsid.service") {
		t.Errorf("unreadable %s must not report listening", procNetUnixPath)
	}

	// skipSystemdCalls: socket found -> systemctl is never called.
	calls := 0
	systemctlExec = func(cmd string, args []string) (string, int, error) {
		calls++
		return "active\n", 0, nil
	}
	procNetUnixPath = filepath.Join(t.TempDir(), "unix")
	if err := os.WriteFile(procNetUnixPath, []byte(header+iscsidListen), 0o644); err != nil {
		t.Fatal(err)
	}
	if !skipSystemdCalls("iscsid.service") || calls != 0 {
		t.Errorf("socket present: want skip without systemctl, got %d systemctl calls", calls)
	}
	// socket not found -> falls back to systemctl is-active.
	if err := os.WriteFile(procNetUnixPath, []byte(header), 0o644); err != nil {
		t.Fatal(err)
	}
	if !skipSystemdCalls("iscsid.service") || calls != 1 {
		t.Errorf("socket absent: want fallback to systemctl is-active, got %d systemctl calls", calls)
	}
}

// fakeSystemctl replaces systemctlExec with fn, hides the daemon sockets so the
// systemctl path is taken, and records the systemctl verbs that were run.
func fakeSystemctl(t *testing.T, fn func(verb string) (string, int, error)) *[]string {
	t.Helper()
	origExec, origAfter, origPath := systemctlExec, systemctlNoAnswerAfter, procNetUnixPath
	t.Cleanup(func() { systemctlExec, systemctlNoAnswerAfter, procNetUnixPath = origExec, origAfter, origPath })
	procNetUnixPath = filepath.Join(t.TempDir(), "missing")
	verbs := &[]string{}
	systemctlExec = func(cmd string, args []string) (string, int, error) {
		*verbs = append(*verbs, args[0])
		return fn(args[0])
	}
	return verbs
}

// ESC-18251: an is-active query that PID 1 did not answer is not "inactive".
// The unit must be left untouched: no daemon-reload, no enable.
func TestSystemdNoAnswerLeavesUnitUntouched(t *testing.T) {
	verbs := fakeSystemctl(t, func(verb string) (string, int, error) {
		time.Sleep(40 * time.Millisecond)
		return "unknown\n", 3, errors.New("command systemctl failed with rc=3 err=unknown")
	})
	systemctlNoAnswerAfter = 20 * time.Millisecond

	if got := systemdUnitState("iscsid.service"); got != unitNoAnswer {
		t.Fatalf("systemdUnitState = %v, want unitNoAnswer", got)
	}
	if IsSystemdUnitActive("iscsid.service") {
		t.Errorf("a query without an answer must not report active")
	}
	*verbs = nil
	if err := enableSystemdService(nil, iscsi); err != nil {
		t.Errorf("enableSystemdService = %v, want nil", err)
	}
	if len(*verbs) != 1 || (*verbs)[0] != "is-active" {
		t.Errorf("want only the is-active query, got %v", *verbs)
	}
}

// A quick "not active" answer still leads to daemon-reload and enable.
func TestEnableSystemdServiceWhenNotActive(t *testing.T) {
	verbs := fakeSystemctl(t, func(verb string) (string, int, error) {
		if verb == "is-active" {
			return "inactive\n", 3, errors.New("command systemctl failed with rc=3 err=inactive")
		}
		return "", 0, nil
	})
	if got := systemdUnitState("iscsid.service"); got != unitNotActive {
		t.Fatalf("systemdUnitState = %v, want unitNotActive", got)
	}
	*verbs = nil
	if err := enableSystemdService(nil, iscsi); err != nil {
		t.Fatalf("enableSystemdService = %v, want nil", err)
	}
	want := []string{"is-active", "daemon-reload", "enable"}
	if strings.Join(*verbs, ",") != strings.Join(want, ",") {
		t.Errorf("systemctl verbs = %v, want %v", *verbs, want)
	}
}

// ESC-18251: a systemctl call that times out is not fatal and nothing more is
// asked of PID 1; any other failure is still returned.
func TestEnableSystemdServiceTimeoutNotFatal(t *testing.T) {
	reloadErr := errors.New("command systemctl failed with rc=1 err=Failed to execute operation: Connection timed out")
	verbs := fakeSystemctl(t, func(verb string) (string, int, error) {
		switch verb {
		case "is-active":
			return "inactive\n", 3, errors.New("command systemctl failed with rc=3 err=inactive")
		case "daemon-reload":
			return "", 1, reloadErr
		}
		return "", 0, nil
	})
	if err := enableSystemdService(nil, iscsi); err != nil {
		t.Errorf("timed-out daemon-reload: enableSystemdService = %v, want nil", err)
	}
	if strings.Join(*verbs, ",") != "is-active,daemon-reload" {
		t.Errorf("enable must not be attempted after a timeout, got %v", *verbs)
	}

	reloadErr = errors.New("command systemctl failed with rc=1 err=Failed to execute operation: Access denied")
	if err := enableSystemdService(nil, iscsi); err == nil {
		t.Errorf("a failure that is not a timeout must still be returned")
	}

	for msg, want := range map[string]bool{
		"command systemctl failed with rc=1 err=Failed to start iscsid.service: Connection timed out": true,
		"error command systemctl with pid: 42 killed as timeout of 60 seconds reached":                true,
		"command systemctl failed with rc=5 err=Failed to start iscsid.service: Unit not found.":      false,
	} {
		if got := isSystemctlTimeout(errors.New(msg)); got != want {
			t.Errorf("isSystemctlTimeout(%q) = %v, want %v", msg, got, want)
		}
	}
	if isSystemctlTimeout(nil) {
		t.Errorf("isSystemctlTimeout(nil) must be false")
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
