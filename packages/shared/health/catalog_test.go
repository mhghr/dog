package health

import (
	"testing"

	"monitoring-platform/packages/shared/domain"
)

func TestSplitParamKeyMapsRealPingMetricKeys(t *testing.T) {
	cases := map[string][]string{
		"ping.reachability":        {"ping.reachability", "reachability"},
		"ping.rtt.avg_ms":          {"rtt_avg_ms", "avg_rtt_ms", "rtt_ms"},
		"ping.rtt.min_ms":          {"rtt_min_ms", "min_rtt_ms"},
		"ping.rtt.max_ms":          {"rtt_max_ms", "max_rtt_ms"},
		"ping.packet_loss_percent": {"packet_loss_percent", "packet_loss"},
		"ping.jitter_ms":           {"jitter_ms", "jitter"},
	}

	for key, want := range cases {
		got := splitParamKey(key)
		if len(got) != len(want) {
			t.Errorf("splitParamKey(%q) = %v, want %v", key, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("splitParamKey(%q)[%d] = %q, want %q", key, i, got[i], want[i])
			}
		}
	}
}

func TestPingPacketLossThresholdsMatchSpec(t *testing.T) {
	var def *ParameterDefinition
	for _, p := range PingParameters {
		if p.Key == "ping.packet_loss_percent" {
			def = &p
			break
		}
	}
	if def == nil {
		t.Fatal("ping.packet_loss_percent not found in PingParameters")
	}
	if def.DefaultWarning == nil || *def.DefaultWarning != 5 {
		t.Fatalf("expected DefaultWarning=5, got %v", def.DefaultWarning)
	}
	if def.DefaultError == nil || *def.DefaultError != 20 {
		t.Fatalf("expected DefaultError=20, got %v", def.DefaultError)
	}
}

func TestExtractParamValueReadsExecutorReachability(t *testing.T) {
	result := domain.ProbeResult{
		Metrics: map[string]any{"reachability": 1, "rtt_ms": 42.0},
	}
	if v, ok := extractParamValue(&result, "ping.reachability"); !ok || v != 1 {
		t.Fatalf("expected reachability value, got %v ok=%v", v, ok)
	}
	if v, ok := extractParamValue(&result, "ping.rtt.avg_ms"); !ok || v != 42.0 {
		t.Fatalf("expected rtt_ms value via avg key, got %v ok=%v", v, ok)
	}
}
