package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/OffchainLabs/prysm/v7/api/client"
	"github.com/OffchainLabs/prysm/v7/api/server/structs"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/pkg/errors"
)

// BeaconNodeClient is a REST client for E2E test evaluators. It wraps the
// existing api/client.Client and provides typed helpers for the beacon API
// endpoints that evaluators need.
type BeaconNodeClient struct {
	c *client.Client
}

// NewBeaconNodeClient creates a REST client pointing at the given beacon node
// base URL (e.g. "http://127.0.0.1:3500").
func NewBeaconNodeClient(baseURL string) (*BeaconNodeClient, error) {
	c, err := client.NewClient(baseURL)
	if err != nil {
		return nil, err
	}
	return &BeaconNodeClient{c: c}, nil
}

// ChainHead holds the same information the gRPC GetChainHead RPC returned,
// assembled from the REST /headers/head and /finality_checkpoints/head
// endpoints.
type ChainHead struct {
	HeadSlot               primitives.Slot
	HeadEpoch              primitives.Epoch
	HeadBlockRoot          string
	FinalizedEpoch         primitives.Epoch
	FinalizedRoot          string
	JustifiedEpoch         primitives.Epoch
	JustifiedRoot          string
	PreviousJustifiedEpoch primitives.Epoch
	PreviousJustifiedRoot  string
}

// GetChainHead returns chain head information by combining two REST calls:
//   - GET /eth/v1/beacon/headers/head     → head slot / root
//   - GET /eth/v1/beacon/finality_checkpoints/head → justified / finalized epochs
func (b *BeaconNodeClient) GetChainHead(ctx context.Context) (*ChainHead, error) {
	// Fetch head header.
	headerBody, err := b.c.Get(ctx, "/eth/v1/beacon/headers/head")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get head header")
	}
	headerResp := &structs.GetBlockHeaderResponse{}
	if err := json.Unmarshal(headerBody, headerResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal head header")
	}
	if headerResp.Data == nil || headerResp.Data.Header == nil || headerResp.Data.Header.Message == nil {
		return nil, errors.New("head header response has nil data")
	}
	headSlot, err := strconv.ParseUint(headerResp.Data.Header.Message.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse head slot")
	}

	// Fetch finality checkpoints.
	finBody, err := b.c.Get(ctx, "/eth/v1/beacon/states/head/finality_checkpoints")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get finality checkpoints")
	}
	finResp := &structs.GetFinalityCheckpointsResponse{}
	if err := json.Unmarshal(finBody, finResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal finality checkpoints")
	}
	if finResp.Data == nil {
		return nil, errors.New("finality checkpoints response has nil data")
	}

	finalizedEpoch, err := strconv.ParseUint(finResp.Data.Finalized.Epoch, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse finalized epoch")
	}
	justifiedEpoch, err := strconv.ParseUint(finResp.Data.CurrentJustified.Epoch, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse justified epoch")
	}
	prevJustifiedEpoch, err := strconv.ParseUint(finResp.Data.PreviousJustified.Epoch, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse previous justified epoch")
	}

	return &ChainHead{
		HeadSlot:               primitives.Slot(headSlot),
		HeadEpoch:              primitives.Epoch(headSlot / uint64(params.BeaconConfig().SlotsPerEpoch)),
		HeadBlockRoot:          headerResp.Data.Root,
		FinalizedEpoch:         primitives.Epoch(finalizedEpoch),
		FinalizedRoot:          finResp.Data.Finalized.Root,
		JustifiedEpoch:         primitives.Epoch(justifiedEpoch),
		JustifiedRoot:          finResp.Data.CurrentJustified.Root,
		PreviousJustifiedEpoch: primitives.Epoch(prevJustifiedEpoch),
		PreviousJustifiedRoot:  finResp.Data.PreviousJustified.Root,
	}, nil
}

// GetGenesis returns genesis information from /eth/v1/beacon/genesis.
func (b *BeaconNodeClient) GetGenesis(ctx context.Context) (*structs.GetGenesisResponse, error) {
	body, err := b.c.Get(ctx, "/eth/v1/beacon/genesis")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get genesis")
	}
	resp := &structs.GetGenesisResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal genesis")
	}
	return resp, nil
}

// GetBlock retrieves a block in JSON from /eth/v2/beacon/blocks/{blockID}.
func (b *BeaconNodeClient) GetBlock(ctx context.Context, blockID string) (*structs.GetBlockV2Response, error) {
	body, err := b.c.Get(ctx, path.Join("/eth/v2/beacon/blocks", blockID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block %s", blockID)
	}
	resp := &structs.GetBlockV2Response{}
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal block")
	}
	return resp, nil
}

// GetBlockSSZ retrieves a block in SSZ encoding from /eth/v2/beacon/blocks/{blockID}.
func (b *BeaconNodeClient) GetBlockSSZ(ctx context.Context, blockID string) ([]byte, error) {
	return b.c.Get(ctx, path.Join("/eth/v2/beacon/blocks", blockID), client.WithSSZEncoding())
}

// ListValidators returns validators from /eth/v1/beacon/states/{stateID}/validators.
func (b *BeaconNodeClient) ListValidators(ctx context.Context, stateID string) (*structs.GetValidatorsResponse, error) {
	body, err := b.c.Get(ctx, fmt.Sprintf("/eth/v1/beacon/states/%s/validators", stateID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list validators for state %s", stateID)
	}
	resp := &structs.GetValidatorsResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal validators")
	}
	return resp, nil
}

// GetValidator returns a single validator from /eth/v1/beacon/states/{stateID}/validators/{validatorID}.
func (b *BeaconNodeClient) GetValidator(ctx context.Context, stateID, validatorID string) (*structs.GetValidatorResponse, error) {
	body, err := b.c.Get(ctx, fmt.Sprintf("/eth/v1/beacon/states/%s/validators/%s", stateID, validatorID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get validator %s", validatorID)
	}
	resp := &structs.GetValidatorResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal validator")
	}
	return resp, nil
}

// GetValidatorParticipation returns participation data from /prysm/v1/validators/{stateID}/participation.
func (b *BeaconNodeClient) GetValidatorParticipation(ctx context.Context, stateID string) (*structs.GetValidatorParticipationResponse, error) {
	body, err := b.c.Get(ctx, fmt.Sprintf("/prysm/v1/validators/%s/participation", stateID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get validator participation for state %s", stateID)
	}
	resp := &structs.GetValidatorParticipationResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal validator participation")
	}
	return resp, nil
}

// GetSyncStatus returns sync status from /eth/v1/node/syncing.
func (b *BeaconNodeClient) GetSyncStatus(ctx context.Context) (*structs.SyncStatusResponse, error) {
	body, err := b.c.Get(ctx, "/eth/v1/node/syncing")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get sync status")
	}
	resp := &structs.SyncStatusResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal sync status")
	}
	return resp, nil
}

// ListPeers returns peers from /eth/v1/node/peers.
func (b *BeaconNodeClient) ListPeers(ctx context.Context) (*structs.GetPeersResponse, error) {
	body, err := b.c.Get(ctx, "/eth/v1/node/peers")
	if err != nil {
		return nil, errors.Wrap(err, "failed to list peers")
	}
	resp := &structs.GetPeersResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal peers")
	}
	return resp, nil
}

// ListDebugPeers returns detailed peer info from /prysm/v1/debug/peers.
func (b *BeaconNodeClient) ListDebugPeers(ctx context.Context) (*structs.DebugPeersResponse, error) {
	body, err := b.c.Get(ctx, "/prysm/v1/debug/peers")
	if err != nil {
		return nil, errors.Wrap(err, "failed to list debug peers")
	}
	resp := &structs.DebugPeersResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal debug peers")
	}
	return resp, nil
}

// GetBeaconStateSSZ retrieves the beacon state in SSZ encoding from
// /eth/v2/debug/beacon/states/{stateID}.
func (b *BeaconNodeClient) GetBeaconStateSSZ(ctx context.Context, stateID string) ([]byte, error) {
	return b.c.Get(ctx, path.Join("/eth/v2/debug/beacon/states", stateID), client.WithSSZEncoding())
}

// GetFinalityCheckpoints returns finality checkpoints from
// /eth/v1/beacon/states/{stateID}/finality_checkpoints.
func (b *BeaconNodeClient) GetFinalityCheckpoints(ctx context.Context, stateID string) (*structs.GetFinalityCheckpointsResponse, error) {
	body, err := b.c.Get(ctx, fmt.Sprintf("/eth/v1/beacon/states/%s/finality_checkpoints", stateID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get finality checkpoints for state %s", stateID)
	}
	resp := &structs.GetFinalityCheckpointsResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal finality checkpoints")
	}
	return resp, nil
}

// SubmitVoluntaryExit posts a signed voluntary exit to /eth/v1/beacon/pool/voluntary_exits.
func (b *BeaconNodeClient) SubmitVoluntaryExit(ctx context.Context, exit *structs.SignedVoluntaryExit) error {
	jsonBytes, err := json.Marshal(exit)
	if err != nil {
		return errors.Wrap(err, "failed to marshal voluntary exit")
	}
	return b.post(ctx, "/eth/v1/beacon/pool/voluntary_exits", jsonBytes)
}

// GetAttesterDuties returns attester duties via POST /eth/v1/validator/duties/attester/{epoch}.
func (b *BeaconNodeClient) GetAttesterDuties(ctx context.Context, epoch primitives.Epoch, validatorIndices []string) (*structs.GetAttesterDutiesResponse, error) {
	jsonBytes, err := json.Marshal(validatorIndices)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal validator indices")
	}
	body, err := b.postAndRead(ctx, fmt.Sprintf("/eth/v1/validator/duties/attester/%d", epoch), jsonBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get attester duties")
	}
	resp := &structs.GetAttesterDutiesResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal attester duties")
	}
	return resp, nil
}

// GetProposerDuties returns proposer duties from /eth/v1/validator/duties/proposer/{epoch}.
func (b *BeaconNodeClient) GetProposerDuties(ctx context.Context, epoch primitives.Epoch) (*structs.GetProposerDutiesResponse, error) {
	body, err := b.c.Get(ctx, fmt.Sprintf("/eth/v1/validator/duties/proposer/%d", epoch))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get proposer duties")
	}
	resp := &structs.GetProposerDutiesResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal proposer duties")
	}
	return resp, nil
}

// GetAttestationData returns attestation data from /eth/v1/validator/attestation_data.
func (b *BeaconNodeClient) GetAttestationData(ctx context.Context, slot primitives.Slot, committeeIndex primitives.CommitteeIndex) (*structs.GetAttestationDataResponse, error) {
	endpoint := fmt.Sprintf("/eth/v1/validator/attestation_data?slot=%d&committee_index=%d", slot, committeeIndex)
	body, err := b.c.Get(ctx, endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get attestation data")
	}
	resp := &structs.GetAttestationDataResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal attestation data")
	}
	return resp, nil
}

// post sends a POST request with JSON body and expects a 200 OK with no meaningful body.
func (b *BeaconNodeClient) post(ctx context.Context, endpoint string, jsonBody []byte) error {
	u := b.c.BaseURL().ResolveReference(&url.URL{Path: endpoint})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(jsonBody))
	if err != nil {
		return errors.Wrap(err, "failed to create POST request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.c.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("POST %s returned status %d", endpoint, resp.StatusCode)
	}
	return nil
}

// postAndRead sends a POST request and returns the response body.
func (b *BeaconNodeClient) postAndRead(ctx context.Context, endpoint string, jsonBody []byte) ([]byte, error) {
	u := b.c.BaseURL().ResolveReference(&url.URL{Path: endpoint})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(jsonBody))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create POST request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("POST %s returned status %d", endpoint, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, client.MaxBodySize))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	return body, nil
}
