package probe

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"time"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/security"
)

// defaultMaxResponseBodyBytes caps how much of a response body a single
// check may download. Overridable per monitor via "max_response_size_bytes".
const defaultMaxResponseBodyBytes = 10 * 1024 * 1024 // 10 MB

type HTTPExecutor struct {
	deps Deps

	// transports pools guarded HTTP transports keyed by the TLS/family
	// configuration so connections (keep-alive, TLS sessions) are reused
	// across checks instead of one TCP/TLS handshake per execution.
	transports sync.Map // key: httpTransportKey, value: *http.Transport
}

type httpTransportKey struct {
	verifyTLS bool
	family    security.IPFamily
}

func NewHTTPExecutor(deps Deps) *HTTPExecutor {
	return &HTTPExecutor{deps: deps}
}

func (e *HTTPExecutor) Type() domain.MonitorType {
	return domain.MonitorHTTP
}

// httpPhases captures the per-phase timings and errors of a single HTTP
// request from the client trace. Durations only become non-zero when the
// corresponding phase actually ran, so failure paths never fabricate values.
type httpPhases struct {
	dnsStart, connectStart, tlsStart time.Time
	requestStart                     time.Time
	wroteRequest                     time.Time
	firstByte                        time.Time
	dns, connect, tls                time.Duration
	requestWrite, ttfb, download     time.Duration
	dnsErr, connectErr, tlsErr       error
}

// transport returns a pooled guarded transport for the given TLS/family
// settings. Transports are shared across executions so TCP connections and
// TLS sessions are reused (enterprise: no handshake per check).
func (e *HTTPExecutor) transport(verifyTLS bool, family security.IPFamily) *http.Transport {
	key := httpTransportKey{verifyTLS: verifyTLS, family: family}
	if cached, ok := e.transports.Load(key); ok {
		return cached.(*http.Transport)
	}

	transport := e.deps.Guard.WithIPFamily(family).WrapTransport(&http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: !verifyTLS,
		},
		// Connection reuse: keep connections alive between checks and bound the
		// idle pool so a large monitor fleet does not accumulate one socket per
		// target. The default client also reuses pooled TLS sessions.
		DisableKeepAlives:   false,
		MaxIdleConns:        256,
		MaxIdleConnsPerHost: 8,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
	})

	actual, _ := e.transports.LoadOrStore(key, transport)
	return actual.(*http.Transport)
}

func (e *HTTPExecutor) Execute(ctx context.Context, job domain.ProbeJob) domain.ProbeResult {
	result := newBaseResult(job)

	// The monitor may override the resource target with an explicit `url`
	// config field. A bare hostname/IP is normalized to https:// so the check
	// works without forcing the user to type a full URL.
	rawTarget := stringConfig(job.Config, "url", job.Target)
	if !strings.HasPrefix(rawTarget, "http://") && !strings.HasPrefix(rawTarget, "https://") {
		rawTarget = "https://" + rawTarget
	}

	parsedURL, err := e.deps.Guard.ValidateURL(rawTarget)
	if err != nil {
		result.Metrics["reachability"] = 0.0
		result.Attributes["error_type"] = "invalid_target"
		return finishFailure(result, "invalid_target", err)
	}

	method := strings.ToUpper(stringConfig(job.Config, "method", http.MethodGet))
	body := stringConfig(job.Config, "body", "")
	if body == "" {
		body = stringConfig(job.Config, "request_body", "")
	}
	verifyTLS := boolConfig(job.Config, "verify_tls", true)
	if raw, ok := job.Config["verify_ssl"]; ok {
		if value, ok := raw.(bool); ok {
			verifyTLS = value
		}
	}
	followRedirects := boolConfig(job.Config, "follow_redirects", true)
	maxRedirects := intConfig(job.Config, "max_redirects", 5)
	family := security.ParseIPFamily(stringConfig(job.Config, "ip_version", string(security.IPFamilyAuto)))
	maxBodyBytes := intConfig(job.Config, "max_response_size_bytes", defaultMaxResponseBodyBytes)
	if maxBodyBytes <= 0 {
		maxBodyBytes = defaultMaxResponseBodyBytes
	}

	client := &http.Client{
		Transport: e.transport(verifyTLS, family),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !followRedirects {
				return http.ErrUseLastResponse
			}
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			if _, err := e.deps.Guard.ValidateURL(req.URL.String()); err != nil {
				return err
			}
			return nil
		},
	}

	var phases httpPhases

	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) {
			phases.dnsStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			phases.dns = time.Since(phases.dnsStart)
			phases.dnsErr = info.Err
		},
		ConnectStart: func(string, string) {
			phases.connectStart = time.Now()
		},
		ConnectDone: func(_ string, _ string, err error) {
			phases.connect = time.Since(phases.connectStart)
			phases.connectErr = err
		},
		TLSHandshakeStart: func() {
			phases.tlsStart = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			phases.tls = time.Since(phases.tlsStart)
			phases.tlsErr = err
		},
		WroteRequest: func(httptrace.WroteRequestInfo) {
			phases.wroteRequest = time.Now()
			if !phases.requestStart.IsZero() {
				phases.requestWrite = time.Since(phases.requestStart)
			}
		},
		GotFirstResponseByte: func() {
			phases.firstByte = time.Now()
			if !phases.requestStart.IsZero() {
				phases.ttfb = time.Since(phases.requestStart)
			}
		},
	}

	req, err := http.NewRequestWithContext(
		httptrace.WithClientTrace(ctx, trace),
		method,
		parsedURL.String(),
		strings.NewReader(body),
	)
	if err != nil {
		result.Metrics["reachability"] = 0.0
		result.Attributes["error_type"] = "invalid_request"
		return finishFailure(result, "invalid_request", err)
	}

	req.Header.Set("User-Agent", "MonitoringPlatform/1.0")
	if rawHeaders, ok := job.Config["headers"].(map[string]any); ok {
		for key, value := range rawHeaders {
			headerValue := fmt.Sprint(value)
			// Secret references are resolved at execution time so raw
			// credentials never live in configuration or result attributes.
			resolved, err := resolveSecrets(ctx, e.deps.Secrets, headerValue)
			if err != nil {
				result.Metrics["reachability"] = 0.0
				result.Attributes["error_type"] = "secret_resolution_failed"
				return finishFailure(result, "secret_resolution_failed", err)
			}
			req.Header.Set(key, resolved)
		}
	}

	result.Attributes["method"] = method
	result.Attributes["url"] = parsedURL.String()
	result.Attributes["ip_version"] = string(family)

	phases.requestStart = time.Now()
	response, err := client.Do(req)
	if err != nil {
		return finishHTTPFailure(result, phases, err)
	}
	defer response.Body.Close()

	readStartedAt := time.Now()
	// Read one byte past the limit so an oversized body is detected
	// deterministically instead of silently truncating (a truncated success
	// would corrupt body assertions and metrics).
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, int64(maxBodyBytes)+1))
	if err != nil {
		result.Attributes["error_type"] = "body_read_failed"
		return finishFailure(result, "body_read_failed", err)
	}
	bodyReadAt := time.Now()

	// Reading limit+1 bytes means the response exceeded the cap — stop the
	// execution and report it; never treat a partial download as success.
	if len(responseBody) > maxBodyBytes {
		result.Metrics["reachability"] = 1.0
		result.Metrics["status_code"] = float64(response.StatusCode)
		result.Attributes["status_code"] = response.StatusCode
		result.Attributes["error_type"] = "response_too_large"
		return finishFailure(
			result,
			"response_too_large",
			fmt.Errorf("response body exceeds limit of %d bytes", maxBodyBytes),
		)
	}

	// Body download time is first-byte → end-of-body. When no first-byte was
	// traced (e.g. an empty response), fall back to the full read window.
	if !phases.firstByte.IsZero() {
		phases.download = bodyReadAt.Sub(phases.firstByte)
	} else {
		phases.download = bodyReadAt.Sub(readStartedAt)
	}
	if phases.download < 0 {
		phases.download = 0
	}

	// Response-level metrics. These are valid whenever a response arrived; a
	// total transport failure never reaches this point, so latency is never
	// fabricated as 0 for a request that never completed.
	result.Metrics["reachability"] = 1.0
	result.Metrics["status_code"] = float64(response.StatusCode)
	result.Metrics["response_time_ms"] = float64(time.Since(phases.requestStart).Milliseconds())
	result.Metrics["response_size_bytes"] = float64(len(responseBody))
	if phases.dns > 0 {
		result.Metrics["dns_duration_ms"] = float64(phases.dns.Milliseconds())
	}
	if phases.connect > 0 {
		result.Metrics["connect_duration_ms"] = float64(phases.connect.Milliseconds())
	}
	if phases.tls > 0 {
		result.Metrics["tls_duration_ms"] = float64(phases.tls.Milliseconds())
	}
	if phases.requestWrite > 0 {
		result.Metrics["request_write_ms"] = float64(phases.requestWrite.Milliseconds())
	}
	if phases.ttfb > 0 {
		result.Metrics["ttfb_ms"] = float64(phases.ttfb.Milliseconds())
	}
	if phases.download > 0 {
		result.Metrics["download_time_ms"] = float64(phases.download.Milliseconds())
	}

	result.Attributes["status_code"] = response.StatusCode
	result.Attributes["content_length"] = len(responseBody)
	result.Attributes["final_url"] = response.Request.URL.String()

	// HTTP monitoring covers availability and behavior only. Certificate /
	// TLS-layer inspection belongs to the dedicated SSL monitoring type; the
	// TLS handshake *timing* is still captured above via the client trace.

	expectedCodes := intSliceConfig(job.Config, "expected_status_codes", nil)
	if len(expectedCodes) == 0 {
		if single := intConfig(job.Config, "expected_status", 0); single > 0 {
			expectedCodes = []int{single}
		}
	}
	if len(expectedCodes) == 0 {
		expectedCodes = []int{http.StatusOK}
	}
	if !containsInt(expectedCodes, response.StatusCode) {
		result.Attributes["error_type"] = "unexpected_status_code"
		return finishFailure(
			result,
			"unexpected_status_code",
			fmt.Errorf("expected one of %v, received %d", expectedCodes, response.StatusCode),
		)
	}

	expectedBody := stringConfig(job.Config, "body_contains", "")
	if expectedBody != "" && !strings.Contains(string(responseBody), expectedBody) {
		result.Metrics["content_assertion"] = 0.0
		result.Attributes["error_type"] = "body_assertion_failed"
		return finishFailure(
			result,
			"body_assertion_failed",
			fmt.Errorf("expected text was not found in response body"),
		)
	}
	result.Metrics["content_assertion"] = 1.0

	return finishSuccess(result)
}

// finishHTTPFailure classifies a transport-level failure into a deterministic
// error_type so health rules and the UI can distinguish DNS failures,
// connection refusals, TLS failures, timeouts, and generic request failures.
// Reachability is always set to 0; timing metrics are left absent (never 0).
func finishHTTPFailure(result domain.ProbeResult, phases httpPhases, err error) domain.ProbeResult {
	result.Metrics["reachability"] = 0.0

	code := "http_request_failed"
	errorType := "http_request_failed"

	switch {
	case phases.dnsErr != nil:
		code = "dns_resolution_failed"
		errorType = "dns_resolution_failed"
	case phases.connectErr != nil:
		code = "connection_failed"
		errorType = "connection_failed"
	case isDialError(err):
		code = "connection_failed"
		errorType = "connection_failed"
	case phases.tlsErr != nil:
		code, errorType = classifyTLSFailure(phases.tlsErr)
	case errors.Is(err, context.DeadlineExceeded):
		code = "timeout"
		errorType = "timeout"
	}

	result.Attributes["error_type"] = errorType
	return finishFailure(result, code, err)
}

// classifyTLSFailure maps TLS handshake errors to precise, deterministic
// error types for professional TLS monitoring.
func classifyTLSFailure(err error) (string, string) {
	code := "tls_handshake_failed"
	errorType := "tls_handshake_failed"

	var certErr x509.CertificateInvalidError
	if errors.As(err, &certErr) {
		switch certErr.Reason {
		case x509.Expired:
			return "tls_certificate_expired", "tls_certificate_expired"
		case x509.NotAuthorizedToSign:
			return "tls_unknown_ca", "tls_unknown_ca"
		}
		return "tls_handshake_failed", "tls_handshake_failed"
	}

	var hostnameErr x509.HostnameError
	if errors.As(err, &hostnameErr) {
		return "tls_hostname_mismatch", "tls_hostname_mismatch"
	}

	var unknownCA x509.UnknownAuthorityError
	if errors.As(err, &unknownCA) {
		return "tls_unknown_ca", "tls_unknown_ca"
	}

	if strings.Contains(strings.ToLower(err.Error()), "certificate is not yet valid") {
		return "tls_certificate_not_yet_valid", "tls_certificate_not_yet_valid"
	}
	if strings.Contains(strings.ToLower(err.Error()), "certificate has expired") {
		return "tls_certificate_expired", "tls_certificate_expired"
	}
	if strings.Contains(strings.ToLower(err.Error()), "doesn't match any of the subject alternative names") ||
		strings.Contains(strings.ToLower(err.Error()), "not valid for") {
		return "tls_hostname_mismatch", "tls_hostname_mismatch"
	}
	if strings.Contains(strings.ToLower(err.Error()), "certificate signed by unknown authority") {
		return "tls_unknown_ca", "tls_unknown_ca"
	}

	return code, errorType
}

// isDialError reports whether the wrapped error is a net.OpError on the dial
// phase (connection refused, host unreachable, port closed, ...).
func isDialError(err error) bool {
	var netErr *net.OpError
	if !errors.As(err, &netErr) {
		return false
	}
	return netErr.Op == "dial"
}
