package probe

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"

	"monitoring-platform/internal/domain"
)

// startFakeTLSServer serves a self-signed certificate valid for validityDays.
func startFakeTLSServer(t *testing.T, validityDays int) string {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost", Organization: []string{"Test"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Duration(validityDays) * 24 * time.Hour),
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certificateDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	certificate := tls.Certificate{
		Certificate: [][]byte{certificateDER},
		PrivateKey:  privateKey,
	}

	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS12,
	})
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
				buffer := make([]byte, 1)
				_, _ = conn.Read(buffer)
				conn.Close()
			}(conn)
		}
	}()

	t.Cleanup(func() { _ = listener.Close() })

	return listener.Addr().String()
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

func TestTLSExecutorSelfSignedInspection(t *testing.T) {
	address := startFakeTLSServer(t, 90)

	executor := NewTLSExecutor(testDeps())
	result := executor.Execute(execCtx(t), tlsJobFor(address, map[string]any{
		"verify_chain":    false,
		"verify_hostname": true,
	}))

	if !result.Success {
		t.Fatalf("expected success, got %+v (error: %s)", result, result.ErrorMessage)
	}

	if result.Attributes["self_signed"] != true {
		t.Fatalf("expected self_signed=true, got %v", result.Attributes["self_signed"])
	}

	days, ok := result.Metrics["days_remaining"].(int)
	if !ok || days < 80 || days > 90 {
		t.Fatalf("unexpected days_remaining %v", result.Metrics["days_remaining"])
	}

	if result.Attributes["fingerprint_sha256"] == "" {
		t.Fatal("expected fingerprint attribute")
	}
}

func TestTLSExecutorChainValidationFailsForSelfSigned(t *testing.T) {
	address := startFakeTLSServer(t, 90)

	executor := NewTLSExecutor(testDeps())
	result := executor.Execute(execCtx(t), tlsJobFor(address, map[string]any{
		"verify_chain": true,
	}))

	if result.Success || result.ErrorCode != "tls_chain_invalid" {
		t.Fatalf("expected tls_chain_invalid, got %+v", result)
	}
}

func TestTLSExecutorExpiryCritical(t *testing.T) {
	address := startFakeTLSServer(t, 3)

	executor := NewTLSExecutor(testDeps())
	result := executor.Execute(execCtx(t), tlsJobFor(address, map[string]any{
		"verify_chain":  false,
		"critical_days": float64(7),
	}))

	if result.Success || result.ErrorCode != "tls_certificate_expiring" {
		t.Fatalf("expected tls_certificate_expiring, got %+v", result)
	}
}

func TestTLSExecutorHostnameMismatch(t *testing.T) {
	address := startFakeTLSServer(t, 90)

	executor := NewTLSExecutor(testDeps())
	result := executor.Execute(execCtx(t), tlsJobFor(address, map[string]any{
		"verify_chain":    false,
		"verify_hostname": true,
		"server_name":     "wrong.example.com",
	}))

	if result.Success || result.ErrorCode != "tls_hostname_invalid" {
		t.Fatalf("expected tls_hostname_invalid, got %+v", result)
	}
}
