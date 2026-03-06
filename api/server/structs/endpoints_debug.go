package structs

import (
	"encoding/json"
)

type GetBeaconStateV2Response struct {
	Version             string          `json:"version"`
	ExecutionOptimistic bool            `json:"execution_optimistic"`
	Finalized           bool            `json:"finalized"`
	Data                json.RawMessage `json:"data"` // represents the state values based on the version
}

type GetForkChoiceHeadsV2Response struct {
	Data []*ForkChoiceHead `json:"data"`
}

type ForkChoiceHead struct {
	Root                string `json:"root"`
	Slot                string `json:"slot"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
}

type GetForkChoiceDumpResponse struct {
	JustifiedCheckpoint *Checkpoint              `json:"justified_checkpoint"`
	FinalizedCheckpoint *Checkpoint              `json:"finalized_checkpoint"`
	ForkChoiceNodes     []*ForkChoiceNode        `json:"fork_choice_nodes"`
	ExtraData           *ForkChoiceDumpExtraData `json:"extra_data"`
}

type ForkChoiceDumpExtraData struct {
	UnrealizedJustifiedCheckpoint *Checkpoint `json:"unrealized_justified_checkpoint"`
	UnrealizedFinalizedCheckpoint *Checkpoint `json:"unrealized_finalized_checkpoint"`
	ProposerBoostRoot             string      `json:"proposer_boost_root"`
	PreviousProposerBoostRoot     string      `json:"previous_proposer_boost_root"`
	HeadRoot                      string      `json:"head_root"`
}

type ForkChoiceNode struct {
	Slot               string                   `json:"slot"`
	BlockRoot          string                   `json:"block_root"`
	ParentRoot         string                   `json:"parent_root"`
	JustifiedEpoch     string                   `json:"justified_epoch"`
	FinalizedEpoch     string                   `json:"finalized_epoch"`
	Weight             string                   `json:"weight"`
	Validity           string                   `json:"validity"`
	ExecutionBlockHash string                   `json:"execution_block_hash"`
	ExtraData          *ForkChoiceNodeExtraData `json:"extra_data"`
}

type ForkChoiceNodeExtraData struct {
	UnrealizedJustifiedEpoch string `json:"unrealized_justified_epoch"`
	UnrealizedFinalizedEpoch string `json:"unrealized_finalized_epoch"`
	Balance                  string `json:"balance"`
	ExecutionOptimistic      bool   `json:"execution_optimistic"`
	TimeStamp                string `json:"timestamp"`
	Target                   string `json:"target"`
}

type GetDebugDataColumnSidecarsResponse struct {
	Version             string               `json:"version"`
	ExecutionOptimistic bool                 `json:"execution_optimistic"`
	Finalized           bool                 `json:"finalized"`
	Data                []*DataColumnSidecar `json:"data"`
}

type DataColumnSidecar struct {
	Index                        string                   `json:"index"`
	Column                       []string                 `json:"column"`
	KzgCommitments               []string                 `json:"kzg_commitments"`
	KzgProofs                    []string                 `json:"kzg_proofs"`
	SignedBeaconBlockHeader      *SignedBeaconBlockHeader `json:"signed_block_header"`
	KzgCommitmentsInclusionProof []string                 `json:"kzg_commitments_inclusion_proof"`
}

type DebugPeersResponse struct {
	Data []*DebugPeer `json:"data"`
}

type DebugPeer struct {
	ListeningAddresses []string         `json:"listening_addresses"`
	Direction          string           `json:"direction"`
	ConnectionState    string           `json:"connection_state"`
	PeerId             string           `json:"peer_id"`
	Enr                string           `json:"enr"`
	PeerInfo           *DebugPeerInfo   `json:"peer_info"`
	PeerStatus         *DebugPeerStatus `json:"peer_status"`
	LastUpdated        string           `json:"last_updated"`
	ScoreInfo          *ScoreInfo       `json:"score_info"`
}

type DebugPeerInfo struct {
	MetadataV0      *DebugPeerMetadataV0 `json:"metadata_v0,omitempty"`
	MetadataV1      *DebugPeerMetadataV1 `json:"metadata_v1,omitempty"`
	MetadataV2      *DebugPeerMetadataV2 `json:"metadata_v2,omitempty"`
	Protocols       []string             `json:"protocols"`
	FaultCount      string               `json:"fault_count"`
	ProtocolVersion string               `json:"protocol_version"`
	AgentVersion    string               `json:"agent_version"`
	PeerLatency     string               `json:"peer_latency"`
}

type DebugPeerMetadataV0 struct {
	SeqNumber string `json:"seq_number"`
	Attnets   string `json:"attnets"`
}

type DebugPeerMetadataV1 struct {
	SeqNumber string `json:"seq_number"`
	Attnets   string `json:"attnets"`
	Syncnets  string `json:"syncnets"`
}

type DebugPeerMetadataV2 struct {
	SeqNumber string `json:"seq_number"`
	Attnets   string `json:"attnets"`
	Syncnets  string `json:"syncnets"`
	Custnets  string `json:"custnets"`
}

type DebugPeerStatus struct {
	ForkDigest     string `json:"fork_digest"`
	FinalizedRoot  string `json:"finalized_root"`
	FinalizedEpoch string `json:"finalized_epoch"`
	HeadRoot       string `json:"head_root"`
	HeadSlot       string `json:"head_slot"`
}

type ScoreInfo struct {
	TopicScores        map[string]*TopicScoreSnapshot `json:"topic_scores"`
	ProcessedBlocks    string                         `json:"processed_blocks"`
	ValidationError    string                         `json:"validation_error"`
	OverallScore       float32                        `json:"overall_score"`
	BlockProviderScore float32                        `json:"block_provider_score"`
	GossipScore        float32                        `json:"gossip_score"`
	BehaviourPenalty   float32                        `json:"behaviour_penalty"`
}

type TopicScoreSnapshot struct {
	TimeInMesh               string  `json:"time_in_mesh_seconds"`
	FirstMessageDeliveries   float32 `json:"first_message_deliveries"`
	MeshMessageDeliveries    float32 `json:"mesh_message_deliveries"`
	InvalidMessageDeliveries float32 `json:"invalid_message_deliveries"`
}
