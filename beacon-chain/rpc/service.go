// Package rpc defines an HTTP server implementing the Ethereum consensus API as needed
// by validator clients and consumers of chain data.
package rpc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/blockchain"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/builder"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/cache"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/cache/depositsnapshot"
	blockfeed "github.com/OffchainLabs/prysm/v7/beacon-chain/core/feed/block"
	opfeed "github.com/OffchainLabs/prysm/v7/beacon-chain/core/feed/operation"
	statefeed "github.com/OffchainLabs/prysm/v7/beacon-chain/core/feed/state"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/db"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/db/filesystem"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/execution"
	lightClient "github.com/OffchainLabs/prysm/v7/beacon-chain/light-client"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/operations/attestations"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/operations/blstoexec"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/operations/slashings"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/operations/synccommittee"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/operations/voluntaryexits"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/p2p"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/rpc/core"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/rpc/eth/rewards"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/rpc/lookup"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/rpc/proposer"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/startup"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state/stategen"
	chainSync "github.com/OffchainLabs/prysm/v7/beacon-chain/sync"
	"github.com/pkg/errors"
)

// Service defining an RPC server for a beacon node.
type Service struct {
	cfg            *Config
	ctx            context.Context
	cancel         context.CancelFunc
	proposerServer *proposer.Server
}

// Config options for the beacon node RPC server.
type Config struct {
	ExecutionReconstructor           execution.Reconstructor
	BeaconMonitoringHost             string
	BeaconMonitoringPort             int
	BeaconDB                         db.HeadAccessDatabase
	ChainInfoFetcher                 blockchain.ChainInfoFetcher
	HeadFetcher                      blockchain.HeadFetcher
	CanonicalFetcher                 blockchain.CanonicalFetcher
	ForkFetcher                      blockchain.ForkFetcher
	ForkchoiceFetcher                blockchain.ForkchoiceFetcher
	FinalizationFetcher              blockchain.FinalizationFetcher
	AttestationReceiver              blockchain.AttestationReceiver
	BlockReceiver                    blockchain.BlockReceiver
	ExecutionPayloadEnvelopeReceiver blockchain.ExecutionPayloadEnvelopeReceiver
	BlobReceiver                     blockchain.BlobReceiver
	DataColumnReceiver               blockchain.DataColumnReceiver
	ExecutionChainService            execution.Chain
	ChainStartFetcher                execution.ChainStartFetcher
	ExecutionChainInfoFetcher        execution.ChainInfoFetcher
	GenesisTimeFetcher               blockchain.TimeFetcher
	GenesisFetcher                   blockchain.GenesisFetcher
	MockEth1Votes                    bool
	EnableDebugRPCEndpoints          bool
	AttestationCache                 *cache.AttestationCache
	AttestationsPool                 attestations.Pool
	ExitPool                         voluntaryexits.PoolManager
	SlashingsPool                    slashings.PoolManager
	SyncCommitteeObjectPool          synccommittee.Pool
	BLSChangesPool                   blstoexec.PoolManager
	SyncService                      chainSync.Checker
	Broadcaster                      p2p.Broadcaster
	PeersFetcher                     p2p.PeersProvider
	PeerManager                      p2p.PeerManager
	MetadataProvider                 p2p.MetadataProvider
	DepositFetcher                   cache.DepositFetcher
	PendingDepositFetcher            depositsnapshot.PendingDepositsFetcher
	StateNotifier                    statefeed.Notifier
	BlockNotifier                    blockfeed.Notifier
	OperationNotifier                opfeed.Notifier
	StateGen                         *stategen.State
	ExecutionEngineCaller            execution.EngineCaller
	OptimisticModeFetcher            blockchain.OptimisticModeFetcher
	BlockBuilder                     builder.BlockBuilder
	Router                           *http.ServeMux
	ClockWaiter                      startup.ClockWaiter
	BlobStorage                      *filesystem.BlobStorage
	DataColumnStorage                *filesystem.DataColumnStorage
	TrackedValidatorsCache           *cache.TrackedValidatorsCache
	PayloadIDCache                   *cache.PayloadIDCache
	LCStore                          *lightClient.Store
	GraffitiInfo                     *execution.GraffitiInfo
}

// NewService instantiates a new RPC service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	s := &Service{
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
	}

	var stateCache stategen.CachedGetter
	if s.cfg.StateGen != nil {
		stateCache = s.cfg.StateGen.CombinedCache()
	}
	withCache := stategen.WithCache(stateCache)
	ch := stategen.NewCanonicalHistory(s.cfg.BeaconDB, s.cfg.ChainInfoFetcher, s.cfg.ChainInfoFetcher, withCache)
	stater := &lookup.BeaconDbStater{
		BeaconDB:           s.cfg.BeaconDB,
		ChainInfoFetcher:   s.cfg.ChainInfoFetcher,
		GenesisTimeFetcher: s.cfg.GenesisTimeFetcher,
		StateGenService:    s.cfg.StateGen,
		ReplayerBuilder:    ch,
	}
	blocker := &lookup.BeaconDbBlocker{
		BeaconDB:           s.cfg.BeaconDB,
		ChainInfoFetcher:   s.cfg.ChainInfoFetcher,
		GenesisTimeFetcher: s.cfg.GenesisTimeFetcher,
		BlobStorage:        s.cfg.BlobStorage,
		DataColumnStorage:  s.cfg.DataColumnStorage,
	}
	rewardFetcher := &rewards.BlockRewardService{Replayer: ch, DB: s.cfg.BeaconDB}
	coreService := &core.Service{
		BeaconDB:              s.cfg.BeaconDB,
		HeadFetcher:           s.cfg.HeadFetcher,
		GenesisTimeFetcher:    s.cfg.GenesisTimeFetcher,
		SyncChecker:           s.cfg.SyncService,
		Broadcaster:           s.cfg.Broadcaster,
		SyncCommitteePool:     s.cfg.SyncCommitteeObjectPool,
		OperationNotifier:     s.cfg.OperationNotifier,
		AttestationCache:      cache.NewAttestationDataCache(),
		StateGen:              s.cfg.StateGen,
		P2P:                   s.cfg.Broadcaster,
		FinalizedFetcher:      s.cfg.FinalizationFetcher,
		ReplayerBuilder:       ch,
		OptimisticModeFetcher: s.cfg.OptimisticModeFetcher,
	}
	proposerServer := &proposer.Server{
		Ctx:                              s.ctx,
		AttestationCache:                 s.cfg.AttestationCache,
		AttPool:                          s.cfg.AttestationsPool,
		ExitPool:                         s.cfg.ExitPool,
		HeadFetcher:                      s.cfg.HeadFetcher,
		ForkFetcher:                      s.cfg.ForkFetcher,
		ForkchoiceFetcher:                s.cfg.ForkchoiceFetcher,
		GenesisFetcher:                   s.cfg.GenesisFetcher,
		FinalizationFetcher:              s.cfg.FinalizationFetcher,
		TimeFetcher:                      s.cfg.GenesisTimeFetcher,
		BlockFetcher:                     s.cfg.ExecutionChainService,
		DepositFetcher:                   s.cfg.DepositFetcher,
		ChainStartFetcher:                s.cfg.ChainStartFetcher,
		Eth1InfoFetcher:                  s.cfg.ExecutionChainService,
		OptimisticModeFetcher:            s.cfg.OptimisticModeFetcher,
		SyncChecker:                      s.cfg.SyncService,
		StateNotifier:                    s.cfg.StateNotifier,
		BlockNotifier:                    s.cfg.BlockNotifier,
		OperationNotifier:                s.cfg.OperationNotifier,
		P2P:                              s.cfg.Broadcaster,
		BlockReceiver:                    s.cfg.BlockReceiver,
		ExecutionPayloadEnvelopeReceiver: s.cfg.ExecutionPayloadEnvelopeReceiver,
		BlobReceiver:                     s.cfg.BlobReceiver,
		DataColumnReceiver:               s.cfg.DataColumnReceiver,
		MockEth1Votes:                    s.cfg.MockEth1Votes,
		Eth1BlockFetcher:                 s.cfg.ExecutionChainService,
		PendingDepositsFetcher:           s.cfg.PendingDepositFetcher,
		SlashingsPool:                    s.cfg.SlashingsPool,
		StateGen:                         s.cfg.StateGen,
		SyncCommitteePool:                s.cfg.SyncCommitteeObjectPool,
		ReplayerBuilder:                  ch,
		ExecutionEngineCaller:            s.cfg.ExecutionEngineCaller,
		BeaconDB:                         s.cfg.BeaconDB,
		BlockBuilder:                     s.cfg.BlockBuilder,
		BLSChangesPool:                   s.cfg.BLSChangesPool,
		ClockWaiter:                      s.cfg.ClockWaiter,
		CoreService:                      coreService,
		TrackedValidatorsCache:           s.cfg.TrackedValidatorsCache,
		PayloadIDCache:                   s.cfg.PayloadIDCache,
		AttestationStateFetcher:          s.cfg.AttestationReceiver,
		GraffitiInfo:                     s.cfg.GraffitiInfo,
	}
	s.proposerServer = proposerServer
	endpoints := s.endpoints(s.cfg.EnableDebugRPCEndpoints, blocker, stater, rewardFetcher, proposerServer, coreService, ch)
	for _, e := range endpoints {
		for i := range e.methods {
			s.cfg.Router.HandleFunc(
				fmt.Sprintf("%s %s", e.methods[i], e.template),
				e.handlerWithMiddleware(),
			)
		}
	}

	return s
}

// paranoid build time check to ensure ChainInfoFetcher implements required interfaces
var _ stategen.CanonicalChecker = blockchain.ChainInfoFetcher(nil)
var _ stategen.CurrentSlotter = blockchain.ChainInfoFetcher(nil)

// Start the service.
func (s *Service) Start() {
}

// Stop the service.
func (s *Service) Stop() error {
	s.cancel()
	return nil
}

// Status returns nil or an error if the service is not healthy.
func (s *Service) Status() error {
	optimistic, err := s.cfg.OptimisticModeFetcher.IsOptimistic(s.ctx)
	if err != nil {
		return errors.Wrap(err, "failed to check if service is optimistic")
	}
	if optimistic {
		return errors.New("service is optimistic, validators can't perform duties " +
			"please check if execution layer is fully synced")
	}
	if s.cfg.SyncService.Syncing() {
		return errors.New("syncing")
	}
	return nil
}
