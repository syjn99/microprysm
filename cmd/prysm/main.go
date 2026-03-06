// Package main defines the unified prysm binary that can run
// a beacon node, validator client, or both as subcommands.
package main

import (
	"os"
	runtimeDebug "runtime/debug"

	beaconapp "github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/app"
	validatorapp "github.com/OffchainLabs/prysm/v7/cmd/validator/app"
	_ "github.com/OffchainLabs/prysm/v7/runtime/maxprocs"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "prysm",
		Usage:   "Ethereum consensus layer client — beacon node, validator, or both",
		Version: version.Version(),
		Commands: []*cli.Command{
			beaconapp.Command(),
			validatorapp.Command(),
			combinedCommand(),
		},
	}

	defer func() {
		if x := recover(); x != nil {
			log.Errorf("Runtime panic: %v\n%v", x, string(runtimeDebug.Stack()))
			panic(x) // lint:nopanic -- This is just resurfacing the original panic.
		}
	}()

	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
	}
}
