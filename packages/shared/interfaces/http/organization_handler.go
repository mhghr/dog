package api

import (
	"net/http"
	"time"

	"monitoring-platform/packages/shared/domain"
)

type organizationResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
