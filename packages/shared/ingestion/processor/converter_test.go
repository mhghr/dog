package processor

import (
	"strings"
	"testing"
	"time"

	"monitoring-platform/packages/shared/domain"
)

func TestConvertToVM_ConvertsAllSamples(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	batch := domain.MetricBatch{
		AgentID:  "agent-1",
		TenantID: "tenant-1",
		Samples: []domain.MetricSample{
			{
				Name:      "cpu.usage.user",
				Value:     12.5,
				Timestamp: ts,
				Labels:    map[string]string{"cpu": "0", "mode": "user"},
			},
			{
				Name:      "mem.usage.bytes",
				Value:     1024,
				Timestamp: ts,
				Labels:    map[string]string{},
			},
		},
	}

	req := ConvertToVM(batch)

	if len(req.Samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(req.Samples))
	}

	first := req.Samples[0]
	if first.Metric != "cpu.usage.user" {
		t.Errorf("expected metric name cpu.usage.user, got %q", first.Metric)
	}
	if first.Value != 12.5 {
		t.Errorf("expected value 12.5, got %g", first.Value)
	}
	if first.Timestamp != ts.UnixMilli() {
		t.Errorf("expected timestamp %d, got %d", ts.UnixMilli(), first.Timestamp)
	}
	if len(first.Labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(first.Labels))
	}
	labels := map[string]string{}
	for _, l := range first.Labels {
		labels[l.Name] = l.Value
	}
	if labels["cpu"] != "0" || labels["mode"] != "user" {
		t.Errorf("expected labels cpu=0 mode=user, got %v", labels)
	}

	second := req.Samples[1]
	if second.Metric != "mem.usage.bytes" {
		t.Errorf("expected metric name mem.usage.bytes, got %q", second.Metric)
	}
	if len(second.Labels) != 0 {
		t.Errorf("expected no labels, got %d", len(second.Labels))
	}
}

func TestToPrometheusText_EscapesQuotesAndBackslashes(t *testing.T) {
	req := VMWriteRequest{Samples: []VMMetric{
		{
			Metric:    "http_requests_total",
			Value:     42,
			Timestamp: 1700000000000,
			Labels: []VMLabel{
				{Name: "method", Value: `GET"POST\`},
				{Name: "path", Value: "line1\nline2"},
			},
		},
	}}

	text := req.ToPrometheusText()

	want := `http_requests_total{method="GET\"POST\\",path="line1\nline2"} 42 1700000000000` + "\n"
	if text != want {
		t.Errorf("unexpected text output\n got: %q\nwant: %q", text, want)
	}
}

func TestToPrometheusText_SampleWithoutLabels(t *testing.T) {
	req := VMWriteRequest{Samples: []VMMetric{
		{Metric: "cpu.usage.user", Value: 1.5, Timestamp: 1700000000000},
	}}

	text := req.ToPrometheusText()

	want := "cpu.usage.user 1.5 1700000000000\n"
	if text != want {
		t.Errorf("unexpected text output\n got: %q\nwant: %q", text, want)
	}
}

func TestToPrometheusText_EmptyBatch(t *testing.T) {
	req := VMWriteRequest{Samples: []VMMetric{}}

	if text := req.ToPrometheusText(); text != "" {
		t.Errorf("expected empty output, got %q", text)
	}

	batch := domain.MetricBatch{}
	if text := ConvertToVM(batch).ToPrometheusText(); text != "" {
		t.Errorf("expected empty output for empty batch, got %q", text)
	}
}

func TestToPrometheusText_RoundTripHasAllSamples(t *testing.T) {
	req := VMWriteRequest{Samples: []VMMetric{
		{Metric: "a", Value: 1, Timestamp: 100},
		{Metric: "b", Value: 2, Timestamp: 200},
	}}

	text := req.ToPrometheusText()

	if got := strings.Count(text, "\n"); got != 2 {
		t.Errorf("expected 2 lines, got %d", got)
	}
}
