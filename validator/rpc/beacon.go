package rpc

import (
	"github.com/OffchainLabs/prysm/v7/api/rest"
	beaconChainClientFactory "github.com/OffchainLabs/prysm/v7/validator/client/beacon-chain-client-factory"
	nodeClientFactory "github.com/OffchainLabs/prysm/v7/validator/client/node-client-factory"
	validatorClientFactory "github.com/OffchainLabs/prysm/v7/validator/client/validator-client-factory"
	validatorHelpers "github.com/OffchainLabs/prysm/v7/validator/helpers"
)

// Initialize a client connection to a beacon node REST endpoint.
func (s *Server) registerBeaconClient() error {
	conn, err := validatorHelpers.NewNodeConnection(
		validatorHelpers.WithREST(s.beaconApiEndpoint,
			rest.WithHttpHeaders(s.beaconApiHeaders),
			rest.WithHttpTimeout(s.beaconApiTimeout),
			rest.WithTracing(),
		),
	)
	if err != nil {
		return err
	}

	s.chainClient = beaconChainClientFactory.NewChainClient(conn)
	s.nodeClient = nodeClientFactory.NewNodeClient(conn)
	s.beaconNodeValidatorClient = validatorClientFactory.NewValidatorClient(conn)
	return nil
}
