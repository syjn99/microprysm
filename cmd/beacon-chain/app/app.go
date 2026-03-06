// Package app exports the beacon-chain CLI command for use by the unified prysm binary.
package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	runtimeDebug "runtime/debug"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/builder"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/node"
	"github.com/OffchainLabs/prysm/v7/cmd"
	blockchaincmd "github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/blockchain"
	das "github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/das"
	dasFlags "github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/das/flags"
	dbcommands "github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/db"
	"github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/execution"
	"github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/flags"
	"github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/genesis"
	jwtcommands "github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/jwt"
	"github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/storage"
	backfill "github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/sync/backfill"
	bflags "github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/sync/backfill/flags"
	"github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/sync/checkpoint"
	cmdcommon "github.com/OffchainLabs/prysm/v7/cmd/common"
	"github.com/OffchainLabs/prysm/v7/config/features"
	"github.com/OffchainLabs/prysm/v7/io/file"
	"github.com/OffchainLabs/prysm/v7/runtime/debug"
	"github.com/OffchainLabs/prysm/v7/runtime/fdlimits"
	"github.com/OffchainLabs/prysm/v7/runtime/tos"
	gethlog "github.com/ethereum/go-ethereum/log"
	golog "github.com/ipfs/go-log/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// AppFlags returns the full set of flags for the beacon-chain command.
func AppFlags() []cli.Flag {
	appFlags := []cli.Flag{
		flags.DepositContractFlag,
		flags.ExecutionEngineEndpoint,
		flags.ExecutionEngineHeaders,
		flags.ExecutionJWTSecretFlag,
		flags.RPCHost,
		flags.RPCPort,
		flags.CertFlag,
		flags.KeyFlag,
		flags.HTTPModules,
		flags.HTTPServerHost,
		flags.HTTPServerPort,
		flags.HTTPServerCorsDomain,
		flags.MinSyncPeers,
		flags.ContractDeploymentBlock,
		flags.SetGCPercent,
		flags.BlockBatchLimit,
		flags.BlockBatchLimitBurstFactor,
		flags.BlobBatchLimit,
		flags.BlobBatchLimitBurstFactor,
		flags.DataColumnBatchLimit,
		flags.DataColumnBatchLimitBurstFactor,
		flags.InteropMockEth1DataVotesFlag,
		flags.SlotsPerArchivedPoint,
		flags.DisableDebugRPCEndpoints,
		flags.SubscribeToAllSubnets,
		flags.Supernode,
		flags.SemiSupernode,
		flags.HistoricalSlasherNode,
		flags.ChainID,
		flags.NetworkID,
		flags.WeakSubjectivityCheckpoint,
		flags.Eth1HeaderReqLimit,
		flags.MinPeersPerSubnet,
		flags.MaxConcurrentDials,
		flags.SuggestedFeeRecipient,
		flags.TerminalTotalDifficultyOverride,
		flags.TerminalBlockHashOverride,
		flags.TerminalBlockHashActivationEpochOverride,
		flags.MevRelayEndpoint,
		flags.MaxBuilderEpochMissedSlots,
		flags.MaxBuilderConsecutiveMissedSlots,
		flags.EngineEndpointTimeoutSeconds,
		flags.LocalBlockValueBoost,
		flags.MinBuilderBid,
		flags.MinBuilderDiff,
		flags.BeaconDBPruning,
		flags.PrunerRetentionEpochs,
		flags.EnableBuilderSSZ,
		cmd.MinimalConfigFlag,
		cmd.E2EConfigFlag,
		cmd.RPCMaxPageSizeFlag,
		cmd.BootstrapNode,
		cmd.NoDiscovery,
		cmd.StaticPeers,
		cmd.RelayNode,
		cmd.P2PUDPPort,
		cmd.P2PQUICPort,
		cmd.P2PTCPPort,
		cmd.P2PIP,
		cmd.P2PHost,
		cmd.P2PHostDNS,
		cmd.P2PMaxPeers,
		cmd.P2PPrivKey,
		cmd.P2PStaticID,
		cmd.P2PAllowList,
		cmd.P2PDenyList,
		cmd.P2PColocationWhitelist,
		cmd.PubsubQueueSize,
		cmd.DataDirFlag,
		cmd.VerbosityFlag,
		cmd.LogVModuleFlag,
		cmd.EnableTracingFlag,
		cmd.TracingProcessNameFlag,
		cmd.TracingEndpointFlag,
		cmd.TraceSampleFractionFlag,
		cmd.MonitoringHostFlag,
		flags.MonitoringPortFlag,
		cmd.DisableMonitoringFlag,
		cmd.ClearDB,
		cmd.ForceClearDB,
		cmd.LogFormat,
		cmd.MaxGoroutines,
		debug.PProfFlag,
		debug.PProfAddrFlag,
		debug.PProfPortFlag,
		debug.MemProfileRateFlag,
		debug.BlockProfileRateFlag,
		debug.MutexProfileFractionFlag,
		cmd.LogFileName,
		cmd.EnableUPnPFlag,
		cmd.ConfigFileFlag,
		cmd.ChainConfigFileFlag,
		cmd.GrpcMaxCallRecvMsgSizeFlag,
		cmd.AcceptTosFlag,
		cmd.RestoreSourceFileFlag,
		cmd.RestoreTargetDirFlag,
		cmd.ValidatorMonitorIndicesFlag,
		cmd.ApiTimeoutFlag,
		checkpoint.BlockPath,
		checkpoint.StatePath,
		checkpoint.RemoteURL,
		genesis.StatePath,
		genesis.BeaconAPIURL,
		flags.SlasherDirFlag,
		flags.SlasherFlag,
		flags.JwtId,
		flags.DisableGetBlobsV2,
		storage.BlobStoragePathFlag,
		storage.DataColumnStoragePathFlag,
		storage.BlobStorageLayout,
		bflags.EnableExperimentalBackfill,
		bflags.BackfillBatchSize,
		bflags.BackfillWorkerCount,
		dasFlags.BackfillOldestSlot,
		dasFlags.BlobRetentionEpochFlag,
		flags.BatchVerifierLimit,
		flags.StateDiffExponents,
		flags.DisableEphemeralLogFile,
	}
	return cmd.WrapFlags(append(appFlags, features.BeaconChainFlags...))
}

// Before returns the cli.BeforeFunc for the beacon-chain command.
// It runs shared logging setup plus beacon-chain-specific initialization.
func Before(appFlags []cli.Flag) cli.BeforeFunc {
	return func(ctx *cli.Context) error {
		// Run the shared logging/debug/feature setup.
		if err := cmdcommon.Before("Prysm Beacon Chain", appFlags, flags.DisableEphemeralLogFile)(ctx); err != nil {
			return err
		}

		// Beacon-chain-specific setup below.
		if err := cmd.ExpandSingleEndpointIfFile(ctx, flags.ExecutionEngineEndpoint); err != nil {
			return errors.Wrap(err, "failed to expand single endpoint")
		}

		if ctx.IsSet(flags.SetGCPercent.Name) {
			runtimeDebug.SetGCPercent(ctx.Int(flags.SetGCPercent.Name))
		}

		if err := fdlimits.SetMaxFdLimits(); err != nil {
			return errors.Wrap(err, "failed to set max fd limits")
		}

		return nil
	}
}

// StartNode starts the beacon node. It handles ToS verification, libp2p logger
// configuration, and beacon node construction.
func StartNode(ctx *cli.Context, cancel context.CancelFunc) error {
	// Fix data dir for Windows users.
	outdatedDataDir := filepath.Join(file.HomeDir(), "AppData", "Roaming", "Eth2")
	currentDataDir := ctx.String(cmd.DataDirFlag.Name)
	if err := cmd.FixDefaultDataDir(outdatedDataDir, currentDataDir); err != nil {
		return err
	}

	// verify if ToS accepted
	if err := tos.VerifyTosAcceptedOrPrompt(ctx); err != nil {
		return err
	}

	verbosity := ctx.String(cmd.VerbosityFlag.Name)
	level, err := logrus.ParseLevel(verbosity)
	if err != nil {
		return err
	}

	// Set libp2p logger to only panic logs for the info level.
	golog.SetAllLoggers(golog.LevelPanic)

	if level == logrus.DebugLevel {
		// Set libp2p logger to error logs for the debug level.
		golog.SetAllLoggers(golog.LevelError)
	}
	if level == logrus.TraceLevel {
		// libp2p specific logging.
		golog.SetAllLoggers(golog.LevelDebug)
		// Geth specific logging.
		gethlog.SetDefault(gethlog.NewLogger(gethlog.NewTerminalHandlerWithLevel(os.Stderr, gethlog.LvlTrace, true)))
	}

	blockchainFlagOpts, err := blockchaincmd.FlagOptions(ctx)
	if err != nil {
		return err
	}
	executionFlagOpts, err := execution.FlagOptions(ctx)
	if err != nil {
		return err
	}
	builderFlagOpts, err := builder.FlagOptions(ctx)
	if err != nil {
		return err
	}
	opts := []node.Option{
		node.WithBlockchainFlagOptions(blockchainFlagOpts),
		node.WithExecutionChainOptions(executionFlagOpts),
		node.WithBuilderFlagOptions(builderFlagOpts),
	}

	optFuncs := []func(*cli.Context) ([]node.Option, error){
		genesis.BeaconNodeOptions,
		checkpoint.BeaconNodeOptions,
		storage.BeaconNodeOptions,
		backfill.BeaconNodeOptions,
		das.BeaconNodeOptions,
	}

	beacon, err := node.New(ctx, cancel, optFuncs, opts...)
	if err != nil {
		return fmt.Errorf("unable to start beacon node: %w", err)
	}
	beacon.Start()
	return nil
}

// Command returns the *cli.Command for the beacon-chain subcommand.
func Command() *cli.Command {
	appFlags := AppFlags()
	return &cli.Command{
		Name:  "beacon-chain",
		Usage: "this is a beacon chain implementation for Ethereum",
		Action: func(ctx *cli.Context) error {
			_, cancel := context.WithCancel(ctx.Context)
			if err := StartNode(ctx, cancel); err != nil {
				log.Fatal(err.Error())
				return err
			}
			return nil
		},
		Subcommands: []*cli.Command{
			dbcommands.Commands,
			jwtcommands.Commands,
		},
		Flags:  appFlags,
		Before: Before(appFlags),
	}
}
