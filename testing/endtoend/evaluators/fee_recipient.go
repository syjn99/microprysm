package evaluators

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/runtime/interop"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/components"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/helpers"
	e2e "github.com/OffchainLabs/prysm/v7/testing/endtoend/params"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/policies"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/types"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var FeeRecipientIsPresent = types.Evaluator{
	Name: "fee_recipient_is_present_%d",
	Policy: func(e primitives.Epoch) bool {
		fEpoch := params.BeaconConfig().BellatrixForkEpoch
		return policies.AfterNthEpoch(fEpoch)(e)
	},
	Evaluation: feeRecipientIsPresent,
}

func lhKeyMap() (map[string]bool, error) {
	if e2e.TestParams.LighthouseBeaconNodeCount == 0 {
		return nil, nil
	}
	pry, lh := e2e.TestParams.BeaconNodeCount, e2e.TestParams.LighthouseBeaconNodeCount
	valPerNode := int(params.BeaconConfig().MinGenesisActiveValidatorCount) / (pry + lh)
	lhOff := valPerNode * pry
	_, keys, err := interop.DeterministicallyGenerateKeys(uint64(lhOff), uint64(valPerNode*lh))
	if err != nil {
		return nil, err
	}

	km := make(map[string]bool)
	for _, k := range keys {
		km[hexutil.Encode(k.Marshal())] = true
	}
	return km, nil
}

func valKeyMap() (map[string]bool, error) {
	nvals := params.BeaconConfig().MinGenesisActiveValidatorCount
	// matches validator start in validator component + validators used for deposits
	_, pubs, err := interop.DeterministicallyGenerateKeys(0, nvals+e2e.DepositCount)
	if err != nil {
		return nil, err
	}
	km := make(map[string]bool)
	for _, k := range pubs {
		km[hexutil.Encode(k.Marshal())] = true
	}
	return km, nil
}

// executionBlockMsg is a minimal struct for parsing the execution payload fields
// common to all post-merge block versions (Bellatrix, Capella, Deneb, Electra, Fulu).
type executionBlockMsg struct {
	ProposerIndex string `json:"proposer_index"`
	Body          struct {
		ExecutionPayload *executionPayloadBase `json:"execution_payload"`
	} `json:"body"`
}

type executionPayloadBase struct {
	FeeRecipient string `json:"fee_recipient"`
	BlockHash    string `json:"block_hash"`
	ParentHash   string `json:"parent_hash"`
}

func feeRecipientIsPresent(_ *types.EvaluationContext, nodeURLs ...string) error {
	client, err := helpers.NewBeaconNodeClient(nodeURLs[0])
	if err != nil {
		return err
	}
	ctx := context.Background()

	chainHead, err := client.GetChainHead(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get chain head")
	}
	epoch := chainHead.HeadEpoch
	if epoch > 0 {
		epoch--
	}

	rpcclient, err := rpc.DialHTTP(fmt.Sprintf("http://127.0.0.1:%d", e2e.TestParams.Ports.Eth1RPCPort))
	if err != nil {
		return err
	}
	defer rpcclient.Close()

	valkeys, err := valKeyMap()
	if err != nil {
		return err
	}
	lhkeys, err := lhKeyMap()
	if err != nil {
		return err
	}

	// Iterate over all slots in the epoch, fetching each block individually.
	startSlot, err := slots.EpochStart(epoch)
	if err != nil {
		return errors.Wrap(err, "failed to compute start slot")
	}
	endSlot := startSlot + params.BeaconConfig().SlotsPerEpoch
	for slot := startSlot; slot < endSlot; slot++ {
		blockResp, err := client.GetBlock(ctx, strconv.FormatUint(uint64(slot), 10))
		if err != nil {
			// Slot may be empty (missed block); skip.
			continue
		}
		if blockResp.Data == nil {
			continue
		}

		version := blockResp.Version
		// Only check post-merge blocks that have execution payloads.
		if version == "phase0" || version == "altair" {
			continue
		}

		var msg executionBlockMsg
		if err := json.Unmarshal(blockResp.Data.Message, &msg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal block message at slot %d", slot)
		}
		if msg.Body.ExecutionPayload == nil {
			continue
		}
		payload := msg.Body.ExecutionPayload

		// If the beacon chain has transitioned to Bellatrix, but the EL hasn't hit TTD, we could see a few slots
		// of blocks with empty payloads.
		emptyHash := hexutil.Encode(make([]byte, 32))
		if payload.BlockHash == emptyHash {
			continue
		}
		if payload.FeeRecipient == "" || payload.FeeRecipient == params.BeaconConfig().EthBurnAddressHex {
			log.WithField("proposerIndex", msg.ProposerIndex).WithField("slot", slot).Error("Fee recipient eval bug")
			return errors.New("fee recipient is not set")
		}

		fr := common.HexToAddress(payload.FeeRecipient)

		proposerIndex := msg.ProposerIndex
		validatorResp, err := client.GetValidator(ctx, strconv.FormatUint(uint64(slot), 10), proposerIndex)
		if err != nil {
			return errors.Wrap(err, "failed to get validators")
		}
		pk := validatorResp.Data.Validator.Pubkey

		if _, ok := lhkeys[pk]; ok {
			// Don't check lighthouse keys.
			continue
		}

		// In e2e we generate deterministic keys by validator index, and then use a slice of their public key bytes
		// as the fee recipient, so that this will also be deterministic, so this test can statelessly verify it.
		// These should be the only keys we see.
		// Otherwise, something has changed in e2e and this test needs to be updated.
		_, knownKey := valkeys[pk]
		if !knownKey {
			log.WithField("pubkey", pk).
				WithField("slot", slot).
				WithField("proposerIndex", proposerIndex).
				WithField("feeRecipient", fr.Hex()).
				Warn("Unknown key observed, not a deterministically generated key")
			return errors.New("unknown key observed, not a deterministically generated key")
		}

		if components.FeeRecipientFromPubkey(pk) != fr.Hex() {
			return fmt.Errorf("publickey %s, fee recipient %s does not match the proposer settings fee recipient %s",
				pk, fr.Hex(), components.FeeRecipientFromPubkey(pk))
		}

		blockHash := common.HexToHash(payload.BlockHash)
		parentHash := common.HexToHash(payload.ParentHash)
		if err := checkRecipientBalance(rpcclient, blockHash, parentHash, fr); err != nil {
			return err
		}
	}

	return nil
}

func checkRecipientBalance(c *rpc.Client, block, parent common.Hash, account common.Address) error {
	web3 := ethclient.NewClient(c)
	ctx := context.Background()
	b, err := web3.BlockByHash(ctx, block)
	if err != nil {
		return err
	}

	bal, err := web3.BalanceAt(ctx, account, b.Number())
	if err != nil {
		return err
	}
	pBlock, err := web3.BlockByHash(ctx, parent)
	if err != nil {
		return err
	}
	pBal, err := web3.BalanceAt(ctx, account, pBlock.Number())
	if err != nil {
		return err
	}
	if b.GasUsed() > 0 && bal.Uint64() <= pBal.Uint64() {
		return errors.Errorf("account balance didn't change after applying fee recipient for account: %s", account.Hex())
	}

	return nil
}
