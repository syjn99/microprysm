package helpers

import (
	"github.com/OffchainLabs/prysm/v7/api/rest"
	"github.com/pkg/errors"
)

// NodeConnection provides access to a REST API connection to a beacon node.
type NodeConnection interface {
	// GetRestConnectionProvider returns the REST connection provider.
	GetRestConnectionProvider() rest.RestConnectionProvider
	// GetRestHandler returns the REST handler for making API requests.
	// Returns nil if no REST provider is configured.
	GetRestHandler() rest.Handler
}

type nodeConnection struct {
	restConnectionProvider rest.RestConnectionProvider
}

func (c *nodeConnection) GetRestConnectionProvider() rest.RestConnectionProvider {
	return c.restConnectionProvider
}

func (c *nodeConnection) GetRestHandler() rest.Handler {
	if c.restConnectionProvider == nil {
		return nil
	}
	return c.restConnectionProvider.Handler()
}

// NodeConnectionOption is a functional option for configuring a NodeConnection.
type NodeConnectionOption func(*nodeConnection) error

// WithREST configures a REST connection provider for the NodeConnection.
// If endpoint is empty, this option is a no-op.
func WithREST(endpoint string, opts ...rest.RestConnectionProviderOption) NodeConnectionOption {
	return func(c *nodeConnection) error {
		if endpoint == "" {
			return nil
		}
		provider, err := rest.NewRestConnectionProvider(endpoint, opts...)
		if err != nil {
			return errors.Wrap(err, "failed to create REST connection provider")
		}
		c.restConnectionProvider = provider
		return nil
	}
}

// WithRestProvider sets a pre-built REST connection provider.
func WithRestProvider(provider rest.RestConnectionProvider) NodeConnectionOption {
	return func(c *nodeConnection) error {
		c.restConnectionProvider = provider
		return nil
	}
}

// NewNodeConnection creates a new NodeConnection with the given options.
// A REST provider must be configured via options.
// Returns an error if no provider is configured.
func NewNodeConnection(opts ...NodeConnectionOption) (NodeConnection, error) {
	c := &nodeConnection{}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	if c.restConnectionProvider == nil {
		return nil, errors.New("beacon node REST API endpoint must be provided (--beacon-rest-api-provider)")
	}

	return c, nil
}
