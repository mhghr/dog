package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"monitoring-platform/internal/alerting"
	"monitoring-platform/internal/domain"
)

type alertPolicyResponse struct {
	ID                 string                  `json:"id"`
	OrganizationID     string                  `json:"organization_id"`
	ProjectID          string                  `json:"project_id"`
	Name               string                  `json:"name"`
	Scope              domain.AlertPolicyScope `json:"scope"`
	Conditions         domain.AlertConditions   `json:"conditions"`
	Severity           string                  `json:"severity"`
	OpeningFailures    int                     `json:"opening_failures"`
	ResolvingSuccesses int                     `json:"resolving_successes"`
	CooldownSeconds    int                     `json:"cooldown_seconds"`
	RenotifySeconds    int                     `json:"renotify_seconds"`
	Enabled            bool                    `json:"enabled"`
	ChannelIDs         []string                `json:"channel_ids"`
	CreatedAt          string                  `json:"created_at"`
	UpdatedAt          string                  `json:"updated_at"`
}

func toAlertPolicyResponse(p alerting.AlertPolicy) alertPolicyResponse {
	return alertPolicyResponse{
		ID:                 p.ID,
		OrganizationID:     p.OrganizationID,
		ProjectID:          p.ProjectID,
		Name:               p.Name,
		Scope:              p.Scope,
		Conditions:         p.Conditions,
		Severity:           p.Severity,
		OpeningFailures:    p.OpeningFailures,
		ResolvingSuccesses: p.ResolvingSuccesses,
		CooldownSeconds:    p.CooldownSeconds,
		RenotifySeconds:    p.RenotifySeconds,
		Enabled:            p.Enabled,
		ChannelIDs:         p.ChannelIDs,
		CreatedAt:          p.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          p.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

type alertEventResponse struct {
	ID                  string  `json:"id"`
	OrganizationID      string  `json:"organization_id"`
	PolicyID            string  `json:"policy_id"`
	MonitorID           string  `json:"monitor_id"`
	State               string  `json:"state"`
	Severity            string  `json:"severity"`
	Title               string  `json:"title"`
	Description         string  `json:"description"`
	DedupKey            string  `json:"dedup_key"`
	ConsecutiveFailures int     `json:"consecutive_failures"`
	ConsecutiveSuccesses int    `json:"consecutive_successes"`
	OpenedAt            *string `json:"opened_at"`
	ResolvedAt          *string `json:"resolved_at"`
	CreatedAt           string  `json:"created_at"`
}

func toAlertEventResponse(a alerting.Alert) alertEventResponse {
	r := alertEventResponse{
		ID:                   a.ID,
		OrganizationID:       a.OrganizationID,
		PolicyID:             a.PolicyID,
		MonitorID:            a.MonitorID,
		State:                a.State,
		Severity:             a.Severity,
		Title:                a.Title,
		Description:          a.Description,
		DedupKey:             a.DedupKey,
		ConsecutiveFailures:  a.ConsecutiveFailures,
		ConsecutiveSuccesses: a.ConsecutiveSuccesses,
		CreatedAt:            a.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if a.OpenedAt != nil {
		s := a.OpenedAt.UTC().Format("2006-01-02T15:04:05Z")
		r.OpenedAt = &s
	}
	if a.ResolvedAt != nil {
		s := a.ResolvedAt.UTC().Format("2006-01-02T15:04:05Z")
		r.ResolvedAt = &s
	}
	return r
}

type notificationChannelResponse struct {
	ID             string            `json:"id"`
	OrganizationID string            `json:"organization_id"`
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	Config         map[string]string `json:"config"`
	Enabled        bool              `json:"enabled"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
}

func toNotificationChannelResponse(ch alerting.NotificationChannel) notificationChannelResponse {
	return notificationChannelResponse{
		ID:             ch.ID,
		OrganizationID: ch.OrganizationID,
		Name:           ch.Name,
		Type:           ch.Type,
		Config:         ch.Config,
		Enabled:        ch.Enabled,
		CreatedAt:      ch.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      ch.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func (h *Handler) listAlertPolicies(w http.ResponseWriter, r *http.Request) {
	orgID, ok := domain.OrgIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "No organization in session", nil)
		return
	}

	policies, err := h.deps.AlertRepo.ListPolicies(r.Context(), orgID)
	if err != nil {
		h.deps.Logger.Error("list alert policies failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	items := make([]alertPolicyResponse, 0, len(policies))
	for _, p := range policies {
		items = append(items, toAlertPolicyResponse(p))
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) createAlertPolicy(w http.ResponseWriter, r *http.Request) {
	orgID, ok := domain.OrgIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "No organization in session", nil)
		return
	}

	var policy alerting.AlertPolicy
	if err := decodeJSON(r, &policy); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}

	policy.OrganizationID = orgID
	if err := h.deps.AlertRepo.CreatePolicy(r.Context(), &policy); err != nil {
		h.deps.Logger.Error("create alert policy failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, toAlertPolicyResponse(policy))
}

func (h *Handler) listNotificationChannels(w http.ResponseWriter, r *http.Request) {
	orgID, ok := domain.OrgIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "No organization in session", nil)
		return
	}

	channels, err := h.deps.ChannelRepo.ListByOrg(r.Context(), orgID)
	if err != nil {
		h.deps.Logger.Error("list notification channels failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	items := make([]notificationChannelResponse, 0, len(channels))
	for _, ch := range channels {
		items = append(items, toNotificationChannelResponse(ch))
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) createNotificationChannel(w http.ResponseWriter, r *http.Request) {
	orgID, ok := domain.OrgIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "No organization in session", nil)
		return
	}

	var channel alerting.NotificationChannel
	if err := decodeJSON(r, &channel); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}

	channel.OrganizationID = orgID
	if err := h.deps.ChannelRepo.CreateChannel(r.Context(), &channel); err != nil {
		h.deps.Logger.Error("create notification channel failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, toNotificationChannelResponse(channel))
}

func (h *Handler) listAlerts(w http.ResponseWriter, r *http.Request) {
	orgID, ok := domain.OrgIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "No organization in session", nil)
		return
	}

	alerts, err := h.deps.AlertRepo.ListAlerts(r.Context(), orgID)
	if err != nil {
		h.deps.Logger.Error("list alerts failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	items := make([]alertEventResponse, 0, len(alerts))
	for _, a := range alerts {
		items = append(items, toAlertEventResponse(a))
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) getAlert(w http.ResponseWriter, r *http.Request) {
	orgID, ok := domain.OrgIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "No organization in session", nil)
		return
	}

	alertID := chi.URLParam(r, "alertID")

	alert, err := h.deps.AlertRepo.GetAlert(r.Context(), alertID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	if alert.OrganizationID != orgID {
		writeDomainError(w, r, domain.ErrNotFound)
		return
	}

	writeJSON(w, http.StatusOK, toAlertEventResponse(alert))
}
