package helpers

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/api/rest"
	"github.com/OffchainLabs/prysm/v7/testing/assert"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func TestNewNodeConnection(t *testing.T) {
	t.Run("with rest provider", func(t *testing.T) {
		restProvider := &rest.MockRestProvider{MockHosts: []string{"http://localhost:3500"}}
		conn, err := NewNodeConnection(WithRestProvider(restProvider))
		require.NoError(t, err)

		assert.Equal(t, restProvider, conn.GetRestConnectionProvider())
	})

	t.Run("with no providers returns error", func(t *testing.T) {
		conn, err := NewNodeConnection()
		require.ErrorContains(t, "beacon node REST API endpoint must be provided", err)
		assert.Equal(t, (NodeConnection)(nil), conn)
	})

	t.Run("with empty endpoint is no-op", func(t *testing.T) {
		conn, err := NewNodeConnection(WithREST(""))
		require.ErrorContains(t, "beacon node REST API endpoint must be provided", err)
		assert.Equal(t, (NodeConnection)(nil), conn)
	})
}

func TestNodeConnection_GetRestHandler(t *testing.T) {
	t.Run("delegates to provider", func(t *testing.T) {
		mockHandler := &rest.MockHandler{}
		restProvider := &rest.MockRestProvider{MockHandler: mockHandler, MockHosts: []string{"http://localhost:3500"}}
		conn, err := NewNodeConnection(WithRestProvider(restProvider))
		require.NoError(t, err)

		assert.Equal(t, mockHandler, conn.GetRestHandler())
	})
}
