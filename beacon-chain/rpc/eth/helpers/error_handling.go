package helpers

import (
	"errors"
	"net/http"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/rpc/eth/shared"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/rpc/lookup"
	"github.com/OffchainLabs/prysm/v7/network/httputil"
)

// HandleIsOptimisticError handles errors from IsOptimistic function calls and writes appropriate HTTP responses.
func HandleIsOptimisticError(w http.ResponseWriter, err error) {
	var fetchErr *lookup.FetchStateError
	if errors.As(err, &fetchErr) {
		shared.WriteStateFetchError(w, err)
		return
	}

	var blockNotFoundErr *lookup.BlockNotFoundError
	if errors.As(err, &blockNotFoundErr) {
		httputil.HandleError(w, "Could not check optimistic status: "+err.Error(), http.StatusNotFound)
		return
	}
	httputil.HandleError(w, "Could not check optimistic status: "+err.Error(), http.StatusInternalServerError)
}

// IndexedVerificationFailure represents a collection of verification failures.
type IndexedVerificationFailure struct {
	Failures []*SingleIndexedVerificationFailure `json:"failures"`
}

// SingleIndexedVerificationFailure represents an issue when verifying a single indexed object e.g. an item in an array.
type SingleIndexedVerificationFailure struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}
