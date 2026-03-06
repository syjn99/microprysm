package evaluators

import (
	"context"
	"strconv"

	"github.com/OffchainLabs/prysm/v7/testing/endtoend/helpers"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/policies"
	"github.com/OffchainLabs/prysm/v7/testing/endtoend/types"
	"github.com/pkg/errors"
)

// PeersCheck performs a check on peer data to ensure that any connected peers
// are not publishing invalid data.
var PeersCheck = types.Evaluator{
	Name:       "peers_check_epoch_%d",
	Policy:     policies.AfterNthEpoch(0),
	Evaluation: peersTest,
}

func peersTest(_ *types.EvaluationContext, nodeURLs ...string) error {
	client, err := helpers.NewBeaconNodeClient(nodeURLs[0])
	if err != nil {
		return err
	}

	peerResponses, err := client.ListDebugPeers(context.Background())
	if err != nil {
		return err
	}
	baseErr := error(nil)
	for _, res := range peerResponses.Data {
		if res.ScoreInfo == nil {
			continue
		}
		if res.ScoreInfo.GossipScore < 0 {
			baseErr = wrapError(baseErr, "Gossip score for peer %s is %f and negative.", res.PeerId, res.ScoreInfo.GossipScore)
		}
		if res.ScoreInfo.BehaviourPenalty > 0 {
			baseErr = wrapError(baseErr, "Behaviour penalty for peer %s is %f and larger than zero.", res.PeerId, res.ScoreInfo.BehaviourPenalty)
		}
		if res.ScoreInfo.BlockProviderScore < 0 {
			baseErr = wrapError(baseErr, "Block provider score for peer %s is %f and negative.", res.PeerId, res.ScoreInfo.BlockProviderScore)
		}
		if res.ScoreInfo.OverallScore < 0 {
			baseErr = wrapError(baseErr, "Overall score for peer %s is %f and negative.", res.PeerId, res.ScoreInfo.OverallScore)
		}
		if res.ScoreInfo.ValidationError != "" {
			baseErr = wrapError(baseErr, "Peer %s has a validation error: %s", res.PeerId, res.ScoreInfo.ValidationError)
		}
		if res.PeerInfo != nil {
			faultCount, err := strconv.ParseUint(res.PeerInfo.FaultCount, 10, 64)
			if err == nil && faultCount > 0 {
				baseErr = wrapError(baseErr, "Peer %s has a non zero fault count: %d", res.PeerId, faultCount)
			}
		}
		for topic, snap := range res.ScoreInfo.TopicScores {
			if snap.InvalidMessageDeliveries > 0 {
				baseErr = wrapError(baseErr, "Peer %s in Topic %s has sent invalid deliveries: %f", res.PeerId, topic, snap.InvalidMessageDeliveries)
			}
		}
	}
	return baseErr
}

func wrapError(err error, format string, args ...any) error {
	if err == nil {
		err = errors.New("")
	}
	return errors.Wrapf(err, format, args...)
}
