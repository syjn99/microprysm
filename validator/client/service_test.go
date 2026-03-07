package client

import (
	"context"
	"testing"
	"time"

	"github.com/OffchainLabs/prysm/v7/runtime"
	"github.com/OffchainLabs/prysm/v7/testing/assert"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

var _ runtime.Service = (*ValidatorService)(nil)

func TestStop_CancelsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	vs := &ValidatorService{
		ctx:    ctx,
		cancel: cancel,
	}

	assert.NoError(t, vs.Stop())

	select {
	case <-time.After(1 * time.Second):
		t.Error("Context not canceled within 1s")
	case <-vs.ctx.Done():
	}
}

func TestNew_WithRESTEndpoint(t *testing.T) {
	_, err := NewValidatorService(t.Context(), &Config{
		BeaconApiEndpoint: "http://localhost:3500",
	})
	require.NoError(t, err)
}

func TestStatus_NoConnectionError(t *testing.T) {
	validatorService := &ValidatorService{}
	assert.ErrorContains(t, "no connection", validatorService.Status())
}
