package p2p

import (
	"context"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/p2p"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	pb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/sirupsen/logrus"
)

var responseCodeSuccess = byte(0x00)

func (c *client) registerHandshakeHandlers() {
	c.registerRPCHandler(p2p.RPCPingTopicV1, c.pingHandler)
	c.registerRPCHandler(p2p.RPCStatusTopicV1, c.statusRPCHandler)
	c.registerRPCHandler(p2p.RPCGoodByeTopicV1, c.goodbyeHandler)
}

// pingHandler reads the incoming ping rpc message from the peer.
func (c *client) pingHandler(_ context.Context, _ any, stream libp2pcore.Stream) error {
	defer closeStream(stream)
	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		return err
	}
	sq := primitives.SSZUint64(c.MetadataSeq())
	if _, err := c.Encoding().EncodeWithMaxLength(stream, &sq); err != nil {
		return err
	}
	return nil
}

func (c *client) goodbyeHandler(_ context.Context, _ any, _ libp2pcore.Stream) error {
	return nil
}

// statusRPCHandler reads the incoming Status RPC from the peer and responds with our version of a status message.
// This handler will disconnect any peer that does not match our fork version.
func (c *client) statusRPCHandler(ctx context.Context, _ any, stream libp2pcore.Stream) error {
	defer closeStream(stream)
	head, err := c.getChainHead(ctx)
	if err != nil {
		return err
	}
	genesis, err := c.getGenesis(ctx)
	if err != nil {
		return err
	}
	currentSlot := slots.CurrentSlot(genesis.GenesisTime)
	currentEpoch := slots.ToEpoch(currentSlot)
	digest := params.ForkDigest(currentEpoch)
	kindOfFork, err := params.Fork(slots.ToEpoch(head.HeadSlot))
	if err != nil {
		return err
	}
	log.WithFields(logrus.Fields{
		"genesisTime":  genesis.GenesisTime,
		"forkDigest":   digest,
		"currentFork":  kindOfFork.CurrentVersion,
		"previousFork": kindOfFork.PreviousVersion,
	}).Info("Responding to status RPC handler")
	status := &pb.Status{
		ForkDigest:     digest[:],
		FinalizedRoot:  head.FinalizedRoot,
		FinalizedEpoch: head.FinalizedEpoch,
		HeadRoot:       head.HeadRoot,
		HeadSlot:       head.HeadSlot,
	}

	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		log.WithError(err).Debug("Could not write to stream")
		return err
	}
	_, err = c.Encoding().EncodeWithMaxLength(stream, status)
	return err
}
