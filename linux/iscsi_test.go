package linux

import (
	"github.com/hpe-storage/common-host-libs/model"
	"testing"
)

func TestDeleteEmptyTarget(t *testing.T) {
	target := &model.IscsiTarget{
		Name:    "",
		Address: "172.10.10.10",
		Port:    "3260",
	}
	err := iscsiDeleteNode(target)
	if err.Error() != "Empty target to delete Node" {
		t.Error("empty target should not be allowed to be deleted")
	}
}


func TestGetIscsiHostNumbersForTargetIqns_EmptyTargets(t *testing.T) {
	hosts, err := GetIscsiHostNumbersForTargetIqns(nil)
	if err != nil {
		t.Errorf("expected no error for nil targets, got %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected empty hosts for nil targets, got %v", hosts)
	}

	hosts, err = GetIscsiHostNumbersForTargetIqns([]string{})
	if err != nil {
		t.Errorf("expected no error for empty targets, got %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected empty hosts for empty targets, got %v", hosts)
	}
}

func TestRescanIscsiHostsForLun_EmptyHosts(t *testing.T) {
	err := RescanIscsiHostsForLun([]string{}, "3")
	if err != nil {
		t.Errorf("expected no error for empty host list, got %v", err)
	}
}

func TestRescanIscsiHostsForLun_NonExistentHost(t *testing.T) {
	// host99999 should not exist on any system
	err := RescanIscsiHostsForLun([]string{"99999"}, "3")
	if err != nil {
		t.Errorf("expected no error for non-existent host (should skip), got %v", err)
	}
}
