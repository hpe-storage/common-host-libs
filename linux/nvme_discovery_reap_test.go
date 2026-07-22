// Copyright 2025 Hewlett Packard Enterprise Development LP

package linux

import (
	"fmt"
	"testing"
)

// TestReapNvmeDiscoveryControllers_DisconnectsDiscoveryNQN verifies the reaper
// issues "nvme disconnect -n <discovery NQN>" so orphaned discovery controllers
// created by "nvme discover" are cleaned up.
func TestReapNvmeDiscoveryControllers_DisconnectsDiscoveryNQN(t *testing.T) {
	orig := nvmeExecCommandOutput
	defer func() { nvmeExecCommandOutput = orig }()

	var gotCmd string
	var gotArgs []string
	nvmeExecCommandOutput = func(cmd string, args []string) (string, int, error) {
		gotCmd = cmd
		gotArgs = args
		return "", 0, nil
	}

	reapNvmeDiscoveryControllers()

	if gotCmd != nvmecmd {
		t.Fatalf("cmd = %q, want %q", gotCmd, nvmecmd)
	}
	want := []string{"disconnect", "-n", nvmeDiscoveryNQN}
	if len(gotArgs) != len(want) {
		t.Fatalf("args = %v, want %v", gotArgs, want)
	}
	for i := range want {
		if gotArgs[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q", i, gotArgs[i], want[i])
		}
	}
	if nvmeDiscoveryNQN != "nqn.2014-08.org.nvmexpress.discovery" {
		t.Fatalf("nvmeDiscoveryNQN = %q, want the well-known discovery NQN", nvmeDiscoveryNQN)
	}
}

// TestReapNvmeDiscoveryControllers_ToleratesError ensures the reaper does not
// panic or propagate when there are no discovery controllers to disconnect.
func TestReapNvmeDiscoveryControllers_ToleratesError(t *testing.T) {
	orig := nvmeExecCommandOutput
	defer func() { nvmeExecCommandOutput = orig }()

	nvmeExecCommandOutput = func(cmd string, args []string) (string, int, error) {
		return "no controllers found", 1, fmt.Errorf("exit status 1")
	}

	reapNvmeDiscoveryControllers() // must not panic
}

// TestDiscoverNvmeEndpoints_ReapsControllers verifies discoverNvmeEndpoints
// reaps discovery controllers even when no discovery IPs are provided is a
// no-op, and that with IPs it always attempts the reap (via the deferred call).
func TestDiscoverNvmeEndpoints_ReapsControllers(t *testing.T) {
	orig := nvmeExecCommandOutput
	defer func() { nvmeExecCommandOutput = orig }()

	var disconnectCalls int
	nvmeExecCommandOutput = func(cmd string, args []string) (string, int, error) {
		if len(args) > 0 && args[0] == "disconnect" {
			disconnectCalls++
		}
		// Return empty discovery output for "discover" calls.
		return "", 0, nil
	}

	_, _ = discoverNvmeEndpoints("nqn.test", []string{"10.0.0.1"})
	if disconnectCalls != 1 {
		t.Fatalf("expected exactly 1 discovery-controller reap, got %d", disconnectCalls)
	}
}
