// Package app exports the validator CLI command for use by the unified prysm binary.
package app

import (
	"path/filepath"

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
	"github.com/OffchainLabs/prysm/v7/runtime/tos"
	"github.com/OffchainLabs/prysm/v7/validator/node"
	"github.com/urfave/cli/v2"
)

// AppFlags returns the full set of flags for the validator command.
func AppFlags() []cli.Flag {
	appFlags := []cli.Flag{
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
	return cmd.WrapFlags(append(appFlags, features.ValidatorFlags...))
}

// Before returns the cli.BeforeFunc for the validator command.
// It runs shared logging setup plus validator-specific initialization.
func Before(appFlags []cli.Flag) cli.BeforeFunc {
	return func(ctx *cli.Context) error {
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
	}
}

// StartNode starts the validator client. It handles ToS verification
// and validator client construction.
func StartNode(ctx *cli.Context) error {
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

// Command returns the *cli.Command for the validator subcommand.
func Command() *cli.Command {
	appFlags := AppFlags()
	return &cli.Command{
		Name:  "validator",
		Usage: "Launches an Ethereum validator client that interacts with a beacon chain, starts proposer and attester services, p2p connections, and more.",
		Action: func(ctx *cli.Context) error {
			if err := StartNode(ctx); err != nil {
				log.Fatal(err.Error())
				return err
			}
			return nil
		},
		Subcommands: []*cli.Command{
			walletcommands.Commands,
			accountcommands.Commands,
			slashingprotectioncommands.Commands,
			dbcommands.Commands,
			web.Commands,
		},
		Flags:  appFlags,
		Before: Before(appFlags),
	}
}
