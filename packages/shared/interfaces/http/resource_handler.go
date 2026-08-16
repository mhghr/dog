package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/geoip"
)

// ── Monitor Types ─────────────────────────────────────────────

func (h *Handler) listMonitorTypesAll(w http.ResponseWriter, r *http.Request) {
	types, err := h.deps.ResourceRepo.ListMonitorTypes(r.Context())
	if err != nil {
		h.deps.Logger.Error("list monitor types failed", "error", err)
		writeDomainError(w, r, err)
		return
	}
	if types == nil {
		types = []domain.MonitorTypeDef{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": types})
}

// ── GeoIP ─────────────────────────────────────────────────────

// geoIpLookup resolves an IP address to its geographic location using the
// geoip provider. Used by the web UI to auto-fill a resource location from
// its target IP. Returns empty country/city for private/undetected ranges.
func (h *Handler) geoIpLookup(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_ip", "ip is required", nil)
		return
	}

	loc, err := geoip.Lookup(ip)
	if err != nil {
		h.deps.Logger.Warn("geoip lookup failed", "ip", ip, "error", err)
		writeError(w, r, http.StatusBadGateway, "geoip_failed", "geo location lookup failed", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"country": loc.Country,
		"city":    loc.City,
		"lat":     loc.Lat,
		"lon":     loc.Lon,
	})
}

// ── Resource Types ────────────────────────────────────────────

func (h *Handler) listResourceTypes(w http.ResponseWriter, r *http.Request) {
	types, err := h.deps.ResourceRepo.ListTypes(r.Context())
	if err != nil {
		h.deps.Logger.Error("list resource types failed", "error", err)
		writeDomainError(w, r, err)
		return
	}
	if types == nil {
		types = []domain.ResourceType{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": types})
}

// ── Resources ─────────────────────────────────────────────────

type resourceResponse struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	WorkspaceID    *string        `json:"workspace_id,omitempty"`
	ResourceTypeID string         `json:"resource_type_id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Target         string         `json:"target"`
	Status         string         `json:"status"`
	Metadata       map[string]any `json:"metadata"`
	TypeName       string         `json:"type_name,omitempty"`
	TypeCategory   string         `json:"type_category,omitempty"`
	TypeIcon       string         `json:"type_icon,omitempty"`
	MonitorsCount  int            `json:"monitors_count,omitempty"`
	HealthStatus   string         `json:"health_status,omitempty"`
	HealthScore    float64        `json:"health_score,omitempty"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
}

func toResourceResponse(res domain.Resource) resourceResponse {
	return resourceResponse{
		ID:             res.ID,
		OrganizationID: res.OrganizationID,
		WorkspaceID:    res.WorkspaceID,
		ResourceTypeID: res.ResourceTypeID,
		Name:           res.Name,
		Description:    res.Description,
		Target:         res.Target,
		Status:         res.Status,
		Metadata:       res.Metadata,
		TypeName:       res.TypeName,
		TypeCategory:   res.TypeCategory,
		TypeIcon:       res.TypeIcon,
		MonitorsCount:  res.MonitorsCount,
		HealthStatus:   res.HealthStatus,
		HealthScore:    res.HealthScore,
		CreatedAt:      res.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      res.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// loadResource parses the resourceID URL param, validates it, and loads the
// resource. It writes the error response and returns ok=false on failure.
func (h *Handler) loadResource(w http.ResponseWriter, r *http.Request) (domain.Resource, bool) {
	resourceID := chi.URLParam(r, "resourceID")
	if _, err := uuid.Parse(resourceID); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "resource id must be a valid UUID", nil)
		return domain.Resource{}, false
	}

	res, err := h.deps.ResourceRepo.GetByID(r.Context(), resourceID)
	if err != nil {
		writeDomainError(w, r, err)
		return domain.Resource{}, false
	}

	return res, true
}

func (h *Handler) listResources(w http.ResponseWriter, r *http.Request) {
	orgID, ok := domain.OrgIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "No organization in session", nil)
		return
	}

	filter := domain.ResourceListFilter{
		OrganizationID: orgID,
		WorkspaceID:    r.URL.Query().Get("workspace_id"),
		ResourceTypeID: r.URL.Query().Get("resource_type_id"),
		Status:         r.URL.Query().Get("status"),
		Search:         r.URL.Query().Get("search"),
		Page:           1,
		PageSize:       50,
	}
	if v := r.URL.Query().Get("page"); v != "" {
		filter.Page = atoiDefault(v, 1)
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		filter.PageSize = atoiDefault(v, 50)
	}

	resources, total, err := h.deps.ResourceRepo.List(r.Context(), filter)
	if err != nil {
		h.deps.Logger.Error("list resources failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	items := make([]resourceResponse, 0, len(resources))
	for _, res := range resources {
		items = append(items, toResourceResponse(res))
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (h *Handler) resourceOverview(w http.ResponseWriter, r *http.Request) {
	orgID, ok := domain.OrgIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "No organization in session", nil)
		return
	}

	filter := domain.ResourceListFilter{OrganizationID: orgID, Page: 1, PageSize: 100}
	resources, _, err := h.deps.ResourceRepo.List(r.Context(), filter)
	if err != nil {
		h.deps.Logger.Error("resource overview failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	counts := map[string]int{"total": len(resources)}
	for _, res := range resources {
		if res.TypeCategory != "" {
			counts[res.TypeCategory]++
		}
		if res.Status != "" {
			key := "status_" + res.Status
			counts[key]++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"counts": counts})
}

func (h *Handler) createResource(w http.ResponseWriter, r *http.Request) {
	orgID, ok := domain.OrgIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "No organization in session", nil)
		return
	}

	var input domain.ResourceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "invalid body", nil)
		return
	}

	if input.ResourceTypeID == "" {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_failed", "resource_type_id is required", nil)
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_failed", "name is required", nil)
		return
	}

	res := &domain.Resource{
		OrganizationID: orgID,
		WorkspaceID:    input.WorkspaceID,
		ResourceTypeID: input.ResourceTypeID,
		Name:           strings.TrimSpace(input.Name),
		Description:    input.Description,
		Target:         input.Target,
		Status:         "active",
		Metadata:       input.Metadata,
	}
	if res.Metadata == nil {
		res.Metadata = map[string]any{}
	}

	if err := h.deps.ResourceRepo.Create(r.Context(), res); err != nil {
		h.deps.Logger.Error("create resource failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	for _, tag := range input.Tags {
		_ = h.deps.ResourceRepo.AttachTag(r.Context(), res.ID, tag.Key, tag.Value)
	}

	writeJSON(w, http.StatusCreated, toResourceResponse(*res))
}

func (h *Handler) getResource(w http.ResponseWriter, r *http.Request) {
	res, ok := h.loadResource(w, r)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, toResourceResponse(res))
}

// getResourceOverview returns a single consolidated snapshot for a resource —
// status, health, and current metric values with a short sparkline trend for
// each monitor. The frontend consumes this with one request to render every
// Metric Card on the resource overview, instead of one request per metric.
func (h *Handler) getResourceOverview(w http.ResponseWriter, r *http.Request) {
	res, ok := h.loadResource(w, r)
	if !ok {
		return
	}

	monitors, err := h.deps.MonitorRepo.ListByResource(r.Context(), res.ID)
	if err != nil {
		h.deps.Logger.Error("overview: list monitors failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	metrics := make(map[string]any, len(monitors))
	trendWindow := time.Now().UTC().Add(-1 * time.Hour)

	for _, m := range monitors {
		latest, err := h.deps.Results.LatestResultsByProbe(r.Context(), m.ID)
		if err != nil || len(latest) == 0 {
			if err != nil {
				h.deps.Logger.Warn("overview: latest results failed", "monitor_id", m.ID, "error", err)
			}
			continue
		}

		entry := monitorOverviewEntry(latest)
		entry["last_checked_at"] = latest[0].FinishedAt
		entry["trend"] = h.sparklineTrend(r.Context(), m.ID, trendWindow)

		key := string(m.Type)
		if key == "" {
			key = m.ID
		}
		metrics[key] = entry
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":              res.ID,
		"name":            res.Name,
		"target":          res.Target,
		"status":          res.Status,
		"health_status":   res.HealthStatus,
		"last_checked_at": res.UpdatedAt,
		"metrics":         metrics,
	})
}

// monitorOverviewEntry aggregates the latest per-probe results into a flat
// map of current metric values (average across probes). The keys are the
// raw probe metric keys (rtt_ms, packet_loss_percent, ...) plus success.
func monitorOverviewEntry(latest []domain.ProbeResult) map[string]any {
	entry := map[string]any{}
	success := 0
	for _, result := range latest {
		if result.Success {
			success++
		}
		for key, value := range result.Metrics {
			if number, ok := toFloat(value); ok {
				if existing, seen := entry[key].(float64); seen {
					entry[key] = (existing + number) / 2
				} else {
					entry[key] = number
				}
			}
		}
	}
	entry["success"] = success > 0 && success == len(latest)
	entry["probe_count"] = len(latest)
	return entry
}

// sparklineTrend returns a short series of averaged values for the monitor's
// primary metric (rtt_ms) over the last hour, suitable for a tiny sparkline.
func (h *Handler) sparklineTrend(ctx context.Context, monitorID string, from time.Time) []float64 {
	series, err := h.deps.Results.SeriesByProbeMetric(ctx, monitorID, "rtt_ms", from, time.Now().UTC(), 60)
	if err != nil || len(series) == 0 {
		return nil
	}

	values := make([]float64, 0, len(series[0].Points))
	for _, point := range series[0].Points {
		values = append(values, point.Value)
	}
	return values
}

func toFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f, true
		}
	}
	return 0, false
}

func (h *Handler) updateResource(w http.ResponseWriter, r *http.Request) {
	res, ok := h.loadResource(w, r)
	if !ok {
		return
	}

	var input struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Target      string         `json:"target"`
		Status      string         `json:"status"`
		Metadata    map[string]any `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "invalid body", nil)
		return
	}

	applyResourceUpdate(&res, input)

	if err := h.deps.ResourceRepo.Update(r.Context(), &res); err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toResourceResponse(res))
}

// applyResourceUpdate overwrites resource fields that were provided in the
// request body. Empty string fields and nil metadata are left untouched.
func applyResourceUpdate(res *domain.Resource, input struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Target      string         `json:"target"`
	Status      string         `json:"status"`
	Metadata    map[string]any `json:"metadata"`
}) {
	if input.Name != "" {
		res.Name = input.Name
	}
	if input.Description != "" {
		res.Description = input.Description
	}
	if input.Target != "" {
		res.Target = input.Target
	}
	if input.Status != "" {
		res.Status = input.Status
	}
	if input.Metadata != nil {
		res.Metadata = input.Metadata
	}
}

func (h *Handler) deleteResource(w http.ResponseWriter, r *http.Request) {
	res, ok := h.loadResource(w, r)
	if !ok {
		return
	}

	if err := h.deps.ResourceRepo.Delete(r.Context(), res.ID); err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Resource Tags ─────────────────────────────────────────────

func (h *Handler) listResourceTags(w http.ResponseWriter, r *http.Request) {
	resourceID := chi.URLParam(r, "resourceID")
	tags, err := h.deps.ResourceRepo.ListTags(r.Context(), resourceID)
	if err != nil {
		h.deps.Logger.Error("list resource tags failed", "error", err)
		writeDomainError(w, r, err)
		return
	}
	if tags == nil {
		tags = []domain.Tag{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": tags})
}

func (h *Handler) attachResourceTag(w http.ResponseWriter, r *http.Request) {
	resourceID := chi.URLParam(r, "resourceID")

	var input domain.TagInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "invalid body", nil)
		return
	}
	if strings.TrimSpace(input.Key) == "" {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_failed", "key is required", nil)
		return
	}

	if err := h.deps.ResourceRepo.AttachTag(r.Context(), resourceID, input.Key, input.Value); err != nil {
		h.deps.Logger.Error("attach resource tag failed", "error", err)
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"key": input.Key, "value": input.Value})
}

func (h *Handler) removeResourceTag(w http.ResponseWriter, r *http.Request) {
	resourceID := chi.URLParam(r, "resourceID")
	tagID := chi.URLParam(r, "tagID")

	if err := h.deps.ResourceRepo.RemoveTag(r.Context(), resourceID, tagID); err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Workspaces ────────────────────────────────────────────────

type workspaceResponse struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	Name           string         `json:"name"`
	Slug           string         `json:"slug"`
	Description    string         `json:"description"`
	Plan           string         `json:"plan"`
	Settings       map[string]any `json:"settings"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
}

func toWorkspaceResponse(ws domain.Workspace) workspaceResponse {
	return workspaceResponse{
		ID:             ws.ID,
		OrganizationID: ws.OrganizationID,
		Name:           ws.Name,
		Slug:           ws.Slug,
		Description:    ws.Description,
		Plan:           string(ws.Plan),
		Settings:       ws.Settings,
		CreatedAt:      ws.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      ws.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func (h *Handler) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	orgID, ok := domain.OrgIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "No organization in session", nil)
		return
	}

	workspaces, err := h.deps.ResourceRepo.ListWorkspaces(r.Context(), orgID)
	if err != nil {
		h.deps.Logger.Error("list workspaces failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	items := make([]workspaceResponse, 0, len(workspaces))
	for _, ws := range workspaces {
		items = append(items, toWorkspaceResponse(ws))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) createWorkspace(w http.ResponseWriter, r *http.Request) {
	orgID, ok := domain.OrgIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "No organization in session", nil)
		return
	}

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Plan        string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "invalid body", nil)
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_failed", "name is required", nil)
		return
	}

	plan := domain.PlanFree
	if input.Plan != "" {
		plan = domain.WorkspacePlan(input.Plan)
	}

	ws := &domain.Workspace{
		OrganizationID: orgID,
		Name:           strings.TrimSpace(input.Name),
		Slug:           slugify(input.Name),
		Description:    input.Description,
		Plan:           plan,
		Settings:       map[string]any{},
	}

	if err := h.deps.ResourceRepo.CreateWorkspace(r.Context(), ws); err != nil {
		h.deps.Logger.Error("create workspace failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, toWorkspaceResponse(*ws))
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func slugify(name string) string {
	return strings.ToLower(strings.NewReplacer(
		" ", "-", "_", "-", ".", "-",
	).Replace(strings.TrimSpace(name)))
}
