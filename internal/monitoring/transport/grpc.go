package transport

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// GRPCConfig configures the agent's gRPC client connection.
type GRPCConfig struct {
	Endpoint string
}

// NewGRPCConnection dials an OTLP gRPC endpoint over TLS.
// The caller is responsible for closing the returned connection.
func NewGRPCConnection(ctx context.Context, cfg GRPCConfig) (*grpc.ClientConn, error) {
	creds := credentials.NewTLS(TLSConfig())

	conn, err := grpc.DialContext(
		ctx,
		cfg.Endpoint,
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(10*1024*1024),
			grpc.MaxCallSendMsgSize(10*1024*1024),
		),
	)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
