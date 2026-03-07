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
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/signing"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/genesis"
	eth "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
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

// GetChainHead returns chain head information using the Prysm-specific
// /prysm/v1/beacon/chain_head endpoint (single call instead of composing
// from headers + finality_checkpoints).
func (b *BeaconNodeClient) GetChainHead(ctx context.Context) (*ChainHead, error) {
	body, err := b.c.Get(ctx, "/prysm/v1/beacon/chain_head")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get chain head")
	}
	resp := &structs.ChainHead{}
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal chain head")
	}

	headSlot, err := strconv.ParseUint(resp.HeadSlot, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse head slot")
	}
	headEpoch, err := strconv.ParseUint(resp.HeadEpoch, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse head epoch")
	}
	finalizedEpoch, err := strconv.ParseUint(resp.FinalizedEpoch, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse finalized epoch")
	}
	justifiedEpoch, err := strconv.ParseUint(resp.JustifiedEpoch, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse justified epoch")
	}
	prevJustifiedEpoch, err := strconv.ParseUint(resp.PreviousJustifiedEpoch, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse previous justified epoch")
	}

	return &ChainHead{
		HeadSlot:               primitives.Slot(headSlot),
		HeadEpoch:              primitives.Epoch(headEpoch),
		HeadBlockRoot:          resp.HeadBlockRoot,
		FinalizedEpoch:         primitives.Epoch(finalizedEpoch),
		FinalizedRoot:          resp.FinalizedBlockRoot,
		JustifiedEpoch:         primitives.Epoch(justifiedEpoch),
		JustifiedRoot:          resp.JustifiedBlockRoot,
		PreviousJustifiedEpoch: primitives.Epoch(prevJustifiedEpoch),
		PreviousJustifiedRoot:  resp.PreviousJustifiedBlockRoot,
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
func (b *BeaconNodeClient) ListValidators(ctx context.Context, stateID string, statuses ...string) (*structs.GetValidatorsResponse, error) {
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
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
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
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("POST %s returned status %d", endpoint, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, client.MaxBodySize))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	return body, nil
}

// SubmitAttestations submits attestations to the beacon pool via
// POST /eth/v2/beacon/pool/attestations. The version header is required.
func (b *BeaconNodeClient) SubmitAttestations(ctx context.Context, version string, jsonAtts []byte) error {
	u := b.c.BaseURL().ResolveReference(&url.URL{Path: "/eth/v2/beacon/pool/attestations"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(jsonAtts))
	if err != nil {
		return errors.Wrap(err, "failed to create POST request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Eth-Consensus-Version", version)
	resp, err := b.c.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, client.MaxBodySize))
		return fmt.Errorf("POST attestations returned status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// PublishBlockV2 publishes a signed block via POST /eth/v2/beacon/blocks.
// The version header is required. The body should be a JSON-encoded signed block.
func (b *BeaconNodeClient) PublishBlockV2(ctx context.Context, version string, jsonBlock []byte) error {
	u := b.c.BaseURL().ResolveReference(&url.URL{Path: "/eth/v2/beacon/blocks"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(jsonBlock))
	if err != nil {
		return errors.Wrap(err, "failed to create POST request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Eth-Consensus-Version", version)
	resp, err := b.c.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, client.MaxBodySize))
		return fmt.Errorf("POST block returned status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ComputeDomainData computes the BLS signature domain for a given domain type
// and epoch entirely client-side, without requiring a gRPC call. This
// replicates the server-side DomainData RPC logic, including the special
// handling for voluntary exits post-Deneb (which must use the Capella fork
// version per EIP-7044).
func ComputeDomainData(epoch primitives.Epoch, domainType [4]byte) ([]byte, error) {
	cfg := params.BeaconConfig()

	var fork *eth.Fork
	if bytes.Equal(domainType[:], cfg.DomainVoluntaryExit[:]) && epoch >= cfg.DenebForkEpoch {
		// EIP-7044: voluntary exits are signed with the Capella fork version
		// regardless of the current fork, starting from Deneb.
		fork = &eth.Fork{
			PreviousVersion: cfg.CapellaForkVersion,
			CurrentVersion:  cfg.CapellaForkVersion,
			Epoch:           cfg.CapellaForkEpoch,
		}
	} else {
		fork = params.ForkFromConfig(cfg, epoch)
	}

	gvr := genesis.ValidatorsRoot()
	return signing.Domain(fork, epoch, domainType, gvr[:])
}

// GetCommittees returns the committees for a given state and optional slot/epoch filter.
func (b *BeaconNodeClient) GetCommittees(ctx context.Context, stateID string, slot primitives.Slot) (*structs.GetCommitteesResponse, error) {
	endpoint := fmt.Sprintf("/eth/v1/beacon/states/%s/committees?slot=%d", stateID, slot)
	body, err := b.c.Get(ctx, endpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get committees for state %s slot %d", stateID, slot)
	}
	resp := &structs.GetCommitteesResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal committees response")
	}
	return resp, nil
}
