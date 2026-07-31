package pipeline

import (
	"strings"
	"testing"

	"monitoring-platform/internal/domain"
)

func sample(name string, labels map[string]string) domain.MetricSample {
	return domain.MetricSample{Name: name, Labels: labels}
}

func TestValidateMetrics_ValidMetricNamesPass(t *testing.T) {
	batch := &domain.MetricBatch{Samples: []domain.MetricSample{
		sample("cpu.usage.user", nil),
		sample("http_requests_total", map[string]string{"method": "GET", "code": "200"}),
		sample("_internal_metric", nil),
		sample("k8s.node/cpu.util", map[string]string{"node": "n1"}),
	}}

	result := ValidateMetrics(batch)

	if !result.Valid {
		t.Fatalf("expected batch to be valid, got errors: %v", result.Errors)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %v", result.Errors)
	}
	if len(batch.Samples) != 4 {
		t.Fatalf("expected 4 samples kept, got %d", len(batch.Samples))
	}
}

func TestValidateMetrics_InvalidMetricNamesDropped(t *testing.T) {
	batch := &domain.MetricBatch{Samples: []domain.MetricSample{
		sample("1bad", nil),
		sample("bad!name", nil),
		sample("", nil),
		sample("cpu.usage.user", nil),
	}}

	result := ValidateMetrics(batch)

	if result.Valid {
		t.Fatal("expected batch to be invalid")
	}
	if len(batch.Samples) != 1 {
		t.Fatalf("expected 1 sample kept, got %d", len(batch.Samples))
	}
	if batch.Samples[0].Name != "cpu.usage.user" {
		t.Fatalf("expected kept sample to be cpu.usage.user, got %q", batch.Samples[0].Name)
	}
	if len(result.Errors) != 3 {
		t.Fatalf("expected 3 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestValidateMetrics_InvalidMetricNamesDroppedWithLabels(t *testing.T) {
	batch := &domain.MetricBatch{Samples: []domain.MetricSample{
		sample("1bad", map[string]string{"method": "GET"}),
	}}

	result := ValidateMetrics(batch)

	if result.Valid {
		t.Fatal("expected batch to be invalid")
	}
	if len(batch.Samples) != 0 {
		t.Fatalf("expected 0 samples kept, got %d", len(batch.Samples))
	}
}

func TestValidateMetrics_ReservedLabelsDropped(t *testing.T) {
	batch := &domain.MetricBatch{Samples: []domain.MetricSample{
		sample("cpu.usage.user", map[string]string{
			"tenant_id": "tenant-1",
			"agent_id":  "agent-1",
			"hostname":  "host-1",
			"env":       "prod",
		}),
	}}

	result := ValidateMetrics(batch)

	if !result.Valid {
		t.Fatalf("expected batch to stay valid, got errors: %v", result.Errors)
	}
	if len(batch.Samples) != 1 {
		t.Fatalf("expected sample kept, got %d", len(batch.Samples))
	}
	labels := batch.Samples[0].Labels
	if _, ok := labels["tenant_id"]; ok {
		t.Error("tenant_id label should have been removed")
	}
	if _, ok := labels["agent_id"]; ok {
		t.Error("agent_id label should have been removed")
	}
	if _, ok := labels["hostname"]; ok {
		t.Error("hostname label should have been removed")
	}
	if labels["env"] != "prod" {
		t.Errorf("expected env label preserved, got %q", labels["env"])
	}
	if result.Skipped != 3 {
		t.Fatalf("expected 3 skipped label removals, got %d", result.Skipped)
	}
}

func TestValidateMetrics_InvalidLabelNameDropsSample(t *testing.T) {
	batch := &domain.MetricBatch{Samples: []domain.MetricSample{
		sample("cpu.usage.user", map[string]string{"1bad": "v", "ok": "yes"}),
		sample("mem.usage.bytes", map[string]string{"region": "us"}),
	}}

	result := ValidateMetrics(batch)

	if result.Valid {
		t.Fatal("expected batch to be invalid")
	}
	if len(batch.Samples) != 1 {
		t.Fatalf("expected 1 sample kept, got %d", len(batch.Samples))
	}
	if batch.Samples[0].Name != "mem.usage.bytes" {
		t.Fatalf("expected mem.usage.bytes kept, got %q", batch.Samples[0].Name)
	}
}

func TestValidateMetrics_LongLabelValueTruncated(t *testing.T) {
	long := strings.Repeat("x", 2048)
	batch := &domain.MetricBatch{Samples: []domain.MetricSample{
		sample("cpu.usage.user", map[string]string{"method": long}),
	}}

	result := ValidateMetrics(batch)

	if !result.Valid {
		t.Fatalf("expected batch to stay valid, got errors: %v", result.Errors)
	}
	if len(batch.Samples) != 1 {
		t.Fatalf("expected sample kept, got %d", len(batch.Samples))
	}
	if got := len(batch.Samples[0].Labels["method"]); got != 1024 {
		t.Fatalf("expected label value truncated to 1024 chars, got %d", got)
	}
}

func TestNormalizeMetricName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"CPU.Usage.User", "cpu_usage_user"},
		{"  http_requests_total  ", "http_requests_total"},
		{"system.CPU Load", "system_cpu_load"},
	}
	for _, c := range cases {
		if got := NormalizeMetricName(c.in); got != c.want {
			t.Errorf("NormalizeMetricName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMetricNormalizer_Normalize(t *testing.T) {
	batch := &domain.MetricBatch{Samples: []domain.MetricSample{
		sample("CPU.Usage.User", map[string]string{"Host.Name": "h1"}),
	}}

	NewMetricNormalizer().Normalize(batch)

	s := batch.Samples[0]
	if s.Name != "cpu_usage_user" {
		t.Errorf("expected normalized name, got %q", s.Name)
	}
	if _, ok := s.Labels["host_name"]; !ok {
		t.Errorf("expected normalized label key host_name, got %v", s.Labels)
	}
	if s.Labels["host_name"] != "h1" {
		t.Errorf("expected label value preserved, got %q", s.Labels["host_name"])
	}
}

func TestMetricEnricher_Enrich(t *testing.T) {
	batch := &domain.MetricBatch{Samples: []domain.MetricSample{
		sample("cpu.usage.user", map[string]string{"env": "prod"}),
	}}
	identity := &AgentIdentity{TenantID: "tenant-1", AgentID: "agent-1", Hostname: "host-1"}

	NewMetricEnricher().Enrich(batch, identity)

	s := batch.Samples[0]
	if s.Labels["tenant_id"] != "tenant-1" || s.Labels["agent_id"] != "agent-1" || s.Labels["hostname"] != "host-1" {
		t.Errorf("expected identity labels set, got %v", s.Labels)
	}
	if s.Labels["env"] != "prod" {
		t.Errorf("expected existing label preserved, got %v", s.Labels)
	}
	if s.TenantID != "tenant-1" || s.AgentID != "agent-1" || s.Hostname != "host-1" {
		t.Errorf("expected identity fields set, got %+v", s)
	}
	if batch.TenantID != "tenant-1" || batch.AgentID != "agent-1" {
		t.Errorf("expected batch identity set, got %+v", batch)
	}
}

func TestMetricEnricher_EmptyHostnameSkipsLabel(t *testing.T) {
	batch := &domain.MetricBatch{Samples: []domain.MetricSample{
		sample("cpu.usage.user", nil),
	}}
	identity := &AgentIdentity{TenantID: "tenant-1", AgentID: "agent-1"}

	NewMetricEnricher().Enrich(batch, identity)

	if _, ok := batch.Samples[0].Labels["hostname"]; ok {
		t.Error("hostname label should not be set when hostname is empty")
	}
}
