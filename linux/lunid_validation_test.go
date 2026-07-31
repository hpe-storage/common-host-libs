// Copyright 2026 Hewlett Packard Enterprise Development LP
package linux

import (
	"testing"
)

func TestValidateLunID(t *testing.T) {
	tests := []struct {
		name    string
		lunID   string
		wantErr bool
	}{
		{
			name:    "empty string is valid (wildcard scan)",
			lunID:   "",
			wantErr: false,
		},
		{
			name:    "zero is valid",
			lunID:   "0",
			wantErr: false,
		},
		{
			name:    "positive integer is valid",
			lunID:   "42",
			wantErr: false,
		},
		{
			name:    "large LUN ID is valid",
			lunID:   "16383",
			wantErr: false,
		},
		{
			name:    "negative one is invalid",
			lunID:   "-1",
			wantErr: true,
		},
		{
			name:    "other negative value is invalid",
			lunID:   "-100",
			wantErr: true,
		},
		{
			name:    "non-numeric string is invalid",
			lunID:   "abc",
			wantErr: true,
		},
		{
			name:    "float string is invalid",
			lunID:   "1.5",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLunID(tt.lunID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLunID(%q) error = %v, wantErr %v", tt.lunID, err, tt.wantErr)
			}
		})
	}
}

func TestRescanIscsiHostsForLun_InvalidLunID(t *testing.T) {
	err := RescanIscsiHostsForLun([]string{"0"}, "-1")
	if err == nil {
		t.Error("RescanIscsiHostsForLun with lunID=-1 should return an error")
	}
}

func TestRescanIscsiHostsForLun_NonNumericLunID(t *testing.T) {
	err := RescanIscsiHostsForLun([]string{"0"}, "abc")
	if err == nil {
		t.Error("RescanIscsiHostsForLun with non-numeric lunID should return an error")
	}
}

func TestRescanIscsiHostsForLun_EmptyLunID(t *testing.T) {
	// Empty lunID is valid (triggers wildcard scan "- - -").
	// This will fail if the sysfs path doesn't exist, but validation should pass.
	// We only verify validation doesn't reject it.
	err := ValidateLunID("")
	if err != nil {
		t.Errorf("ValidateLunID with empty lunID should not fail validation: %v", err)
	}
}

func TestRescanFcTarget_InvalidLunID(t *testing.T) {
	err := RescanFcTarget("-1")
	if err == nil {
		t.Error("RescanFcTarget with lunID=-1 should return an error")
	}
}

func TestRescanScsiHosts_InvalidLunID(t *testing.T) {
	err := RescanScsiHosts([]string{"host0"}, "-1")
	if err == nil {
		t.Error("RescanScsiHosts with lunID=-1 should return an error")
	}
}

func TestRescanFcHostsForLun_InvalidLunID(t *testing.T) {
	err := RescanFcHostsForLun([]string{"0"}, "-1")
	if err == nil {
		t.Error("RescanFcHostsForLun with lunID=-1 should return an error")
	}
}
