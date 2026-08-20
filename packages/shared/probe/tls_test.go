package probe

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"testing"
	"time"

	"monitoring-platform/packages/shared/domain"
)

// ── Certificate / TLS server helpers ──────────────────────────────────────

type testCertificate struct {
	tls  tls.Certificate
	leaf *x509.Certificate
	key  *ecdsa.PrivateKey
}

func generateTestKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}

// newCertificate creates a certificate from a template. When parent is nil the
// certificate is self-signed; otherwise it is signed by the parent.
func newCertificate(t *testing.T, template, parent *x509.Certificate, parentKey *ecdsa.PrivateKey) testCertificate {
	t.Helper()

	key := generateTestKey(t)
	if parent == nil {
		parent = template
		parentKey = key
	}

	der, err := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, parentKey)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}

	return testCertificate{
		tls: tls.Certificate{
			Certificate: [][]byte{der},
			PrivateKey:  key,
		},
		leaf: leaf,
		key:  key,
	}
}

func certTemplate(commonName string, dnsNames []string, notBefore, notAfter time.Time, isCA bool) *x509.Certificate {
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: commonName, Organization: []string{"Test Org"}},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	if !isCA {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}
	if len(dnsNames) > 0 {
		template.DNSNames = dnsNames
	}
	if isCA {
		template.IsCA = true
		template.BasicConstraintsValid = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}
	return template
}

func selfSignedLeaf(t *testing.T, commonName string, dnsNames []string, notBefore, notAfter time.Time) testCertificate {
	t.Helper()
	return newCertificate(t, certTemplate(commonName, dnsNames, notBefore, notAfter, false), nil, nil)
}

// startTLSServer serves a tls.Config over TCP on 127.0.0.1.
func startTLSServer(t *testing.T, config *tls.Config) (net.Listener, string) {
	t.Helper()

	listener, err := tls.Listen("tcp", "127.0.0.1:0", config)
	if err != nil {
		t.Fatalf("tls listen: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				buffer := make([]byte, 1)
				_, _ = conn.Read(buffer)
			}(conn)
		}
	}()

	t.Cleanup(func() { _ = listener.Close() })
	return listener, listener.Addr().String()
}

func tlsJobFor(address string, config map[string]any) domain.ProbeJob {
	host, portRaw, _ := net.SplitHostPort(address)
	if config == nil {
		config = map[string]any{}
	}
	config["port"] = portRaw
	if _, exists := config["server_name"]; !exists {
		config["server_name"] = "localhost"
	}

	return testJob(domain.MonitorTLS, host, config)
}

// ── Executor behavior ─────────────────────────────────────────────────────

func TestTLSExecutorValidCertificateInspection(t *testing.T) {
	cert := selfSignedLeaf(t, "localhost", []string{"localhost"}, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))
	listener, address := startTLSServer(t, &tls.Config{Certificates: []tls.Certificate{cert.tls}})
	_ = listener

	executor := NewTLSExecutor(testDeps())
	result := executor.Execute(execCtx(t), tlsJobFor(address, map[string]any{
		"verify_tls": false,
	}))

	if !result.Success {
		t.Fatalf("expected success, got %+v (error: %s)", result, result.ErrorMessage)
	}
	if result.Metrics["reachability"] != 1.0 {
		t.Fatalf("expected reachability=1, got %v", result.Metrics["reachability"])
	}
	if _, ok := result.Metrics["handshake_time_ms"]; !ok {
		t.Fatal("expected handshake_time_ms metric")
	}
	days, ok := result.Metrics["certificate_expiry_days"].(float64)
	if !ok || days < 80 || days > 90 {
		t.Fatalf("unexpected certificate_expiry_days %v", result.Metrics["certificate_expiry_days"])
	}
	if result.Attributes["certificate_subject"] == nil {
		t.Fatal("expected certificate_subject attribute")
	}
	if result.Attributes["certificate_issuer"] == nil {
		t.Fatal("expected certificate_issuer attribute")
	}
	if result.Attributes["certificate_not_after"] == nil {
		t.Fatal("expected certificate_not_after attribute")
	}
	if result.Attributes["tls_version"] == nil || result.Attributes["tls_version"] == "" {
		t.Fatal("expected tls_version attribute")
	}
	if result.Attributes["cipher_suite"] == nil {
		t.Fatal("expected cipher_suite attribute")
	}
	if result.Attributes["fingerprint_sha256"] == nil || result.Attributes["fingerprint_sha256"] == "" {
		t.Fatal("expected fingerprint_sha256 attribute")
	}
	if result.Attributes["certificate_serial_number"] == nil || result.Attributes["certificate_serial_number"] == "" {
		t.Fatal("expected certificate_serial_number attribute")
	}
	if result.Metrics["certificate_valid"] != 1.0 {
		t.Fatalf("expected certificate_valid=1, got %v", result.Metrics["certificate_valid"])
	}
	// Verification disabled must be explicit, never implied trusted.
	if result.Attributes["verified"] != false {
		t.Fatalf("expected verified=false when verification disabled, got %v", result.Attributes["verified"])
	}
	if result.Attributes["verification_enabled"] != false {
		t.Fatalf("expected verification_enabled=false, got %v", result.Attributes["verification_enabled"])
	}
	// Chain/hostname metrics are intentionally absent when verification is off.
	if _, ok := result.Metrics["chain_valid"]; ok {
		t.Fatalf("chain_valid must not be emitted when verification is disabled: %v", result.Metrics)
	}
}

func TestTLSExecutorValidationMetrics(t *testing.T) {
	cert := selfSignedLeaf(t, "localhost", []string{"localhost"}, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))
	_, address := startTLSServer(t, &tls.Config{Certificates: []tls.Certificate{cert.tls}})

	executor := NewTLSExecutor(testDeps())

	// Hostname-only verification on a matching cert: success with hostname_match=1.
	ok := executor.Execute(execCtx(t), tlsJobFor(address, map[string]any{
		"verify_chain":    false,
		"verify_hostname": true,
	}))
	if !ok.Success {
		t.Fatalf("expected success, got %+v", ok)
	}
	if ok.Metrics["hostname_match"] != 1.0 {
		t.Fatalf("expected hostname_match=1, got %v", ok.Metrics["hostname_match"])
	}
	if ok.Metrics["certificate_valid"] != 1.0 {
		t.Fatalf("expected certificate_valid=1, got %v", ok.Metrics["certificate_valid"])
	}

	// Hostname mismatch → failure with hostname_match=0, certificate_valid=1.
	mismatch := executor.Execute(execCtx(t), tlsJobFor(address, map[string]any{
		"verify_chain":    false,
		"verify_hostname": true,
		"server_name":     "wrong.example.com",
	}))
	if mismatch.Success || mismatch.ErrorCode != "hostname_mismatch" {
		t.Fatalf("expected hostname_mismatch, got %+v", mismatch)
	}
	if mismatch.Metrics["hostname_match"] != 0.0 {
		t.Fatalf("expected hostname_match=0, got %v", mismatch.Metrics["hostname_match"])
	}
	if mismatch.Metrics["certificate_valid"] != 1.0 {
		t.Fatalf("expected certificate_valid=1, got %v", mismatch.Metrics["certificate_valid"])
	}
}

func TestTLSExecutorVerificationDisabledExplicitlyUnverified(t *testing.T) {
	cert := selfSignedLeaf(t, "localhost", []string{"localhost"}, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))
	_, address := startTLSServer(t, &tls.Config{Certificates: []tls.Certificate{cert.tls}})

	executor := NewTLSExecutor(testDeps())
	result := executor.Execute(execCtx(t), tlsJobFor(address, map[string]any{
		"verify_tls": false,
	}))

	if result.Attributes["verified"] != false {
		t.Fatalf("expected verified=false, got %v", result.Attributes["verified"])
	}
	if result.Attributes["verification_enabled"] != false {
		t.Fatalf("expected verification_enabled=false, got %v", result.Attributes["verification_enabled"])
	}
	if result.Success != true {
		t.Fatalf("handshake succeeded so result must be up even when unverified, got %+v", result)
	}
}

func TestTLSExecutorUnknownCA(t *testing.T) {
	cert := selfSignedLeaf(t, "localhost", []string{"localhost"}, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))
	_, address := startTLSServer(t, &tls.Config{Certificates: []tls.Certificate{cert.tls}})

	executor := NewTLSExecutor(testDeps())
	result := executor.Execute(execCtx(t), tlsJobFor(address, map[string]any{
		"verify_tls": true,
	}))

	if result.Success || result.ErrorCode != "unknown_ca" {
		t.Fatalf("expected unknown_ca, got %+v", result)
	}
	if result.Attributes["verified"] != false {
		t.Fatalf("expected verified=false, got %v", result.Attributes["verified"])
	}
	if result.Metrics["chain_valid"] != 0.0 {
		t.Fatalf("expected chain_valid=0 for untrusted chain, got %v", result.Metrics["chain_valid"])
	}
	if result.Metrics["hostname_match"] != 1.0 {
		t.Fatalf("expected hostname_match=1, got %v", result.Metrics["hostname_match"])
	}
	if result.Metrics["certificate_valid"] != 1.0 {
		t.Fatalf("expected certificate_valid=1, got %v", result.Metrics["certificate_valid"])
	}
}

func TestTLSExecutorExpiredCertificate(t *testing.T) {
	cert := selfSignedLeaf(t, "localhost", []string{"localhost"}, time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour))
	_, address := startTLSServer(t, &tls.Config{Certificates: []tls.Certificate{cert.tls}})

	executor := NewTLSExecutor(testDeps())
	result := executor.Execute(execCtx(t), tlsJobFor(address, map[string]any{
		"verify_tls": true,
	}))

	if result.Success || result.ErrorCode != "certificate_expired" {
		t.Fatalf("expected certificate_expired, got %+v", result)
	}
	if result.Metrics["certificate_expiry_days"].(float64) >= 0 {
		t.Fatalf("expected negative expiry days, got %v", result.Metrics["certificate_expiry_days"])
	}
	if result.Metrics["certificate_valid"] != 0.0 {
		t.Fatalf("expected certificate_valid=0 for expired certificate, got %v", result.Metrics["certificate_valid"])
	}
}

func TestTLSExecutorNotYetValidCertificate(t *testing.T) {
	cert := selfSignedLeaf(t, "localhost", []string{"localhost"}, time.Now().Add(24*time.Hour), time.Now().Add(90*24*time.Hour))
	_, address := startTLSServer(t, &tls.Config{Certificates: []tls.Certificate{cert.tls}})

	executor := NewTLSExecutor(testDeps())
	result := executor.Execute(execCtx(t), tlsJobFor(address, map[string]any{
		"verify_tls": true,
	}))

	if result.Success || result.ErrorCode != "certificate_not_yet_valid" {
		t.Fatalf("expected certificate_not_yet_valid, got %+v", result)
	}
}

func TestTLSExecutorHostnameMismatch(t *testing.T) {
	cert := selfSignedLeaf(t, "localhost", []string{"localhost"}, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))
	_, address := startTLSServer(t, &tls.Config{Certificates: []tls.Certificate{cert.tls}})

	executor := NewTLSExecutor(testDeps())
	result := executor.Execute(execCtx(t), tlsJobFor(address, map[string]any{
		"verify_chain":    false,
		"verify_hostname": true,
		"server_name":     "wrong.example.com",
	}))

	if result.Success || result.ErrorCode != "hostname_mismatch" {
		t.Fatalf("expected hostname_mismatch, got %+v", result)
	}
}

func TestTLSExecutorHandshakeFailure(t *testing.T) {
	// A plain TCP listener never speaks TLS: the handshake must fail.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				buffer := make([]byte, 256)
				_, _ = conn.Read(buffer)
				conn.Close()
			}(conn)
		}
	}()

	executor := NewTLSExecutor(testDeps())
	result := executor.Execute(execCtx(t), tlsJobFor(listener.Addr().String(), map[string]any{}))

	if result.Success || result.ErrorCode != "tls_handshake_failed" {
		t.Fatalf("expected tls_handshake_failed, got %+v", result)
	}
	if result.Metrics["reachability"] != 0.0 {
		t.Fatalf("expected reachability=0, got %v", result.Metrics["reachability"])
	}
}

func TestTLSExecutorHandshakeTimeout(t *testing.T) {
	// A listener that accepts and never responds to the ClientHello.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// Hold the connection open without reading: the client's
			// handshake will wait until the per-check timeout expires.
			go func(conn net.Conn) {
				time.Sleep(5 * time.Second)
				conn.Close()
			}(conn)
		}
	}()

	executor := NewTLSExecutor(testDeps())
	result := executor.Execute(execCtx(t), tlsJobFor(listener.Addr().String(), map[string]any{
		"timeout_ms": float64(150),
	}))

	if result.Success {
		t.Fatal("expected failure on handshake timeout")
	}
	if result.ErrorCode != "timeout" && result.ErrorCode != "tls_handshake_failed" {
		t.Fatalf("expected timeout, got %s", result.ErrorCode)
	}
}

func TestTLSExecutorSNI(t *testing.T) {
	cert := selfSignedLeaf(t, "localhost", []string{"localhost"}, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))

	// The server only completes the handshake when the correct SNI is sent.
	_, address := startTLSServer(t, &tls.Config{
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			if hello.ServerName != "localhost" {
				return nil, errors.New("unexpected SNI")
			}
			return &tls.Config{Certificates: []tls.Certificate{cert.tls}}, nil
		},
	})

	executor := NewTLSExecutor(testDeps())
	good := executor.Execute(execCtx(t), tlsJobFor(address, map[string]any{
		"verify_tls": false,
		"server_name": "localhost",
	}))
	if !good.Success {
		t.Fatalf("expected success with matching SNI, got %+v", good)
	}

	bad := executor.Execute(execCtx(t), tlsJobFor(address, map[string]any{
		"verify_tls": false,
		"server_name": "other.example.com",
	}))
	if bad.Success {
		t.Fatal("expected failure when SNI does not match the virtual host")
	}
}

func TestTLSExecutorDefaultPort443FromConfig(t *testing.T) {
	// With no port in config, the executor must default to 443. Pointing the
	// check at 127.0.0.1 (a non-TLS port) should yield a connection-level
	// failure, not a config error, proving the default was applied.
	executor := NewTLSExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTLS, "127.0.0.1", map[string]any{
		"timeout_ms": float64(150),
		"server_name": "localhost",
	}))

	if result.Success {
		t.Fatal("expected failure connecting to 127.0.0.1:443")
	}
	if result.ErrorCode == "invalid_target" {
		t.Fatalf("port default not applied: %+v", result)
	}
	if result.Attributes["port"] != 443 {
		t.Fatalf("expected port 443, got %v", result.Attributes["port"])
	}
}

func TestTLSExecutorIPv6Selection(t *testing.T) {
	key := generateTestKey(t)
	leaf := newCertificate(t, certTemplate("localhost", []string{"localhost"}, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour), false), nil, nil)
	_ = key
	_ = leaf

	listener, err := tls.Listen("tcp", "[::1]:0", &tls.Config{
		Certificates: []tls.Certificate{leaf.tls},
	})
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				buffer := make([]byte, 1)
				_, _ = conn.Read(buffer)
				conn.Close()
			}(conn)
		}
	}()

	host, portRaw, _ := net.SplitHostPort(listener.Addr().String())

	executor := NewTLSExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTLS, host, map[string]any{
		"port":        portRaw,
		"verify_tls":  false,
		"server_name": "localhost",
		"ip_version":  "ipv6",
	}))

	if !result.Success {
		t.Fatalf("expected ipv6 success, got %+v", result)
	}
}

func TestTLSExecutorIPv4ForcesFailureOnIPv6OnlyTarget(t *testing.T) {
	leaf := selfSignedLeaf(t, "localhost", []string{"localhost"}, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))
	listener, err := tls.Listen("tcp", "[::1]:0", &tls.Config{Certificates: []tls.Certificate{leaf.tls}})
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	defer listener.Close()

	host, portRaw, _ := net.SplitHostPort(listener.Addr().String())

	executor := NewTLSExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTLS, host, map[string]any{
		"port":        portRaw,
		"verify_tls":  false,
		"server_name": "localhost",
		"ip_version":  "ipv4",
	}))

	if result.Success {
		t.Fatal("expected failure when family filter excludes all addresses")
	}
	if result.ErrorCode != "blocked_target" && result.ErrorCode != "connection_failed" {
		t.Fatalf("unexpected error code %s", result.ErrorCode)
	}
}

// ── Security: private / reserved destinations are blocked at dial time ────

func TestTLSExecutorBlocksPrivateTargets(t *testing.T) {
	executor := NewTLSExecutor(restrictiveDeps())

	cases := []struct {
		name   string
		target string
	}{
		{"loopback", "127.0.0.1:443"},
		{"private ipv4", "10.0.0.1:443"},
		{"private ipv4 c", "192.168.1.1:443"},
		{"link-local ipv4", "169.254.169.254:443"},
		{"ipv6 loopback", "[::1]:443"},
		{"ipv6 unique local", "[fc00::1]:443"},
		{"ipv6 link-local", "[fe80::1]:443"},
		{"ipv6 multicast", "[ff02::1]:443"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := executor.Execute(execCtx(t), testJob(domain.MonitorTLS, tc.target, nil))
			if result.Success {
				t.Fatalf("expected blocked failure, got success: %+v", result)
			}
			if result.ErrorCode != "blocked_target" {
				t.Fatalf("expected blocked_target, got %s (error: %s)", result.ErrorCode, result.ErrorMessage)
			}
		})
	}
}

// ── Chain verification against a trusted root (positive path) ─────────────

func TestVerifyChainWithTrustedRoot(t *testing.T) {
	now := time.Now()
	ca := newCertificate(t, certTemplate("Test CA", nil, now.Add(-24*time.Hour), now.Add(365*24*time.Hour), true), nil, nil)

	leaf := newCertificate(t,
		certTemplate("localhost", []string{"localhost"}, now.Add(-time.Hour), now.Add(90*24*time.Hour), false),
		ca.leaf, ca.key,
	)

	roots := x509.NewCertPool()
	roots.AddCert(ca.leaf)

	valid, err := verifyChainWith(leaf.leaf, []*x509.Certificate{leaf.leaf}, now, roots)
	if err != nil || !valid {
		t.Fatalf("expected trusted chain to verify, valid=%v err=%v", valid, err)
	}

	invalid, err := verifyChainWith(leaf.leaf, []*x509.Certificate{leaf.leaf}, now, x509.NewCertPool())
	if invalid || err == nil {
		t.Fatalf("expected untrusted chain to fail verification, valid=%v err=%v", invalid, err)
	}
}
