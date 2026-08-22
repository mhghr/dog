package engines

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthMux returns the standard engine health endpoints. /health/ready pings
// Postgres; NATS connectivity is maintained by the message bus and failures
// are surfaced through logs per subscription.
func HealthMux(pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")

		if pool == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unavailable","dependencies":{"postgres":"unconfigured"}}`))
			return
		}
		if err := pool.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unavailable","dependencies":{"postgres":"error"}}`))
			return
		}

		_, _ = w.Write([]byte(`{"status":"ok","dependencies":{"postgres":"ok"}}`))
	})

	return mux
}
