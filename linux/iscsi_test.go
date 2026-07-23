package linux

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hpe-storage/common-host-libs/model"
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

// TestGetIscsiHostNumbersForTargetIqns_MatchAndScope exercises the os.ReadDir
// session enumeration and target-IQN matching against a fake sysfs tree.
func TestGetIscsiHostNumbersForTargetIqns_MatchAndScope(t *testing.T) {
	scsiRoot := t.TempDir()
	iscsiRoot := t.TempDir()
	origScsi, origIscsi := scsiHostBasePath, iscsiHostRootPath
	scsiHostBasePath = scsiRoot
	iscsiHostRootPath = iscsiRoot
	defer func() { scsiHostBasePath, iscsiHostRootPath = origScsi, origIscsi }()

	// Host enumeration source: /sys/class/iscsi_host/host<N>
	if err := os.MkdirAll(filepath.Join(iscsiRoot, "host33"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Session/target source: /sys/class/scsi_host/host<N>/device/session<S>/iscsi_session/session<S>/targetname
	writeTarget := func(host, sess, iqn string) {
		dir := filepath.Join(scsiRoot, "host"+host, "device", "session"+sess, "iscsi_session", "session"+sess)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "targetname"), []byte(iqn+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeTarget("33", "10", "iqn.2023-24.com.hpe:target1")
	// A stray non-session dir under device/ that must be ignored.
	if err := os.MkdirAll(filepath.Join(scsiRoot, "host33", "device", "power"), 0o755); err != nil {
		t.Fatal(err)
	}

	hosts, err := GetIscsiHostNumbersForTargetIqns([]string{"iqn.2023-24.com.hpe:target1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 1 || hosts[0] != "33" {
		t.Fatalf("expected host [33], got %v", hosts)
	}

	// Non-matching IQN -> (nil, nil), caller falls back to full rescan.
	hosts, err = GetIscsiHostNumbersForTargetIqns([]string{"iqn.2023-24.com.hpe:other"})
	if err != nil {
		t.Fatalf("unexpected error for no-match: %v", err)
	}
	if len(hosts) != 0 {
		t.Fatalf("expected no hosts for no-match, got %v", hosts)
	}
}
