package probe

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"monitoring-platform/packages/shared/domain"
)

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

	host := job.Target
	port := intConfig(job.Config, "port", 443)
	serverName := stringConfig(job.Config, "server_name", host)
	verifyChain := boolConfig(job.Config, "verify_chain", true)
	verifyHostname := boolConfig(job.Config, "verify_hostname", true)
	warningDays := intConfig(job.Config, "warning_days", 30)
	criticalDays := intConfig(job.Config, "critical_days", 7)

	address := net.JoinHostPort(host, strconv.Itoa(port))

	rawConn, err := e.deps.Guard.DialContext(ctx, "tcp", address)
	if err != nil {
		return finishFailure(result, "tls_connect_failed", err)
	}
	defer rawConn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = rawConn.SetDeadline(deadline)
	}

	handshakeStart := time.Now()
	tlsConn := tls.Client(rawConn, &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: true, // verification is performed explicitly below
		MinVersion:         tls.VersionTLS10,
	})

	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return finishFailure(result, "tls_handshake_failed", err)
	}
	handshakeDuration := time.Since(handshakeStart)

	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return finishFailure(result, "tls_handshake_failed", fmt.Errorf("server presented no certificates"))
	}

	leaf := state.PeerCertificates[0]
	now := time.Now()

	fingerprint := sha256.Sum256(leaf.Raw)
	fingerprintHex := hex.EncodeToString(fingerprint[:])

	daysRemaining := int(leaf.NotAfter.Sub(now).Hours() / 24)

	selfSigned := leaf.Subject.String() == leaf.Issuer.String()

	chainValid := true
	var chainErr error
	if verifyChain {
		intermediates := x509.NewCertPool()
		for _, certificate := range state.PeerCertificates[1:] {
			intermediates.AddCert(certificate)
		}

		_, chainErr = leaf.Verify(x509.VerifyOptions{
			Intermediates: intermediates,
			CurrentTime:   now,
		})
		chainValid = chainErr == nil
	}

	hostnameValid := true
	var hostnameErr error
	if verifyHostname {
		hostnameErr = leaf.VerifyHostname(serverName)
		hostnameValid = hostnameErr == nil
	}

	result.Attributes["issuer"] = leaf.Issuer.String()
	result.Attributes["subject"] = leaf.Subject.String()
	result.Attributes["dns_names"] = leaf.DNSNames
	result.Attributes["not_before"] = leaf.NotBefore.UTC()
	result.Attributes["not_after"] = leaf.NotAfter.UTC()
	result.Attributes["tls_version"] = tlsVersionName(state.Version)
	result.Attributes["cipher_suite"] = tls.CipherSuiteName(state.CipherSuite)
	result.Attributes["fingerprint_sha256"] = fingerprintHex
	result.Attributes["self_signed"] = selfSigned
	result.Attributes["chain_valid"] = chainValid
	result.Attributes["hostname_valid"] = hostnameValid
	result.Attributes["days_remaining"] = daysRemaining

	result.Metrics["handshake_duration_ms"] = handshakeDuration.Milliseconds()
	result.Metrics["days_remaining"] = daysRemaining
	result.Metrics["certificate_valid"] = boolToInt(chainValid && hostnameValid && now.Before(leaf.NotAfter))
	result.Metrics["hostname_valid"] = boolToInt(hostnameValid)
	result.Metrics["chain_valid"] = boolToInt(chainValid)

	if now.After(leaf.NotAfter) {
		return finishFailure(
			result,
			"tls_certificate_expired",
			fmt.Errorf("certificate expired at %s", leaf.NotAfter.UTC().Format(time.RFC3339)),
		)
	}

	if now.Before(leaf.NotBefore) {
		return finishFailure(
			result,
			"tls_chain_invalid",
			fmt.Errorf("certificate is not valid before %s", leaf.NotBefore.UTC().Format(time.RFC3339)),
		)
	}

	if verifyChain && !chainValid {
		return finishFailure(result, "tls_chain_invalid", chainErr)
	}

	if verifyHostname && !hostnameValid {
		return finishFailure(result, "tls_hostname_invalid", hostnameErr)
	}

	minVersionText := stringConfig(job.Config, "minimum_tls_version", "1.2")
	if minVersion, ok := tlsVersionFromText(minVersionText); ok && state.Version < minVersion {
		return finishFailure(
			result,
			"tls_version_too_old",
			fmt.Errorf("negotiated %s is older than required TLS %s", tlsVersionName(state.Version), minVersionText),
		)
	}

	if expectedIssuer := stringConfig(job.Config, "expected_issuer_contains", ""); expectedIssuer != "" {
		if !strings.Contains(strings.ToLower(leaf.Issuer.String()), strings.ToLower(expectedIssuer)) {
			return finishFailure(
				result,
				"tls_issuer_mismatch",
				fmt.Errorf("issuer %q does not contain %q", leaf.Issuer.String(), expectedIssuer),
			)
		}
	}

	if expectedFingerprint := stringConfig(job.Config, "expected_fingerprint_sha256", ""); expectedFingerprint != "" {
		normalized := strings.ToLower(strings.ReplaceAll(expectedFingerprint, ":", ""))
		if normalized != fingerprintHex {
			return finishFailure(
				result,
				"tls_fingerprint_mismatch",
				fmt.Errorf("certificate fingerprint does not match the expected value"),
			)
		}
	}

	if daysRemaining <= criticalDays {
		return finishFailure(
			result,
			"tls_certificate_expiring",
			fmt.Errorf("certificate expires in %d days (critical threshold %d)", daysRemaining, criticalDays),
		)
	}

	if daysRemaining <= warningDays {
		result.Attributes["expiry_warning"] = true
	}

	return finishSuccess(result)
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

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
