package evaluators

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/helpers"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/policies"
	e2etypes "github.com/OffchainLabs/prysm/v7/testing/endtoend/types"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Minimum blob utilization percentage required to pass the evaluator.
// We expect the transaction generator to produce enough blobs to hit at least this threshold.
const minBlobUtilizationPercent = 50

// BlobsIncludedInBlocks verifies that blocks contain blobs and that the blob count
// does not exceed the maximum allowed for the epoch (respecting BPO schedule).
var BlobsIncludedInBlocks = e2etypes.Evaluator{
	Name: "blobs_included_in_blocks_epoch_%d",
	Policy: func(currentEpoch primitives.Epoch) bool {
		// Only run from Deneb onwards
		return policies.OnwardsNthEpoch(params.BeaconConfig().DenebForkEpoch)(currentEpoch)
	},
	Evaluation: blobsIncludedInBlocks,
}

// BlobLimitsRespected verifies that the BPO (Blob Parameter Optimization) limits
// are correctly enforced after Fulu fork, checking that:
// 1. Blocks don't exceed MaxBlobsPerBlock for their respective epoch
// 2. We're utilizing the increased capacity (at least 50% utilization)
// 3. When BPO activates (limit increases), blocks actually use the new higher limit
var BlobLimitsRespected = e2etypes.Evaluator{
	Name: "blob_limits_respected_epoch_%d",
	Policy: func(currentEpoch primitives.Epoch) bool {
		// Only run from Fulu onwards where BPO is active
		return policies.OnwardsNthEpoch(params.BeaconConfig().FuluForkEpoch)(currentEpoch)
	},
	Evaluation: blobLimitsRespected,
}

// blobBlockMsg is a minimal struct for parsing blob-related fields from any post-Deneb block.
type blobBlockMsg struct {
	Slot string `json:"slot"`
	Body struct {
		BlobKzgCommitments []string `json:"blob_kzg_commitments"`
	} `json:"body"`
}

func blobsIncludedInBlocks(_ *e2etypes.EvaluationContext, nodeURLs ...string) error {
	client, err := helpers.NewBeaconNodeClient(nodeURLs[0])
	if err != nil {
		return err
	}
	ctx := context.Background()

	chainHead, err := client.GetChainHead(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get chain head")
	}

	// Check blocks from the previous epoch
	epoch := chainHead.HeadEpoch
	if epoch > 0 {
		epoch = epoch - 1
	}

	// Skip if we're before Deneb
	if epoch < params.BeaconConfig().DenebForkEpoch {
		return nil
	}

	startSlot, err := slots.EpochStart(epoch)
	if err != nil {
		return errors.Wrap(err, "failed to compute start slot")
	}
	endSlot := startSlot + params.BeaconConfig().SlotsPerEpoch

	blocksWithBlobs := 0
	for slot := startSlot; slot < endSlot; slot++ {
		blockResp, err := client.GetBlock(ctx, strconv.FormatUint(uint64(slot), 10))
		if err != nil {
			// Slot may be empty (missed block); skip.
			continue
		}
		if blockResp.Data == nil {
			continue
		}

		// Skip blocks before Deneb
		v, err := version.FromString(blockResp.Version)
		if err != nil || v < version.Deneb {
			continue
		}

		var msg blobBlockMsg
		if err := json.Unmarshal(blockResp.Data.Message, &msg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal block message at slot %d", slot)
		}

		if len(msg.Body.BlobKzgCommitments) > 0 {
			blocksWithBlobs++
		}
	}

	// We expect at least some blocks to have blobs since we're sending blob transactions
	if blocksWithBlobs == 0 {
		return errors.Errorf("no blocks with blobs found in epoch %d", epoch)
	}

	return nil
}

func blobLimitsRespected(_ *e2etypes.EvaluationContext, nodeURLs ...string) error {
	client, err := helpers.NewBeaconNodeClient(nodeURLs[0])
	if err != nil {
		return err
	}
	ctx := context.Background()

	genesisResp, err := client.GetGenesis(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get genesis")
	}
	genesisTimeSec, err := strconv.ParseInt(genesisResp.Data.GenesisTime, 10, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse genesis time")
	}
	genesisTime := time.Unix(genesisTimeSec, 0)

	currSlot := slots.CurrentSlot(genesisTime)
	currEpoch := slots.ToEpoch(currSlot)

	// Check the previous epoch to ensure blocks are finalized
	epochToCheck := currEpoch
	if epochToCheck > 0 {
		epochToCheck = epochToCheck - 1
	}

	// Skip if we're before Fulu
	if epochToCheck < params.BeaconConfig().FuluForkEpoch {
		return nil
	}

	startSlot, err := slots.EpochStart(epochToCheck)
	if err != nil {
		return errors.Wrap(err, "failed to compute start slot")
	}
	endSlot := startSlot + params.BeaconConfig().SlotsPerEpoch

	cfg := params.BeaconConfig()
	maxBlobsForEpoch := cfg.MaxBlobsPerBlockAtEpoch(epochToCheck)

	// Get the previous epoch's limit to detect BPO transitions
	var prevEpochMaxBlobs int
	if epochToCheck > 0 {
		prevEpochMaxBlobs = cfg.MaxBlobsPerBlockAtEpoch(epochToCheck - 1)
	}

	// Check if this is a BPO transition epoch (limit increased from previous epoch)
	isBPOTransitionEpoch := maxBlobsForEpoch > prevEpochMaxBlobs

	var totalBlobs int
	var maxBlobsInBlock int
	var blockCount int

	for slot := startSlot; slot < endSlot; slot++ {
		blockResp, err := client.GetBlock(ctx, strconv.FormatUint(uint64(slot), 10))
		if err != nil {
			// Slot may be empty (missed block); skip.
			continue
		}
		if blockResp.Data == nil {
			continue
		}

		// Skip blocks before Deneb (shouldn't happen if we're checking post-Fulu epochs)
		v, err := version.FromString(blockResp.Version)
		if err != nil || v < version.Deneb {
			continue
		}

		var msg blobBlockMsg
		if err := json.Unmarshal(blockResp.Data.Message, &msg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal block message at slot %d", slot)
		}

		parsedSlot, err := strconv.ParseUint(msg.Slot, 10, 64)
		if err != nil {
			return errors.Wrapf(err, "failed to parse slot at %d", slot)
		}
		blockEpoch := slots.ToEpoch(primitives.Slot(parsedSlot))

		blobCount := len(msg.Body.BlobKzgCommitments)

		// Verify we don't exceed the limit
		if blobCount > maxBlobsForEpoch {
			return errors.Errorf(
				"block at slot %d (epoch %d) has %d blobs, exceeding max allowed %d for this epoch",
				parsedSlot, blockEpoch, blobCount, maxBlobsForEpoch,
			)
		}

		totalBlobs += blobCount
		blockCount++
		if blobCount > maxBlobsInBlock {
			maxBlobsInBlock = blobCount
		}
	}

	// Calculate utilization
	if blockCount == 0 {
		return errors.Errorf("no blocks found in epoch %d", epochToCheck)
	}

	utilizationPercent := (maxBlobsInBlock * 100) / maxBlobsForEpoch

	logrus.WithFields(logrus.Fields{
		"epoch":                epochToCheck,
		"maxBlobsForEpoch":     maxBlobsForEpoch,
		"prevEpochMaxBlobs":    prevEpochMaxBlobs,
		"maxBlobsInBlock":      maxBlobsInBlock,
		"totalBlobs":           totalBlobs,
		"blockCount":           blockCount,
		"utilizationPercent":   utilizationPercent,
		"isBPOTransitionEpoch": isBPOTransitionEpoch,
	}).Info("Blob utilization stats for epoch")

	// For BPO transition epochs, verify that BPO actually activated by checking
	// that at least one block exceeded the previous epoch's limit
	if isBPOTransitionEpoch {
		if maxBlobsInBlock <= prevEpochMaxBlobs {
			return errors.Errorf(
				"BPO transition at epoch %d: limit increased from %d to %d, but no block exceeded the old limit. "+
					"Max blobs in any block was %d. This indicates BPO may not be working correctly or "+
					"the transaction generator is not producing enough blobs to test the increased limit.",
				epochToCheck, prevEpochMaxBlobs, maxBlobsForEpoch, maxBlobsInBlock,
			)
		}
		logrus.WithFields(logrus.Fields{
			"epoch":           epochToCheck,
			"prevLimit":       prevEpochMaxBlobs,
			"newLimit":        maxBlobsForEpoch,
			"maxBlobsInBlock": maxBlobsInBlock,
		}).Info("BPO activation verified: blocks are using capacity beyond previous limit")
	}

	// For all BPO epochs (where limit > Electra's 9), verify we're utilizing the capacity
	electraMax := cfg.DeprecatedMaxBlobsPerBlockElectra
	if maxBlobsForEpoch > electraMax {
		if utilizationPercent < minBlobUtilizationPercent {
			return errors.Errorf(
				"BPO epoch %d has max blobs %d but only achieved %d%% utilization (max blobs in any block: %d). "+
					"Expected at least %d%% utilization to verify BPO is working. "+
					"This may indicate the transaction generator is not producing enough blobs.",
				epochToCheck, maxBlobsForEpoch, utilizationPercent, maxBlobsInBlock, minBlobUtilizationPercent,
			)
		}
	}

	return nil
}
