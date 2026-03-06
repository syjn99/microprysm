package debug

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OffchainLabs/prysm/v7/api/server/structs"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/p2p/peers"
	mockp2p "github.com/OffchainLabs/prysm/v7/beacon-chain/p2p/testing"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/assert"
	"github.com/OffchainLabs/prysm/v7/testing/require"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2ptest "github.com/libp2p/go-libp2p/p2p/host/peerstore/test"
	ma "github.com/multiformats/go-multiaddr"
)

func TestListPeers(t *testing.T) {
	peersProvider := &mockp2p.MockPeersProvider{}
	mP2P := mockp2p.NewTestP2P(t)
	s := &Server{
		PeersFetcher: peersProvider,
		PeerManager:  &mockp2p.MockPeerManager{BHost: mP2P.BHost},
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.com/prysm/v1/debug/peers", nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.ListPeers(writer, request)
	require.Equal(t, http.StatusOK, writer.Code)

	resp := &structs.DebugPeersResponse{}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
	// MockPeersProvider creates 2 peers by default.
	require.Equal(t, 2, len(resp.Data))

	// Check that we have one inbound and one outbound.
	hasInbound := false
	hasOutbound := false
	for _, p := range resp.Data {
		if p.Direction == "INBOUND" {
			hasInbound = true
		}
		if p.Direction == "OUTBOUND" {
			hasOutbound = true
		}
		// All mock peers are connected.
		assert.Equal(t, "CONNECTED", p.ConnectionState)
		// Each peer should have a listening address.
		assert.Equal(t, true, len(p.ListeningAddresses) > 0, "Expected peer to have addresses")
		// Peer info should be present.
		require.NotNil(t, p.PeerInfo)
		require.NotNil(t, p.PeerStatus)
		require.NotNil(t, p.ScoreInfo)
		// PeerId should be non-empty.
		assert.NotEqual(t, "", p.PeerId)
	}
	assert.Equal(t, true, hasInbound, "Expected an inbound peer")
	assert.Equal(t, true, hasOutbound, "Expected an outbound peer")
}

func TestListPeers_NoPeers(t *testing.T) {
	peersProvider := &mockp2p.MockPeersProvider{}
	peersProvider.ClearPeers()
	mP2P := mockp2p.NewTestP2P(t)
	s := &Server{
		PeersFetcher: peersProvider,
		PeerManager:  &mockp2p.MockPeerManager{BHost: mP2P.BHost},
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.com/prysm/v1/debug/peers", nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.ListPeers(writer, request)
	require.Equal(t, http.StatusOK, writer.Code)

	resp := &structs.DebugPeersResponse{}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
	assert.Equal(t, 0, len(resp.Data))
}

func TestListPeers_NilMetadata(t *testing.T) {
	peersProvider := &mockp2p.MockPeersProvider{}
	peersProvider.ClearPeers()
	peerStatus := peersProvider.Peers()

	ids := libp2ptest.GeneratePeerIDs(1)
	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/13000")
	require.NoError(t, err)
	peerStatus.Add(nil, ids[0], addr, network.DirInbound)
	peerStatus.SetConnectionState(ids[0], peers.Connected)

	mP2P := mockp2p.NewTestP2P(t)
	s := &Server{
		PeersFetcher: peersProvider,
		PeerManager:  &mockp2p.MockPeerManager{BHost: mP2P.BHost},
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.com/prysm/v1/debug/peers", nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.ListPeers(writer, request)
	require.Equal(t, http.StatusOK, writer.Code)

	resp := &structs.DebugPeersResponse{}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
	require.Equal(t, 1, len(resp.Data))
	// Metadata fields should all be nil when metadata is not set.
	assert.Equal(t, (*structs.DebugPeerMetadataV0)(nil), resp.Data[0].PeerInfo.MetadataV0)
	assert.Equal(t, (*structs.DebugPeerMetadataV1)(nil), resp.Data[0].PeerInfo.MetadataV1)
	assert.Equal(t, (*structs.DebugPeerMetadataV2)(nil), resp.Data[0].PeerInfo.MetadataV2)
}

func TestListPeers_MissingChainState(t *testing.T) {
	peersProvider := &mockp2p.MockPeersProvider{}
	peersProvider.ClearPeers()
	peerStatus := peersProvider.Peers()

	ids := libp2ptest.GeneratePeerIDs(1)
	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/13000")
	require.NoError(t, err)
	peerStatus.Add(nil, ids[0], addr, network.DirOutbound)
	peerStatus.SetConnectionState(ids[0], peers.Connected)
	// Intentionally do NOT set chain state.

	mP2P := mockp2p.NewTestP2P(t)
	s := &Server{
		PeersFetcher: peersProvider,
		PeerManager:  &mockp2p.MockPeerManager{BHost: mP2P.BHost},
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.com/prysm/v1/debug/peers", nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.ListPeers(writer, request)
	require.Equal(t, http.StatusOK, writer.Code)

	resp := &structs.DebugPeersResponse{}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
	require.Equal(t, 1, len(resp.Data))
	// Should have zero-value chain state, not an error.
	assert.Equal(t, "OUTBOUND", resp.Data[0].Direction)
	assert.Equal(t, "0x", resp.Data[0].PeerStatus.ForkDigest)
	assert.Equal(t, "0", resp.Data[0].PeerStatus.FinalizedEpoch)
	assert.Equal(t, "0", resp.Data[0].PeerStatus.HeadSlot)
}

func TestListPeers_ScoreInfo(t *testing.T) {
	peersProvider := &mockp2p.MockPeersProvider{}
	peersProvider.ClearPeers()
	peerStatus := peersProvider.Peers()

	ids := libp2ptest.GeneratePeerIDs(1)
	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/13000")
	require.NoError(t, err)
	peerStatus.Add(nil, ids[0], addr, network.DirInbound)
	peerStatus.SetConnectionState(ids[0], peers.Connected)
	peerStatus.SetChainState(ids[0], &ethpb.StatusV2{
		ForkDigest:     []byte{0x01, 0x02, 0x03, 0x04},
		FinalizedEpoch: 100,
		FinalizedRoot:  make([]byte, 32),
		HeadRoot:       make([]byte, 32),
		HeadSlot:       500,
	})

	mP2P := mockp2p.NewTestP2P(t)
	s := &Server{
		PeersFetcher: peersProvider,
		PeerManager:  &mockp2p.MockPeerManager{BHost: mP2P.BHost},
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.com/prysm/v1/debug/peers", nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.ListPeers(writer, request)
	require.Equal(t, http.StatusOK, writer.Code)

	resp := &structs.DebugPeersResponse{}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
	require.Equal(t, 1, len(resp.Data))

	p := resp.Data[0]
	assert.Equal(t, ids[0].String(), p.PeerId)
	assert.Equal(t, "INBOUND", p.Direction)
	assert.Equal(t, "CONNECTED", p.ConnectionState)
	assert.Equal(t, "0x01020304", p.PeerStatus.ForkDigest)
	assert.Equal(t, "100", p.PeerStatus.FinalizedEpoch)
	assert.Equal(t, "500", p.PeerStatus.HeadSlot)
	// Score info should be present with default values.
	require.NotNil(t, p.ScoreInfo)
	assert.Equal(t, "0", p.ScoreInfo.ProcessedBlocks)

	// Verify JSON shape includes PeerId at correct key.
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &raw))
	var dataArr []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw["data"], &dataArr))
	_, hasPeerId := dataArr[0]["peer_id"]
	_, hasScoreInfo := dataArr[0]["score_info"]
	_, hasPeerInfo := dataArr[0]["peer_info"]
	assert.Equal(t, true, hasPeerId, "JSON should have peer_id key")
	assert.Equal(t, true, hasScoreInfo, "JSON should have score_info key")
	assert.Equal(t, true, hasPeerInfo, "JSON should have peer_info key")
}

// Helpers used by other tests in this package (from handlers_test.go).
// These are defined to ensure test compatibility.
var _ peer.ID // ensure peer import is used
