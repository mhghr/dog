package domain

import "testing"

func intPtr(value int) *int { return &value }

func TestValidateMonitorInputHTTP(t *testing.T) {
	monitor, fieldErrors := ValidateMonitorInput(MonitorInput{
		Name:            "Main Website",
		Type:            "http",
		Target:          "https://example.com",
		IntervalSeconds: intPtr(60),
		Config: map[string]any{
			"method":                "GET",
			"expected_status_codes": []any{float64(200)},
			"verify_tls":            true,
		},
	})

	if len(fieldErrors) > 0 {
		t.Fatalf("unexpected field errors: %v", fieldErrors)
	}

	if monitor.Type != MonitorHTTP || monitor.TimeoutMillis != 5000 || monitor.Retries != 1 || !monitor.Enabled {
		t.Fatalf("defaults not applied: %+v", monitor)
	}

	if monitor.LastStatus != StatusUnknown {
		t.Fatalf("expected unknown status, got %s", monitor.LastStatus)
	}
}

func TestValidateMonitorInputRejectsBadTargets(t *testing.T) {
	cases := []MonitorInput{
		{Name: "bad http", Type: "http", Target: "example.com"},
		{Name: "bad http creds", Type: "http", Target: "https://user:pass@example.com"},
		{Name: "bad tcp", Type: "tcp", Target: "db.example.com/path"},
		{Name: "tcp no port", Type: "tcp", Target: "db.example.com"},
		{Name: "bad type", Type: "gopher", Target: "example.com"},
		{Name: "bad dns record", Type: "dns", Target: "example.com", Config: map[string]any{"record_type": "BOGUS"}},
		{Name: "bad smtp mode", Type: "smtp", Target: "mail.example.com", Config: map[string]any{"mode": "tls13"}},
		{Name: "bad ntp version", Type: "ntp", Target: "time.example.com", Config: map[string]any{"version": float64(7)}},
		{Name: "domain ip", Type: "domain_expiration", Target: "8.8.8.8"},
	}

	for _, input := range cases {
		if _, fieldErrors := ValidateMonitorInput(input); len(fieldErrors) == 0 {
			t.Errorf("expected validation errors for %q", input.Name)
		}
	}
}

func TestValidateMonitorInputEnforcesPerTypeMinInterval(t *testing.T) {
	_, fieldErrors := ValidateMonitorInput(MonitorInput{
		Name:            "TLS too fast",
		Type:            "tls",
		Target:          "example.com",
		IntervalSeconds: intPtr(30),
	})

	if len(fieldErrors["interval_seconds"]) == 0 {
		t.Fatal("expected interval_seconds error for tls monitor below 300s")
	}
}

func TestValidateMonitorInputRejectsUnknownConfigKeys(t *testing.T) {
	_, fieldErrors := ValidateMonitorInput(MonitorInput{
		Name:   "Unknown key",
		Type:   "http",
		Target: "https://example.com",
		Config: map[string]any{"tls_verify": true},
	})

	if len(fieldErrors["config.tls_verify"]) == 0 {
		t.Fatalf("expected unknown key error, got %v", fieldErrors)
	}
}

func TestValidateMonitorInputDefaultsIntervalPerType(t *testing.T) {
	monitor, fieldErrors := ValidateMonitorInput(MonitorInput{
		Name:   "Domain",
		Type:   "domain_expiration",
		Target: "example.com",
	})

	if len(fieldErrors) > 0 {
		t.Fatalf("unexpected errors: %v", fieldErrors)
	}

	if monitor.IntervalSeconds != 43200 {
		t.Fatalf("expected 12h default interval, got %d", monitor.IntervalSeconds)
	}
}
