package proposer

import (
	"context"

	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errOptimisticMode = errors.New("the node is currently optimistic and cannot serve validators")

func (vs *Server) optimisticStatus(ctx context.Context) error {
	if slots.ToEpoch(vs.TimeFetcher.CurrentSlot()) < params.BeaconConfig().BellatrixForkEpoch {
		return nil
	}
	optimistic, err := vs.OptimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "Could not determine if the node is a optimistic node: %v", err)
	}
	if !optimistic {
		return nil
	}

	return status.Errorf(codes.Unavailable, "error=%v", errOptimisticMode)
}

// ValidatorIndex returns the validator index for a given public key.
func (vs *Server) ValidatorIndex(ctx context.Context, req *ethpb.ValidatorIndexRequest) (*ethpb.ValidatorIndexResponse, error) {
	st, err := vs.HeadFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine head state: %v", err)
	}
	if st == nil || st.IsNil() {
		return nil, status.Errorf(codes.Internal, "head state is empty")
	}
	index, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(req.PublicKey))
	if !ok {
		return nil, status.Errorf(codes.NotFound, "Could not find validator index for public key %#x", req.PublicKey)
	}

	return &ethpb.ValidatorIndexResponse{Index: index}, nil
}
