package domain

import (
	"regexp"
	"strings"
	"time"
)

// StatusPage is a public-facing page that exposes the live status of a
// selected set of monitors under a unique slug.
type StatusPage struct {
	ID          string
	Slug        string
	Name        string
	Description string
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Components  []StatusPageComponent
}

type StatusPageComponent struct {
	ID          string
	MonitorID   string
	MonitorName string
	DisplayName string
	SortOrder   int
}

type StatusPageInput struct {
	Slug        string                     `json:"slug"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Enabled     *bool                      `json:"enabled"`
	Components  []StatusPageComponentInput `json:"components"`
}

type StatusPageComponentInput struct {
	MonitorID   string `json:"monitor_id"`
	DisplayName string `json:"display_name"`
}

// Overall status levels for the public page.
const (
	StatusPageOperational   = "operational"
	StatusPagePartialOutage = "partial_outage"
	StatusPageMajorOutage   = "major_outage"
)

var statusPageSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

const maxStatusPageComponents = 50

func ValidateStatusPageInput(input StatusPageInput) (StatusPage, map[string][]string) {
	fieldErrors := map[string][]string{}

	slug := strings.ToLower(strings.TrimSpace(input.Slug))
	if len(slug) < 2 || len(slug) > 100 || !statusPageSlugPattern.MatchString(slug) {
		fieldErrors["slug"] = append(
			fieldErrors["slug"],
			"slug must be 2-100 characters of lowercase letters, digits and dashes",
		)
	}

	name := strings.TrimSpace(input.Name)
	if nameLength := len([]rune(name)); nameLength < 2 || nameLength > 200 {
		fieldErrors["name"] = append(fieldErrors["name"], "name must be between 2 and 200 characters")
	}

	description := strings.TrimSpace(input.Description)
	if len([]rune(description)) > 500 {
		fieldErrors["description"] = append(fieldErrors["description"], "description must be at most 500 characters")
	}

	if len(input.Components) == 0 {
		fieldErrors["components"] = append(fieldErrors["components"], "at least one monitor is required")
	}
	if len(input.Components) > maxStatusPageComponents {
		fieldErrors["components"] = append(fieldErrors["components"], "too many monitors")
	}

	seen := make(map[string]bool, len(input.Components))
	for _, component := range input.Components {
		monitorID := strings.TrimSpace(component.MonitorID)
		if monitorID == "" {
			fieldErrors["components"] = append(fieldErrors["components"], "monitor_id is required for every component")
			break
		}
		if seen[monitorID] {
			fieldErrors["components"] = append(fieldErrors["components"], "duplicate monitor in components")
			break
		}
		seen[monitorID] = true
	}

	if len(fieldErrors) > 0 {
		return StatusPage{}, fieldErrors
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	components := make([]StatusPageComponent, 0, len(input.Components))
	for index, component := range input.Components {
		components = append(components, StatusPageComponent{
			MonitorID:   strings.TrimSpace(component.MonitorID),
			DisplayName: strings.TrimSpace(component.DisplayName),
			SortOrder:   index,
		})
	}

	return StatusPage{
		Slug:        slug,
		Name:        name,
		Description: description,
		Enabled:     enabled,
		Components:  components,
	}, nil
}

// PublicStatusComponent is the anonymous, public view of one component.
type PublicStatusComponent struct {
	Name      string        `json:"name"`
	Status    MonitorStatus `json:"status"`
	Uptime24h *float64      `json:"uptime_24h"`
	Uptime7d  *float64      `json:"uptime_7d"`
	Uptime30d *float64      `json:"uptime_30d"`
}

type PublicStatusPage struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Status      string                  `json:"status"`
	Components  []PublicStatusComponent `json:"components"`
	CheckedAt   time.Time               `json:"checked_at"`
}

// ComputeOverallStatus derives the page-level status from component states.
// Paused and unknown components are excluded from the outage calculation.
func ComputeOverallStatus(components []PublicStatusComponent) string {
	active := 0
	down := 0

	for _, component := range components {
		if component.Status != StatusUp && component.Status != StatusDown {
			continue
		}
		active++
		if component.Status == StatusDown {
			down++
		}
	}

	switch {
	case active == 0 || down == 0:
		return StatusPageOperational
	case down == active:
		return StatusPageMajorOutage
	default:
		return StatusPagePartialOutage
	}
}
