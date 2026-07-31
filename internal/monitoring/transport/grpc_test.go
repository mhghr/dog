package transport

import (
	"crypto/tls"
	"testing"
)

func TestTLSConfigDefaults(t *testing.T) {
	cfg := TLSConfig()
	if cfg == nil {
		t.Fatal("TLSConfig() returned nil")
	}
	if cfg.MinVersion < tls.VersionTLS12 {
		t.Errorf("MinVersion = %d, want >= TLS12 (%d)", cfg.MinVersion, tls.VersionTLS12)
	}
	if len(cfg.CipherSuites) == 0 {
		t.Error("CipherSuites is empty")
	}
	hasX25519 := false
	for _, c := range cfg.CurvePreferences {
		if c == tls.X25519 {
			hasX25519 = true
		}
	}
	if !hasX25519 {
		t.Error("X25519 not in CurvePreferences")
	}
}
