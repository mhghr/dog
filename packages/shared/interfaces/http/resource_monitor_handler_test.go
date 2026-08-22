package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"monitoring-platform/packages/shared/domain"
)

func TestAttributeInt(t *testing.T) {
	cases := []struct {
		name string
		raw  any
		want *int
	}{
		{"float64", float64(200), intPtr(200)},
		{"float32", float32(404), intPtr(404)},
		{"int", 301, intPtr(301)},
		{"int64", int64(500), intPtr(500)},
		{"numeric string", "418", intPtr(418)},
		{"non-numeric string", "abc", nil},
		{"nil", nil, nil},
		{"bool", true, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := attributeInt(tc.raw)
			if tc.want == nil {
				if got != nil {
					t.Fatalf("attributeInt(%v) = %v, want nil", tc.raw, got)
				}
				return
			}
			if got == nil || *got != *tc.want {
				t.Fatalf("attributeInt(%v) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func intPtr(v int) *int { return &v }

func TestAttachLastStatusToProbes(t *testing.T) {
	latest := []domain.ProbeResult{
		{
			ProbeLocationID: "p1",
			Success:         true,
			StartedAt:       time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			Attributes:      map[string]any{"status_code": float64(200)},
		},
		{
			ProbeLocationID: "p2",
			Success:         false,
			StartedAt:       time.Date(2026, 1, 1, 12, 1, 0, 0, time.UTC),
			Attributes:      map[string]any{},
		},
	}

	probes := []domain.ProbeAggregateMetrics{
		{ProbeID: "p1", ProbeName: "frankfurt"},
		{ProbeID: "p2", ProbeName: "tehran"},
		{ProbeID: "p3", ProbeName: "unknown"}, // no latest result -> untouched
	}

	attachLastStatusToProbes(probes, latest)

	if probes[0].LastSuccess != true {
		t.Fatalf("p1 LastSuccess = %v, want true", probes[0].LastSuccess)
	}
	if probes[0].LastStatusCode == nil || *probes[0].LastStatusCode != 200 {
		t.Fatalf("p1 LastStatusCode = %v, want 200", probes[0].LastStatusCode)
	}
	if probes[0].LastCheckedAt == nil || !probes[0].LastCheckedAt.Equal(latest[0].StartedAt) {
		t.Fatalf("p1 LastCheckedAt = %v, want %v", probes[0].LastCheckedAt, latest[0].StartedAt)
	}
	if probes[1].LastSuccess != false {
		t.Fatalf("p2 LastSuccess = %v, want false", probes[1].LastSuccess)
	}
	if probes[1].LastStatusCode != nil {
		t.Fatalf("p2 LastStatusCode = %v, want nil", probes[1].LastStatusCode)
	}
	if probes[2].LastCheckedAt != nil || probes[2].LastSuccess != false {
		t.Fatalf("p3 should stay untouched, got %+v", probes[2])
	}
}

func TestParseMetricsRange(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("defaults to trailing 24h", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		from, to, ok := parseMetricsRange(httptest.NewRecorder(), req, url.Values{})
		if !ok {
			t.Fatal("expected ok")
		}
		if !to.After(from) {
			t.Fatalf("to %v must be after from %v", to, from)
		}
		if d := to.Sub(from); d != 24*time.Hour {
			t.Fatalf("window = %v, want 24h", d)
		}
	})

	t.Run("parses explicit from/to", func(t *testing.T) {
		fromStr := base.Add(-2 * time.Hour).Format(time.RFC3339)
		toStr := base.Format(time.RFC3339)
		values := url.Values{"from": {fromStr}, "to": {toStr}}
		from, to, ok := parseMetricsRange(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/metrics", nil), values)
		if !ok {
			t.Fatal("expected ok")
		}
		if !from.Equal(base.Add(-2 * time.Hour)) {
			t.Fatalf("from = %v", from)
		}
		if !to.Equal(base) {
			t.Fatalf("to = %v", to)
		}
	})

	t.Run("rejects invalid to", func(t *testing.T) {
		values := url.Values{"to": {"not-a-time"}}
		if _, _, ok := parseMetricsRange(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/metrics", nil), values); ok {
			t.Fatal("expected failure")
		}
	})

	t.Run("rejects from after to", func(t *testing.T) {
		fromStr := base.Add(time.Hour).Format(time.RFC3339)
		toStr := base.Format(time.RFC3339)
		values := url.Values{"from": {fromStr}, "to": {toStr}}
		if _, _, ok := parseMetricsRange(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/metrics", nil), values); ok {
			t.Fatal("expected failure")
		}
	})
}
