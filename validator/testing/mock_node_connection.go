package testing

import (
	"github.com/OffchainLabs/prysm/v7/api/rest"
	"github.com/OffchainLabs/prysm/v7/validator/helpers"
)

// MockNodeConnection creates a minimal NodeConnection for testing.
func MockNodeConnection() helpers.NodeConnection {
	conn, _ := helpers.NewNodeConnection(
		helpers.WithRestProvider(&rest.MockRestProvider{
			MockHosts: []string{"http://mock:3500"},
		}),
	)
	return conn
}
