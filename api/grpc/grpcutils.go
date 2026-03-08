package grpc

import (
	"context"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// LogRequests logs the gRPC backend as well as request duration when the log level is set to debug
// or higher.
func LogRequests(
	ctx context.Context,
	method string, req,
	reply any,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	// Shortcut when debug logging is not enabled.
	if logrus.GetLevel() < logrus.DebugLevel {
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	var header metadata.MD
	opts = append(
		opts,
		grpc.Header(&header),
	)
	start := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)
	log.WithField("backend", header["x-backend"]).
		WithField("method", method).WithField("duration", time.Since(start)).
		Debug("gRPC request finished.")
	return err
}

// LogStream prints the method at DEBUG level at the start of the stream.
func LogStream(
	ctx context.Context,
	sd *grpc.StreamDesc,
	conn *grpc.ClientConn,
	method string,
	streamer grpc.Streamer,
	opts ...grpc.CallOption,
) (grpc.ClientStream, error) {
	// Shortcut when debug logging is not enabled.
	if logrus.GetLevel() < logrus.DebugLevel {
		return streamer(ctx, sd, conn, method, opts...)
	}

	var header metadata.MD
	opts = append(
		opts,
		grpc.Header(&header),
	)
	strm, err := streamer(ctx, sd, conn, method, opts...)
	log.WithField("backend", header["x-backend"]).
		WithField("method", method).
		Debug("gRPC stream started.")
	return strm, err
}

// NoOpClientStream is a grpc.ClientStream implementation with no-op methods.
// Use it to satisfy interfaces that embed grpc.ClientStream when only specific
// methods (like Recv) are actually needed, e.g. in HTTP-based beacon API clients.
type NoOpClientStream struct{}

func (NoOpClientStream) Header() (metadata.MD, error) { return nil, nil }
func (NoOpClientStream) Trailer() metadata.MD         { return nil }
func (NoOpClientStream) CloseSend() error             { return nil }
func (NoOpClientStream) Context() context.Context     { return context.Background() }
func (NoOpClientStream) SendMsg(any) error            { return nil }
func (NoOpClientStream) RecvMsg(any) error            { return nil }

// AppendHeaders parses the provided GRPC headers
// and attaches them to the provided context.
func AppendHeaders(parent context.Context, headers []string) context.Context {
	for _, h := range headers {
		if h != "" {
			keyValue := strings.Split(h, "=")
			if len(keyValue) < 2 {
				log.Warnf("Incorrect gRPC header flag format. Skipping %v", keyValue[0])
				continue
			}
			parent = metadata.AppendToOutgoingContext(parent, keyValue[0], strings.Join(keyValue[1:], "=")) // nolint:fatcontext
		}
	}
	return parent
}
