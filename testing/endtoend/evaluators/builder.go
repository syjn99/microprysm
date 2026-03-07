package evaluators

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/ssz"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/helpers"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/policies"
	e2etypes "github.com/OffchainLabs/prysm/v7/testing/endtoend/types"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/pkg/errors"
)

// BuilderIsActive checks that the builder is indeed producing the respective payloads
var BuilderIsActive = e2etypes.Evaluator{
	Name: "builder_is_active_at_epoch_%d",
	Policy: func(e primitives.Epoch) bool {
		fEpoch := params.BeaconConfig().BellatrixForkEpoch
		return policies.OnwardsNthEpoch(fEpoch)(e)
	},
	Evaluation: builderActive,
}

// maxNonBuilderBlocks is the maximum number of blocks that can be built locally
// instead of by the builder before the test fails. This allows tolerance for
// occasional builder timeouts or failures.
const maxNonBuilderBlocks = 2

// builderBlockMsg is a minimal struct for parsing execution payload fields
// relevant to the builder evaluator from any post-merge block version.
type builderBlockMsg struct {
	Slot string `json:"slot"`
	Body struct {
		ExecutionPayload *builderExecPayload `json:"execution_payload"`
	} `json:"body"`
}

type builderExecPayload struct {
	ExtraData    string   `json:"extra_data"`
	GasLimit     string   `json:"gas_limit"`
	Transactions []string `json:"transactions"`
}

func builderActive(_ *e2etypes.EvaluationContext, nodeURLs ...string) error {
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

	if lowestBound < params.BeaconConfig().BellatrixForkEpoch {
		lowestBound = params.BeaconConfig().BellatrixForkEpoch
	}
	emptyRt, err := ssz.TransactionsRoot([][]byte{})
	if err != nil {
		return err
	}

	nonBuilderBlocks := 0
	builderBlocks := 0

	if err := checkBuilderBlocksForEpoch(ctx, client, lowestBound, emptyRt, &nonBuilderBlocks, &builderBlocks); err != nil {
		return err
	}
	if lowestBound == currEpoch {
		if nonBuilderBlocks > maxNonBuilderBlocks {
			return errors.Errorf("too many non-builder blocks: %d (max allowed: %d), builder blocks: %d", nonBuilderBlocks, maxNonBuilderBlocks, builderBlocks)
		}
		return nil
	}
	if err := checkBuilderBlocksForEpoch(ctx, client, currEpoch, emptyRt, &nonBuilderBlocks, &builderBlocks); err != nil {
		return err
	}
	if nonBuilderBlocks > maxNonBuilderBlocks {
		return errors.Errorf("too many non-builder blocks: %d (max allowed: %d), builder blocks: %d", nonBuilderBlocks, maxNonBuilderBlocks, builderBlocks)
	}
	return nil
}

func checkBuilderBlocksForEpoch(
	ctx context.Context,
	client *helpers.BeaconNodeClient,
	epoch primitives.Epoch,
	emptyRt [32]byte,
	nonBuilderBlocks, builderBlocks *int,
) error {
	startSlot, err := slots.EpochStart(epoch)
	if err != nil {
		return errors.Wrap(err, "failed to compute start slot")
	}
	endSlot := startSlot + params.BeaconConfig().SlotsPerEpoch
	forkStartSlot, err := slots.EpochStart(params.BeaconConfig().BellatrixForkEpoch)
	if err != nil {
		return err
	}

	for slot := startSlot; slot < endSlot; slot++ {
		blockResp, err := client.GetBlock(ctx, strconv.FormatUint(uint64(slot), 10))
		if err != nil {
			// Only treat 'not found' errors as empty slots (missed blocks).
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "not found") {
				continue
			}
			return errors.Wrapf(err, "failed to get block at slot %d", slot)
		}
		if blockResp.Data == nil {
			continue
		}

		// Skip pre-merge blocks.
		v, err := version.FromString(blockResp.Version)
		if err != nil || v < version.Bellatrix {
			continue
		}

		var msg builderBlockMsg
		if err := json.Unmarshal(blockResp.Data.Message, &msg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal block message at slot %d", slot)
		}
		if msg.Body.ExecutionPayload == nil {
			return errors.New("nil block provided")
		}

		blockSlot, err := strconv.ParseUint(msg.Slot, 10, 64)
		if err != nil {
			return errors.Wrapf(err, "failed to parse slot")
		}

		if primitives.Slot(blockSlot) == forkStartSlot || primitives.Slot(blockSlot) == forkStartSlot+1 || epoch <= 1 {
			// Skip fork slot and the next one, as we don't send FCUs yet.
			continue
		}

		payload := msg.Body.ExecutionPayload

		// Decode transactions and compute root.
		txs := make([][]byte, len(payload.Transactions))
		for i, t := range payload.Transactions {
			txs[i], err = hex.DecodeString(strings.TrimPrefix(t, "0x"))
			if err != nil {
				return errors.Wrapf(err, "failed to decode transaction %d", i)
			}
		}
		txRoot, err := ssz.TransactionsRoot(txs)
		if err != nil {
			return err
		}

		// Decode extra data.
		extraBytes, err := hex.DecodeString(strings.TrimPrefix(payload.ExtraData, "0x"))
		if err != nil {
			return errors.Wrap(err, "failed to decode extra data")
		}
		extraData := string(extraBytes)

		if txRoot == emptyRt && extraData != "prysm-builder" {
			// If a local payload is built with 0 transactions, builder cannot build a payload with more transactions
			// since they both utilize the same EL.
			continue
		}
		if extraData != "prysm-builder" {
			*nonBuilderBlocks++
			continue
		}
		*builderBlocks++

		gasLimit, err := strconv.ParseUint(payload.GasLimit, 10, 64)
		if err != nil {
			return errors.Wrap(err, "failed to parse gas limit")
		}
		if gasLimit == 0 {
			return errors.Errorf("%s block with slot %d has a gas limit of 0, when it should be in the 30M range", blockResp.Version, blockSlot)
		}
	}
	return nil
}
