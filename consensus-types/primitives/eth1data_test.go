package primitives_test

import (
	"encoding/json"
	"testing"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func TestEth1Data_JSONRoundTrip(t *testing.T) {
	original := &primitives.Eth1Data{
		DepositRoot:  [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		DepositCount: 12345,
		BlockHash:    [32]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded primitives.Eth1Data
	require.NoError(t, json.Unmarshal(data, &decoded))

	require.DeepEqual(t, original.DepositRoot, decoded.DepositRoot)
	require.Equal(t, original.DepositCount, decoded.DepositCount)
	require.DeepEqual(t, original.BlockHash, decoded.BlockHash)
}

func TestEth1Data_JSONCompatibleWithProto(t *testing.T) {
	// Same data in both formats
	root := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	hash := []byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}

	native := &primitives.Eth1Data{
		DepositCount: 42,
	}
	copy(native.DepositRoot[:], root)
	copy(native.BlockHash[:], hash)

	nativeJSON, err := json.Marshal(native)
	require.NoError(t, err)

	// Verify the JSON matches Beacon API format
	var rawMap map[string]string
	require.NoError(t, json.Unmarshal(nativeJSON, &rawMap))
	require.Equal(t, "0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20", rawMap["deposit_root"])
	require.Equal(t, "42", rawMap["deposit_count"])
	require.Equal(t, "0x201f1e1d1c1b1a191817161514131211100f0e0d0c0b0a090807060504030201", rawMap["block_hash"])

	// Verify proto SSZ compatibility
	proto := &ethpb.Eth1Data{
		DepositRoot:  root,
		DepositCount: 42,
		BlockHash:    hash,
	}
	protoSSZ, err := proto.MarshalSSZ()
	require.NoError(t, err)

	nativeSSZ, err := native.MarshalSSZTo(make([]byte, 0, native.SizeSSZ()))
	require.NoError(t, err)
	require.DeepEqual(t, protoSSZ, nativeSSZ, "SSZ encoding must be byte-identical between proto and native")
}

func TestEth1Data_Copy(t *testing.T) {
	original := &primitives.Eth1Data{
		DepositRoot:  [32]byte{1},
		DepositCount: 100,
		BlockHash:    [32]byte{2},
	}
	cp := original.Copy()
	require.DeepEqual(t, original, cp)

	// Mutate copy, original should be unchanged
	cp.DepositCount = 999
	require.Equal(t, uint64(100), original.DepositCount)
}
