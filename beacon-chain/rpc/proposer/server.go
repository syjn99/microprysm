package proposer

import (
	"sync"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/blockchain"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/builder"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/cache"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/cache/depositsnapshot"
	blockfeed "github.com/OffchainLabs/prysm/v7/beacon-chain/core/feed/block"
	opfeed "github.com/OffchainLabs/prysm/v7/beacon-chain/core/feed/operation"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/db"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/execution"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/operations/attestations"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/operations/blstoexec"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/operations/slashings"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/operations/synccommittee"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/operations/voluntaryexits"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/p2p"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/rpc/core"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state/stategen"
	prysmSync "github.com/OffchainLabs/prysm/v7/beacon-chain/sync"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
)

// Server defines a server implementation for block proposal and broadcasting,
// providing the proposer logic used by REST API handlers.
type Server struct {
	PayloadIDCache                   *cache.PayloadIDCache
	TrackedValidatorsCache           *cache.TrackedValidatorsCache
	executionPayloadEnvelopeMu       sync.RWMutex
	executionPayloadEnvelope         *ethpb.ExecutionPayloadEnvelope
	HeadFetcher                      blockchain.HeadFetcher
	ForkFetcher                      blockchain.ForkFetcher
	ForkchoiceFetcher                blockchain.ForkchoiceFetcher
	GenesisFetcher                   blockchain.GenesisFetcher
	FinalizationFetcher              blockchain.FinalizationFetcher
	TimeFetcher                      blockchain.TimeFetcher
	BlockFetcher                     execution.POWBlockFetcher
	DepositFetcher                   cache.DepositFetcher
	ChainStartFetcher                execution.ChainStartFetcher
	Eth1InfoFetcher                  execution.ChainInfoFetcher
	OptimisticModeFetcher            blockchain.OptimisticModeFetcher
	SyncChecker                      prysmSync.Checker
	BlockNotifier                    blockfeed.Notifier
	P2P                              p2p.Broadcaster
	AttestationCache                 *cache.AttestationCache
	AttPool                          attestations.Pool
	SlashingsPool                    slashings.PoolManager
	ExitPool                         voluntaryexits.PoolManager
	SyncCommitteePool                synccommittee.Pool
	BlockReceiver                    blockchain.BlockReceiver
	ExecutionPayloadEnvelopeReceiver blockchain.ExecutionPayloadEnvelopeReceiver
	BlobReceiver                     blockchain.BlobReceiver
	DataColumnReceiver               blockchain.DataColumnReceiver
	MockEth1Votes                    bool
	Eth1BlockFetcher                 execution.POWBlockFetcher
	PendingDepositsFetcher           depositsnapshot.PendingDepositsFetcher
	OperationNotifier                opfeed.Notifier
	StateGen                         stategen.StateManager
	ReplayerBuilder                  stategen.ReplayerBuilder
	BeaconDB                         db.HeadAccessDatabase
	ExecutionEngineCaller            execution.EngineCaller
	BlockBuilder                     builder.BlockBuilder
	BLSChangesPool                   blstoexec.PoolManager
	CoreService                      *core.Service
	AttestationStateFetcher          blockchain.AttestationStateFetcher
	GraffitiInfo                     *execution.GraffitiInfo
}
