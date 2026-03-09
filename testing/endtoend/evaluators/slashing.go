package evaluators

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/OffchainLabs/prysm/v7/api/server/structs"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/signing"
	fieldparams "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/container/slice"
	"github.com/OffchainLabs/prysm/v7/crypto/bls"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	eth "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/helpers"
	e2e "github.com/OffchainLabs/prysm/v7/testing/endtoend/params"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/policies"
	e2eTypes "github.com/OffchainLabs/prysm/v7/testing/endtoend/types"
	"github.com/OffchainLabs/prysm/v7/testing/util"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
)

// InjectDoubleVoteOnEpoch broadcasts a double vote into the beacon node pool for the slasher to detect.
var InjectDoubleVoteOnEpoch = func(n primitives.Epoch) e2eTypes.Evaluator {
	return e2eTypes.Evaluator{
		Name:       "inject_double_vote_%d",
		Policy:     policies.OnEpoch(n),
		Evaluation: insertDoubleAttestationIntoPool,
	}
}

// InjectDoubleBlockOnEpoch proposes a double block to the beacon node for the slasher to detect.
var InjectDoubleBlockOnEpoch = func(n primitives.Epoch) e2eTypes.Evaluator {
	return e2eTypes.Evaluator{
		Name:       "inject_double_block_%d",
		Policy:     policies.OnEpoch(n),
		Evaluation: proposeDoubleBlock,
	}
}

// ValidatorsSlashedAfterEpoch ensures the expected amount of validators are slashed.
var ValidatorsSlashedAfterEpoch = func(n primitives.Epoch) e2eTypes.Evaluator {
	return e2eTypes.Evaluator{
		Name:       "validators_slashed_epoch_%d",
		Policy:     policies.AfterNthEpoch(n),
		Evaluation: validatorsSlashed,
	}
}

// SlashedValidatorsLoseBalanceAfterEpoch checks if the validators slashed lose the right balance.
var SlashedValidatorsLoseBalanceAfterEpoch = func(n primitives.Epoch) e2eTypes.Evaluator {
	return e2eTypes.Evaluator{
		Name:       "slashed_validators_lose_balance_epoch_%d",
		Policy:     policies.AfterNthEpoch(n),
		Evaluation: validatorsLoseBalance,
	}
}

var slashedIndices []uint64

func validatorsSlashed(_ *e2eTypes.EvaluationContext, nodeURLs ...string) error {
	client, err := helpers.NewBeaconNodeClient(nodeURLs[0])
	if err != nil {
		return err
	}
	ctx := context.Background()

	actualSlashedIndices := 0
	for _, slashedIndex := range slashedIndices {
		valResp, err := client.GetValidator(ctx, "head", fmt.Sprintf("%d", slashedIndex))
		if err != nil {
			return err
		}
		if valResp.Data.Validator.Slashed {
			actualSlashedIndices++
		}
	}

	if actualSlashedIndices != len(slashedIndices) {
		return fmt.Errorf("expected %d indices to be slashed, received %d", len(slashedIndices), actualSlashedIndices)
	}
	return nil
}

func validatorsLoseBalance(_ *e2eTypes.EvaluationContext, nodeURLs ...string) error {
	client, err := helpers.NewBeaconNodeClient(nodeURLs[0])
	if err != nil {
		return err
	}
	ctx := context.Background()

	for i, slashedIndex := range slashedIndices {
		valResp, err := client.GetValidator(ctx, "head", fmt.Sprintf("%d", slashedIndex))
		if err != nil {
			return err
		}
		effectiveBalance, err := strconv.ParseUint(valResp.Data.Validator.EffectiveBalance, 10, 64)
		if err != nil {
			return errors.Wrap(err, "failed to parse effective balance")
		}

		slashedPenalty := params.BeaconConfig().MaxEffectiveBalance / params.BeaconConfig().MinSlashingPenaltyQuotient
		slashedBal := params.BeaconConfig().MaxEffectiveBalance - slashedPenalty + params.BeaconConfig().EffectiveBalanceIncrement/10
		if effectiveBalance >= slashedBal {
			return fmt.Errorf(
				"expected slashed validator %d balance to be less than %d, received %d",
				i,
				slashedBal,
				effectiveBalance,
			)
		}
	}
	return nil
}

func insertDoubleAttestationIntoPool(_ *e2eTypes.EvaluationContext, nodeURLs ...string) error {
	client0, err := helpers.NewBeaconNodeClient(nodeURLs[0])
	if err != nil {
		return err
	}

	ctx := context.Background()

	h := doubleAttestationHelper{client: client0}
	if err := h.setup(ctx); err != nil {
		return errors.Wrap(err, "could not setup doubleAttestationHelper")
	}

	client1, err := helpers.NewBeaconNodeClient(nodeURLs[1])
	if err != nil {
		return err
	}

	valsToSlash := uint64(2)
	for i := uint64(0); i < valsToSlash; i++ {
		valIdx := h.validatorIndexAtCommitteeIndex(i)

		if len(slice.IntersectionUint64(slashedIndices, []uint64{uint64(valIdx)})) > 0 {
			valsToSlash++
			continue
		}

		// Need to send attestation to both beacon nodes to avoid flakiness.
		// See: https://github.com/prysmaticlabs/prysm/issues/12415#issuecomment-1874643269
		att, err := h.getSlashableAttestation(i)
		if err != nil {
			return err
		}
		jsonAtts, err := marshalAttestationsJSON(att)
		if err != nil {
			return errors.Wrap(err, "could not marshal attestation")
		}
		if err := client0.SubmitAttestations(ctx, "phase0", jsonAtts); err != nil {
			return errors.Wrap(err, "could not submit attestation to node 0")
		}

		att1, err := h.getSlashableAttestation(i)
		if err != nil {
			return err
		}
		jsonAtts1, err := marshalAttestationsJSON(att1)
		if err != nil {
			return errors.Wrap(err, "could not marshal attestation")
		}
		if err := client1.SubmitAttestations(ctx, "phase0", jsonAtts1); err != nil {
			return errors.Wrap(err, "could not submit attestation to node 1")
		}

		slashedIndices = append(slashedIndices, uint64(valIdx))
	}
	return nil
}

// marshalAttestationsJSON converts a proto Attestation to the REST JSON array format.
func marshalAttestationsJSON(att *eth.Attestation) ([]byte, error) {
	restAtt := &structs.Attestation{
		AggregationBits: hexutil.Encode(att.AggregationBits),
		Data: &structs.AttestationData{
			Slot:            fmt.Sprintf("%d", att.Data.Slot),
			CommitteeIndex:  fmt.Sprintf("%d", att.Data.CommitteeIndex),
			BeaconBlockRoot: hexutil.Encode(att.Data.BeaconBlockRoot),
			Source: &structs.Checkpoint{
				Epoch: fmt.Sprintf("%d", att.Data.Source.Epoch),
				Root:  hexutil.Encode(att.Data.Source.Root),
			},
			Target: &structs.Checkpoint{
				Epoch: fmt.Sprintf("%d", att.Data.Target.Epoch),
				Root:  hexutil.Encode(att.Data.Target.Root),
			},
		},
		Signature: hexutil.Encode(att.Signature),
	}
	return json.Marshal([]*structs.Attestation{restAtt})
}

func proposeDoubleBlock(_ *e2eTypes.EvaluationContext, nodeURLs ...string) error {
	client, err := helpers.NewBeaconNodeClient(nodeURLs[0])
	if err != nil {
		return err
	}
	ctx := context.Background()
	chainHead, err := client.GetChainHead(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get chain head")
	}
	_, privKeys, err := util.DeterministicDepositsAndKeys(params.BeaconConfig().MinGenesisActiveValidatorCount)
	if err != nil {
		return err
	}

	dutiesResp, err := client.GetProposerDuties(ctx, chainHead.HeadEpoch)
	if err != nil {
		return errors.Wrap(err, "could not get proposer duties")
	}

	targetSlot := fmt.Sprintf("%d", uint64(chainHead.HeadSlot)-1)
	var proposerIndex primitives.ValidatorIndex
	for _, duty := range dutiesResp.Data {
		if duty.Slot == targetSlot {
			idx, err := strconv.ParseUint(duty.ValidatorIndex, 10, 64)
			if err != nil {
				return errors.Wrap(err, "could not parse proposer index")
			}
			proposerIndex = primitives.ValidatorIndex(idx)
			break
		}
	}

	validatorNum := int(params.BeaconConfig().MinGenesisActiveValidatorCount)
	beaconNodeNum := e2e.TestParams.BeaconNodeCount
	if validatorNum%beaconNodeNum != 0 {
		return errors.New("validator count is not easily divisible by beacon node count")
	}
	validatorsPerNode := validatorNum / beaconNodeNum

	// If the proposer index is in the second validator client, we connect to
	// the corresponding beacon node instead.
	submitClient := client
	if proposerIndex >= primitives.ValidatorIndex(uint64(validatorsPerNode)) {
		submitClient, err = helpers.NewBeaconNodeClient(nodeURLs[1])
		if err != nil {
			return err
		}
	}

	jsonBlock, err := generateSignedBeaconBlockJSON(chainHead, proposerIndex, privKeys, "bad state root")
	if err != nil {
		return err
	}
	if err = submitClient.PublishBlockV2(ctx, "phase0", jsonBlock); err == nil {
		return errors.New("expected block to fail processing")
	}

	jsonBlock, err = generateSignedBeaconBlockJSON(chainHead, proposerIndex, privKeys, "bad state root 2")
	if err != nil {
		return err
	}
	if err = submitClient.PublishBlockV2(ctx, "phase0", jsonBlock); err == nil {
		return errors.New("expected block to fail processing")
	}

	slashedIndices = append(slashedIndices, uint64(proposerIndex))
	return nil
}

func generateSignedBeaconBlockJSON(
	chainHead *helpers.ChainHead,
	proposerIndex primitives.ValidatorIndex,
	privKeys []bls.SecretKey,
	stateRoot string,
) ([]byte, error) {
	parentRoot, err := hexutil.Decode(chainHead.HeadBlockRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode head block root")
	}

	hashLen := 32
	blk := &eth.BeaconBlock{
		Slot:          chainHead.HeadSlot - 1,
		ParentRoot:    parentRoot,
		StateRoot:     bytesutil.PadTo([]byte(stateRoot), hashLen),
		ProposerIndex: proposerIndex,
		Body: &eth.BeaconBlockBody{
			Eth1Data: &eth.Eth1Data{
				BlockHash:    bytesutil.PadTo([]byte("bad block hash"), hashLen),
				DepositRoot:  bytesutil.PadTo([]byte("bad deposit root"), hashLen),
				DepositCount: 1,
			},
			RandaoReveal:      bytesutil.PadTo([]byte("bad randao"), fieldparams.BLSSignatureLength),
			Graffiti:          bytesutil.PadTo([]byte("teehee"), hashLen),
			ProposerSlashings: []*eth.ProposerSlashing{},
			AttesterSlashings: []*eth.AttesterSlashing{},
			Attestations:      []*eth.Attestation{},
			Deposits:          []*eth.Deposit{},
			VoluntaryExits:    []*eth.SignedVoluntaryExit{},
		},
	}

	domain, err := helpers.ComputeDomainData(chainHead.HeadEpoch, params.BeaconConfig().DomainBeaconProposer)
	if err != nil {
		return nil, errors.Wrap(err, "could not get domain data")
	}
	signingRoot, err := signing.ComputeSigningRoot(blk, domain)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute signing root")
	}
	sig := privKeys[proposerIndex].Sign(signingRoot[:]).Marshal()

	// Convert to REST JSON structs for PublishBlockV2.
	jsonBlock := &structs.SignedBeaconBlock{
		Message: &structs.BeaconBlock{
			Slot:          fmt.Sprintf("%d", chainHead.HeadSlot-1),
			ProposerIndex: fmt.Sprintf("%d", proposerIndex),
			ParentRoot:    chainHead.HeadBlockRoot,
			StateRoot:     hexutil.Encode(bytesutil.PadTo([]byte(stateRoot), hashLen)),
			Body: &structs.BeaconBlockBody{
				RandaoReveal: hexutil.Encode(bytesutil.PadTo([]byte("bad randao"), fieldparams.BLSSignatureLength)),
				Eth1Data: &eth.Eth1Data{
					BlockHash:    bytesutil.PadTo([]byte("bad block hash"), hashLen),
					DepositRoot:  bytesutil.PadTo([]byte("bad deposit root"), hashLen),
					DepositCount: 1,
				},
				Graffiti:          hexutil.Encode(bytesutil.PadTo([]byte("teehee"), hashLen)),
				ProposerSlashings: []*structs.ProposerSlashing{},
				AttesterSlashings: []*structs.AttesterSlashing{},
				Attestations:      []*structs.Attestation{},
				Deposits:          []*structs.Deposit{},
				VoluntaryExits:    []*structs.SignedVoluntaryExit{},
			},
		},
		Signature: hexutil.Encode(sig),
	}

	return json.Marshal(jsonBlock)
}
