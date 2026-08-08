package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) getAgentConfig(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")
	if agentID == "" {
		writeError(w, r, http.StatusBadRequest, "missing_agent_id", "agent_id is required", nil)
		return
	}

	cfg, err := h.deps.AgentConfigs.GetActive(r.Context(), agentID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"version": cfg.Version,
		"config": map[string]any{
			"version":                     cfg.Version,
			"collection_interval_seconds": cfg.CollectionIntervalSeconds,
			"batch_size":                  cfg.BatchSize,
			"export_interval_seconds":     cfg.ExportIntervalSeconds,
			"enabled_receivers":           cfg.EnabledReceivers,
			"max_metrics_per_batch":       cfg.MaxMetricsPerBatch,
			"max_label_count":             cfg.MaxLabelCount,
			"max_label_length":            cfg.MaxLabelLength,
			"feature_flags":               cfg.FeatureFlags,
			"otlp_endpoint":               cfg.OTLPEndpoint,
			"compress":                    cfg.Compress,
			"retry_initial_interval_ms":   cfg.RetryInitialIntervalMs,
			"retry_max_interval_ms":       cfg.RetryMaxIntervalMs,
			"retry_max_elapsed_ms":        cfg.RetryMaxElapsedMs,
			"log_level":                   cfg.LogLevel,
		},
	})
}
