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
