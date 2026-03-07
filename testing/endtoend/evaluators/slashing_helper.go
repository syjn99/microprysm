package evaluators

import (
	"context"
	"crypto/rand"
	"strconv"

	"github.com/OffchainLabs/go-bitfield"
	"github.com/OffchainLabs/prysm/v7/api/server/structs"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/signing"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/crypto/bls"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	eth "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/helpers"
	"github.com/OffchainLabs/prysm/v7/testing/util"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
)

type doubleAttestationHelper struct {
	client   *helpers.BeaconNodeClient
	privKeys []bls.SecretKey
	pubKeys  [][]byte
	domain   []byte
	attData  *eth.AttestationData

	committee []primitives.ValidatorIndex
}

// setup initializes the helper with details needed to make a double attestation.
func (h *doubleAttestationHelper) setup(ctx context.Context) error {
	chainHead, err := h.client.GetChainHead(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get chain head")
	}
	_, privKeys, err := util.DeterministicDepositsAndKeys(params.BeaconConfig().MinGenesisActiveValidatorCount)
	if err != nil {
		return errors.Wrap(err, "could not get depositsandkeys")
	}

	pubKeys := make([][]byte, len(privKeys))
	for i, priv := range privKeys {
		pubKeys[i] = priv.PublicKey().Marshal()
	}

	// Use the committees API to get the committee directly from the server.
	committeesResp, err := h.client.GetCommittees(ctx, "head", chainHead.HeadSlot)
	if err != nil {
		return errors.Wrap(err, "could not get committees")
	}
	if len(committeesResp.Data) == 0 {
		return errors.New("no committees found for head slot")
	}

	// Use the first committee at the head slot.
	firstCommittee := committeesResp.Data[0]
	ci, err := strconv.ParseUint(firstCommittee.Index, 10, 64)
	if err != nil {
		return errors.Wrap(err, "could not parse committee index")
	}
	committeeIndex := primitives.CommitteeIndex(ci)

	committee := make([]primitives.ValidatorIndex, len(firstCommittee.Validators))
	for i, valIdxStr := range firstCommittee.Validators {
		vi, err := strconv.ParseUint(valIdxStr, 10, 64)
		if err != nil {
			return errors.Wrapf(err, "could not parse validator index at position %d", i)
		}
		committee[i] = primitives.ValidatorIndex(vi)
	}

	attDataResp, err := h.client.GetAttestationData(ctx, chainHead.HeadSlot, committeeIndex)
	if err != nil {
		return err
	}

	// Convert REST attestation data to proto for signing.
	attData, err := attestationDataFromREST(attDataResp.Data)
	if err != nil {
		return errors.Wrap(err, "could not convert attestation data")
	}

	domain, err := helpers.ComputeDomainData(chainHead.HeadEpoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		return errors.Wrap(err, "could not get domain data")
	}

	h.privKeys = privKeys
	h.pubKeys = pubKeys
	h.domain = domain
	h.committee = committee
	h.attData = attData

	return nil
}

// attestationDataFromREST converts REST attestation data JSON to proto.
func attestationDataFromREST(data *structs.AttestationData) (*eth.AttestationData, error) {
	slot, err := strconv.ParseUint(data.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse slot")
	}
	ci, err := strconv.ParseUint(data.CommitteeIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse committee index")
	}
	blockRoot, err := hexutil.Decode(data.BeaconBlockRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode beacon block root")
	}
	sourceEpoch, err := strconv.ParseUint(data.Source.Epoch, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse source epoch")
	}
	sourceRoot, err := hexutil.Decode(data.Source.Root)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode source root")
	}
	targetEpoch, err := strconv.ParseUint(data.Target.Epoch, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse target epoch")
	}
	targetRoot, err := hexutil.Decode(data.Target.Root)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode target root")
	}
	return &eth.AttestationData{
		Slot:            primitives.Slot(slot),
		CommitteeIndex:  primitives.CommitteeIndex(ci),
		BeaconBlockRoot: blockRoot,
		Source: &eth.Checkpoint{
			Epoch: primitives.Epoch(sourceEpoch),
			Root:  sourceRoot,
		},
		Target: &eth.Checkpoint{
			Epoch: primitives.Epoch(targetEpoch),
			Root:  targetRoot,
		},
	}, nil
}

// validatorIndexAtCommitteeIndex returns the validatorIndex at position idx in the committee.
func (h *doubleAttestationHelper) validatorIndexAtCommitteeIndex(idx uint64) primitives.ValidatorIndex {
	return h.committee[idx]
}

// getSlashableAttestation returns an attestation with a random block root, signed by the
// validator at position idx in the committee. The random block root ensures P2P uniqueness.
func (h *doubleAttestationHelper) getSlashableAttestation(idx uint64) (*eth.Attestation, error) {
	// msg must be unique so they are not filtered by P2P
	randVal := make([]byte, 4)
	_, err := rand.Read(randVal)
	if err != nil {
		return nil, errors.Wrap(err, "error reading random val")
	}
	blockRoot := bytesutil.ToBytes32(append(randVal, []byte("muahahahaha evil validator")...))
	h.attData.BeaconBlockRoot = blockRoot[:]

	signingRoot, err := signing.ComputeSigningRoot(h.attData, h.domain)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute signing root")
	}

	valIdx := h.validatorIndexAtCommitteeIndex(idx)

	attBitfield := bitfield.NewBitlist(uint64(len(h.committee)))
	attBitfield.SetBitAt(idx, true)
	att := &eth.Attestation{
		AggregationBits: attBitfield,
		Data:            h.attData,
		Signature:       h.privKeys[valIdx].Sign(signingRoot[:]).Marshal(),
	}
	return att, nil
}
