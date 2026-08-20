package probe

import (
	"io"
	"log/slog"
	"testing"

	"monitoring-platform/packages/shared/security"
)

// restrictiveDeps builds executor dependencies with the SSRF Guard enforcing
// public-only destinations (allowPrivate=false), mirroring production config.
func restrictiveDeps() Deps {
	return Deps{
		Guard:  security.NewGuard(false),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestStringConfig(t *testing.T) {
	config := map[string]any{"method": "POST", "empty": ""}

	if got := stringConfig(config, "method", "GET"); got != "POST" {
		t.Errorf("expected POST, got %s", got)
	}
	if got := stringConfig(config, "missing", "GET"); got != "GET" {
		t.Errorf("expected default GET, got %s", got)
	}
	if got := stringConfig(config, "empty", "GET"); got != "GET" {
		t.Errorf("expected default for empty string, got %s", got)
	}
}

func TestBoolConfig(t *testing.T) {
	config := map[string]any{"verify_tls": false, "wrong": "yes"}

	if got := boolConfig(config, "verify_tls", true); got {
		t.Error("expected false")
	}
	if got := boolConfig(config, "missing", true); !got {
		t.Error("expected default true")
	}
	if got := boolConfig(config, "wrong", true); !got {
		t.Error("expected default for non-bool value")
	}
}

func TestIntConfig(t *testing.T) {
	config := map[string]any{
		"float":  float64(42),
		"int":    7,
		"string": "8080",
		"bad":    "abc",
	}

	if got := intConfig(config, "float", 0); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
	if got := intConfig(config, "int", 0); got != 7 {
		t.Errorf("expected 7, got %d", got)
	}
	if got := intConfig(config, "string", 0); got != 8080 {
		t.Errorf("expected 8080, got %d", got)
	}
	if got := intConfig(config, "bad", 5); got != 5 {
		t.Errorf("expected default 5, got %d", got)
	}
	if got := intConfig(config, "missing", 9); got != 9 {
		t.Errorf("expected default 9, got %d", got)
	}
}

func TestIntSliceConfig(t *testing.T) {
	config := map[string]any{
		"codes": []any{float64(200), float64(204), 301},
		"empty": []any{},
	}

	got := intSliceConfig(config, "codes", []int{200})
	if len(got) != 3 || got[0] != 200 || got[1] != 204 || got[2] != 301 {
		t.Errorf("unexpected slice %v", got)
	}

	if got := intSliceConfig(config, "empty", []int{418}); len(got) != 1 || got[0] != 418 {
		t.Errorf("expected default for empty slice, got %v", got)
	}
}

func TestStringSliceConfig(t *testing.T) {
	config := map[string]any{"values": []any{"a", "b", 3}}

	got := stringSliceConfig(config, "values", nil)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("unexpected slice %v", got)
	}
}

func TestHasCommonValue(t *testing.T) {
	if !hasCommonValue([]string{"2001:db8::1"}, []string{"2001:0db8:0000:0000:0000:0000:0000:0001"}) {
		t.Error("expected IP normalization match")
	}
	if !hasCommonValue([]string{"ns1.example.com."}, []string{"NS1.example.com"}) {
		t.Error("expected case/dot-insensitive match")
	}
	if hasCommonValue([]string{"a"}, []string{"b"}) {
		t.Error("expected no match")
	}
}
