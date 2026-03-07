package evaluators

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/helpers"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/policies"
	e2etypes "github.com/OffchainLabs/prysm/v7/testing/endtoend/types"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/pkg/errors"
)

// AltairForkTransition ensures that the Altair hard fork has occurred successfully.
var AltairForkTransition = e2etypes.Evaluator{
	Name: "altair_fork_transition_%d",
	Policy: func(e primitives.Epoch) bool {
		if e2etypes.GenesisFork() >= version.Altair {
			return false
		}
		return policies.OnEpoch(params.BeaconConfig().AltairForkEpoch)(e)
	},
	Evaluation: altairForkOccurs,
}

// BellatrixForkTransition ensures that the Bellatrix hard fork has occurred successfully.
var BellatrixForkTransition = e2etypes.Evaluator{
	Name: "bellatrix_fork_transition_%d",
	Policy: func(e primitives.Epoch) bool {
		if e2etypes.GenesisFork() >= version.Bellatrix {
			return false
		}
		return policies.OnEpoch(params.BeaconConfig().BellatrixForkEpoch)(e)
	},
	Evaluation: bellatrixForkOccurs,
}

// CapellaForkTransition ensures that the Capella hard fork has occurred successfully.
var CapellaForkTransition = e2etypes.Evaluator{
	Name: "capella_fork_transition_%d",
	Policy: func(e primitives.Epoch) bool {
		if e2etypes.GenesisFork() >= version.Capella {
			return false
		}
		return policies.OnEpoch(params.BeaconConfig().CapellaForkEpoch)(e)
	},
	Evaluation: capellaForkOccurs,
}

// DenebForkTransition ensures that the Deneb hard fork has occurred successfully
var DenebForkTransition = e2etypes.Evaluator{
	Name: "deneb_fork_transition_%d",
	Policy: func(e primitives.Epoch) bool {
		if e2etypes.GenesisFork() >= version.Deneb {
			return false
		}
		return policies.OnEpoch(params.BeaconConfig().DenebForkEpoch)(e)
	},
	Evaluation: denebForkOccurs,
}

// ElectraForkTransition ensures that the electra hard fork has occurred successfully
var ElectraForkTransition = e2etypes.Evaluator{
	Name: "electra_fork_transition_%d",
	Policy: func(e primitives.Epoch) bool {
		if e2etypes.GenesisFork() >= version.Electra {
			return false
		}
		return policies.OnEpoch(params.BeaconConfig().ElectraForkEpoch)(e)
	},
	Evaluation: electraForkOccurs,
}

// FuluForkTransition ensures that the fulu hard fork has occurred successfully
var FuluForkTransition = e2etypes.Evaluator{
	Name: "fulu_fork_transition_%d",
	Policy: func(e primitives.Epoch) bool {
		if e2etypes.GenesisFork() >= version.Fulu {
			return false
		}
		return policies.OnEpoch(params.BeaconConfig().FuluForkEpoch)(e)
	},
	Evaluation: fuluForkOccurs,
}

func altairForkOccurs(_ *e2etypes.EvaluationContext, nodeURLs ...string) error {
	return forkOccurs(nodeURLs[0], version.Altair, params.BeaconConfig().AltairForkEpoch)
}

func bellatrixForkOccurs(_ *e2etypes.EvaluationContext, nodeURLs ...string) error {
	return forkOccurs(nodeURLs[0], version.Bellatrix, params.BeaconConfig().BellatrixForkEpoch)
}

func capellaForkOccurs(_ *e2etypes.EvaluationContext, nodeURLs ...string) error {
	return forkOccurs(nodeURLs[0], version.Capella, params.BeaconConfig().CapellaForkEpoch)
}

func denebForkOccurs(_ *e2etypes.EvaluationContext, nodeURLs ...string) error {
	return forkOccurs(nodeURLs[0], version.Deneb, params.BeaconConfig().DenebForkEpoch)
}

func electraForkOccurs(_ *e2etypes.EvaluationContext, nodeURLs ...string) error {
	return forkOccurs(nodeURLs[0], version.Electra, params.BeaconConfig().ElectraForkEpoch)
}

func fuluForkOccurs(_ *e2etypes.EvaluationContext, nodeURLs ...string) error {
	return forkOccurs(nodeURLs[0], version.Fulu, params.BeaconConfig().FuluForkEpoch)
}

// forkOccurs checks that the head block is at or past the expected fork version.
func forkOccurs(nodeURL string, expectedVersion int, forkEpoch primitives.Epoch) error {
	client, err := helpers.NewBeaconNodeClient(nodeURL)
	if err != nil {
		return err
	}
	ctx := context.Background()

	blockResp, err := client.GetBlock(ctx, "head")
	if err != nil {
		return errors.Wrap(err, "failed to get head block")
	}
	if blockResp.Data == nil || blockResp.Data.Message == nil {
		return errors.New("nil block returned by beacon node")
	}

	blockVersion, err := version.FromString(blockResp.Version)
	if err != nil {
		return errors.Wrap(err, "could not parse block version")
	}
	if blockVersion < expectedVersion {
		return fmt.Errorf("expected %s block but got %s", version.String(expectedVersion), blockResp.Version)
	}

	// Parse slot from the block message.
	var msg struct {
		Slot string `json:"slot"`
	}
	if err := json.Unmarshal(blockResp.Data.Message, &msg); err != nil {
		return errors.Wrap(err, "could not unmarshal block message")
	}
	slot, err := strconv.ParseUint(msg.Slot, 10, 64)
	if err != nil {
		return errors.Wrap(err, "could not parse block slot")
	}

	fSlot, err := slots.EpochStart(forkEpoch)
	if err != nil {
		return err
	}
	if primitives.Slot(slot) < fSlot {
		return fmt.Errorf("wanted a block at slot >= %d but received %d", fSlot, slot)
	}
	return nil
}
