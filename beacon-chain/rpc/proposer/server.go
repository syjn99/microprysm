// Package proposer implements block proposal logic extracted from the
// v1alpha1 validator gRPC service.
package proposer

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/blockchain"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/builder"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/cache"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/cache/depositsnapshot"
	blockfeed "github.com/OffchainLabs/prysm/v7/beacon-chain/core/feed/block"
	opfeed "github.com/OffchainLabs/prysm/v7/beacon-chain/core/feed/operation"
	statefeed "github.com/OffchainLabs/prysm/v7/beacon-chain/core/feed/state"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/db"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/execution"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/operations/attestations"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/operations/blstoexec"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/operations/slashings"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/operations/synccommittee"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/operations/voluntaryexits"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/p2p"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/rpc/core"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/startup"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state/stategen"
	prysmSync "github.com/OffchainLabs/prysm/v7/beacon-chain/sync"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/time/slots"
)

var errOptimisticMode = errors.New("the node is currently optimistic and cannot serve validators")

// Server defines a server implementation for block proposal logic.
type Server struct {
	Ctx                              context.Context
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
	StateNotifier                    statefeed.Notifier
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
	ClockWaiter                      startup.ClockWaiter
	CoreService                      *core.Service
	AttestationStateFetcher          blockchain.AttestationStateFetcher
	GraffitiInfo                     *execution.GraffitiInfo
}

// ValidatorIndex is called by a validator to get its index location in the beacon state.
func (vs *Server) ValidatorIndex(ctx context.Context, req *ethpb.ValidatorIndexRequest) (*ethpb.ValidatorIndexResponse, error) {
	st, err := vs.HeadFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		return nil, fmt.Errorf("Could not determine head state: %v", err)
	}
	if st == nil || st.IsNil() {
		return nil, errors.New("head state is empty")
	}
	index, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(req.PublicKey))
	if !ok {
		return nil, fmt.Errorf("Could not find validator index for public key %#x", req.PublicKey)
	}

	return &ethpb.ValidatorIndexResponse{Index: index}, nil
}

// optimisticStatus returns an error if the node is currently optimistic with respect to head.
func (vs *Server) optimisticStatus(ctx context.Context) error {
	if slots.ToEpoch(vs.TimeFetcher.CurrentSlot()) < params.BeaconConfig().BellatrixForkEpoch {
		return nil
	}
	optimistic, err := vs.OptimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		return fmt.Errorf("Could not determine if the node is a optimistic node: %v", err)
	}
	if !optimistic {
		return nil
	}

	return fmt.Errorf("error=%v", errOptimisticMode)
}
