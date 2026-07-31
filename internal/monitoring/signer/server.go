package signer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/proto/otlp/collector/metrics/v1"

	"monitoring-platform/internal/monitoring/credential"
)

// Proxy is a local OTLP MetricsService that signs each export request with the
// agent credential and forwards it to the remote ingestion gateway.
type Proxy struct {
	v1.UnimplementedMetricsServiceServer
	credMgr *credential.Manager
	remote  v1.MetricsServiceClient
	conn    *grpc.ClientConn
	logger  *slog.Logger
}

// New creates a signing proxy that forwards signed requests over the given
// connection. The caller is responsible for building the connection to the
// otel-ingest gateway (e.g. via transport.NewGRPCConnection) and for closing
// it, either directly or via Proxy.Close.
func New(conn *grpc.ClientConn, credMgr *credential.Manager, logger *slog.Logger) (*Proxy, error) {
	if conn == nil {
		return nil, fmt.Errorf("signer: nil client conn")
	}
	return &Proxy{
		credMgr: credMgr,
		remote:  v1.NewMetricsServiceClient(conn),
		conn:    conn,
		logger:  logger,
	}, nil
}

// Close closes the underlying gateway connection.
func (p *Proxy) Close() error {
	if p.conn == nil {
		return nil
	}
	return p.conn.Close()
}

// Export signs the incoming OTLP request with the agent credential and
// forwards it to the remote gateway.
func (p *Proxy) Export(ctx context.Context, req *v1.ExportMetricsServiceRequest) (*v1.ExportMetricsServiceResponse, error) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	md := metadata.Pairs(
		"x-agent-id", p.credMgr.AgentID(),
		"x-timestamp", timestamp,
		"x-signature", p.credMgr.SignMessage(p.credMgr.AgentID(), timestamp),
	)
	outCtx := metadata.NewOutgoingContext(ctx, md)
	return p.remote.Export(outCtx, req)
}
