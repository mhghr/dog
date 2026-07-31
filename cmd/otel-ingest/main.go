package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/proto/otlp/collector/metrics/v1"

	"monitoring-platform/internal/config"
	"monitoring-platform/internal/domain"
	"monitoring-platform/internal/ingestion/messagebus"
	"monitoring-platform/internal/ingestion/pipeline"
	"monitoring-platform/internal/logging"
	"monitoring-platform/internal/postgres"
	"monitoring-platform/internal/security"
)

type otlpServer struct {
	v1.UnimplementedMetricsServiceServer
	publisher *pipeline.IngestionPublisher
	logger    *slog.Logger
}

func (s *otlpServer) Export(ctx context.Context, req *v1.ExportMetricsServiceRequest) (*v1.ExportMetricsServiceResponse, error) {
	if len(req.GetResourceMetrics()) == 0 {
		return &v1.ExportMetricsServiceResponse{}, nil
	}

	samples := pipeline.ConvertOTLPMetrics(req, time.Now().UTC())
	if len(samples) == 0 {
		return &v1.ExportMetricsServiceResponse{}, nil
	}

	batch := &domain.MetricBatch{
		Samples:    samples,
		ReceivedAt: time.Now().UTC(),
	}

	if err := s.publisher.PublishBatch(ctx, batch); err != nil {
		s.logger.Error("publish batch failed", "error", err)

		var authErr *pipeline.AuthError
		switch {
		case errors.As(err, &authErr):
			return nil, status.Error(codes.Unauthenticated, "invalid agent credentials")
		default:
			return nil, status.Error(codes.Internal, "failed to ingest metrics")
		}
	}

	return &v1.ExportMetricsServiceResponse{}, nil
}

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.LogLevel, cfg.LogFormat, "otel-ingest")

	if err := cfg.Require("OTEL_INGEST_ADDRESS", "DATABASE_URL", "NATS_URL", "AGENT_SECRET_ENCRYPTION_KEY"); err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	agentsRepo := postgres.NewMonitoringAgentRepository(pool)

	bus, err := messagebus.NewNATSBus(messagebus.NATSConfig{
		URL:       cfg.NATSURL,
		Reconnect: true,
		MaxReconn: 10,
	}, logger)
	if err != nil {
		logger.Error("failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer bus.Close()

	// HMAC auth: getSecret decrypts the stored encrypted secret.
	auth := pipeline.NewHMACAuthenticator(func(ctx context.Context, agentID string) (string, error) {
		agent, err := agentsRepo.GetByAgentID(ctx, agentID)
		if err != nil {
			return "", err
		}
		secret, err := security.DecryptSecret(cfg.AgentSecretEncryptionKey, agent.SecretEncrypted)
		if err != nil {
			return "", fmt.Errorf("decrypt agent secret: %w", err)
		}
		return secret, nil
	})

	tenantRes := pipeline.NewTenantResolver(func(ctx context.Context, agentID string) (string, error) {
		agent, err := agentsRepo.GetByAgentID(ctx, agentID)
		if err != nil {
			return "", err
		}
		return agent.TenantID, nil
	})

	publisher := pipeline.NewIngestionPublisher(
		bus,
		auth,
		tenantRes,
		pipeline.NewMetricNormalizer(),
		pipeline.NewMetricEnricher(),
		logger,
	)

	lis, err := net.Listen("tcp", cfg.OTELIngestAddress)
	if err != nil {
		logger.Error("failed to listen", "address", cfg.OTELIngestAddress, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	v1.RegisterMetricsServiceServer(grpcServer, &otlpServer{
		publisher: publisher,
		logger:    logger,
	})
	reflection.Register(grpcServer)

	go func() {
		logger.Info("otel-ingest listening", "address", cfg.OTELIngestAddress)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("grpc server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down otel-ingest")
	grpcServer.GracefulStop()
	logger.Info("otel-ingest stopped")
}
