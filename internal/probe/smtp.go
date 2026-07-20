package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"monitoring-platform/internal/domain"
)

type SMTPExecutor struct {
	deps Deps
}

func NewSMTPExecutor(deps Deps) *SMTPExecutor {
	return &SMTPExecutor{deps: deps}
}

func (e *SMTPExecutor) Type() domain.MonitorType {
	return domain.MonitorSMTP
}

// Execute performs a full SMTP handshake without authenticating or sending
// mail: connect, banner, EHLO, optional STARTTLS, NOOP, QUIT.
func (e *SMTPExecutor) Execute(ctx context.Context, job domain.ProbeJob) domain.ProbeResult {
	result := newBaseResult(job)

	host := job.Target
	mode := strings.ToLower(stringConfig(job.Config, "mode", "plain"))

	defaultPort := 25
	switch mode {
	case "starttls":
		defaultPort = 587
	case "implicit_tls":
		defaultPort = 465
	}

	port := intConfig(job.Config, "port", defaultPort)
	ehloDomain := stringConfig(job.Config, "ehlo_domain", "monitoring.local")
	verifyTLS := boolConfig(job.Config, "verify_tls", true)
	requireStartTLS := boolConfig(job.Config, "require_starttls", mode == "starttls")

	address := net.JoinHostPort(host, strconv.Itoa(port))

	connectStart := time.Now()
	conn, err := e.deps.Guard.DialContext(ctx, "tcp", address)
	if err != nil {
		return finishFailure(result, "smtp_connect_failed", err)
	}
	defer conn.Close()
	result.Metrics["connect_duration_ms"] = time.Since(connectStart).Milliseconds()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	tlsConfig := &tls.Config{
		ServerName:         host,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: !verifyTLS,
	}

	if mode == "implicit_tls" {
		tlsConn := tls.Client(conn, tlsConfig)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			return finishFailure(result, "smtp_tls_invalid", err)
		}
		conn = tlsConn
		result.Attributes["tls_version"] = tlsVersionName(tlsConn.ConnectionState().Version)
	}

	text := textproto.NewConn(conn)

	bannerStart := time.Now()
	code, banner, err := readSMTPResponse(text)
	if err != nil {
		return finishFailure(result, "smtp_banner_timeout", err)
	}
	result.Metrics["banner_duration_ms"] = time.Since(bannerStart).Milliseconds()
	result.Attributes["banner"] = banner

	if code != 220 {
		return finishFailure(result, "smtp_invalid_banner", fmt.Errorf("expected 220 banner, received %d %s", code, banner))
	}

	if expectedBanner := stringConfig(job.Config, "expected_banner_contains", ""); expectedBanner != "" {
		if !strings.Contains(banner, expectedBanner) {
			return finishFailure(
				result,
				"smtp_invalid_banner",
				fmt.Errorf("banner does not contain %q", expectedBanner),
			)
		}
	}

	ehloStart := time.Now()
	capabilities, err := smtpEHLO(text, ehloDomain)
	if err != nil {
		return finishFailure(result, "smtp_ehlo_failed", err)
	}
	result.Metrics["ehlo_duration_ms"] = time.Since(ehloStart).Milliseconds()

	startTLSAvailable := capabilities["STARTTLS"]
	result.Metrics["starttls_available"] = boolToInt(startTLSAvailable)

	if mode == "starttls" {
		if !startTLSAvailable {
			if requireStartTLS {
				return finishFailure(result, "smtp_starttls_unavailable", fmt.Errorf("server does not advertise STARTTLS"))
			}
		} else {
			startTLSStart := time.Now()

			if err := text.PrintfLine("STARTTLS"); err != nil {
				return finishFailure(result, "smtp_starttls_failed", err)
			}
			code, message, err := readSMTPResponse(text)
			if err != nil || code != 220 {
				if err == nil {
					err = fmt.Errorf("STARTTLS rejected: %d %s", code, message)
				}
				return finishFailure(result, "smtp_starttls_failed", err)
			}

			tlsConn := tls.Client(conn, tlsConfig)
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				return finishFailure(result, "smtp_tls_invalid", err)
			}
			conn = tlsConn
			text = textproto.NewConn(conn)
			result.Attributes["tls_version"] = tlsVersionName(tlsConn.ConnectionState().Version)
			result.Metrics["starttls_duration_ms"] = time.Since(startTLSStart).Milliseconds()

			capabilities, err = smtpEHLO(text, ehloDomain)
			if err != nil {
				return finishFailure(result, "smtp_ehlo_failed", err)
			}
		}
	}

	capabilityNames := make([]string, 0, len(capabilities))
	for capability := range capabilities {
		capabilityNames = append(capabilityNames, capability)
	}
	result.Attributes["capabilities"] = capabilityNames

	for _, expected := range stringSliceConfig(job.Config, "expected_capabilities", nil) {
		if !capabilities[strings.ToUpper(strings.TrimSpace(expected))] {
			return finishFailure(
				result,
				"smtp_capability_missing",
				fmt.Errorf("capability %q is not advertised", expected),
			)
		}
	}

	if err := text.PrintfLine("NOOP"); err != nil {
		return finishFailure(result, "smtp_noop_failed", err)
	}
	if code, message, err := readSMTPResponse(text); err != nil || code != 250 {
		if err == nil {
			err = fmt.Errorf("NOOP rejected: %d %s", code, message)
		}
		return finishFailure(result, "smtp_noop_failed", err)
	}

	_ = text.PrintfLine("QUIT")
	_, _, _ = readSMTPResponse(text)

	return finishSuccess(result)
}

func smtpEHLO(text *textproto.Conn, ehloDomain string) (map[string]bool, error) {
	if err := text.PrintfLine("EHLO %s", ehloDomain); err != nil {
		return nil, err
	}

	code, message, err := readSMTPResponse(text)
	if err != nil {
		return nil, err
	}
	if code != 250 {
		return nil, fmt.Errorf("EHLO rejected: %d %s", code, message)
	}

	capabilities := make(map[string]bool)
	lines := strings.Split(message, "\n")
	for index, line := range lines {
		if index == 0 {
			continue // first line is the server greeting
		}
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) > 0 {
			capabilities[strings.ToUpper(fields[0])] = true
		}
	}

	return capabilities, nil
}

func readSMTPResponse(text *textproto.Conn) (int, string, error) {
	code, message, err := text.ReadResponse(-1)
	if err != nil {
		if protocolErr, ok := err.(*textproto.Error); ok {
			return protocolErr.Code, protocolErr.Msg, nil
		}
		return 0, "", err
	}

	return code, message, nil
}
