package helpers

import (
	"bytes"
	"testing"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/signing"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/genesis"
	eth "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func TestComputeDomainData_MatchesServer(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())

	cfg := params.BeaconConfig()
	vr := cfg.GenesisValidatorsRoot
	genesis.StoreDuringTest(t, genesis.GenesisData{ValidatorsRoot: vr})
	gvr := genesis.ValidatorsRoot()

	tests := []struct {
		name       string
		epoch      primitives.Epoch
		domainType [4]byte
	}{
		{
			name:       "beacon proposer at epoch 0",
			epoch:      0,
			domainType: cfg.DomainBeaconProposer,
		},
		{
			name:       "beacon attester at epoch 5",
			epoch:      5,
			domainType: cfg.DomainBeaconAttester,
		},
		{
			name:       "voluntary exit pre-Deneb",
			epoch:      0,
			domainType: cfg.DomainVoluntaryExit,
		},
		{
			name:       "voluntary exit post-Deneb",
			epoch:      cfg.DenebForkEpoch + 1,
			domainType: cfg.DomainVoluntaryExit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ComputeDomainData(tt.epoch, tt.domainType)
			require.NoError(t, err)

			// Compute expected using the same logic as the gRPC server.
			var fork *eth.Fork
			if bytes.Equal(tt.domainType[:], cfg.DomainVoluntaryExit[:]) && tt.epoch >= cfg.DenebForkEpoch {
				fork = &eth.Fork{
					PreviousVersion: cfg.CapellaForkVersion,
					CurrentVersion:  cfg.CapellaForkVersion,
					Epoch:           cfg.CapellaForkEpoch,
				}
			} else {
				fork = params.ForkFromConfig(cfg, tt.epoch)
			}
			expected, err := signing.Domain(fork, tt.epoch, tt.domainType, gvr[:])
			require.NoError(t, err)
			require.DeepEqual(t, expected, got)
		})
	}
}
