package postgres

import (
	"testing"
	"time"
)

func TestGetAttemptFromAttributes(t *testing.T) {
	cases := []struct {
		name     string
		attrs    map[string]any
		expected int
	}{
		{name: "nil attrs", attrs: nil, expected: 1},
		{name: "no attempt key", attrs: map[string]any{}, expected: 1},
		{name: "float64", attrs: map[string]any{"attempt": 2.0}, expected: 2},
		{name: "int", attrs: map[string]any{"attempt": 3}, expected: 3},
		{name: "int64", attrs: map[string]any{"attempt": int64(4)}, expected: 4},
		{name: "unsupported type", attrs: map[string]any{"attempt": "three"}, expected: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := getAttemptFromAttributes(tc.attrs); got != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, got)
			}
		})
	}
}

func TestGroupProbeSeries(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	raw := []rawPoint{
		{probeID: "p1", probeName: "Frankfurt", location: "DE", ts: now, value: 10},
		{probeID: "p1", probeName: "Frankfurt", location: "DE", ts: now.Add(time.Minute), value: 20},
		{probeID: "p2", probeName: "Tokyo", location: "JP", ts: now, value: 30},
		{probeID: "p2", probeName: "Tokyo", location: "JP", ts: now.Add(time.Minute), value: 40},
	}

	series := groupProbeSeries(raw, "rtt")

	if len(series) != 2 {
		t.Fatalf("expected 2 series, got %d", len(series))
	}

	first := series[0]
	if first.ProbeID != "p1" || first.ProbeName != "Frankfurt" || first.Location != "DE" || first.MetricKey != "rtt" {
		t.Errorf("unexpected first series %+v", first)
	}
	if len(first.Points) != 2 || first.Points[0].Value != 10 || first.Points[1].Value != 20 {
		t.Errorf("unexpected first series points %+v", first.Points)
	}
	if len(first.Values) != len(first.Points) {
		t.Errorf("expected Values to mirror Points, got %d vs %d", len(first.Values), len(first.Points))
	}

	second := series[1]
	if second.ProbeID != "p2" || second.ProbeName != "Tokyo" {
		t.Errorf("unexpected second series %+v", second)
	}
	if len(second.Points) != 2 || second.Points[0].Value != 30 || second.Points[1].Value != 40 {
		t.Errorf("unexpected second series points %+v", second.Points)
	}
}

func TestGroupProbeSeriesEmpty(t *testing.T) {
	if series := groupProbeSeries(nil, ""); len(series) != 0 {
		t.Errorf("expected no series for empty raw, got %d", len(series))
	}
}

func TestGroupProbeSeriesMetricKeyPreserved(t *testing.T) {
	raw := []rawPoint{{probeID: "p1", probeName: "Frankfurt", location: "DE", ts: time.Now(), value: 1}}
	if got := groupProbeSeries(raw, "status"); len(got) != 1 || got[0].MetricKey != "status" {
		t.Errorf("expected MetricKey status preserved, got %+v", got)
	}
}
