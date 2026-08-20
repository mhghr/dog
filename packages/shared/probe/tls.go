package probe

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/security"
)

// TLSExecutor inspects a host:port TLS endpoint: it performs a real TLS
// handshake (with correct SNI), verifies the presented certificate chain and
// hostname when verification is enabled, and reports certificate attributes
// and expiry. Sensitive material (private keys, credentials) never appears in
// the result; certificate subject/issuer and validity windows are exposed as
// diagnostic attributes, never as high-cardinality metric labels.
type TLSExecutor struct {
	deps Deps
}

func NewTLSExecutor(deps Deps) *TLSExecutor {
	return &TLSExecutor{deps: deps}
}

func (e *TLSExecutor) Type() domain.MonitorType {
	return domain.MonitorTLS
}

func (e *TLSExecutor) Execute(ctx context.Context, job domain.ProbeJob) domain.ProbeResult {
	result := newBaseResult(job)

	host, port, err := parseTLSTarget(job)
	if err != nil {
		result.Metrics["reachability"] = 0.0
		result.Attributes["error_type"] = "invalid_target"
		return finishFailure(result, "invalid_target", err)
	}

	serverName := stringConfig(job.Config, "server_name", host)
	verifyTLS := boolConfig(job.Config, "verify_tls", true)
	// Legacy keys override verify_tls granularly.
	verifyChain := boolConfig(job.Config, "verify_chain", verifyTLS)
	verifyHostname := boolConfig(job.Config, "verify_hostname", verifyTLS)

	family := security.ParseIPFamily(stringConfig(job.Config, "ip_version", string(security.IPFamilyAuto)))
	guard := e.deps.Guard.WithIPFamily(family)

	if timeout := intConfig(job.Config, "timeout_ms", 0); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
		defer cancel()
	}

	result.Attributes["hostname"] = host
	result.Attributes["port"] = port
	result.Attributes["server_name"] = serverName
	result.Attributes["ip_version"] = string(family)

	rawConn, err := guard.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return finishTLSConnectFailure(result, err)
	}
	defer rawConn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = rawConn.SetDeadline(deadline)
	}

	var minVersion uint16 = tls.VersionTLS10
	if text := stringConfig(job.Config, "min_tls_version", ""); text != "" {
		if parsed, ok := tlsVersionFromText(text); ok {
			minVersion = parsed
		}
	}

	handshakeStart := time.Now()
	tlsConn := tls.Client(rawConn, &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: true, // verification is performed explicitly below
		MinVersion:         minVersion,
	})
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return finishTLSHandshakeFailure(result, err)
	}
	handshakeDuration := time.Since(handshakeStart)

	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		result.Metrics["reachability"] = 0.0
		return finishFailure(result, "tls_handshake_failed", fmt.Errorf("server presented no certificates"))
	}

	leaf := state.PeerCertificates[0]
	now := time.Now().UTC()
	daysRemaining := int(leaf.NotAfter.Sub(now).Hours() / 24)

	fingerprint := sha256.Sum256(leaf.Raw)
	fingerprintHex := hex.EncodeToString(fingerprint[:])

	result.Attributes["tls_version"] = tlsVersionName(state.Version)
	result.Attributes["cipher_suite"] = tls.CipherSuiteName(state.CipherSuite)
	result.Attributes["certificate_subject"] = leaf.Subject.String()
	result.Attributes["certificate_issuer"] = leaf.Issuer.String()
	result.Attributes["certificate_serial_number"] = leaf.SerialNumber.String()
	result.Attributes["certificate_not_before"] = leaf.NotBefore.UTC()
	result.Attributes["certificate_not_after"] = leaf.NotAfter.UTC()
	result.Attributes["certificate_dns_names"] = leaf.DNSNames
	result.Attributes["self_signed"] = leaf.Subject.String() == leaf.Issuer.String()
	result.Attributes["fingerprint_sha256"] = fingerprintHex
	result.Attributes["verification_enabled"] = verifyChain || verifyHostname

	certificateValid := !now.Before(leaf.NotBefore) && !now.After(leaf.NotAfter)

	result.Metrics["reachability"] = 1.0
	result.Metrics["handshake_time_ms"] = float64(handshakeDuration.Milliseconds())
	result.Metrics["certificate_expiry_days"] = float64(daysRemaining)
	result.Metrics["certificate_valid"] = float64(boolToInt(certificateValid))

	if !verifyChain && !verifyHostname {
		// Verification disabled: report the certificate facts but never claim
		// the chain or hostname is trusted. The result is still "up" because
		// the handshake itself succeeded, but verified=false is explicit and
		// the validity metrics are intentionally absent.
		result.Attributes["verified"] = false
		return finishSuccess(result)
	}

	chainValid, chainErr := verifyCertificateChain(leaf, state.PeerCertificates, now)
	hostnameValid := verifyHostnameFor(leaf, serverName)

	result.Metrics["chain_valid"] = float64(boolToInt(chainValid))
	result.Metrics["hostname_match"] = float64(boolToInt(hostnameValid))

	if now.After(leaf.NotAfter) {
		return finishTLSCertFailure(
			result,
			"certificate_expired",
			fmt.Errorf("certificate expired at %s", leaf.NotAfter.UTC().Format(time.RFC3339)),
		)
	}
	if now.Before(leaf.NotBefore) {
		return finishTLSCertFailure(
			result,
			"certificate_not_yet_valid",
			fmt.Errorf("certificate is not valid before %s", leaf.NotBefore.UTC().Format(time.RFC3339)),
		)
	}

	if verifyChain && !chainValid {
		code, message := classifyChainError(chainErr)
		return finishTLSCertFailure(result, code, fmt.Errorf("%s: %v", message, chainErr))
	}

	if verifyHostname && !hostnameValid {
		hostnameErr := leaf.VerifyHostname(serverName)
		return finishTLSCertFailure(
			result,
			"hostname_mismatch",
			fmt.Errorf("certificate is not valid for hostname %q: %v", serverName, hostnameErr),
		)
	}

	result.Attributes["verified"] = true
	return finishSuccess(result)
}

// verifyCertificateChain validates the leaf against the presented
// intermediates and the system trust store.
func verifyCertificateChain(leaf *x509.Certificate, peerCertificates []*x509.Certificate, now time.Time) (bool, error) {
	return verifyChainWith(leaf, peerCertificates, now, nil)
}

// verifyChainWith validates the leaf against the presented intermediates and
// the given root pool (nil uses the system trust store). Splitting the root
// pool out keeps the trust path testable without touching the host store.
func verifyChainWith(leaf *x509.Certificate, peerCertificates []*x509.Certificate, now time.Time, roots *x509.CertPool) (bool, error) {
	intermediates := x509.NewCertPool()
	for _, certificate := range peerCertificates[1:] {
		intermediates.AddCert(certificate)
	}

	_, err := leaf.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		CurrentTime:   now,
	})
	return err == nil, err
}

func verifyHostnameFor(leaf *x509.Certificate, serverName string) bool {
	return leaf.VerifyHostname(serverName) == nil
}

// classifyChainError maps certificate chain verification failures to the
// deterministic TLS error taxonomy.
func classifyChainError(err error) (code, message string) {
	if err == nil {
		return "tls_handshake_failed", "certificate chain verification failed"
	}

	var unknownCA x509.UnknownAuthorityError
	if errors.As(err, &unknownCA) {
		return "unknown_ca", "certificate is signed by an unknown authority"
	}

	var certErr x509.CertificateInvalidError
	if errors.As(err, &certErr) {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "has expired") {
			return "certificate_expired", "certificate has expired"
		}
		if strings.Contains(lower, "not yet valid") {
			return "certificate_not_yet_valid", "certificate is not yet valid"
		}
		return "unknown_ca", "certificate chain is not trusted"
	}

	return "unknown_ca", "certificate chain is not trusted"
}

// finishTLSConnectFailure handles dial-level failures (refused, unreachable,
// timeout, blocked target). Reachability is always 0.
func finishTLSConnectFailure(result domain.ProbeResult, err error) domain.ProbeResult {
	result.Metrics["reachability"] = 0.0

	code := classifyDialError(err)
	if isBlockedError(err) {
		code = "blocked_target"
	}
	result.Attributes["error_type"] = code

	return finishFailure(result, code, err)
}

// finishTLSHandshakeFailure classifies TLS handshake errors into the
// deterministic taxonomy. Reachability is 0; handshake time is never
// fabricated for a handshake that never completed.
func finishTLSHandshakeFailure(result domain.ProbeResult, err error) domain.ProbeResult {
	result.Metrics["reachability"] = 0.0

	code := "tls_handshake_failed"
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		code = "timeout"
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "protocol version") || strings.Contains(lower, "incompatible version") ||
		strings.Contains(lower, "version not supported") {
		code = "protocol_version_error"
	}

	result.Attributes["error_type"] = code
	return finishFailure(result, code, err)
}

// finishTLSCertFailure marks a certificate-level failure while preserving the
// already-recorded certificate attributes and metrics.
func finishTLSCertFailure(result domain.ProbeResult, code string, err error) domain.ProbeResult {
	result.Metrics["reachability"] = 0.0
	result.Attributes["error_type"] = code
	result.Attributes["verified"] = false
	return finishFailure(result, code, err)
}

// parseTLSTarget resolves the target host and port. The resource target may
// carry a port (host:port); otherwise the port comes from the configuration
// and defaults to 443.
func parseTLSTarget(job domain.ProbeJob) (string, int, error) {
	if host, portRaw, err := net.SplitHostPort(job.Target); err == nil {
		port, err := parsePort(portRaw)
		if err != nil {
			return "", 0, err
		}
		return host, port, nil
	}

	host := job.Target
	if configuredHost := stringConfig(job.Config, "host", ""); configuredHost != "" {
		host = configuredHost
	}
	if host == "" {
		return "", 0, fmt.Errorf("TLS target host is required")
	}

	port, err := parsePort(strconv.Itoa(intConfig(job.Config, "port", 443)))
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}

func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "1.0"
	case tls.VersionTLS11:
		return "1.1"
	case tls.VersionTLS12:
		return "1.2"
	case tls.VersionTLS13:
		return "1.3"
	default:
		return fmt.Sprintf("unknown (0x%04x)", version)
	}
}

func tlsVersionFromText(text string) (uint16, bool) {
	switch text {
	case "1.0":
		return tls.VersionTLS10, true
	case "1.1":
		return tls.VersionTLS11, true
	case "1.2":
		return tls.VersionTLS12, true
	case "1.3":
		return tls.VersionTLS13, true
	default:
		return 0, false
	}
}
