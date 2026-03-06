package main

import (
	"context"
	"fmt"
	"net"
	"time"

	beaconapp "github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/app"
	beaconflags "github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/flags"
	validatorapp "github.com/OffchainLabs/prysm/v7/cmd/validator/app"
	validatorflags "github.com/OffchainLabs/prysm/v7/cmd/validator/flags"
	"github.com/urfave/cli/v2"
)

// combinedCommand returns the *cli.Command for running both BN and VC in a single process.
func combinedCommand() *cli.Command {
	combinedFlags := buildCombinedFlags()
	return &cli.Command{
		Name:  "combined",
		Usage: "Run both beacon node and validator client in a single process",
		Action: func(ctx *cli.Context) error {
			return runCombined(ctx)
		},
		Flags:  combinedFlags,
		Before: beaconapp.Before(combinedFlags),
	}
}

// buildCombinedFlags returns the union of BN and VC flags.
// VC-specific flags that collide with BN flags (--monitoring-port)
// are added with a --vc- prefix.
func buildCombinedFlags() []cli.Flag {
	// Start with all BN flags.
	bnFlags := beaconapp.AppFlags()

	// Collect VC-only flags (exclude ones already in BN set).
	bnFlagNames := make(map[string]bool)
	for _, f := range bnFlags {
		for _, name := range f.Names() {
			bnFlagNames[name] = true
		}
	}

	vcFlags := validatorapp.AppFlags()
	for _, f := range vcFlags {
		names := f.Names()
		if len(names) == 0 {
			continue
		}
		if bnFlagNames[names[0]] {
			continue // Skip flags already present in BN set.
		}
		bnFlags = append(bnFlags, f)
	}

	// Add VC-specific prefixed flags for collisions.
	bnFlags = append(bnFlags,
		&cli.IntFlag{
			Name:  "vc-monitoring-port",
			Usage: "Validator client monitoring port (default is BN monitoring-port + 1).",
			Value: 0, // 0 means auto-derive from BN monitoring port.
		},
	)

	return bnFlags
}

func runCombined(ctx *cli.Context) error {
	// Determine BN gRPC address.
	bnRPCHost := "127.0.0.1"
	if ctx.IsSet(beaconflags.RPCHost.Name) {
		bnRPCHost = ctx.String(beaconflags.RPCHost.Name)
	}
	bnRPCPort := 4000
	if ctx.IsSet(beaconflags.RPCPort.Name) {
		bnRPCPort = ctx.Int(beaconflags.RPCPort.Name)
	}
	bnGRPCAddr := fmt.Sprintf("%s:%d", bnRPCHost, bnRPCPort)

	// Start beacon node in a goroutine. Both BN and VC Start() methods block
	// until shutdown and install their own signal handlers, so on SIGINT/SIGTERM
	// both will gracefully shut down independently.
	_, bnCancel := context.WithCancel(ctx.Context)
	defer bnCancel()

	bnErrCh := make(chan error, 1)
	go func() {
		bnErrCh <- beaconapp.StartNode(ctx, bnCancel)
	}()

	// Wait for BN gRPC to become ready.
	if err := waitForGRPC(ctx.Context, bnGRPCAddr, 120*time.Second); err != nil {
		bnCancel()
		return fmt.Errorf("beacon node gRPC did not become ready: %w", err)
	}
	log.WithField("address", bnGRPCAddr).Info("Beacon node gRPC is ready, starting validator client")

	// Override VC's beacon-rpc-provider to point at the local BN.
	if err := ctx.Set(validatorflags.BeaconRPCProviderFlag.Name, bnGRPCAddr); err != nil {
		log.WithError(err).Warn("Could not auto-set beacon-rpc-provider for validator")
	}

	// Set VC monitoring port. The BN has already captured its monitoring port
	// during startup, so we can safely change the shared flag value for the VC.
	vcMonPort := ctx.Int(beaconflags.MonitoringPortFlag.Name) + 1
	if ctx.IsSet("vc-monitoring-port") && ctx.Int("vc-monitoring-port") != 0 {
		vcMonPort = ctx.Int("vc-monitoring-port")
	}
	if err := ctx.Set(beaconflags.MonitoringPortFlag.Name, fmt.Sprintf("%d", vcMonPort)); err != nil {
		log.WithError(err).Warn("Could not set validator monitoring port")
	}

	// Start validator client in a goroutine.
	vcErrCh := make(chan error, 1)
	go func() {
		vcErrCh <- validatorapp.StartNode(ctx)
	}()

	// Wait for either to finish (both block until shutdown).
	select {
	case err := <-bnErrCh:
		if err != nil {
			return fmt.Errorf("beacon node error: %w", err)
		}
	case err := <-vcErrCh:
		if err != nil {
			return fmt.Errorf("validator client error: %w", err)
		}
	}

	return nil
}

// waitForGRPC polls the given address until it accepts a TCP connection or timeout.
func waitForGRPC(ctx context.Context, addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			if err := conn.Close(); err != nil {
				return err
			}
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s after %v", addr, timeout)
}
