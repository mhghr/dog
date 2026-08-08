package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/health"
)

func (h *Handler) listMonitorTypeParameters(w http.ResponseWriter, r *http.Request) {
	monitorType := chi.URLParam(r, "type")

	params, err := h.deps.MonitorTypeParams.ListByMonitorType(r.Context(), monitorType)
	if err != nil {
		h.deps.Logger.Error("list monitor type parameters failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal", err.Error(), nil)
		return
	}

	if params == nil {
		params = []domain.MonitorTypeParameter{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"monitor_type": monitorType,
		"parameters":   params,
	})
}

func (h *Handler) listParameterRules(w http.ResponseWriter, r *http.Request) {
	monitorID := paramFromURL(r, "monitorID")
	if monitorID == "" {
		return
	}

	rules, err := h.deps.HealthRepo.ListParameterRules(r.Context(), monitorID)
	if err != nil {
		h.deps.Logger.Error("list parameter rules failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	if rules == nil {
		rules = []health.ParameterRule{}
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": rules})
}

func (h *Handler) getParameterRule(w http.ResponseWriter, r *http.Request) {
	monitorID := paramFromURL(r, "monitorID")
	parameterKey := paramFromURL(r, "parameterKey")
	if monitorID == "" || parameterKey == "" {
		return
	}

	rule, err := h.deps.HealthRepo.GetParameterRule(r.Context(), monitorID, parameterKey)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, rule)
}

func (h *Handler) putParameterRule(w http.ResponseWriter, r *http.Request) {
	monitorID := paramFromURL(r, "monitorID")
	parameterKey := paramFromURL(r, "parameterKey")
	if monitorID == "" || parameterKey == "" {
		return
	}

	var rule health.ParameterRule
	if err := decodeJSON(r, &rule); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}

	rule.MonitorID = monitorID
	rule.ParameterKey = parameterKey

	if rule.Aggregation == "" {
		rule.Aggregation = "avg"
	}
	if rule.WindowType == "" {
		rule.WindowType = "checks"
	}
	if rule.WindowValue == 0 {
		rule.WindowValue = 3
	}
	if rule.WarningOperator == "" {
		rule.WarningOperator = "gte"
	}
	if rule.ErrorOperator == "" {
		rule.ErrorOperator = "gte"
	}
	if rule.MinimumSamples == 0 {
		rule.MinimumSamples = 3
	}
	if rule.MissingDataPolicy == "" {
		rule.MissingDataPolicy = "IGNORE"
	}
	if rule.MissedChecks == 0 {
		rule.MissedChecks = 3
	}

	if err := h.deps.HealthRepo.UpsertParameterRule(r.Context(), &rule); err != nil {
		h.deps.Logger.Error("upsert parameter rule failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, rule)
}

func (h *Handler) deleteParameterRule(w http.ResponseWriter, r *http.Request) {
	monitorID := paramFromURL(r, "monitorID")
	parameterKey := paramFromURL(r, "parameterKey")
	if monitorID == "" || parameterKey == "" {
		return
	}

	if err := h.deps.HealthRepo.DeleteParameterRule(r.Context(), monitorID, parameterKey); err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) listHealthNotificationChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := h.deps.HealthRepo.ListNotificationChannels(r.Context())
	if err != nil {
		h.deps.Logger.Error("list health notification channels failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	if channels == nil {
		channels = []health.HealthNotificationChannel{}
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": channels})
}

func (h *Handler) createHealthNotificationChannel(w http.ResponseWriter, r *http.Request) {
	var channel health.HealthNotificationChannel
	if err := decodeJSON(r, &channel); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}

	if channel.Name == "" || channel.Type == "" {
		writeError(w, r, http.StatusBadRequest, "validation_error", "name and type are required", nil)
		return
	}

	if err := h.deps.HealthRepo.CreateNotificationChannel(r.Context(), &channel); err != nil {
		h.deps.Logger.Error("create health notification channel failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, channel)
}

func (h *Handler) updateHealthNotificationChannel(w http.ResponseWriter, r *http.Request) {
	channelID := paramFromURL(r, "channelId")
	if channelID == "" {
		return
	}

	var channel health.HealthNotificationChannel
	if err := decodeJSON(r, &channel); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}

	channel.ID = channelID
	if err := h.deps.HealthRepo.UpdateNotificationChannel(r.Context(), &channel); err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, channel)
}

func (h *Handler) deleteHealthNotificationChannel(w http.ResponseWriter, r *http.Request) {
	channelID := paramFromURL(r, "channelId")
	if channelID == "" {
		return
	}

	if err := h.deps.HealthRepo.DeleteNotificationChannel(r.Context(), channelID); err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) testHealthNotificationChannel(w http.ResponseWriter, r *http.Request) {
	channelID := paramFromURL(r, "channelId")
	if channelID == "" {
		return
	}

	if err := h.deps.HealthNotifier.SendTest(r.Context(), channelID); err != nil {
		writeError(w, r, http.StatusBadRequest, "test_failed", err.Error(), nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "test_dispatched"})
}

func (h *Handler) listNotificationPolicies(w http.ResponseWriter, r *http.Request) {
	monitorID := paramFromURL(r, "monitorID")
	if monitorID == "" {
		return
	}

	policies, err := h.deps.HealthRepo.ListNotificationPolicies(r.Context(), monitorID)
	if err != nil {
		h.deps.Logger.Error("list notification policies failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	if policies == nil {
		policies = []health.NotificationPolicy{}
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": policies})
}

func (h *Handler) createNotificationPolicy(w http.ResponseWriter, r *http.Request) {
	monitorID := paramFromURL(r, "monitorID")
	if monitorID == "" {
		return
	}

	var policy health.NotificationPolicy
	if err := decodeJSON(r, &policy); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}

	policy.MonitorID = &monitorID
	if err := h.deps.HealthRepo.CreateNotificationPolicy(r.Context(), &policy); err != nil {
		h.deps.Logger.Error("create notification policy failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, policy)
}

func (h *Handler) updateNotificationPolicy(w http.ResponseWriter, r *http.Request) {
	policyID := paramFromURL(r, "policyId")
	if policyID == "" {
		return
	}

	var policy health.NotificationPolicy
	if err := decodeJSON(r, &policy); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}

	policy.ID = policyID
	if err := h.deps.HealthRepo.UpdateNotificationPolicy(r.Context(), &policy); err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, policy)
}

func (h *Handler) deleteNotificationPolicy(w http.ResponseWriter, r *http.Request) {
	policyID := paramFromURL(r, "policyId")
	if policyID == "" {
		return
	}

	if err := h.deps.HealthRepo.DeleteNotificationPolicy(r.Context(), policyID); err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func paramFromURL(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}
