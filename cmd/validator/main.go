// Package main defines a validator client, a critical actor in Ethereum which manages
// a keystore of private keys, connects to a beacon node to receive assignments,
// and submits blocks/attestations as needed.
package main

import (
	"os"
	"path/filepath"
	runtimeDebug "runtime/debug"

	"github.com/OffchainLabs/prysm/v7/cmd"
	cmdcommon "github.com/OffchainLabs/prysm/v7/cmd/common"
	accountcommands "github.com/OffchainLabs/prysm/v7/cmd/validator/accounts"
	dbcommands "github.com/OffchainLabs/prysm/v7/cmd/validator/db"
	"github.com/OffchainLabs/prysm/v7/cmd/validator/flags"
	slashingprotectioncommands "github.com/OffchainLabs/prysm/v7/cmd/validator/slashing-protection"
	walletcommands "github.com/OffchainLabs/prysm/v7/cmd/validator/wallet"
	"github.com/OffchainLabs/prysm/v7/cmd/validator/web"
	"github.com/OffchainLabs/prysm/v7/config/features"
	"github.com/OffchainLabs/prysm/v7/io/file"
	"github.com/OffchainLabs/prysm/v7/runtime/debug"
	_ "github.com/OffchainLabs/prysm/v7/runtime/maxprocs"
	"github.com/OffchainLabs/prysm/v7/runtime/tos"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/OffchainLabs/prysm/v7/validator/node"
	"github.com/urfave/cli/v2"
)

func startNode(ctx *cli.Context) error {
	// Verify if ToS is accepted.
	if err := tos.VerifyTosAcceptedOrPrompt(ctx); err != nil {
		return err
	}

	validatorClient, err := node.NewValidatorClient(ctx)
	if err != nil {
		return err
	}
	validatorClient.Start()
	return nil
}

var appFlags = []cli.Flag{
	flags.BeaconRPCProviderFlag,
	flags.BeaconRESTApiProviderFlag,
	flags.BeaconRESTApiHeaders,
	flags.CertFlag,
	flags.GraffitiFlag,
	flags.DisablePenaltyRewardLogFlag,
	flags.InteropStartIndex,
	flags.InteropNumValidators,
	flags.EnableRPCFlag,
	flags.RPCHost,
	flags.RPCPort,
	flags.HTTPServerPort,
	flags.HTTPServerHost,
	flags.GRPCRetriesFlag,
	flags.GRPCRetryDelayFlag,
	flags.GRPCHeadersFlag,
	flags.HTTPServerCorsDomain,
	flags.DisableAccountMetricsFlag,
	flags.MonitoringPortFlag,
	flags.SlasherRPCProviderFlag,
	flags.SlasherCertFlag,
	flags.WalletPasswordFileFlag,
	flags.WalletDirFlag,
	flags.GraffitiFileFlag,
	flags.EnableDistributed,
	flags.AuthTokenPathFlag,
	flags.DisableDutiesPolling,
	flags.MaxHealthChecksFlag,
	// Consensys' Web3Signer flags
	flags.Web3SignerURLFlag,
	flags.Web3SignerPublicValidatorKeysFlag,
	flags.Web3SignerKeyFileFlag,
	flags.SuggestedFeeRecipientFlag,
	flags.ProposerSettingsURLFlag,
	flags.ProposerSettingsFlag,
	flags.EnableBuilderFlag,
	flags.BuilderGasLimitFlag,
	flags.ValidatorsRegistrationBatchSizeFlag,
	////////////////////
	cmd.DisableMonitoringFlag,
	cmd.MonitoringHostFlag,
	cmd.EnableBackupWebhookFlag,
	cmd.MinimalConfigFlag,
	cmd.E2EConfigFlag,
	cmd.VerbosityFlag,
	cmd.LogVModuleFlag,
	cmd.DataDirFlag,
	cmd.ClearDB,
	cmd.ForceClearDB,
	cmd.EnableTracingFlag,
	cmd.TracingProcessNameFlag,
	cmd.TracingEndpointFlag,
	cmd.TraceSampleFractionFlag,
	cmd.LogFormat,
	cmd.LogFileName,
	cmd.ConfigFileFlag,
	cmd.ChainConfigFileFlag,
	cmd.GrpcMaxCallRecvMsgSizeFlag,
	cmd.ApiTimeoutFlag,
	debug.PProfFlag,
	debug.PProfAddrFlag,
	debug.PProfPortFlag,
	debug.MemProfileRateFlag,
	debug.BlockProfileRateFlag,
	debug.MutexProfileFractionFlag,
	cmd.AcceptTosFlag,
	flags.DisableEphemeralLogFile,
}

func init() {
	appFlags = cmd.WrapFlags(append(appFlags, features.ValidatorFlags...))
}

func main() {
	app := cli.App{
		Name:    "validator",
		Usage:   "Launches an Ethereum validator client that interacts with a beacon chain, starts proposer and attester services, p2p connections, and more.",
		Version: version.Version(),
		Action: func(ctx *cli.Context) error {
			if err := startNode(ctx); err != nil {
				log.Fatal(err.Error())
				return err
			}
			return nil
		},
		Commands: []*cli.Command{
			walletcommands.Commands,
			accountcommands.Commands,
			slashingprotectioncommands.Commands,
			dbcommands.Commands,
			web.Commands,
			cmd.CompletionCommand("validator"),
		},
		Flags:                appFlags,
		EnableBashCompletion: true,
		Before: func(ctx *cli.Context) error {
			// Run the shared logging/debug/feature setup.
			if err := cmdcommon.Before("Prysm Validator", appFlags, flags.DisableEphemeralLogFile)(ctx); err != nil {
				return err
			}

			// Validator-specific setup: fix data dir for Windows users.
			outdatedDataDir := filepath.Join(file.HomeDir(), "AppData", "Roaming", "Eth2Validators")
			currentDataDir := flags.DefaultValidatorDir()
			if err := cmd.FixDefaultDataDir(outdatedDataDir, currentDataDir); err != nil {
				log.WithError(err).Error("Cannot update data directory")
			}

			return nil
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
