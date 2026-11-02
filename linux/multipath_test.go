// Copyright 2026 Hewlett Packard Enterprise Development LP
package linux

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hpe-storage/common-host-libs/model"
	"github.com/hpe-storage/common-host-libs/util"
)

// ---- matchesWWID tests ----

var matchesWWIDTests = []struct {
	name   string
	wwid   string
	serial string
	expect bool
}{
	{
		name:   "NAA-6 prefix: WWID has leading type byte, serial does not",
		wwid:   "360002ac0000000000200d34e0007f544",
		serial: "60002ac0000000000200d34e0007f544",
		expect: true,
	},
	{
		name:   "exact match: no NAA prefix",
		wwid:   "60002ac0000000000200d34e0007f544",
		serial: "60002ac0000000000200d34e0007f544",
		expect: true,
	},
	{
		name:   "full WWID passed as serial",
		wwid:   "360002ac0000000000200d34e0007f544",
		serial: "360002ac0000000000200d34e0007f544",
		expect: true,
	},
	{
		name:   "different serial: different array",
		wwid:   "360002ac0000000000200d34e0007f544",
		serial: "60002ac0000000000001e24a0002b114",
		expect: false,
	},
	{
		name:   "empty serial: backward compat, always matches",
		wwid:   "360002ac0000000000200d34e0007f544",
		serial: "",
		expect: true,
	},
	{
		name:   "case insensitive: uppercase WWID, lowercase serial",
		wwid:   "360002AC0000000000200D34E0007F544",
		serial: "60002ac0000000000200d34e0007f544",
		expect: true,
	},
	{
		name:   "case insensitive: lowercase WWID, uppercase serial",
		wwid:   "360002ac0000000000200d34e0007f544",
		serial: "60002AC0000000000200D34E0007F544",
		expect: true,
	},
	{
		name:   "partial serial mismatch: prefix matches but suffix differs",
		wwid:   "360002ac0000000000200d34e0007f544",
		serial: "60002ac0000000000200d34e0007f999",
		expect: false,
	},
	{
		name:   "NAA-5 prefix: Nimble style WWID",
		wwid:   "2a17a5500c8fec6d16000000000000001",
		serial: "a17a5500c8fec6d16000000000000001",
		expect: true,
	},
}

func TestMatchesWWID(t *testing.T) {
	for _, tc := range matchesWWIDTests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchesWWID(tc.wwid, tc.serial)
			if got != tc.expect {
				t.Errorf("matchesWWID(%q, %q) = %v, want %v",
					tc.wwid, tc.serial, got, tc.expect)
			}
		})
	}
}

func TestLoadMultipathdTimeouts(t *testing.T) {
	tests := []struct {
		name               string
		value              string
		wantUxsockTimeout  int
		wantCommandTimeout int
	}{
		{name: "default", wantUxsockTimeout: 300000, wantCommandTimeout: 305},
		{name: "configured", value: "60000", wantUxsockTimeout: 60000, wantCommandTimeout: 65},
		{name: "rounds partial second up", value: "60001", wantUxsockTimeout: 60001, wantCommandTimeout: 66},
		{name: "zero falls back", value: "0", wantUxsockTimeout: 300000, wantCommandTimeout: 305},
		{name: "negative falls back", value: "-1", wantUxsockTimeout: 300000, wantCommandTimeout: 305},
		{name: "invalid falls back", value: "invalid", wantUxsockTimeout: 300000, wantCommandTimeout: 305},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(multipathUxsockTimeoutEnvVar, test.value)
			if test.name == "default" {
				if err := os.Unsetenv(multipathUxsockTimeoutEnvVar); err != nil {
					t.Fatalf("unset %s: %v", multipathUxsockTimeoutEnvVar, err)
				}
			}

			gotUxsockTimeout, gotCommandTimeout := loadMultipathdTimeouts()
			if gotUxsockTimeout != test.wantUxsockTimeout {
				t.Fatalf("loadMultipathdTimeouts() uxsock timeout = %d, want %d", gotUxsockTimeout, test.wantUxsockTimeout)
			}
			if gotCommandTimeout != test.wantCommandTimeout {
				t.Fatalf("loadMultipathdTimeouts() command timeout = %d, want %d", gotCommandTimeout, test.wantCommandTimeout)
			}
		})
	}
}

func TestMultipathdCommandTimeoutIsCached(t *testing.T) {
	wantUxsockTimeout := MultipathdUxsockTimeout()
	wantCommandTimeout := MultipathdCommandTimeout()
	t.Setenv(multipathUxsockTimeoutEnvVar, "1")

	if got := MultipathdUxsockTimeout(); got != wantUxsockTimeout {
		t.Fatalf("MultipathdUxsockTimeout() = %d after environment change, want cached value %d", got, wantUxsockTimeout)
	}
	if got := MultipathdCommandTimeout(); got != wantCommandTimeout {
		t.Fatalf("MultipathdCommandTimeout() = %d after environment change, want cached value %d", got, wantCommandTimeout)
	}
}

// ---- orphanPathsPattern regex tests ----

// Simulated multipathd show paths lines in format: %w %d %t %i %o %T %z %s %m
var orphanRegexTests = []struct {
	name       string
	line       string
	wantMatch  bool
	wantWWID   string
	wantHost   string
	wantChan   string
	wantTarget string
	wantLun    string
}{
	{
		name:       "3PARdata orphan: typical FC path",
		line:       "360002ac0000000000200d34e0007f544 sdc active 33:0:0:1 running ready tur 3PARdata,VV [orphan]",
		wantMatch:  true,
		wantWWID:   "360002ac0000000000200d34e0007f544",
		wantHost:   "33",
		wantChan:   "0",
		wantTarget: "0",
		wantLun:    "1",
	},
	{
		name:       "Nimble orphan path",
		line:       "2a17a5500c8fec6d160000000000000001 sdl active 38:0:0:3 running ready tur Nimble,CS [orphan]",
		wantMatch:  true,
		wantWWID:   "2a17a5500c8fec6d160000000000000001",
		wantHost:   "38",
		wantChan:   "0",
		wantTarget: "0",
		wantLun:    "3",
	},
	{
		name:      "non-orphan path: belongs to mpathcd",
		line:      "360002ac0000000000200d34e0007f544 sdh active 33:0:0:1 running ready tur 3PARdata,VV mpathcd",
		wantMatch: false,
	},
	{
		name:      "unknown vendor: should not match",
		line:      "360002ac0000000000200d34e0007f544 sdc active 33:0:0:1 running ready tur UNKNOWN,VV [orphan]",
		wantMatch: false,
	},
	{
		name:       "TrueNAS orphan path",
		line:       "naa.6589cfc00000012345 sda active 5:0:0:0 running ready tur TrueNAS,iSCSI [orphan]",
		wantMatch:  true,
		wantWWID:   "naa.6589cfc00000012345",
		wantHost:   "5",
		wantChan:   "0",
		wantTarget: "0",
		wantLun:    "0",
	},
}

func TestOrphanPathsRegex(t *testing.T) {
	re := regexp.MustCompile(getOrphanPathsPattern())
	for _, tc := range orphanRegexTests {
		t.Run(tc.name, func(t *testing.T) {
			result := util.FindStringSubmatchMap(tc.line, re)
			if tc.wantMatch {
				if len(result) == 0 {
					t.Fatalf("expected regex to match line %q, but got no match", tc.line)
				}
				if result["wwid"] != tc.wantWWID {
					t.Errorf("wwid: got %q, want %q", result["wwid"], tc.wantWWID)
				}
				if result["host"] != tc.wantHost {
					t.Errorf("host: got %q, want %q", result["host"], tc.wantHost)
				}
				if result["channel"] != tc.wantChan {
					t.Errorf("channel: got %q, want %q", result["channel"], tc.wantChan)
				}
				if result["target"] != tc.wantTarget {
					t.Errorf("target: got %q, want %q", result["target"], tc.wantTarget)
				}
				if result["lun"] != tc.wantLun {
					t.Errorf("lun: got %q, want %q", result["lun"], tc.wantLun)
				}
			} else {
				if len(result) > 0 {
					t.Errorf("expected no match for line %q, but got %v", tc.line, result)
				}
			}
		})
	}
}

// ---- End-to-end orphan filtering tests ----
// These simulate what multipathGetOrphanPathsBySerialAndLunID does
// without shelling out to multipathd.

func filterOrphanLines(lines []string, serialNumber, lunID string) []string {
	re := regexp.MustCompile(getOrphanPathsPattern())
	var hctls []string
	for _, line := range lines {
		result := util.FindStringSubmatchMap(line, re)
		if len(result) == 0 {
			continue
		}
		wwid := result["wwid"]
		lun := result["lun"]
		if lun == lunID && matchesWWID(wwid, serialNumber) {
			hctl := result["host"] + ":" + result["channel"] + ":" + result["target"] + ":" + lun
			hctls = append(hctls, hctl)
		}
	}
	return hctls
}

var orphanFilterTests = []struct {
	name      string
	lines     []string
	serial    string
	lunID     string
	wantHCTLs []string
}{
	{
		name: "array1 serial matches only array1 orphans at same LUN",
		lines: []string{
			"360002ac0000000000200d34e0007f544 sdc active 33:0:0:1 running ready tur 3PARdata,VV [orphan]",
			"360002ac0000000000200d34e0007f544 sdg active 34:0:0:1 running ready tur 3PARdata,VV [orphan]",
			"360002ac0000000000001e24a0002b114 sde active 35:0:0:1 running ready tur 3PARdata,VV [orphan]",
			"360002ac0000000000001e24a0002b114 sdi active 36:0:0:1 running ready tur 3PARdata,VV [orphan]",
		},
		serial:    "60002ac0000000000200d34e0007f544",
		lunID:     "1",
		wantHCTLs: []string{"33:0:0:1", "34:0:0:1"},
	},
	{
		name: "array2 serial matches only array2 orphans at same LUN",
		lines: []string{
			"360002ac0000000000200d34e0007f544 sdc active 33:0:0:1 running ready tur 3PARdata,VV [orphan]",
			"360002ac0000000000200d34e0007f544 sdg active 34:0:0:1 running ready tur 3PARdata,VV [orphan]",
			"360002ac0000000000001e24a0002b114 sde active 35:0:0:1 running ready tur 3PARdata,VV [orphan]",
			"360002ac0000000000001e24a0002b114 sdi active 36:0:0:1 running ready tur 3PARdata,VV [orphan]",
		},
		serial:    "60002ac0000000000001e24a0002b114",
		lunID:     "1",
		wantHCTLs: []string{"35:0:0:1", "36:0:0:1"},
	},
	{
		name: "LUN mismatch: no matches even with correct serial",
		lines: []string{
			"360002ac0000000000200d34e0007f544 sdc active 33:0:0:1 running ready tur 3PARdata,VV [orphan]",
		},
		serial:    "60002ac0000000000200d34e0007f544",
		lunID:     "2",
		wantHCTLs: nil,
	},
	{
		name: "empty serial: backward compat matches all orphans at LUN",
		lines: []string{
			"360002ac0000000000200d34e0007f544 sdc active 33:0:0:1 running ready tur 3PARdata,VV [orphan]",
			"360002ac0000000000001e24a0002b114 sde active 35:0:0:1 running ready tur 3PARdata,VV [orphan]",
		},
		serial:    "",
		lunID:     "1",
		wantHCTLs: []string{"33:0:0:1", "35:0:0:1"},
	},
	{
		name: "non-orphan lines are ignored",
		lines: []string{
			"360002ac0000000000200d34e0007f544 sdc active 33:0:0:1 running ready tur 3PARdata,VV mpathcd",
			"360002ac0000000000200d34e0007f544 sdg active 34:0:0:1 running ready tur 3PARdata,VV [orphan]",
		},
		serial:    "60002ac0000000000200d34e0007f544",
		lunID:     "1",
		wantHCTLs: []string{"34:0:0:1"},
	},
	{
		name: "mixed vendors: only matching vendor+serial+LUN",
		lines: []string{
			"360002ac0000000000200d34e0007f544 sdc active 33:0:0:1 running ready tur 3PARdata,VV [orphan]",
			"2a17a5500c8fec6d160000000000000001 sdl active 38:0:0:1 running ready tur Nimble,CS [orphan]",
		},
		serial:    "60002ac0000000000200d34e0007f544",
		lunID:     "1",
		wantHCTLs: []string{"33:0:0:1"},
	},
	{
		name:      "no orphan lines: empty result",
		lines:     []string{},
		serial:    "60002ac0000000000200d34e0007f544",
		lunID:     "1",
		wantHCTLs: nil,
	},
}

func TestOrphanPathFiltering(t *testing.T) {
	for _, tc := range orphanFilterTests {
		t.Run(tc.name, func(t *testing.T) {
			got := filterOrphanLines(tc.lines, tc.serial, tc.lunID)
			if len(got) != len(tc.wantHCTLs) {
				t.Fatalf("got %d HCTLs %v, want %d %v",
					len(got), got, len(tc.wantHCTLs), tc.wantHCTLs)
			}
			for i, hctl := range got {
				if hctl != tc.wantHCTLs[i] {
					t.Errorf("HCTL[%d]: got %q, want %q", i, hctl, tc.wantHCTLs[i])
				}
			}
		})
	}
}

// ---- checkIfDeviceCanBeDeleted open-descriptor guard tests ----
// A block-mode/direct-attached volume (e.g. a raw-block disk opened directly
// by a VM) has no mount and no dm holder, so only the open-count check below
// can tell that it is still in use.

func TestCheckIfDeviceCanBeDeleted_OpenDescriptorGuard(t *testing.T) {
	origGetDmOpenCount := getDmOpenCountFunc
	defer func() { getDmOpenCountFunc = origGetDmOpenCount }()

	// a name that cannot collide with any real mount entry on the test host
	dev := &model.Device{
		MpathName:       "mpathtest-con5018-not-a-real-device",
		AltFullPathName: "/dev/mapper/mpathtest-con5018-not-a-real-device",
		Pathname:        "/dev/mapper/mpathtest-con5018-not-a-real-device",
	}

	tests := []struct {
		name        string
		stub        func(string) (int, error)
		wantErr     bool
		wantBusyMsg bool
	}{
		{
			name:    "not open: teardown allowed",
			stub:    func(string) (int, error) { return 0, nil },
			wantErr: false,
		},
		{
			name:        "open: teardown refused",
			stub:        func(string) (int, error) { return 1, nil },
			wantErr:     true,
			wantBusyMsg: true,
		},
		{
			name:        "open count unreadable: fails safe, teardown refused",
			stub:        func(string) (int, error) { return 0, fmt.Errorf("dmsetup: boom") },
			wantErr:     true,
			wantBusyMsg: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getDmOpenCountFunc = tc.stub
			err := checkIfDeviceCanBeDeleted(dev)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tc.wantBusyMsg && !strings.Contains(err.Error(), multipathDeviceBusy) {
				t.Errorf("expected error to contain %q, got %q", multipathDeviceBusy, err.Error())
			}
		})
	}
}

// ---- getDmOpenCount parsing tests ----

func TestGetDmOpenCount_EmptyMpathName(t *testing.T) {
	if _, err := getDmOpenCount(""); err == nil {
		t.Fatal("expected error for empty mpath name")
	}
}

// ---- cleanupDeviceAndSlaves / retryCleanupDeviceAndSlaves TOCTOU guard tests
// (review follow-up Medium-1b/d) ----
// cleanupDeviceAndSlaves is the single choke point every teardown path (remap
// cleanup, zombie unstage, retries) funnels through, unlike
// checkIfDeviceCanBeDeleted which only guards the DeleteDevice entry point.
// These tests prove the open-count guard added there refuses teardown before
// any destructive step runs, and that retryCleanupDeviceAndSlaves treats that
// refusal as terminal instead of retrying for up to 50s.

func TestCleanupDeviceAndSlaves_OpenGuardRefusesTeardown(t *testing.T) {
	origGetDmOpenCount := getDmOpenCountFunc
	defer func() { getDmOpenCountFunc = origGetDmOpenCount }()
	getDmOpenCountFunc = func(string) (int, error) { return 1, nil }

	dev := &model.Device{MpathName: "mpathtest-con5018-not-a-real-device"}
	err := cleanupDeviceAndSlaves(dev)
	if err == nil {
		t.Fatal("expected open map to refuse teardown")
	}
	if !strings.Contains(err.Error(), multipathDeviceBusy) {
		t.Errorf("expected error to contain %q, got %q", multipathDeviceBusy, err.Error())
	}
}

func TestRetryCleanupDeviceAndSlaves_BusyIsTerminalNotRetried(t *testing.T) {
	origGetDmOpenCount := getDmOpenCountFunc
	defer func() { getDmOpenCountFunc = origGetDmOpenCount }()
	getDmOpenCountFunc = func(string) (int, error) { return 1, nil }

	dev := &model.Device{MpathName: "mpathtest-con5018-not-a-real-device"}
	start := time.Now()
	err := retryCleanupDeviceAndSlaves(dev)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected busy error")
	}
	if !strings.Contains(err.Error(), multipathDeviceBusy) {
		t.Errorf("expected error to contain %q, got %q", multipathDeviceBusy, err.Error())
	}
	// the 10x5s retry loop would take >=50s; a busy/in-use result must return
	// immediately instead of holding the caller's stageLock for that long.
	if elapsed >= 5*time.Second {
		t.Errorf("expected busy result to short-circuit the retry loop, took %s", elapsed)
	}
}

// ---- multipathRemoveDmDevice tests (review follow-up Medium-1c) ----
// multipathRemoveDmDevice now uses `multipath -f` instead of `dmsetup remove
// --force`, which installs an error table on an open map before reporting
// busy (the exact ESC-18223/CON-4959 teardown trigger). The exec is stubbed via
// execMultipathFlush so these branches run without a real multipath binary.

func TestMultipathRemoveDmDevice_Branches(t *testing.T) {
	origFlush := execMultipathFlush
	defer func() { execMultipathFlush = origFlush }()

	dev := &model.Device{MpathName: "mpathtest-con5018-not-a-real-device"}

	tests := []struct {
		name        string
		stub        func(string) (string, error)
		wantErr     bool
		wantBusyMsg bool
	}{
		{
			name:    "flush succeeds",
			stub:    func(string) (string, error) { return "", nil },
			wantErr: false,
		},
		{
			name:    "already removed (No such device) is not an error",
			stub:    func(string) (string, error) { return deviceDoesNotExist, fmt.Errorf("exit status 1: %s", deviceDoesNotExist) },
			wantErr: false,
		},
		{
			name:        "in use via libdevmapper busy string surfaces busy sentinel",
			stub:        func(string) (string, error) { return "device-mapper: remove ioctl failed: " + multipathDeviceBusy, fmt.Errorf("exit status 1") },
			wantErr:     true,
			wantBusyMsg: true,
		},
		{
			name:        "in use via multipath 'map in use' string surfaces busy sentinel",
			stub:        func(string) (string, error) { return "mpathx: " + multipathMapInUse, fmt.Errorf("exit status 1") },
			wantErr:     true,
			wantBusyMsg: true,
		},
		{
			name:    "other failure is a non-busy error",
			stub:    func(string) (string, error) { return "some unexpected failure", fmt.Errorf("exit status 1") },
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			execMultipathFlush = tc.stub
			err := multipathRemoveDmDevice(dev)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tc.wantBusyMsg && !strings.Contains(err.Error(), multipathDeviceBusy) {
				t.Errorf("expected busy sentinel %q, got %q", multipathDeviceBusy, err.Error())
			}
			if err != nil && !tc.wantBusyMsg && strings.Contains(err.Error(), multipathDeviceBusy) {
				t.Errorf("non-busy failure must not be reported as busy, got %q", err.Error())
			}
		})
	}
}
