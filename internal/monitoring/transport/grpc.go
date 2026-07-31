package transport

import (
	"context"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// GRPCConfig configures the agent's gRPC client connection.
type GRPCConfig struct {
	Endpoint   string
	CACertFile string // optional: path to a CA cert used to verify the gateway
}

// NewGRPCConnection dials an OTLP gRPC endpoint over TLS.
// The caller is responsible for closing the returned connection.
func NewGRPCConnection(ctx context.Context, cfg GRPCConfig) (*grpc.ClientConn, error) {
	tlsConfig := TLSConfig()
	if cfg.CACertFile != "" {
		pem, err := os.ReadFile(cfg.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("failed to parse CA cert %s", cfg.CACertFile)
		}
		tlsConfig.RootCAs = pool
	}
	creds := credentials.NewTLS(tlsConfig)

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
