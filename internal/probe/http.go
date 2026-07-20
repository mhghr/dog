package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"monitoring-platform/internal/domain"
)

const maxResponseBodyBytes = 1024 * 1024

type HTTPExecutor struct {
	deps Deps
}

func NewHTTPExecutor(deps Deps) *HTTPExecutor {
	return &HTTPExecutor{deps: deps}
}

func (e *HTTPExecutor) Type() domain.MonitorType {
	return domain.MonitorHTTP
}

func (e *HTTPExecutor) Execute(ctx context.Context, job domain.ProbeJob) domain.ProbeResult {
	result := newBaseResult(job)

	parsedURL, err := e.deps.Guard.ValidateURL(job.Target)
	if err != nil {
		return finishFailure(result, "invalid_target", err)
	}

	method := strings.ToUpper(stringConfig(job.Config, "method", http.MethodGet))
	body := stringConfig(job.Config, "body", "")
	verifyTLS := boolConfig(job.Config, "verify_tls", true)
	followRedirects := boolConfig(job.Config, "follow_redirects", true)
	maxRedirects := intConfig(job.Config, "max_redirects", 5)

	transport := e.deps.Guard.WrapTransport(&http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: !verifyTLS,
		},
		DisableKeepAlives: true,
	})

	client := &http.Client{
		Transport: transport,
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

	var (
		dnsStart, connectStart, tlsStart, requestStart time.Time
		dnsDuration, connectDuration, tlsDuration      time.Duration
		timeToFirstByte                                time.Duration
	)

	trace := &httptrace.ClientTrace{
		DNSStart:          func(httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:           func(httptrace.DNSDoneInfo) { dnsDuration = time.Since(dnsStart) },
		ConnectStart:      func(string, string) { connectStart = time.Now() },
		ConnectDone:       func(string, string, error) { connectDuration = time.Since(connectStart) },
		TLSHandshakeStart: func() { tlsStart = time.Now() },
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			tlsDuration = time.Since(tlsStart)
		},
		GotFirstResponseByte: func() {
			timeToFirstByte = time.Since(requestStart)
		},
	}

	req, err := http.NewRequestWithContext(
		httptrace.WithClientTrace(ctx, trace),
		method,
		parsedURL.String(),
		strings.NewReader(body),
	)
	if err != nil {
		return finishFailure(result, "invalid_request", err)
	}

	req.Header.Set("User-Agent", "MonitoringPlatform/1.0")
	if rawHeaders, ok := job.Config["headers"].(map[string]any); ok {
		for key, value := range rawHeaders {
			req.Header.Set(key, fmt.Sprint(value))
		}
	}

	requestStart = time.Now()
	response, err := client.Do(req)
	if err != nil {
		return finishFailure(result, "http_request_failed", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes))
	if err != nil {
		return finishFailure(result, "body_read_failed", err)
	}

	result.Attributes["status_code"] = response.StatusCode
	result.Attributes["content_length"] = len(responseBody)
	result.Attributes["final_url"] = response.Request.URL.String()

	if dnsDuration > 0 {
		result.Metrics["dns_duration_ms"] = dnsDuration.Milliseconds()
	}
	if connectDuration > 0 {
		result.Metrics["connect_duration_ms"] = connectDuration.Milliseconds()
	}
	if tlsDuration > 0 {
		result.Metrics["tls_duration_ms"] = tlsDuration.Milliseconds()
	}
	if timeToFirstByte > 0 {
		result.Metrics["ttfb_ms"] = timeToFirstByte.Milliseconds()
	}

	if response.TLS != nil && len(response.TLS.PeerCertificates) > 0 {
		certificate := response.TLS.PeerCertificates[0]
		result.Attributes["tls_issuer"] = certificate.Issuer.String()
		result.Attributes["tls_expires_at"] = certificate.NotAfter
		result.Attributes["tls_days_remaining"] = int(time.Until(certificate.NotAfter).Hours() / 24)
	}

	expectedCodes := intSliceConfig(job.Config, "expected_status_codes", []int{http.StatusOK})
	if !containsInt(expectedCodes, response.StatusCode) {
		return finishFailure(
			result,
			"unexpected_status_code",
			fmt.Errorf("expected one of %v, received %d", expectedCodes, response.StatusCode),
		)
	}

	expectedBody := stringConfig(job.Config, "body_contains", "")
	if expectedBody != "" && !strings.Contains(string(responseBody), expectedBody) {
		return finishFailure(
			result,
			"body_assertion_failed",
			fmt.Errorf("expected text was not found in response body"),
		)
	}

	return finishSuccess(result)
}
