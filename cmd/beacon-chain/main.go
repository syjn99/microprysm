// Package beacon-chain defines the entire runtime of an Ethereum beacon node.
package main

import (
	"context"
	"os"
	runtimeDebug "runtime/debug"

	"github.com/OffchainLabs/prysm/v7/cmd"
	"github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/app"
	dbcommands "github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/db"
	jwtcommands "github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/jwt"
	_ "github.com/OffchainLabs/prysm/v7/runtime/maxprocs"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/urfave/cli/v2"
)

var appFlags = app.AppFlags()

func main() {
	// rctx = root context with cancellation.
	// note other instances of ctx in this func are *cli.Context.
	rctx, cancel := context.WithCancel(context.Background())
	cliApp := cli.App{
		Name:  "beacon-chain",
		Usage: "this is a beacon chain implementation for Ethereum",
		Action: func(ctx *cli.Context) error {
			if err := app.StartNode(ctx, cancel); err != nil {
				log.Fatal(err.Error())
				return err
			}
			return nil
		},
		Version: version.Version(),
		Commands: []*cli.Command{
			dbcommands.Commands,
			jwtcommands.Commands,
			cmd.CompletionCommand("beacon-chain"),
		},
		Flags:                appFlags,
		Before:               app.Before(appFlags),
		EnableBashCompletion: true,
	}

	defer func() {
		if x := recover(); x != nil {
			log.Errorf("Runtime panic: %v\n%v", x, string(runtimeDebug.Stack()))
			panic(x) // lint:nopanic -- This is just resurfacing the original panic.
		}
	}()

	if err := cliApp.RunContext(rctx, os.Args); err != nil {
		log.Error(err.Error())
	}
}
