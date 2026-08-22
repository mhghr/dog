package snmp

import (
	"context"
	"strings"
	"testing"

	"monitoring-platform/packages/shared/domain"
)

func TestBuildParams_DefaultsAndValidation(t *testing.T) {
	// v2c with defaults.
	cfg := &domain.SNMPDeviceConfig{Host: "10.0.0.1", Version: domain.SNMPv2c, Community: "public"}
	params, err := BuildParams(cfg)
	if err != nil {
		t.Fatalf("BuildParams: %v", err)
	}
	if params.Port != 161 {
		t.Fatalf("default port should be 161, got %d", params.Port)
	}
	if params.Timeout.Seconds() != 3 {
		t.Fatalf("default timeout should be 3s, got %v", params.Timeout)
	}
	if params.Version != domain.SNMPv2c {
		t.Fatalf("default version should be 2c, got %v", params.Version)
	}

	// Empty host.
	if _, err := BuildParams(&domain.SNMPDeviceConfig{}); err == nil {
		t.Fatal("expected error for empty host")
	}

	// Unsupported version.
	if _, err := BuildParams(&domain.SNMPDeviceConfig{Host: "h", Version: "9"}); err == nil || !strings.Contains(err.Error(), "unsupported snmp version") {
		t.Fatalf("expected unsupported version error, got %v", err)
	}

	// v3 with authPriv requires secrets + protocols.
	cfg3 := &domain.SNMPDeviceConfig{
		Host: "10.0.0.2", Version: domain.SNMPv3, Username: "monitor",
		SecurityLevel: domain.SNMPAuthPriv, AuthenticationProto: "SHA", PrivacyProto: "AES",
		AuthenticationSecret: "s3cret", PrivacySecret: "p3riv",
	}
	if _, err := BuildParams(cfg3); err != nil {
		t.Fatalf("valid authPriv config rejected: %v", err)
	}

	// v3 authPriv missing privacy secret.
	bad := *cfg3
	bad.PrivacySecret = ""
	if _, err := BuildParams(&bad); err == nil {
		t.Fatal("expected error for authPriv without privacy secret")
	}

	// v3 authNoPriv missing auth secret.
	bad2 := *cfg3
	bad2.SecurityLevel = domain.SNMPAuthNoPriv
	bad2.PrivacySecret = ""
	bad2.AuthenticationSecret = ""
	if _, err := BuildParams(&bad2); err == nil {
		t.Fatal("expected error for authNoPriv without auth secret")
	}
}

func TestAuthPrivProtocolMapping(t *testing.T) {
	if _, ok := authProtocol("MD5"); !ok {
		t.Fatal("MD5 should be supported")
	}
	if _, ok := authProtocol("sha-256"); !ok {
		t.Fatal("SHA-256 should be supported")
	}
	if _, ok := authProtocol("SHA512"); ok {
		t.Fatal("SHA512 is not in the supported set")
	}
	if _, ok := privProtocol("AES-256"); !ok {
		t.Fatal("AES-256 should be supported")
	}
	if _, ok := privProtocol("AES192"); ok {
		t.Fatal("AES192 is not in the supported set")
	}
}

func TestErrorClassification(t *testing.T) {
	auth := ResponseErrorState(context.Canceled)
	_ = auth

	if got := DialErrorState(context.Canceled); got != domain.SNMPStateDeviceUnreachable {
		t.Fatalf("unexpected dial state for context.Canceled: %v", got)
	}

	timeoutErr := &requestTimeoutErr{}
	if got := ResponseErrorState(timeoutErr); got != domain.SNMPStateTimeout {
		t.Fatalf("expected timeout state, got %v", got)
	}

	noAccess := &fakeError{msg: "no access to that OID"}
	if got := ResponseErrorState(noAccess); got != domain.SNMPStateAuthorization {
		t.Fatalf("expected authorization state, got %v", got)
	}

	authErr := &fakeError{msg: "authentication failure (incorrect community)"}
	if got := ResponseErrorState(authErr); got != domain.SNMPStateAuthentication {
		t.Fatalf("expected authentication state, got %v", got)
	}

	unsupported := &fakeError{msg: "no such object, unsupported"}
	if got := ResponseErrorState(unsupported); got != domain.SNMPStateUnsupported {
		t.Fatalf("expected unsupported state, got %v", got)
	}
}

type requestTimeoutErr struct{}

func (*requestTimeoutErr) Error() string { return "Request timeout" }

type fakeError struct{ msg string }

func (e *fakeError) Error() string { return e.msg }

func TestSanitizeError_NoSecrets(t *testing.T) {
	// Even if an underlying error contains a credential, the sanitized
	// message must never leak it.
	secretErr := &fakeError{msg: "community 'public-string-secret' authentication failure, packet: 0xdeadbeef"}
	out := SanitizeError(secretErr)
	if strings.Contains(out, "public-string-secret") || strings.Contains(out, "0xdeadbeef") {
		t.Fatalf("sanitized error leaked sensitive material: %q", out)
	}
}
