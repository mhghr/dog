package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"monitoring-platform/internal/domain"
)

func (h *Handler) postAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")
	if agentID == "" {
		writeError(w, r, http.StatusBadRequest, "missing_agent_id", "agent_id is required", nil)
		return
	}

	var req struct {
		CPUPercent           float64 `json:"cpu_percent"`
		MemoryPercent        float64 `json:"memory_percent"`
		DiskPercent          float64 `json:"disk_percent"`
		UptimeSeconds        int64   `json:"uptime_seconds"`
		MetricsSent          int64   `json:"metrics_sent"`
		MetricsQueued        int64   `json:"metrics_queued"`
		CollectorUptimeSecs  int64   `json:"collector_uptime_seconds"`
		PublicIP             string  `json:"public_ip"`
		CurrentConfigVersion int     `json:"current_config_version"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_body", "Invalid request body", nil)
		return
	}

	hb := domain.AgentHeartbeat{
		AgentID:                agentID,
		CPUPercent:             req.CPUPercent,
		MemoryPercent:          req.MemoryPercent,
		DiskPercent:            req.DiskPercent,
		UptimeSeconds:          req.UptimeSeconds,
		MetricsSent:            req.MetricsSent,
		MetricsQueued:          req.MetricsQueued,
		CollectorUptimeSeconds: req.CollectorUptimeSecs,
		PublicIP:               req.PublicIP,
	}

	var warning string
	if err := h.deps.MonitoringAgents.UpdateHeartbeat(r.Context(), agentID, hb); err != nil {
		h.deps.Logger.Warn("agent heartbeat record failed", "agent_id", agentID, "error", err)
		if errors.Is(err, domain.ErrNotFound) {
			warning = "agent record not found; heartbeat not recorded"
		} else {
			warning = "heartbeat not recorded"
		}
	}

	configChanged := false
	newConfigVersion := 0
	latest, err := h.deps.AgentConfigs.GetActive(r.Context(), agentID)
	if err != nil {
		h.deps.Logger.Warn("get active agent config failed", "agent_id", agentID, "error", err)
	} else {
		newConfigVersion = latest.Version
		if latest.Version > req.CurrentConfigVersion {
			configChanged = true
		}
	}

	resp := map[string]any{
		"config_changed":     configChanged,
		"new_config_version": newConfigVersion,
		"update_available":   configChanged,
		"action":             "none",
	}
	if warning != "" {
		resp["warning"] = warning
	}

	writeJSON(w, http.StatusOK, resp)
}
