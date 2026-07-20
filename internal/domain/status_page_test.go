package domain

import "testing"

func TestValidateStatusPageInputAcceptsValidInput(t *testing.T) {
	enabled := false
	page, fieldErrors := ValidateStatusPageInput(StatusPageInput{
		Slug:        "  Main-Status ",
		Name:        "Public Status",
		Description: "Everything important",
		Enabled:     &enabled,
		Components: []StatusPageComponentInput{
			{MonitorID: "9f6f2c9a-1111-4222-8333-444455556666", DisplayName: "API"},
			{MonitorID: "9f6f2c9a-1111-4222-8333-444455557777"},
		},
	})

	if len(fieldErrors) != 0 {
		t.Fatalf("expected no field errors, got %v", fieldErrors)
	}
	if page.Slug != "main-status" {
		t.Fatalf("expected normalized slug, got %q", page.Slug)
	}
	if page.Enabled {
		t.Fatal("expected enabled=false to be honored")
	}
	if len(page.Components) != 2 || page.Components[1].SortOrder != 1 {
		t.Fatalf("expected ordered components, got %+v", page.Components)
	}
}

func TestValidateStatusPageInputRejectsBadSlugAndDuplicates(t *testing.T) {
	_, fieldErrors := ValidateStatusPageInput(StatusPageInput{
		Slug: "Bad Slug!",
		Name: "x",
		Components: []StatusPageComponentInput{
			{MonitorID: "a"},
			{MonitorID: "a"},
		},
	})

	if len(fieldErrors["slug"]) == 0 {
		t.Fatal("expected slug error")
	}
	if len(fieldErrors["name"]) == 0 {
		t.Fatal("expected name error")
	}
	if len(fieldErrors["components"]) == 0 {
		t.Fatal("expected components error")
	}
}

func TestComputeOverallStatus(t *testing.T) {
	up := PublicStatusComponent{Status: StatusUp}
	down := PublicStatusComponent{Status: StatusDown}
	paused := PublicStatusComponent{Status: StatusPaused}

	cases := []struct {
		name       string
		components []PublicStatusComponent
		expected   string
	}{
		{"all up", []PublicStatusComponent{up, up}, StatusPageOperational},
		{"empty", nil, StatusPageOperational},
		{"paused only", []PublicStatusComponent{paused}, StatusPageOperational},
		{"partial", []PublicStatusComponent{up, down}, StatusPagePartialOutage},
		{"major", []PublicStatusComponent{down, down, paused}, StatusPageMajorOutage},
	}

	for _, testCase := range cases {
		if got := ComputeOverallStatus(testCase.components); got != testCase.expected {
			t.Fatalf("%s: expected %s, got %s", testCase.name, testCase.expected, got)
		}
	}
}
