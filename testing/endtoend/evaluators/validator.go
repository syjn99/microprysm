package evaluators

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/bits"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/OffchainLabs/prysm/v7/api/server/structs"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/altair"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	"github.com/OffchainLabs/prysm/v7/network/httputil"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/helpers"
	e2eparams "github.com/OffchainLabs/prysm/v7/testing/endtoend/params"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/policies"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/types"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/pkg/errors"
)

var expectedParticipation = 0.98

var expectedMulticlientParticipation = 0.95

var expectedSyncParticipation = 0.95

// ValidatorsAreActive ensures the expected amount of validators are active.
var ValidatorsAreActive = types.Evaluator{
	Name:       "validators_active_epoch_%d",
	Policy:     policies.AllEpochs,
	Evaluation: validatorsAreActive,
}

// ValidatorsParticipatingAtEpoch ensures the expected amount of validators are participating.
var ValidatorsParticipatingAtEpoch = func(epoch primitives.Epoch) types.Evaluator {
	return types.Evaluator{
		Name:       "validators_participating_epoch_%d",
		Policy:     policies.AfterNthEpoch(epoch),
		Evaluation: validatorsParticipating,
	}
}

// ValidatorSyncParticipation ensures the expected amount of sync committee participants
// are active.
var ValidatorSyncParticipation = types.Evaluator{
	Name: "validator_sync_participation_%d",
	Policy: func(e primitives.Epoch) bool {
		fEpoch := params.BeaconConfig().AltairForkEpoch
		return policies.OnwardsNthEpoch(fEpoch)(e)
	},
	Evaluation: validatorsSyncParticipation,
}

func validatorsAreActive(ec *types.EvaluationContext, nodeURLs ...string) error {
	client, err := helpers.NewBeaconNodeClient(nodeURLs[0])
	if err != nil {
		return err
	}
	ctx := context.Background()

	validatorsResp, err := client.ListValidators(ctx, "head", "active")
	if err != nil {
		return errors.Wrap(err, "failed to get validators")
	}

	// Count should be MinGenesisActiveValidatorCount minus any validators that have exited.
	receivedCount := uint64(len(validatorsResp.Data))
	maxExpected := params.BeaconConfig().MinGenesisActiveValidatorCount
	minExpected := maxExpected - uint64(len(ec.ExitedVals))

	if receivedCount > maxExpected {
		return fmt.Errorf("validator count %d exceeds genesis count %d", receivedCount, maxExpected)
	}
	if receivedCount < minExpected {
		return fmt.Errorf("validator count %d is less than expected minimum %d (genesis %d - %d submitted exits)",
			receivedCount, minExpected, maxExpected, len(ec.ExitedVals))
	}

	effBalanceLowCount := 0
	exitEpochWrongCount := 0
	withdrawEpochWrongCount := 0
	farFutureEpoch := strconv.FormatUint(uint64(params.BeaconConfig().FarFutureEpoch), 10)
	for _, item := range validatorsResp.Data {
		pk, err := hex.DecodeString(strings.TrimPrefix(item.Validator.Pubkey, "0x"))
		if err != nil {
			return errors.Wrap(err, "failed to decode validator pubkey")
		}
		if _, exited := ec.ExitedVals[bytesutil.ToBytes48(pk)]; exited {
			continue
		}
		effBalance, err := strconv.ParseUint(item.Validator.EffectiveBalance, 10, 64)
		if err != nil {
			return errors.Wrap(err, "failed to parse effective balance")
		}
		if effBalance < params.BeaconConfig().MaxEffectiveBalance {
			effBalanceLowCount++
		}
		if item.Validator.ExitEpoch != farFutureEpoch {
			exitEpochWrongCount++
		}
		if item.Validator.WithdrawableEpoch != farFutureEpoch {
			withdrawEpochWrongCount++
		}
	}

	if effBalanceLowCount > 0 {
		return fmt.Errorf(
			"%d validators did not have genesis validator effective balance of %d",
			effBalanceLowCount,
			params.BeaconConfig().MaxEffectiveBalance,
		)
	} else if exitEpochWrongCount > 0 {
		return fmt.Errorf("%d validators did not have genesis validator exit epoch of far future epoch", exitEpochWrongCount)
	} else if withdrawEpochWrongCount > 0 {
		return fmt.Errorf("%d validators did not have genesis validator withdrawable epoch of far future epoch", withdrawEpochWrongCount)
	}

	return nil
}

// validatorsParticipating ensures the validators have an acceptable participation rate.
func validatorsParticipating(_ *types.EvaluationContext, nodeURLs ...string) error {
	client, err := helpers.NewBeaconNodeClient(nodeURLs[0])
	if err != nil {
		return err
	}
	ctx := context.Background()

	participation, err := client.GetValidatorParticipation(ctx, "head")
	if err != nil {
		return errors.Wrap(err, "failed to get validator participation")
	}

	partRate, err := strconv.ParseFloat(participation.Participation.GlobalParticipationRate, 32)
	if err != nil {
		return errors.Wrap(err, "failed to parse participation rate")
	}
	partEpoch, err := strconv.ParseUint(participation.Epoch, 10, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse participation epoch")
	}
	epoch := primitives.Epoch(partEpoch)

	expected := float64(expectedParticipation)
	if e2eparams.TestParams.LighthouseBeaconNodeCount != 0 {
		expected = float64(expectedMulticlientParticipation)
	}
	if epoch == params.BeaconConfig().ElectraForkEpoch {
		// The first slot of Electra will be missed due to the switching of attestation types
		// 5/6 slots =~0.83
		// validator REST always is slightly reduced at ~0.82
		expected = 0.82
	}
	if epoch > 0 && epoch.Sub(1) == params.BeaconConfig().BellatrixForkEpoch {
		// Reduce Participation requirement to 95% to account for longer EE calls for
		// the merge block. Target and head will likely be missed for a few validators at
		// slot 0.
		expected = 0.95
	}
	if partRate < expected {
		path := fmt.Sprintf("http://localhost:%d/eth/v2/debug/beacon/states/head", e2eparams.TestParams.Ports.PrysmBeaconNodeHTTPPort)
		resp := structs.GetBeaconStateV2Response{}
		httpResp, err := http.Get(path) // #nosec G107 -- path can't be constant because it depends on port param
		if err != nil {
			return err
		}
		if httpResp.StatusCode != http.StatusOK {
			e := httputil.DefaultJsonError{}
			if err = json.NewDecoder(httpResp.Body).Decode(&e); err != nil {
				return err
			}
			return fmt.Errorf("%s (status code %d)", e.Message, e.Code)
		}
		if err = json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
			return err
		}

		var respPrevEpochParticipation []string
		switch resp.Version {
		case version.String(version.Phase0):
		// Do Nothing
		case version.String(version.Altair):
			st := &structs.BeaconStateAltair{}
			if err = json.Unmarshal(resp.Data, st); err != nil {
				return err
			}
			respPrevEpochParticipation = st.PreviousEpochParticipation
		case version.String(version.Bellatrix):
			st := &structs.BeaconStateBellatrix{}
			if err = json.Unmarshal(resp.Data, st); err != nil {
				return err
			}
			respPrevEpochParticipation = st.PreviousEpochParticipation
		case version.String(version.Capella):
			st := &structs.BeaconStateCapella{}
			if err = json.Unmarshal(resp.Data, st); err != nil {
				return err
			}
			respPrevEpochParticipation = st.PreviousEpochParticipation
		case version.String(version.Deneb):
			st := &structs.BeaconStateDeneb{}
			if err = json.Unmarshal(resp.Data, st); err != nil {
				return err
			}
			respPrevEpochParticipation = st.PreviousEpochParticipation
		case version.String(version.Electra):
			st := &structs.BeaconStateElectra{}
			if err = json.Unmarshal(resp.Data, st); err != nil {
				return err
			}
			respPrevEpochParticipation = st.PreviousEpochParticipation
		default:
			return fmt.Errorf("unrecognized version %s", resp.Version)
		}

		prevEpochParticipation := make([]byte, len(respPrevEpochParticipation))
		for i, p := range respPrevEpochParticipation {
			n, err := strconv.ParseUint(p, 10, 64)
			if err != nil {
				return err
			}
			prevEpochParticipation[i] = byte(n)
		}
		missSrcVals, missTgtVals, missHeadVals, err := findMissingValidators(prevEpochParticipation)
		if err != nil {
			return errors.Wrap(err, "failed to get missing validators")
		}

		return fmt.Errorf(
			"validator participation was below for epoch %d, expected %f, received: %f."+
				" Missing Source,Target and Head validators are %v, %v, %v",
			epoch,
			expected,
			partRate,
			missSrcVals,
			missTgtVals,
			missHeadVals,
		)
	}
	return nil
}

// syncBlockMsg is a minimal struct for parsing sync aggregate fields from any post-Altair block.
type syncBlockMsg struct {
	Slot string `json:"slot"`
	Body struct {
		SyncAggregate *syncAggregateMsg `json:"sync_aggregate"`
	} `json:"body"`
}

type syncAggregateMsg struct {
	SyncCommitteeBits string `json:"sync_committee_bits"`
}

// countSyncBits decodes a hex-encoded bitvector and returns (set bits count, total bits count).
func countSyncBits(hexBits string) (uint64, uint64, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(hexBits, "0x"))
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to decode sync committee bits")
	}
	var count uint64
	for _, byt := range b {
		count += uint64(bits.OnesCount8(byt))
	}
	return count, uint64(len(b)) * 8, nil
}

// validatorsSyncParticipation ensures the validators have an acceptable participation rate for
// sync committee assignments.
func validatorsSyncParticipation(_ *types.EvaluationContext, nodeURLs ...string) error {
	client, err := helpers.NewBeaconNodeClient(nodeURLs[0])
	if err != nil {
		return err
	}
	ctx := context.Background()

	genesisResp, err := client.GetGenesis(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get genesis data")
	}
	genesisTimeSec, err := strconv.ParseInt(genesisResp.Data.GenesisTime, 10, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse genesis time")
	}
	genesisTime := time.Unix(genesisTimeSec, 0)

	currSlot := slots.CurrentSlot(genesisTime)
	currEpoch := slots.ToEpoch(currSlot)
	lowestBound := primitives.Epoch(0)
	if currEpoch >= 1 {
		lowestBound = currEpoch - 1
	}

	if lowestBound < params.BeaconConfig().AltairForkEpoch {
		lowestBound = params.BeaconConfig().AltairForkEpoch
	}

	if err := checkSyncParticipationForEpoch(ctx, client, lowestBound, true); err != nil {
		return err
	}
	if lowestBound == currEpoch {
		return nil
	}
	return checkSyncParticipationForEpoch(ctx, client, currEpoch, false)
}

func checkSyncParticipationForEpoch(
	ctx context.Context,
	client *helpers.BeaconNodeClient,
	epoch primitives.Epoch,
	isLowestBound bool,
) error {
	startSlot, err := slots.EpochStart(epoch)
	if err != nil {
		return errors.Wrap(err, "failed to compute start slot")
	}
	endSlot := startSlot + params.BeaconConfig().SlotsPerEpoch
	forkStartSlot, err := slots.EpochStart(params.BeaconConfig().AltairForkEpoch)
	if err != nil {
		return err
	}

	for slot := startSlot; slot < endSlot; slot++ {
		blockResp, err := client.GetBlock(ctx, strconv.FormatUint(uint64(slot), 10))
		if err != nil {
			// Slot may be empty (missed block); skip.
			continue
		}
		if blockResp.Data == nil {
			continue
		}

		// Skip Phase0 blocks (no sync aggregate).
		v, err := version.FromString(blockResp.Version)
		if err != nil || v < version.Altair {
			continue
		}

		var msg syncBlockMsg
		if err := json.Unmarshal(blockResp.Data.Message, &msg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal block message at slot %d", slot)
		}
		if msg.Body.SyncAggregate == nil {
			return errors.New("nil block provided")
		}

		blockSlot, err := strconv.ParseUint(msg.Slot, 10, 64)
		if err != nil {
			return errors.Wrapf(err, "failed to parse slot")
		}

		if isLowestBound {
			if primitives.Slot(blockSlot) == forkStartSlot {
				// Skip fork slot.
				continue
			}
			// Skip slots 1-2 at genesis - validators need time to ramp up after chain start
			// due to doppelganger protection. This is a startup timing issue, not a fork transition issue.
			if blockSlot < 3 {
				continue
			}
			expectedPart := expectedSyncParticipation
			switch slots.ToEpoch(primitives.Slot(blockSlot)) {
			case params.BeaconConfig().AltairForkEpoch:
				// Drop expected sync participation figure.
				expectedPart = 0.90
			default:
				// no-op
			}
			count, length, err := countSyncBits(msg.Body.SyncAggregate.SyncCommitteeBits)
			if err != nil {
				return err
			}
			threshold := uint64(float64(length) * expectedPart)
			if count < threshold {
				return errors.Errorf("In block of slot %d ,the aggregate bitvector with length of %d only got a count of %d", blockSlot, threshold, count)
			}
		} else {
			// For the current epoch, skip fork transition slots.
			forkEpochs := []primitives.Epoch{
				params.BeaconConfig().AltairForkEpoch,
				params.BeaconConfig().BellatrixForkEpoch,
				params.BeaconConfig().CapellaForkEpoch,
				params.BeaconConfig().DenebForkEpoch,
				params.BeaconConfig().ElectraForkEpoch,
				params.BeaconConfig().FuluForkEpoch,
			}
			skipSlot := false
			for _, forkEpoch := range forkEpochs {
				if forkEpoch == params.BeaconConfig().FarFutureEpoch {
					continue
				}
				forkSlot, err := slots.EpochStart(forkEpoch)
				if err != nil {
					return err
				}
				if primitives.Slot(blockSlot) == forkSlot || primitives.Slot(blockSlot) == forkSlot+1 {
					skipSlot = true
					break
				}
			}
			if skipSlot {
				continue
			}
			count, length, err := countSyncBits(msg.Body.SyncAggregate.SyncCommitteeBits)
			if err != nil {
				return err
			}
			threshold := uint64(float64(length) * expectedSyncParticipation)
			if count < threshold {
				return errors.Errorf("In block of slot %d ,the aggregate bitvector with length of %d only got a count of %d", blockSlot, threshold, count)
			}
		}
	}
	return nil
}

func findMissingValidators(participation []byte) ([]uint64, []uint64, []uint64, error) {
	cfg := params.BeaconConfig()
	sourceFlagIndex := cfg.TimelySourceFlagIndex
	targetFlagIndex := cfg.TimelyTargetFlagIndex
	headFlagIndex := cfg.TimelyHeadFlagIndex
	var missingSourceValidators []uint64
	var missingHeadValidators []uint64
	var missingTargetValidators []uint64
	for i, b := range participation {
		hasSource, err := altair.HasValidatorFlag(b, sourceFlagIndex)
		if err != nil {
			return nil, nil, nil, err
		}
		if !hasSource {
			missingSourceValidators = append(missingSourceValidators, uint64(i))
		}
		hasTarget, err := altair.HasValidatorFlag(b, targetFlagIndex)
		if err != nil {
			return nil, nil, nil, err
		}
		if !hasTarget {
			missingTargetValidators = append(missingTargetValidators, uint64(i))
		}
		hasHead, err := altair.HasValidatorFlag(b, headFlagIndex)
		if err != nil {
			return nil, nil, nil, err
		}
		if !hasHead {
			missingHeadValidators = append(missingHeadValidators, uint64(i))
		}
	}
	return missingSourceValidators, missingTargetValidators, missingHeadValidators, nil
}
