package helpers

import (
	"testing"

	state_native "github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/consensus-types/validator"
	ethpbalpha "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func Test_ValidatorStatus(t *testing.T) {
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch

	type args struct {
		validator *ethpbalpha.Validator
		epoch     primitives.Epoch
	}
	tests := []struct {
		name    string
		args    args
		want    validator.Status
		wantErr bool
	}{
		{
			name: "pending initialized",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch:            farFutureEpoch,
					ActivationEligibilityEpoch: farFutureEpoch,
				},
				epoch: primitives.Epoch(5),
			},
			want: validator.Pending,
		},
		{
			name: "pending queued",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch:            10,
					ActivationEligibilityEpoch: 2,
				},
				epoch: primitives.Epoch(5),
			},
			want: validator.Pending,
		},
		{
			name: "active ongoing",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       farFutureEpoch,
				},
				epoch: primitives.Epoch(5),
			},
			want: validator.Active,
		},
		{
			name: "active slashed",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       30,
					Slashed:         true,
				},
				epoch: primitives.Epoch(5),
			},
			want: validator.Active,
		},
		{
			name: "active exiting",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       30,
					Slashed:         false,
				},
				epoch: primitives.Epoch(5),
			},
			want: validator.Active,
		},
		{
			name: "exited slashed",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					Slashed:           true,
				},
				epoch: primitives.Epoch(35),
			},
			want: validator.Exited,
		},
		{
			name: "exited unslashed",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					Slashed:           false,
				},
				epoch: primitives.Epoch(35),
			},
			want: validator.Exited,
		},
		{
			name: "withdrawal possible",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
					Slashed:           false,
				},
				epoch: primitives.Epoch(45),
			},
			want: validator.Withdrawal,
		},
		{
			name: "withdrawal done",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					EffectiveBalance:  0,
					Slashed:           false,
				},
				epoch: primitives.Epoch(45),
			},
			want: validator.Withdrawal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readOnlyVal, err := state_native.NewValidator(tt.args.validator)
			require.NoError(t, err)
			got, err := ValidatorStatus(readOnlyVal, tt.args.epoch)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("validatorStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ValidatorSubStatus(t *testing.T) {
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch

	type args struct {
		validator *ethpbalpha.Validator
		epoch     primitives.Epoch
	}
	tests := []struct {
		name    string
		args    args
		want    validator.Status
		wantErr bool
	}{
		{
			name: "pending initialized",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch:            farFutureEpoch,
					ActivationEligibilityEpoch: farFutureEpoch,
				},
				epoch: primitives.Epoch(5),
			},
			want: validator.PendingInitialized,
		},
		{
			name: "pending queued",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch:            10,
					ActivationEligibilityEpoch: 2,
				},
				epoch: primitives.Epoch(5),
			},
			want: validator.PendingQueued,
		},
		{
			name: "active ongoing",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       farFutureEpoch,
				},
				epoch: primitives.Epoch(5),
			},
			want: validator.ActiveOngoing,
		},
		{
			name: "active slashed",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       30,
					Slashed:         true,
				},
				epoch: primitives.Epoch(5),
			},
			want: validator.ActiveSlashed,
		},
		{
			name: "active exiting",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       30,
					Slashed:         false,
				},
				epoch: primitives.Epoch(5),
			},
			want: validator.ActiveExiting,
		},
		{
			name: "exited slashed",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					Slashed:           true,
				},
				epoch: primitives.Epoch(35),
			},
			want: validator.ExitedSlashed,
		},
		{
			name: "exited unslashed",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					Slashed:           false,
				},
				epoch: primitives.Epoch(35),
			},
			want: validator.ExitedUnslashed,
		},
		{
			name: "withdrawal possible",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
					Slashed:           false,
				},
				epoch: primitives.Epoch(45),
			},
			want: validator.WithdrawalPossible,
		},
		{
			name: "withdrawal done",
			args: args{
				validator: &ethpbalpha.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					EffectiveBalance:  0,
					Slashed:           false,
				},
				epoch: primitives.Epoch(45),
			},
			want: validator.WithdrawalDone,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readOnlyVal, err := state_native.NewValidator(tt.args.validator)
			require.NoError(t, err)
			got, err := ValidatorSubStatus(readOnlyVal, tt.args.epoch)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("validatorSubStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}
