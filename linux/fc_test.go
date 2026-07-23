package linux

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hpe-storage/common-host-libs/model"
)

// ---- getFcTargetWwpns tests ----

func TestGetFcTargetWwpns_FcVolume(t *testing.T) {
	vol := &model.Volume{
		AccessProtocol: "fc",
		TargetAddress:  "50060b0000c26270,50060b0000c26271",
	}
	wwpns := getFcTargetWwpns(vol)
	if len(wwpns) != 2 {
		t.Fatalf("expected 2 WWPNs, got %d: %v", len(wwpns), wwpns)
	}
	if wwpns[0] != "50060b0000c26270" || wwpns[1] != "50060b0000c26271" {
		t.Errorf("unexpected WWPNs: %v", wwpns)
	}
}

func TestGetFcTargetWwpns_IscsiVolume(t *testing.T) {
	vol := &model.Volume{
		AccessProtocol: "iscsi",
		TargetAddress:  "iqn.2023-24.com.hpe:target1",
	}
	wwpns := getFcTargetWwpns(vol)
	if wwpns != nil {
		t.Errorf("expected nil for iSCSI volume, got %v", wwpns)
	}
}

func TestGetFcTargetWwpns_EmptyTargetAddress(t *testing.T) {
	vol := &model.Volume{
		AccessProtocol: "fc",
		TargetAddress:  "",
	}
	wwpns := getFcTargetWwpns(vol)
	if wwpns != nil {
		t.Errorf("expected nil for empty target address, got %v", wwpns)
	}
}

func TestGetFcTargetWwpns_SingleWwpn(t *testing.T) {
	vol := &model.Volume{
		AccessProtocol: "fc",
		TargetAddress:  "50060b0000c26270",
	}
	wwpns := getFcTargetWwpns(vol)
	if len(wwpns) != 1 {
		t.Fatalf("expected 1 WWPN, got %d", len(wwpns))
	}
	if wwpns[0] != "50060b0000c26270" {
		t.Errorf("unexpected WWPN: %s", wwpns[0])
	}
}

func TestGetFcTargetWwpns_TrailingComma(t *testing.T) {
	vol := &model.Volume{
		AccessProtocol: "fc",
		TargetAddress:  "50060b0000c26270,",
	}
	wwpns := getFcTargetWwpns(vol)
	if len(wwpns) != 1 {
		t.Fatalf("expected 1 WWPN (trailing comma ignored), got %d: %v", len(wwpns), wwpns)
	}
}

// ---- GetFcHostNumbersForTargetWwpns tests ----

func TestGetFcHostNumbersForTargetWwpns_EmptyInput(t *testing.T) {
	hosts, err := GetFcHostNumbersForTargetWwpns(nil)
	if err != nil {
		t.Errorf("expected no error for nil input, got %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected empty hosts for nil input, got %v", hosts)
	}
	hosts, err = GetFcHostNumbersForTargetWwpns([]string{})
	if err != nil {
		t.Errorf("expected no error for empty input, got %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected empty hosts for empty input, got %v", hosts)
	}
}

func TestGetFcHostNumbersForTargetWwpns_NoRemotePorts(t *testing.T) {
	// On a system without FC, /sys/class/fc_remote_ports/ won't exist
	_, err := GetFcHostNumbersForTargetWwpns([]string{"50060b0000c26270"})
	if err == nil {
		t.Log("system has fc_remote_ports, test inconclusive")
	}
}

// ---- RescanFcHostsForLun tests ----

func TestRescanFcHostsForLun_EmptyHosts(t *testing.T) {
	err := RescanFcHostsForLun([]string{}, "3")
	if err != nil {
		t.Errorf("expected no error for empty host list, got %v", err)
	}
}

func TestRescanFcHostsForLun_NonExistentHost(t *testing.T) {
	// host99999 should not exist on any system
	err := RescanFcHostsForLun([]string{"99999"}, "3")
	if err != nil {
		t.Errorf("expected no error for non-existent host (should skip), got %v", err)
	}
}

func TestRescanFcHostsForLun_NilHosts(t *testing.T) {
	err := RescanFcHostsForLun(nil, "3")
	if err != nil {
		t.Errorf("expected no error for nil host list, got %v", err)
	}
}

// ---- normalizeWwpn tests ----

func TestNormalizeWwpn(t *testing.T) {
	cases := map[string]string{
		"20410002AC07EE45":        "20410002ac07ee45",
		"0x20410002ac07ee45":      "20410002ac07ee45",
		"20:41:00:02:ac:07:ee:45": "20410002ac07ee45",
		" 0X20410002AC07EE45 ":    "20410002ac07ee45",
	}
	for in, want := range cases {
		if got := normalizeWwpn(in); got != want {
			t.Errorf("normalizeWwpn(%q) = %q, want %q", in, got, want)
		}
	}
}

// ---- GetFcHostNumbersForTargetWwpns matching against a fake sysfs tree ----

func TestGetFcHostNumbersForTargetWwpns_MatchAndScope(t *testing.T) {
	tmp := t.TempDir()
	orig := fcRemotePortBasePath
	fcRemotePortBasePath = tmp
	defer func() { fcRemotePortBasePath = orig }()

	// rport-<host>:<bus>-<target>/port_name — sysfs form is lowercase 0x-prefixed.
	writePort := func(rport, wwpn string) {
		dir := filepath.Join(tmp, rport)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "port_name"), []byte(wwpn+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writePort("rport-6:0-0", "0x20410002ac07ee45")
	writePort("rport-9:0-1", "0x20420002ac07ee45")
	// A non-rport entry that must be ignored.
	if err := os.MkdirAll(filepath.Join(tmp, "fc_host"), 0o755); err != nil {
		t.Fatal(err)
	}

	// CSP-style WWPN (uppercase, no 0x) must still match via normalizeWwpn.
	hosts, err := GetFcHostNumbersForTargetWwpns([]string{"20410002AC07EE45"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 1 || hosts[0] != "6" {
		t.Fatalf("expected host [6], got %v", hosts)
	}

	// No matching target WWPN -> (nil, nil), caller falls back to full rescan.
	hosts, err = GetFcHostNumbersForTargetWwpns([]string{"dead0000dead0000"})
	if err != nil {
		t.Fatalf("unexpected error for no-match: %v", err)
	}
	if len(hosts) != 0 {
		t.Fatalf("expected no hosts for no-match, got %v", hosts)
	}
}
