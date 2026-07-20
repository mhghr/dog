package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"monitoring-platform/internal/domain"
)

type statusPageComponentResponse struct {
	ID          string `json:"id"`
	MonitorID   string `json:"monitor_id"`
	MonitorName string `json:"monitor_name"`
	DisplayName string `json:"display_name"`
	SortOrder   int    `json:"sort_order"`
}

type statusPageResponse struct {
	ID             string                        `json:"id"`
	Slug           string                        `json:"slug"`
	Name           string                        `json:"name"`
	Description    string                        `json:"description"`
	Enabled        bool                          `json:"enabled"`
	ComponentCount int                           `json:"component_count"`
	Components     []statusPageComponentResponse `json:"components,omitempty"`
	CreatedAt      time.Time                     `json:"created_at"`
	UpdatedAt      time.Time                     `json:"updated_at"`
}

func toStatusPageResponse(page domain.StatusPage, includeComponents bool) statusPageResponse {
	response := statusPageResponse{
		ID:             page.ID,
		Slug:           page.Slug,
		Name:           page.Name,
		Description:    page.Description,
		Enabled:        page.Enabled,
		ComponentCount: len(page.Components),
		CreatedAt:      page.CreatedAt,
		UpdatedAt:      page.UpdatedAt,
	}

	if includeComponents {
		response.Components = make([]statusPageComponentResponse, 0, len(page.Components))
		for _, component := range page.Components {
			response.Components = append(response.Components, statusPageComponentResponse{
				ID:          component.ID,
				MonitorID:   component.MonitorID,
				MonitorName: component.MonitorName,
				DisplayName: component.DisplayName,
				SortOrder:   component.SortOrder,
			})
		}
	}

	return response
}

func (h *Handler) listStatusPages(w http.ResponseWriter, r *http.Request) {
	pages, err := h.deps.StatusPages.List(r.Context())
	if err != nil {
		h.deps.Logger.Error("list status pages failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	items := make([]statusPageResponse, 0, len(pages))
	for _, page := range pages {
		items = append(items, toStatusPageResponse(page, false))
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) createStatusPage(w http.ResponseWriter, r *http.Request) {
	var input domain.StatusPageInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}

	page, fieldErrors := domain.ValidateStatusPageInput(input)
	if len(fieldErrors) > 0 {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_failed", "The request contains invalid fields", fieldErrors)
		return
	}

	if err := h.deps.StatusPages.Create(r.Context(), &page); err != nil {
		h.deps.Logger.Error("create status page failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	created, err := h.deps.StatusPages.GetByID(r.Context(), page.ID)
	if err != nil {
		writeJSON(w, http.StatusCreated, toStatusPageResponse(page, false))
		return
	}

	writeJSON(w, http.StatusCreated, toStatusPageResponse(created, true))
}

func (h *Handler) statusPageIDFromRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := chi.URLParam(r, "statusPageID")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "The status page id is not valid", nil)
		return "", false
	}
	return id, true
}

func (h *Handler) getStatusPage(w http.ResponseWriter, r *http.Request) {
	id, ok := h.statusPageIDFromRequest(w, r)
	if !ok {
		return
	}

	page, err := h.deps.StatusPages.GetByID(r.Context(), id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toStatusPageResponse(page, true))
}

func (h *Handler) updateStatusPage(w http.ResponseWriter, r *http.Request) {
	id, ok := h.statusPageIDFromRequest(w, r)
	if !ok {
		return
	}

	var input domain.StatusPageInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}

	page, fieldErrors := domain.ValidateStatusPageInput(input)
	if len(fieldErrors) > 0 {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_failed", "The request contains invalid fields", fieldErrors)
		return
	}
	page.ID = id

	if err := h.deps.StatusPages.Update(r.Context(), &page); err != nil {
		h.deps.Logger.Error("update status page failed", "error", err, "status_page_id", id)
		writeDomainError(w, r, err)
		return
	}

	updated, err := h.deps.StatusPages.GetByID(r.Context(), id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toStatusPageResponse(updated, true))
}

func (h *Handler) deleteStatusPage(w http.ResponseWriter, r *http.Request) {
	id, ok := h.statusPageIDFromRequest(w, r)
	if !ok {
		return
	}

	if err := h.deps.StatusPages.Delete(r.Context(), id); err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

// publicStatusPage is unauthenticated by design: it exposes only display
// names, statuses and uptime percentages of monitors explicitly published
// on an enabled status page.
func (h *Handler) publicStatusPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	page, err := h.deps.StatusPages.PublicBySlug(r.Context(), slug)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	page.CheckedAt = time.Now().UTC()

	w.Header().Set("Cache-Control", "public, max-age=30")
	writeJSON(w, http.StatusOK, page)
}
