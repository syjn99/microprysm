package beacon

import (
	"testing"

	chainMock "github.com/OffchainLabs/prysm/v7/beacon-chain/blockchain/testing"
	dbTest "github.com/OffchainLabs/prysm/v7/beacon-chain/db/testing"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/rpc/core"
	state_native "github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native"
	fieldparams "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/blocks"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/assert"
	"github.com/OffchainLabs/prysm/v7/testing/require"
	"github.com/OffchainLabs/prysm/v7/testing/util"
	"github.com/OffchainLabs/prysm/v7/time/slots"
)

// ensures that if any of the checkpoints are zero-valued, an error will be generated without genesis being present
func TestServer_GetChainHead_NoGenesis(t *testing.T) {
	db := dbTest.SetupDB(t)

	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetSlot(1))

	genBlock := util.NewBeaconBlock()
	genBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, fieldparams.RootLength)
	util.SaveBlock(t, t.Context(), db, genBlock)
	gRoot, err := genBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	cases := []struct {
		name       string
		zeroSetter func(val *ethpb.Checkpoint) error
	}{
		{
			name:       "zero-value prev justified",
			zeroSetter: s.SetPreviousJustifiedCheckpoint,
		},
		{
			name:       "zero-value current justified",
			zeroSetter: s.SetCurrentJustifiedCheckpoint,
		},
		{
			name:       "zero-value finalized",
			zeroSetter: s.SetFinalizedCheckpoint,
		},
	}
	finalized := &ethpb.Checkpoint{Epoch: 1, Root: gRoot[:]}
	prevJustified := &ethpb.Checkpoint{Epoch: 2, Root: gRoot[:]}
	justified := &ethpb.Checkpoint{Epoch: 3, Root: gRoot[:]}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.NoError(t, s.SetPreviousJustifiedCheckpoint(prevJustified))
			require.NoError(t, s.SetCurrentJustifiedCheckpoint(justified))
			require.NoError(t, s.SetFinalizedCheckpoint(finalized))
			require.NoError(t, c.zeroSetter(&ethpb.Checkpoint{Epoch: 0, Root: params.BeaconConfig().ZeroHash[:]}))
		})
		wsb, err := blocks.NewSignedBeaconBlock(genBlock)
		require.NoError(t, err)
		bs := &Server{
			CoreService: &core.Service{
				BeaconDB:    db,
				HeadFetcher: &chainMock.ChainService{Block: wsb, State: s},
				FinalizedFetcher: &chainMock.ChainService{
					FinalizedCheckPoint:         s.FinalizedCheckpoint(),
					CurrentJustifiedCheckPoint:  s.CurrentJustifiedCheckpoint(),
					PreviousJustifiedCheckPoint: s.PreviousJustifiedCheckpoint(),
				},
				OptimisticModeFetcher: &chainMock.ChainService{},
			},
		}
		_, err = bs.GetChainHead(t.Context(), nil)
		require.ErrorContains(t, "could not get genesis block", err)
	}
}

func TestServer_GetChainHead_NoFinalizedBlock(t *testing.T) {
	db := dbTest.SetupDB(t)

	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetSlot(1))
	require.NoError(t, s.SetPreviousJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: 3, Root: bytesutil.PadTo([]byte{'A'}, fieldparams.RootLength)}))
	require.NoError(t, s.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: 2, Root: bytesutil.PadTo([]byte{'B'}, fieldparams.RootLength)}))
	require.NoError(t, s.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'C'}, fieldparams.RootLength)}))

	genBlock := util.NewBeaconBlock()
	genBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, fieldparams.RootLength)
	util.SaveBlock(t, t.Context(), db, genBlock)
	gRoot, err := genBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(t.Context(), gRoot))

	wsb, err := blocks.NewSignedBeaconBlock(genBlock)
	require.NoError(t, err)

	bs := &Server{
		CoreService: &core.Service{
			BeaconDB:    db,
			HeadFetcher: &chainMock.ChainService{Block: wsb, State: s},
			FinalizedFetcher: &chainMock.ChainService{
				FinalizedCheckPoint:         s.FinalizedCheckpoint(),
				CurrentJustifiedCheckPoint:  s.CurrentJustifiedCheckpoint(),
				PreviousJustifiedCheckPoint: s.PreviousJustifiedCheckpoint()},
			OptimisticModeFetcher: &chainMock.ChainService{},
		},
	}

	_, err = bs.GetChainHead(t.Context(), nil)
	require.ErrorContains(t, "could not get finalized block", err)
}

func TestServer_GetChainHead_NoHeadBlock(t *testing.T) {
	bs := &Server{
		CoreService: &core.Service{
			HeadFetcher:           &chainMock.ChainService{Block: nil},
			OptimisticModeFetcher: &chainMock.ChainService{},
		},
	}
	_, err := bs.GetChainHead(t.Context(), nil)
	assert.ErrorContains(t, "head block of chain was nil", err)
}

func TestServer_GetChainHead(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())

	db := dbTest.SetupDB(t)
	genBlock := util.NewBeaconBlock()
	genBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, fieldparams.RootLength)
	util.SaveBlock(t, t.Context(), db, genBlock)
	gRoot, err := genBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(t.Context(), gRoot))

	finalizedBlock := util.NewBeaconBlock()
	finalizedBlock.Block.Slot = 1
	finalizedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'A'}, fieldparams.RootLength)
	util.SaveBlock(t, t.Context(), db, finalizedBlock)
	fRoot, err := finalizedBlock.Block.HashTreeRoot()
	require.NoError(t, err)

	justifiedBlock := util.NewBeaconBlock()
	justifiedBlock.Block.Slot = 2
	justifiedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'B'}, fieldparams.RootLength)
	util.SaveBlock(t, t.Context(), db, justifiedBlock)
	jRoot, err := justifiedBlock.Block.HashTreeRoot()
	require.NoError(t, err)

	prevJustifiedBlock := util.NewBeaconBlock()
	prevJustifiedBlock.Block.Slot = 3
	prevJustifiedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'C'}, fieldparams.RootLength)
	util.SaveBlock(t, t.Context(), db, prevJustifiedBlock)
	pjRoot, err := prevJustifiedBlock.Block.HashTreeRoot()
	require.NoError(t, err)

	s, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
		Slot:                        1,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Epoch: 3, Root: pjRoot[:]},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Epoch: 2, Root: jRoot[:]},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Epoch: 1, Root: fRoot[:]},
	})
	require.NoError(t, err)

	b := util.NewBeaconBlock()
	b.Block.Slot, err = slots.EpochStart(s.PreviousJustifiedCheckpoint().Epoch)
	require.NoError(t, err)
	b.Block.Slot++
	wsb, err := blocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	bs := &Server{
		CoreService: &core.Service{
			BeaconDB:              db,
			HeadFetcher:           &chainMock.ChainService{Block: wsb, State: s},
			OptimisticModeFetcher: &chainMock.ChainService{},
			FinalizedFetcher: &chainMock.ChainService{
				FinalizedCheckPoint:         s.FinalizedCheckpoint(),
				CurrentJustifiedCheckPoint:  s.CurrentJustifiedCheckpoint(),
				PreviousJustifiedCheckPoint: s.PreviousJustifiedCheckpoint()},
		},
	}

	head, err := bs.GetChainHead(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, primitives.Epoch(3), head.PreviousJustifiedEpoch, "Unexpected PreviousJustifiedEpoch")
	assert.Equal(t, primitives.Epoch(2), head.JustifiedEpoch, "Unexpected JustifiedEpoch")
	assert.Equal(t, primitives.Epoch(1), head.FinalizedEpoch, "Unexpected FinalizedEpoch")
	assert.Equal(t, primitives.Slot(24), head.PreviousJustifiedSlot, "Unexpected PreviousJustifiedSlot")
	assert.Equal(t, primitives.Slot(16), head.JustifiedSlot, "Unexpected JustifiedSlot")
	assert.Equal(t, primitives.Slot(8), head.FinalizedSlot, "Unexpected FinalizedSlot")
	assert.DeepEqual(t, pjRoot[:], head.PreviousJustifiedBlockRoot, "Unexpected PreviousJustifiedBlockRoot")
	assert.DeepEqual(t, jRoot[:], head.JustifiedBlockRoot, "Unexpected JustifiedBlockRoot")
	assert.DeepEqual(t, fRoot[:], head.FinalizedBlockRoot, "Unexpected FinalizedBlockRoot")
	assert.Equal(t, false, head.OptimisticStatus)
}
