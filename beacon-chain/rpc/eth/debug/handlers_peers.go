package debug

import (
	"net/http"
	"strconv"

	"github.com/OffchainLabs/prysm/v7/api/server/structs"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/p2p"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/p2p/peers"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/libp2p/go-libp2p/core/network"
	libpeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/OffchainLabs/prysm/v7/monitoring/tracing/trace"
	"github.com/OffchainLabs/prysm/v7/network/httputil"
)

// ListPeers returns all peers known to the host node with detailed debug information
// including scores, gossip info, fault counts, and topic scores.
func (s *Server) ListPeers(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "debug.ListPeers")
	defer span.End()

	peerStatus := s.PeersFetcher.Peers()
	ps := s.PeerManager.Host().Peerstore()
	allIds := peerStatus.All()
	debugPeers := make([]*structs.DebugPeer, 0, len(allIds))

	for _, pid := range allIds {
		dp, err := debugPeerInfo(pid, peerStatus, ps)
		if err != nil {
			httputil.HandleError(w, "Could not get peer info: "+err.Error(), http.StatusInternalServerError)
			return
		}
		debugPeers = append(debugPeers, dp)
	}

	httputil.WriteJson(w, &structs.DebugPeersResponse{Data: debugPeers})
}

func debugPeerInfo(pid libpeer.ID, peerStatus *peers.Status, ps peerstore.Peerstore) (*structs.DebugPeer, error) {
	addr, err := peerStatus.Address(pid)
	if err != nil {
		return nil, err
	}
	dir, err := peerStatus.Direction(pid)
	if err != nil {
		return nil, err
	}
	dirStr := "UNKNOWN"
	switch dir {
	case network.DirInbound:
		dirStr = "INBOUND"
	case network.DirOutbound:
		dirStr = "OUTBOUND"
	}

	connState, err := peerStatus.ConnectionState(pid)
	if err != nil {
		return nil, err
	}
	connStateStr := ethpb.ConnectionState_name[int32(connState)]

	record, err := peerStatus.ENR(pid)
	if err != nil {
		return nil, err
	}
	enr := ""
	if record != nil {
		enr, err = p2p.SerializeENR(record)
		if err != nil {
			return nil, err
		}
	}

	metadata, err := peerStatus.Metadata(pid)
	if err != nil {
		return nil, err
	}

	protocols, err := ps.GetProtocols(pid)
	if err != nil {
		return nil, err
	}

	faultCount, err := peerStatus.Scorers().BadResponsesScorer().Count(pid)
	if err != nil {
		return nil, err
	}

	rawPversion, err := ps.Get(pid, "ProtocolVersion")
	pVersion, ok := rawPversion.(string)
	if err != nil || !ok {
		pVersion = ""
	}
	rawAversion, err := ps.Get(pid, "AgentVersion")
	aVersion, ok := rawAversion.(string)
	if err != nil || !ok {
		aVersion = ""
	}

	peerInfo := &structs.DebugPeerInfo{
		Protocols:       protocol.ConvertToStrings(protocols),
		FaultCount:      strconv.FormatUint(uint64(faultCount), 10),
		ProtocolVersion: pVersion,
		AgentVersion:    aVersion,
		PeerLatency:     strconv.FormatUint(uint64(ps.LatencyEWMA(pid).Milliseconds()), 10),
	}
	if metadata != nil && !metadata.IsNil() {
		switch {
		case metadata.MetadataObjV0() != nil:
			v0 := metadata.MetadataObjV0()
			peerInfo.MetadataV0 = &structs.DebugPeerMetadataV0{
				SeqNumber: strconv.FormatUint(v0.SeqNumber, 10),
				Attnets:   hexutil.Encode(v0.Attnets),
			}
		case metadata.MetadataObjV1() != nil:
			v1 := metadata.MetadataObjV1()
			peerInfo.MetadataV1 = &structs.DebugPeerMetadataV1{
				SeqNumber: strconv.FormatUint(v1.SeqNumber, 10),
				Attnets:   hexutil.Encode(v1.Attnets),
				Syncnets:  hexutil.Encode(v1.Syncnets),
			}
		case metadata.MetadataObjV2() != nil:
			v2 := metadata.MetadataObjV2()
			peerInfo.MetadataV2 = &structs.DebugPeerMetadataV2{
				SeqNumber: strconv.FormatUint(v2.SeqNumber, 10),
				Attnets:   hexutil.Encode(v2.Attnets),
				Syncnets:  hexutil.Encode(v2.Syncnets),
				Custnets:  strconv.FormatUint(v2.CustodyGroupCount, 10),
			}
		}
	}

	addresses := ps.Addrs(pid)
	var stringAddrs []string
	if addr != nil {
		stringAddrs = append(stringAddrs, addr.String())
	}
	for _, a := range addresses {
		if addr != nil && addr.String() == a.String() {
			continue
		}
		stringAddrs = append(stringAddrs, a.String())
	}

	pStatus, err := peerStatus.ChainState(pid)
	if err != nil {
		pStatus = new(ethpb.StatusV2)
	}
	lastUpdated, err := peerStatus.ChainStateLastUpdated(pid)
	if err != nil {
		return nil, err
	}
	unixTime := uint64(0)
	if !lastUpdated.IsZero() {
		unixTime = uint64(lastUpdated.Unix())
	}

	gScore, bPenalty, topicMaps, err := peerStatus.Scorers().GossipScorer().GossipData(pid)
	if err != nil {
		return nil, err
	}
	topicScores := make(map[string]*structs.TopicScoreSnapshot, len(topicMaps))
	for k, v := range topicMaps {
		topicScores[k] = &structs.TopicScoreSnapshot{
			TimeInMesh:               strconv.FormatUint(v.TimeInMesh, 10),
			FirstMessageDeliveries:   v.FirstMessageDeliveries,
			MeshMessageDeliveries:    v.MeshMessageDeliveries,
			InvalidMessageDeliveries: v.InvalidMessageDeliveries,
		}
	}

	scoreInfo := &structs.ScoreInfo{
		OverallScore:       float32(peerStatus.Scorers().Score(pid)),
		ProcessedBlocks:    strconv.FormatUint(peerStatus.Scorers().BlockProviderScorer().ProcessedBlocks(pid), 10),
		BlockProviderScore: float32(peerStatus.Scorers().BlockProviderScorer().Score(pid)),
		TopicScores:        topicScores,
		GossipScore:        float32(gScore),
		BehaviourPenalty:   float32(bPenalty),
		ValidationError:    errorToString(peerStatus.Scorers().ValidationError(pid)),
	}

	debugPeerStatus := &structs.DebugPeerStatus{
		ForkDigest:     hexutil.Encode(pStatus.ForkDigest),
		FinalizedRoot:  hexutil.Encode(pStatus.FinalizedRoot),
		FinalizedEpoch: strconv.FormatUint(uint64(pStatus.FinalizedEpoch), 10),
		HeadRoot:       hexutil.Encode(pStatus.HeadRoot),
		HeadSlot:       strconv.FormatUint(uint64(pStatus.HeadSlot), 10),
	}

	return &structs.DebugPeer{
		ListeningAddresses: stringAddrs,
		Direction:          dirStr,
		ConnectionState:    connStateStr,
		PeerId:             pid.String(),
		Enr:                enr,
		PeerInfo:           peerInfo,
		PeerStatus:         debugPeerStatus,
		LastUpdated:        strconv.FormatUint(unixTime, 10),
		ScoreInfo:          scoreInfo,
	}, nil
}

func errorToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
