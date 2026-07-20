// Package httpserver provides graceful HTTP server lifecycle helpers shared
// by all binaries, plus the container healthcheck subcommand.
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

func New(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// Run serves until the context is cancelled, then shuts down gracefully.
func Run(ctx context.Context, server *http.Server, logger *slog.Logger) error {
	errChan := make(chan error, 1)

	go func() {
		logger.Info("http server listening", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	logger.Info("shutting down http server", "address", server.Addr)
	return server.Shutdown(shutdownCtx)
}

// RunHealthcheckCommand implements `<binary> healthcheck` for distroless
// containers that have no shell or curl.
func RunHealthcheckCommand(address string) {
	if len(os.Args) < 2 || os.Args[1] != "healthcheck" {
		return
	}

	url := fmt.Sprintf("http://127.0.0.1%s/health/live", normalizeAddress(address))

	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck failed:", err)
		os.Exit(1)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck failed with status:", response.Status)
		os.Exit(1)
	}

	os.Exit(0)
}

func normalizeAddress(address string) string {
	if strings.HasPrefix(address, ":") {
		return address
	}

	if index := strings.LastIndex(address, ":"); index >= 0 {
		return address[index:]
	}

	return ":" + address
}
