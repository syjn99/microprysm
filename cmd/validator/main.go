// Package main defines a validator client, a critical actor in Ethereum which manages
// a keystore of private keys, connects to a beacon node to receive assignments,
// and submits blocks/attestations as needed.
package main

import (
	"os"
	runtimeDebug "runtime/debug"

	"github.com/OffchainLabs/prysm/v7/cmd"
	validatorapp "github.com/OffchainLabs/prysm/v7/cmd/validator/app"
	_ "github.com/OffchainLabs/prysm/v7/runtime/maxprocs"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/urfave/cli/v2"
)

var appFlags = validatorapp.AppFlags()

func main() {
	// Get the full command definition from the app package.
	appCmd := validatorapp.Command()

	cliApp := cli.App{
		Name:    "validator",
		Usage:   appCmd.Usage,
		Version: version.Version(),
		Action: func(ctx *cli.Context) error {
			if err := validatorapp.StartNode(ctx); err != nil {
				log.Fatal(err.Error())
				return err
			}
			return nil
		},
		Commands:             append(appCmd.Subcommands, cmd.CompletionCommand("validator")),
		Flags:                appFlags,
		EnableBashCompletion: true,
		Before:               validatorapp.Before(appFlags),
	}

	defer func() {
		if x := recover(); x != nil {
			log.Errorf("Runtime panic: %v\n%v", x, string(runtimeDebug.Stack()))
			panic(x) // lint:nopanic -- This is just resurfacing the original panic.
		}
	}()

	if err := cliApp.Run(os.Args); err != nil {
		log.Error(err.Error())
	}
}
