package proposer

import (
	"encoding/binary"
	"testing"

	mockChain "github.com/OffchainLabs/prysm/v7/beacon-chain/blockchain/testing"
	"github.com/OffchainLabs/prysm/v7/config/params"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/assert"
	"github.com/OffchainLabs/prysm/v7/testing/require"
	"github.com/OffchainLabs/prysm/v7/testing/util"
)

func pubKey(i uint64) []byte {
	pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
	binary.LittleEndian.PutUint64(pubKey, i)
	return pubKey
}

func TestValidatorIndex_OK(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)

	pk := pubKey(1)

	err = st.SetValidators([]*ethpb.Validator{{PublicKey: pk}})
	require.NoError(t, err)

	Server := &Server{
		HeadFetcher: &mockChain.ChainService{State: st},
	}

	req := &ethpb.ValidatorIndexRequest{
		PublicKey: pk,
	}
	_, err = Server.ValidatorIndex(t.Context(), req)
	assert.NoError(t, err, "Could not get validator index")
}

func TestValidatorIndex_StateEmpty(t *testing.T) {
	Server := &Server{
		HeadFetcher: &mockChain.ChainService{},
	}
	pk := pubKey(1)
	req := &ethpb.ValidatorIndexRequest{
		PublicKey: pk,
	}
	_, err := Server.ValidatorIndex(t.Context(), req)
	assert.ErrorContains(t, "head state is empty", err)
}
