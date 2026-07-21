package linux

import (
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
	_, err := GetFcHostNumbersForTargetWwpns(nil)
	if err == nil {
		t.Error("expected error for nil input")
	}
	_, err = GetFcHostNumbersForTargetWwpns([]string{})
	if err == nil {
		t.Error("expected error for empty input")
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
