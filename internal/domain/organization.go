package domain

import (
	"context"
	"strings"
	"time"
)

type Organization struct {
	ID        string
	Name      string
	Slug      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Project struct {
	ID             string
	OrganizationID string
	Name           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type OrganizationInput struct {
	Name string `json:"name"`
}

func ValidateOrganizationInput(input OrganizationInput) (Organization, map[string][]string) {
	fieldErrors := map[string][]string{}

	name := strings.TrimSpace(input.Name)
	if nameLen := len([]rune(name)); nameLen < 2 || nameLen > 200 {
		fieldErrors["name"] = append(fieldErrors["name"], "name must be between 2 and 200 characters")
	}

	if len(fieldErrors) > 0 {
		return Organization{}, fieldErrors
	}

	slug := generateSlug(name)
	return Organization{Name: name, Slug: slug}, nil
}

type ProjectInput struct {
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
}

func ValidateProjectInput(input ProjectInput) (Project, map[string][]string) {
	fieldErrors := map[string][]string{}

	if input.OrganizationID == "" {
		fieldErrors["organization_id"] = append(fieldErrors["organization_id"], "organization is required")
	}

	name := strings.TrimSpace(input.Name)
	if nameLen := len([]rune(name)); nameLen < 2 || nameLen > 200 {
		fieldErrors["name"] = append(fieldErrors["name"], "name must be between 2 and 200 characters")
	}

	if len(fieldErrors) > 0 {
		return Project{}, fieldErrors
	}

	return Project{OrganizationID: input.OrganizationID, Name: name}, nil
}

var slugBadChars = strings.NewReplacer(
	" ", "-", "?", "", "!", "", "@", "", "#", "", "$", "", "%", "", "^", "",
	"&", "", "*", "", "(", "", ")", "", "+", "", "=", "", ":", "", ";", "",
	",", "", ".", "", "/", "", "\\", "", "|", "", "~", "", "`", "",
	"[", "",  "]", "",  "{", "",  "}", "",  "<", "",  ">", "",  "'", "",  "\"", "",
)

func generateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = slugBadChars.Replace(slug)
	parts := strings.Fields(slug)
	if len(parts) > 8 {
		parts = parts[:8]
	}
	return strings.Join(parts, "-")
}

type OrgContextKey string

const OrgIDContextKey OrgContextKey = "org.id"
const ProjectIDContextKey OrgContextKey = "project.id"

func OrgIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(OrgIDContextKey).(string)
	return id, ok && id != ""
}

func ProjectIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ProjectIDContextKey).(string)
	return id, ok && id != ""
}
