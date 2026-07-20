package api

import (
	"context"
	"net/http"
	"time"
)

func (h *Handler) healthLive(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// healthReady verifies critical dependencies. PostgreSQL and Redis gate
// readiness; VictoriaMetrics is reported but non-fatal (metrics degrade
// gracefully while probe scheduling keeps working).
func (h *Handler) healthReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	dependencies := map[string]string{
		"postgres":        "ok",
		"redis":           "ok",
		"victoriametrics": "ok",
	}

	healthy := true

	if err := h.deps.Pool.Ping(ctx); err != nil {
		dependencies["postgres"] = "error"
		healthy = false
	}

	if err := h.deps.Redis.Ping(ctx).Err(); err != nil {
		dependencies["redis"] = "error"
		healthy = false
	}

	if err := h.deps.Victoria.Health(ctx); err != nil {
		dependencies["victoriametrics"] = "degraded"
	}

	status := "ok"
	statusCode := http.StatusOK
	if !healthy {
		status = "unavailable"
		statusCode = http.StatusServiceUnavailable
	}

	writeJSON(w, statusCode, map[string]any{
		"status":       status,
		"dependencies": dependencies,
	})
}
