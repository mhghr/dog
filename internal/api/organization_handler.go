package api

import (
	"net/http"
	"time"

	"monitoring-platform/internal/domain"
)

type organizationResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type projectResponse struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (h *Handler) createOrganization(w http.ResponseWriter, r *http.Request) {
	var input domain.OrganizationInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}

	org, fieldErrors := domain.ValidateOrganizationInput(input)
	if len(fieldErrors) > 0 {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_failed", "The request contains invalid fields", fieldErrors)
		return
	}

	if err := h.deps.Orgs.Create(r.Context(), &org); err != nil {
		h.deps.Logger.Error("create organization failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, organizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	})
}

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	orgID, ok := domain.OrgIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "No organization in session", nil)
		return
	}

	projects, err := h.deps.Projects.ListByOrganization(r.Context(), orgID)
	if err != nil {
		h.deps.Logger.Error("list projects failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	items := make([]projectResponse, 0, len(projects))
	for _, project := range projects {
		items = append(items, projectResponse{
			ID:             project.ID,
			OrganizationID: project.OrganizationID,
			Name:           project.Name,
			CreatedAt:      project.CreatedAt,
			UpdatedAt:      project.UpdatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	orgID, ok := domain.OrgIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "No organization in session", nil)
		return
	}

	var input domain.ProjectInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}
	input.OrganizationID = orgID

	project, fieldErrors := domain.ValidateProjectInput(input)
	if len(fieldErrors) > 0 {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_failed", "The request contains invalid fields", fieldErrors)
		return
	}

	if err := h.deps.Projects.Create(r.Context(), &project); err != nil {
		h.deps.Logger.Error("create project failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, projectResponse{
		ID:             project.ID,
		OrganizationID: project.OrganizationID,
		Name:           project.Name,
		CreatedAt:      project.CreatedAt,
		UpdatedAt:      project.UpdatedAt,
	})
}
