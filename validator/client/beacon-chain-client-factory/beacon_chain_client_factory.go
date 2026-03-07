package beacon_chain_client_factory

import (
	beaconApi "github.com/OffchainLabs/prysm/v7/validator/client/beacon-api"
	"github.com/OffchainLabs/prysm/v7/validator/client/iface"
	nodeClientFactory "github.com/OffchainLabs/prysm/v7/validator/client/node-client-factory"
	validatorHelpers "github.com/OffchainLabs/prysm/v7/validator/helpers"
)

func NewChainClient(validatorConn validatorHelpers.NodeConnection) iface.ChainClient {
	// REST is the default transport. The WithFallback wrapper is preserved
	// so unimplemented REST methods can fall back to a secondary client.
	// Pass nil for now — no gRPC fallback.
	return beaconApi.NewBeaconApiChainClientWithFallback(validatorConn.GetRestHandler(), nil)
}

func NewPrysmChainClient(validatorConn validatorHelpers.NodeConnection) iface.PrysmChainClient {
	return beaconApi.NewPrysmChainClient(validatorConn.GetRestHandler(), nodeClientFactory.NewNodeClient(validatorConn))
}
