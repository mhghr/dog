package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type GatewayConfig struct {
	ListenAddress string
	HealthAddress string
	TLSCertFile   string
	TLSKeyFile    string
	CACertFile    string
	DatabaseURL   string
}

type Gateway struct {
	cfg    GatewayConfig
	logger *slog.Logger
	http   *http.Server
}

func New(cfg GatewayConfig, logger *slog.Logger) (*Gateway, error) {
	return &Gateway{cfg: cfg, logger: logger}, nil
}

func (g *Gateway) Close() {
	if g.http != nil {
		_ = g.http.Close()
	}
}

func (g *Gateway) ListenAndServe(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	lis, err := net.Listen("tcp", g.cfg.ListenAddress)
	if err != nil {
		return fmt.Errorf("gateway listen: %w", err)
	}

	g.http = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	g.logger.Info("agent-gateway listening", "address", g.cfg.ListenAddress)

	errCh := make(chan error, 1)
	go func() {
		errCh <- g.http.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = g.http.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}
