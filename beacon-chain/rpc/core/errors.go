package core

import (
	"net/http"
)

type ErrorReason uint8

const (
	Internal = iota
	Unavailable
	BadRequest
	NotFound
	// Add more errors as needed
)

type RpcError struct {
	Err    error
	Reason ErrorReason
}

func ErrorReasonToHTTP(reason ErrorReason) int {
	switch reason {
	case Internal:
		return http.StatusInternalServerError
	case Unavailable:
		return http.StatusServiceUnavailable
	case BadRequest:
		return http.StatusBadRequest
	case NotFound:
		return http.StatusNotFound
	// Add more cases for other error reasons as needed
	default:
		return http.StatusInternalServerError
	}
}
