package linux

import (
	"os"
	"path/filepath"
	"testing"
)

var parseHctlTests = []struct {
	name      string
	lunString string
	host      string
	channel   string
	target    string
	lun       string
	err       bool
}{
	{"parse test/lun0", "Host: scsi7 Channel: 00 Id: 00 Lun: 00", "7", "0", "0", "0", false},
	{"parse test/multi digits", "Host: scsi10 Channel: 20 Id: 30 Lun: 255", "10", "20", "30", "255", false},
	{"parse test/begin with space", " Host: scsi7 Channel: 00 Id: 00 Lun: 00", "7", "0", "0", "0", false},
	{"parse test/no match", "not a valid lun string", "", "", "", "", true},
}

func TestParseHctl(t *testing.T) {
	for _, tc := range parseHctlTests {
		t.Run(tc.name, func(t *testing.T) {
			host, channel, target, lun, err := parseHctl(tc.lunString)
			if (err != nil) != tc.err {
				t.Error(
					"For:", tc.name,
					"expected:", "No error",
					"got:", err,
				)
			}
			if host != tc.host {
				t.Error(
					"For:", tc.name,
					"expected host:", tc.host,
					"got host:", host,
				)
			}
			if channel != tc.channel {
				t.Error(
					"For:", tc.name,
					"expected channel:", tc.channel,
					"got channel:", channel,
				)
			}
			if target != tc.target {
				t.Error(
					"For:", tc.name,
					"expected target:", tc.target,
					"got target:", target,
				)
			}
			if lun != tc.lun {
				t.Error(
					"For:", tc.name,
					"expected lun:", tc.lun,
					"got lun:", lun,
				)
			}
		})
	}
}

// ---- isPathInRemapScope tests (ESC-18223 defect #1) ----

func TestIsPathInRemapScope(t *testing.T) {
	cases := []struct {
		name                       string
		h, c, tg                   string
		scopedHostSet              map[string]bool
		scopedHostChannelTargetSet map[string]bool
		want                       bool
	}{
		{"no scoping configured", "6", "0", "0", nil, nil, true},
		{"host matches, no host:channel:target set", "6", "0", "0", map[string]bool{"6": true}, nil, true},
		{"host does not match", "9", "0", "0", map[string]bool{"6": true}, nil, false},
		{"host matches but different target (ESC-18223)", "6", "0", "1", map[string]bool{"6": true}, map[string]bool{"6:0:0": true}, false},
		{"host and host:channel:target both match", "6", "0", "0", map[string]bool{"6": true}, map[string]bool{"6:0:0": true}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isPathInRemapScope(tc.h, tc.c, tc.tg, tc.scopedHostSet, tc.scopedHostChannelTargetSet)
			if got != tc.want {
				t.Errorf("isPathInRemapScope(%s,%s,%s) = %v, want %v", tc.h, tc.c, tc.tg, got, tc.want)
			}
		})
	}
}

// ---- checkRemappedLunPath tests (ESC-18223 defect #2) ----

func writeScsiDeviceAttr(t *testing.T, base, hctl, attr, content string) {
	t.Helper()
	dir := filepath.Join(base, hctl, "device")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, attr), []byte(content+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRemappedLunPath_3PAR_SameSerial_NotRemapped(t *testing.T) {
	tmp := t.TempDir()
	orig := scsiDeviceBasePath
	scsiDeviceBasePath = tmp
	defer func() { scsiDeviceBasePath = orig }()

	writeScsiDeviceAttr(t, tmp, "6:0:0:5", "state", "running")
	writeScsiDeviceAttr(t, tmp, "6:0:0:5", "vendor", "3PARdata")
	writeScsiDeviceAttr(t, tmp, "6:0:0:5", "wwid", "naa.60002ac0000000000a0066ef0001db2c")

	found, _, err := checkRemappedLunPath("6", "0", "0", "5", "60002ac0000000000a0066ef0001db2c", "5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatalf("expected not remapped when wwid already matches the expected serial")
	}
}

func TestCheckRemappedLunPath_3PAR_Offline_Remapped(t *testing.T) {
	tmp := t.TempDir()
	orig := scsiDeviceBasePath
	scsiDeviceBasePath = tmp
	defer func() { scsiDeviceBasePath = orig }()

	writeScsiDeviceAttr(t, tmp, "6:0:0:5", "state", "offline")
	writeScsiDeviceAttr(t, tmp, "6:0:0:5", "vendor", "3PARdata")
	writeScsiDeviceAttr(t, tmp, "6:0:0:5", "wwid", "naa.oldserial")

	found, oldSerial, err := checkRemappedLunPath("6", "0", "0", "5", "newserial", "5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || oldSerial != "oldserial" {
		t.Fatalf("expected remapped=true oldSerial=%q, got remapped=%v oldSerial=%q", "oldserial", found, oldSerial)
	}
}

// TestCheckRemappedLunPath_3PAR_ForeignArray_NotRemapped reproduces the
// ESC-18223 defect #2 false positive: a stable, healthy, DIFFERENT array's
// volume shares this host LUN ID. The wwid mismatches the volume being
// staged, but a live re-read of VPD page 0x80 shows the same value as the
// cached sysfs reading (nothing has actually changed at this address since
// the kernel last probed it), so this must NOT be classified as a remap.
func TestCheckRemappedLunPath_3PAR_ForeignArray_NotRemapped(t *testing.T) {
	tmp := t.TempDir()
	origBase := scsiDeviceBasePath
	scsiDeviceBasePath = tmp
	defer func() { scsiDeviceBasePath = origBase }()

	origFunc := getDeviceSerialByHctlFunc
	defer func() { getDeviceSerialByHctlFunc = origFunc }()

	writeScsiDeviceAttr(t, tmp, "6:0:1:5", "state", "running")
	writeScsiDeviceAttr(t, tmp, "6:0:1:5", "vendor", "3PARdata")
	writeScsiDeviceAttr(t, tmp, "6:0:1:5", "wwid", "naa.foreignarrayserial")
	writeScsiDeviceAttr(t, tmp, "6:0:1:5", "vpd_pg80", "x stableid123")

	getDeviceSerialByHctlFunc = func(h, c, tg, l string) (string, error) {
		return "stableid123", nil // live read matches cached -> nothing has changed here
	}

	found, _, err := checkRemappedLunPath("6", "0", "1", "5", "myexpectedserial", "5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatalf("expected foreign array's stable volume to NOT be classified as remapped (ESC-18223 defect #2)")
	}
}

// TestCheckRemappedLunPath_3PAR_GenuineRemap_Remapped verifies that when the
// live VPD80 read diverges from the cached one (the physical device at this
// address has actually changed since the kernel's last probe), the mismatch
// is still correctly confirmed as a genuine remap, preserving pre-existing
// 3PAR remap-detection capability.
func TestCheckRemappedLunPath_3PAR_GenuineRemap_Remapped(t *testing.T) {
	tmp := t.TempDir()
	origBase := scsiDeviceBasePath
	scsiDeviceBasePath = tmp
	defer func() { scsiDeviceBasePath = origBase }()

	origFunc := getDeviceSerialByHctlFunc
	defer func() { getDeviceSerialByHctlFunc = origFunc }()

	writeScsiDeviceAttr(t, tmp, "6:0:0:5", "state", "running")
	writeScsiDeviceAttr(t, tmp, "6:0:0:5", "vendor", "3PARdata")
	writeScsiDeviceAttr(t, tmp, "6:0:0:5", "wwid", "naa.oldserial")
	writeScsiDeviceAttr(t, tmp, "6:0:0:5", "vpd_pg80", "x cachedid")

	getDeviceSerialByHctlFunc = func(h, c, tg, l string) (string, error) {
		return "livedivergedid", nil // live read differs from cached -> something changed here
	}

	found, oldSerial, err := checkRemappedLunPath("6", "0", "0", "5", "newserial", "5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || oldSerial != "oldserial" {
		t.Fatalf("expected genuine remap confirmed with oldSerial %q, got found=%v oldSerial=%q", "oldserial", found, oldSerial)
	}
}

// TestCheckRemappedLunPath_NonVendor_StableMismatch_NotRemapped is a regression
// check that the pre-existing non-3PAR confirmation behavior is unaffected.
func TestCheckRemappedLunPath_NonVendor_StableMismatch_NotRemapped(t *testing.T) {
	tmp := t.TempDir()
	origBase := scsiDeviceBasePath
	scsiDeviceBasePath = tmp
	defer func() { scsiDeviceBasePath = origBase }()

	origFunc := getDeviceSerialByHctlFunc
	defer func() { getDeviceSerialByHctlFunc = origFunc }()

	writeScsiDeviceAttr(t, tmp, "4:0:0:2", "state", "running")
	writeScsiDeviceAttr(t, tmp, "4:0:0:2", "vendor", "NIMBLE")
	writeScsiDeviceAttr(t, tmp, "4:0:0:2", "vpd_pg80", "x othervolumeid")

	getDeviceSerialByHctlFunc = func(h, c, tg, l string) (string, error) {
		return "othervolumeid", nil // live re-read still doesn't match our expected volume
	}

	found, _, err := checkRemappedLunPath("4", "0", "0", "2", "myexpectedserial", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatalf("expected non-3PAR confirm step to reject a stable mismatch")
	}
}
