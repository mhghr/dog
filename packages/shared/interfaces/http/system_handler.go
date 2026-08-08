package api

import (
	"context"
	"net/http"
	"time"

	"monitoring-platform/packages/shared/heartbeat"
	"monitoring-platform/packages/shared/domain"
)

func (h *Handler) dashboardSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.deps.Results.DashboardSummary(r.Context())
	if err != nil {
		h.deps.Logger.Error("dashboard summary failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

type locationResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *Handler) listLocations(w http.ResponseWriter, r *http.Request) {
	locations, err := h.deps.Locations.List(r.Context())
	if err != nil {
		h.deps.Logger.Error("list locations failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	items := make([]locationResponse, 0, len(locations))
	for _, location := range locations {
		items = append(items, locationResponse{
			ID:        location.ID,
			Name:      location.Name,
			Code:      location.Code,
			Enabled:   location.Enabled,
			CreatedAt: location.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) createLocation(w http.ResponseWriter, r *http.Request) {
	var input domain.LocationInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}

	location, fieldErrors := domain.ValidateLocationInput(input)
	if len(fieldErrors) > 0 {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_failed", "The request contains invalid fields", fieldErrors)
		return
	}

	if err := h.deps.Locations.Create(r.Context(), &location); err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, locationResponse{
		ID:        location.ID,
		Name:      location.Name,
		Code:      location.Code,
		Enabled:   location.Enabled,
		CreatedAt: location.CreatedAt,
	})
}

type componentHealth struct {
	Name     string     `json:"name"`
	Status   string     `json:"status"`
	LastSeen *time.Time `json:"last_seen,omitempty"`
}

// systemHealth aggregates control-plane and distributed component health:
// direct dependency checks plus Redis-based heartbeats and queue stats.
func (h *Handler) systemHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	components := []componentHealth{{Name: "api", Status: "healthy"}}

	postgresStatus := "healthy"
	if err := h.deps.Pool.Ping(ctx); err != nil {
		postgresStatus = "unhealthy"
	}
	components = append(components, componentHealth{Name: "postgres", Status: postgresStatus})

	redisStatus := "healthy"
	redisHealthy := true
	if err := h.deps.Redis.Ping(ctx).Err(); err != nil {
		redisStatus = "unhealthy"
		redisHealthy = false
	}
	components = append(components, componentHealth{Name: "redis", Status: redisStatus})

	victoriaStatus := "healthy"
	if err := h.deps.Victoria.Health(ctx); err != nil {
		victoriaStatus = "unhealthy"
	}
	components = append(components, componentHealth{Name: "victoriametrics", Status: victoriaStatus})

	schedulerStatus := "unknown"
	var schedulerSeen *time.Time
	if redisHealthy {
		alive, lastSeen := heartbeat.Alive(ctx, h.deps.Redis, "scheduler", "scheduler")
		schedulerStatus = "unhealthy"
		if alive {
			schedulerStatus = "healthy"
			if !lastSeen.IsZero() {
				schedulerSeen = &lastSeen
			}
		}
	}
	components = append(components, componentHealth{Name: "scheduler", Status: schedulerStatus, LastSeen: schedulerSeen})

	workers := []componentHealth{}
	if redisHealthy {
		infos, err := heartbeat.List(ctx, h.deps.Redis, "worker")
		if err == nil {
			for _, info := range infos {
				lastSeen := info.LastSeen
				workers = append(workers, componentHealth{
					Name:     info.Name,
					Status:   "healthy",
					LastSeen: &lastSeen,
				})
			}
		}
	}

	queueStats, err := h.deps.Queue.Stats(ctx)
	if err != nil {
		h.deps.Logger.Warn("queue stats failed", "error", err)
	}

	overall := "healthy"
	if postgresStatus != "healthy" || redisStatus != "healthy" {
		overall = "unhealthy"
	} else if victoriaStatus != "healthy" || schedulerStatus != "healthy" || len(workers) == 0 {
		overall = "degraded"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":     overall,
		"components": components,
		"workers":    workers,
		"queue": map[string]int64{
			"lag":     queueStats.Lag,
			"pending": queueStats.Pending,
		},
		"checked_at": time.Now().UTC(),
	})
}
