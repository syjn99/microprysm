package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/OffchainLabs/prysm/v7/api"
	"github.com/OffchainLabs/prysm/v7/monitoring/tracing/trace"
	"github.com/OffchainLabs/prysm/v7/network/httputil"
	pb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetVersion returns the beacon node and validator client versions
func (s *Server) GetVersion(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.web.health.Version")
	defer span.End()

	beacon, err := s.nodeClient.Version(ctx, &emptypb.Empty{})
	if err != nil {
		httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	httputil.WriteJson(w, struct {
		Beacon    string `json:"beacon"`
		Validator string `json:"validator"`
	}{
		Beacon:    beacon.Version,
		Validator: version.Version(),
	})
}

// StreamBeaconLogs from the beacon node via server-side events.
// TODO: Re-implement using REST/SSE once the beacon node exposes a log streaming HTTP endpoint.
func (s *Server) StreamBeaconLogs(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "validator.web.health.StreamBeaconLogs")
	defer span.End()
	httputil.HandleError(w, "beacon log streaming is not yet available via REST API", http.StatusNotImplemented)
}

// StreamValidatorLogs from the validator client via server-side events.
func (s *Server) StreamValidatorLogs(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.web.health.StreamValidatorLogs")
	defer span.End()

	// Ensure that the writer supports flushing.
	flusher, ok := w.(http.Flusher)
	if !ok {
		httputil.HandleError(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	ch := make(chan []byte, s.logStreamerBufferSize)
	sub := s.logStreamer.LogsFeed().Subscribe(ch)
	defer func() {
		sub.Unsubscribe()
		close(ch)
	}()
	// Set up SSE response headers
	w.Header().Set("Content-Type", api.EventStreamMediaType)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", api.KeepAlive)

	recentLogs := s.logStreamer.GetLastFewLogs()
	logStrings := make([]string, len(recentLogs))
	for i, l := range recentLogs {
		logStrings[i] = string(l)
	}
	ls := &pb.LogsResponse{
		Logs: logStrings,
	}
	jsonLogs, err := json.Marshal(ls)
	if err != nil {
		httputil.HandleError(w, "Failed to marshal logs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = fmt.Fprintf(w, "%s\n", jsonLogs)
	if err != nil {
		httputil.HandleError(w, "Error sending data: "+err.Error(), http.StatusInternalServerError)
		return
	}
	flusher.Flush()

	for {
		select {
		case log := <-ch:
			// Set up SSE response headers
			ls = &pb.LogsResponse{
				Logs: []string{string(log)},
			}
			jsonLogs, err = json.Marshal(ls)
			if err != nil {
				httputil.HandleError(w, "Failed to marshal logs: "+err.Error(), http.StatusInternalServerError)
				return
			}
			_, err = fmt.Fprintf(w, "%s\n", jsonLogs)
			if err != nil {
				httputil.HandleError(w, "Error sending data: "+err.Error(), http.StatusInternalServerError)
				return
			}

			flusher.Flush()
		case <-s.ctx.Done():
			return
		case err := <-sub.Err():
			httputil.HandleError(w, "Subscriber error: "+err.Error(), http.StatusInternalServerError)
			return
		case <-ctx.Done():
			return
		}
	}
}
