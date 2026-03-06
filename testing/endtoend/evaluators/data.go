package evaluators

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/helpers"
	e2etypes "github.com/OffchainLabs/prysm/v7/testing/endtoend/types"
)

const epochToCheck = 50 // must be more than 46 (32 hot states + 16 chkpt interval)

// ColdStateCheckpoint checks data from the database using cold state storage.
var ColdStateCheckpoint = e2etypes.Evaluator{
	Name: "cold_state_assignments_from_epoch_%d",
	Policy: func(currentEpoch primitives.Epoch) bool {
		return currentEpoch == epochToCheck
	},
	Evaluation: checkColdStateCheckpoint,
}

// Checks the first node for an old checkpoint using cold state storage.
func checkColdStateCheckpoint(ec *e2etypes.EvaluationContext, nodeURLs ...string) error {
	ctx := context.Background()
	client, err := helpers.NewBeaconNodeClient(nodeURLs[0])
	if err != nil {
		return err
	}

	for i := range primitives.Epoch(epochToCheck) {
		res, err := client.GetProposerDuties(ctx, i)
		if err != nil {
			return err
		}
		if res == nil || res.Data == nil {
			return fmt.Errorf("failed to return proposer duties for epoch %d "+
				"using cold state storage from the database", i)
		}
	}

	return nil
}
