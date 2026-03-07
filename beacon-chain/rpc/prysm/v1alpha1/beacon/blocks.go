package beacon

import (
	"context"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/rpc/core"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Deprecated: The gRPC API will remain the default and fully supported through v8 (expected in 2026) but will be eventually removed in favor of REST API.
//
// GetChainHead retrieves information about the head of the beacon chain from
// the view of the beacon chain node.
//
// This includes the head block slot and root as well as information about
// the most recent finalized and justified slots.
func (bs *Server) GetChainHead(ctx context.Context, _ *emptypb.Empty) (*ethpb.ChainHead, error) {
	ch, err := bs.CoreService.ChainHead(ctx)
	if err != nil {
		return nil, status.Errorf(core.ErrorReasonToGRPC(err.Reason), "Could not retrieve chain head: %v", err.Err)
	}
	return ch, nil
}
