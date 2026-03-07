package node_client_factory

import (
	beaconApi "github.com/OffchainLabs/prysm/v7/validator/client/beacon-api"
	"github.com/OffchainLabs/prysm/v7/validator/client/iface"
	validatorHelpers "github.com/OffchainLabs/prysm/v7/validator/helpers"
)

func NewNodeClient(validatorConn validatorHelpers.NodeConnection) iface.NodeClient {
	// REST is the default transport. The WithFallback wrapper is preserved
	// so unimplemented REST methods can fall back to a secondary client.
	// Pass nil for now — no gRPC fallback.
	return beaconApi.NewNodeClientWithFallback(validatorConn.GetRestHandler(), nil)
}
