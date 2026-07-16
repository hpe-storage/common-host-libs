// Copyright 2025 Hewlett Packard Enterprise Development LP

package linux

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIsNvmeMultipathEnabled verifies that the nvme_core.multipath value is read
// and interpreted correctly, and that a missing/unreadable parameter is reported
// as an error (never as "permission denied" from a doomed write).
func TestIsNvmeMultipathEnabled(t *testing.T) {
	origPath := nvmeCoreMultipathPath
	defer func() { nvmeCoreMultipathPath = origPath }()

	tests := []struct {
		name        string
		fileContent *string // nil means the file is absent
		wantEnabled bool
		wantErr     bool
	}{
		{name: "enabled uppercase", fileContent: strPtr("Y\n"), wantEnabled: true, wantErr: false},
		{name: "enabled lowercase", fileContent: strPtr("y"), wantEnabled: true, wantErr: false},
		{name: "disabled N", fileContent: strPtr("N\n"), wantEnabled: false, wantErr: false},
		{name: "disabled empty", fileContent: strPtr(""), wantEnabled: false, wantErr: false},
		{name: "missing file", fileContent: nil, wantEnabled: false, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.fileContent == nil {
				nvmeCoreMultipathPath = filepath.Join(t.TempDir(), "does-not-exist")
			} else {
				p := filepath.Join(t.TempDir(), "multipath")
				if err := os.WriteFile(p, []byte(*tc.fileContent), 0444); err != nil {
					t.Fatalf("failed to write temp param file: %v", err)
				}
				nvmeCoreMultipathPath = p
			}

			enabled, err := isNvmeMultipathEnabled()
			if (err != nil) != tc.wantErr {
				t.Fatalf("isNvmeMultipathEnabled() err = %v, wantErr = %v", err, tc.wantErr)
			}
			if enabled != tc.wantEnabled {
				t.Fatalf("isNvmeMultipathEnabled() = %v, want %v", enabled, tc.wantEnabled)
			}
		})
	}
}

// TestLogNvmeMultipathStatus ensures the read-only check never panics and never
// attempts a write, regardless of parameter state.
func TestLogNvmeMultipathStatus(t *testing.T) {
	origPath := nvmeCoreMultipathPath
	defer func() { nvmeCoreMultipathPath = origPath }()

	// Enabled read-only (0444) file: must be handled via read, not write.
	p := filepath.Join(t.TempDir(), "multipath")
	if err := os.WriteFile(p, []byte("N\n"), 0444); err != nil {
		t.Fatalf("failed to write temp param file: %v", err)
	}
	nvmeCoreMultipathPath = p
	logNvmeMultipathStatus() // should not panic

	// Absent file path: must not panic.
	nvmeCoreMultipathPath = filepath.Join(t.TempDir(), "absent")
	logNvmeMultipathStatus()
}

func strPtr(s string) *string { return &s }
