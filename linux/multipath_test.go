// Copyright 2026 Hewlett Packard Enterprise Development LP
package linux

import (
	"regexp"
	"testing"

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
		name:       "non-orphan path: belongs to mpathcd",
		line:       "360002ac0000000000200d34e0007f544 sdh active 33:0:0:1 running ready tur 3PARdata,VV mpathcd",
		wantMatch:  false,
	},
	{
		name:       "unknown vendor: should not match",
		line:       "360002ac0000000000200d34e0007f544 sdc active 33:0:0:1 running ready tur UNKNOWN,VV [orphan]",
		wantMatch:  false,
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
	name       string
	lines      []string
	serial     string
	lunID      string
	wantHCTLs  []string
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
