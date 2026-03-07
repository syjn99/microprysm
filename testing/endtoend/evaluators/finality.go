package evaluators

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/helpers"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/policies"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/types"
)

// FinalizationOccurs is an evaluator to make sure finalization is performing as it should.
// Requires to be run after at least 4 epochs have passed.
var FinalizationOccurs = func(epoch primitives.Epoch) types.Evaluator {
	return types.Evaluator{
		Name:       "finalizes_at_epoch_%d",
		Policy:     policies.AfterNthEpoch(epoch),
		Evaluation: finalizationOccurs,
	}
}

func finalizationOccurs(_ *types.EvaluationContext, nodeURLs ...string) error {
	client, err := helpers.NewBeaconNodeClient(nodeURLs[0])
	if err != nil {
		return err
	}
	chainHead, err := client.GetChainHead(context.Background())
	if err != nil {
		return err
	}
	currentEpoch := chainHead.HeadEpoch
	finalizedEpoch := chainHead.FinalizedEpoch

	expectedFinalizedEpoch := currentEpoch - 2
	if expectedFinalizedEpoch != finalizedEpoch {
		return fmt.Errorf(
			"expected finalized epoch to be %d, received: %d",
			expectedFinalizedEpoch,
			finalizedEpoch,
		)
	}
	previousJustifiedEpoch := chainHead.PreviousJustifiedEpoch
	currentJustifiedEpoch := chainHead.JustifiedEpoch
	if previousJustifiedEpoch+1 != currentJustifiedEpoch {
		return fmt.Errorf(
			"there should be no gaps between current and previous justified epochs, received current %d and previous %d",
			currentJustifiedEpoch,
			previousJustifiedEpoch,
		)
	}
	if currentJustifiedEpoch+1 != currentEpoch {
		return fmt.Errorf(
			"there should be no gaps between current epoch and current justified epoch, received current %d and justified %d",
			currentEpoch,
			currentJustifiedEpoch,
		)
	}
	return nil
}
