package main

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/OffchainLabs/prysm/v7/testing/assert"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func TestBuildCombinedFlags(t *testing.T) {
	flags := buildCombinedFlags()

	// Should contain BN-specific flags.
	flagNames := make(map[string]bool)
	for _, f := range flags {
		for _, name := range f.Names() {
			flagNames[name] = true
		}
	}

	// BN-specific flags should be present.
	assert.Equal(t, true, flagNames["execution-endpoint"], "missing BN flag: execution-endpoint")
	assert.Equal(t, true, flagNames["rpc-port"], "missing BN flag: rpc-port")

	// VC-specific flags should be present.
	assert.Equal(t, true, flagNames["beacon-rpc-provider"], "missing VC flag: beacon-rpc-provider")
	assert.Equal(t, true, flagNames["wallet-dir"], "missing VC flag: wallet-dir")
	assert.Equal(t, true, flagNames["graffiti"], "missing VC flag: graffiti")

	// Combined-specific flags should be present.
	assert.Equal(t, true, flagNames["vc-monitoring-port"], "missing combined flag: vc-monitoring-port")

	// Shared flags should appear only once (no duplicates).
	nameCount := make(map[string]int)
	for _, f := range flags {
		for _, name := range f.Names() {
			nameCount[name]++
		}
	}
	assert.Equal(t, 1, nameCount["verbosity"], "shared flag 'verbosity' should appear exactly once")
	assert.Equal(t, 1, nameCount["datadir"], "shared flag 'datadir' should appear exactly once")
}

func TestWaitForGRPC_Success(t *testing.T) {
	// Start a TCP listener.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, ln.Close())
	}()

	ctx := context.Background()
	err = waitForGRPC(ctx, ln.Addr().String(), 5*time.Second)
	require.NoError(t, err)
}

func TestWaitForGRPC_Timeout(t *testing.T) {
	// Use a port that nothing is listening on.
	ctx := context.Background()
	err := waitForGRPC(ctx, "127.0.0.1:1", 500*time.Millisecond)
	require.ErrorContains(t, "timeout waiting for", err)
}

func TestWaitForGRPC_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := waitForGRPC(ctx, "127.0.0.1:1", 5*time.Second)
	require.ErrorContains(t, "context canceled", err)
}
