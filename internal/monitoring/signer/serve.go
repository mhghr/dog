package signer

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"

	"go.opentelemetry.io/proto/otlp/collector/metrics/v1"
)

// Serve starts the local signing proxy on the given address (e.g.
// "127.0.0.1:4319") and blocks until the server stops. Returns the grpc.Server
// for lifecycle management via a server field.
func Serve(ctx context.Context, address string, p *Proxy, logger *slog.Logger) (*grpc.Server, net.Listener, error) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return nil, nil, fmt.Errorf("listen %s: %w", address, err)
	}

	server := grpc.NewServer()
	v1.RegisterMetricsServiceServer(server, p)

	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()

	go func() {
		if err := server.Serve(lis); err != nil && ctx.Err() == nil {
			logger.Error("signing proxy serve error", "error", err)
		}
	}()

	return server, lis, nil
}
