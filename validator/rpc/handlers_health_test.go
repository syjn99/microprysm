package rpc

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/OffchainLabs/prysm/v7/api"
	"github.com/OffchainLabs/prysm/v7/io/logs/mock"
	pb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/require"
	validatormock "github.com/OffchainLabs/prysm/v7/testing/validator-mock"
	"go.uber.org/mock/gomock"
)

type flushableResponseRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flushableResponseRecorder) Flush() {
	f.flushed = true
}

func TestStreamBeaconLogs_NotImplemented(t *testing.T) {
	s := Server{
		ctx: t.Context(),
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v2/validator/health/logs/beacon/stream", nil)
	s.StreamBeaconLogs(w, r)
	resp := w.Result()
	require.Equal(t, http.StatusNotImplemented, resp.StatusCode)
}

func TestStreamValidatorLogs(t *testing.T) {
	ctx := t.Context()
	mockLogs := [][]byte{
		[]byte("[2023-10-31 10:00:00] INFO: Starting server..."),
		[]byte("[2023-10-31 10:01:23] DEBUG: Database connection established."),
		[]byte("[2023-10-31 10:05:45] WARN: High memory usage detected."),
		[]byte("[2023-10-31 10:10:12] INFO: New user registered: user123."),
		[]byte("[2023-10-31 10:15:30] ERROR: Failed to send email."),
	}
	logStreamer := mock.NewMockStreamer(mockLogs)
	// Setting up the mock in the server struct
	s := Server{
		ctx:                   ctx,
		logStreamer:           logStreamer,
		logStreamerBufferSize: 100,
	}

	w := &flushableResponseRecorder{
		ResponseRecorder: httptest.NewRecorder(),
	}
	r := httptest.NewRequest("GET", "/v2/validator/health/logs/validator/stream", nil)
	go func() {
		s.StreamValidatorLogs(w, r)
	}()
	// wait for initiation of StreamValidatorLogs
	time.Sleep(100 * time.Millisecond)
	logStreamer.LogsFeed().Send([]byte("Some mock event data"))
	// wait for feed
	time.Sleep(100 * time.Millisecond)
	s.ctx.Done()
	// Assert the results
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK but got %v", resp.StatusCode)
	}
	ct, ok := resp.Header["Content-Type"]
	require.Equal(t, ok, true)
	require.Equal(t, ct[0], api.EventStreamMediaType)
	cn, ok := resp.Header["Connection"]
	require.Equal(t, ok, true)
	require.Equal(t, cn[0], api.KeepAlive)
	// Check if data was written
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotNil(t, body)

	require.StringContains(t, `{"logs":["[2023-10-31 10:00:00] INFO: Starting server...","[2023-10-31 10:01:23] DEBUG: Database connection established.",`+
		`"[2023-10-31 10:05:45] WARN: High memory usage detected.","[2023-10-31 10:10:12] INFO: New user registered: user123.","[2023-10-31 10:15:30] ERROR: Failed to send email."]}`, string(body))
	require.StringContains(t, `{"logs":["Some mock event data"]}`, string(body))

	// Check if Flush was called
	if !w.flushed {
		t.Fatal("Flush was not called")
	}

}

func TestServer_GetVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := t.Context()
	mockNodeClient := validatormock.NewMockNodeClient(ctrl)
	s := Server{
		ctx:        ctx,
		nodeClient: mockNodeClient,
	}
	mockNodeClient.EXPECT().Version(gomock.Any(), gomock.Any()).Return(&pb.Version{
		Version:  "4.10.1",
		Metadata: "beacon node",
	}, nil)
	r := httptest.NewRequest("GET", "/v2/validator/health/version", nil)
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	s.GetVersion(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK but got %v", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotNil(t, body)
	require.StringContains(t, `{"beacon":"4.10.1","validator":"Prysm/Unknown/Local build. Built at: Moments ago"}`, string(body))
}
