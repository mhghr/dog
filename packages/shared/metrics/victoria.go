package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"monitoring-platform/packages/shared/domain"
)

// VictoriaClient ships probe metrics to VictoriaMetrics using the Prometheus
// import format. Writes are asynchronous and batched so ingestion latency is
// never coupled to the metrics store.
type VictoriaClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
	lines      chan string
	done       chan struct{}
}

func NewVictoriaClient(baseURL string, logger *slog.Logger) *VictoriaClient {
	return &VictoriaClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
		lines:      make(chan string, 4096),
		done:       make(chan struct{}),
	}
}

// Start launches the background flusher. Call Close for a final flush.
func (c *VictoriaClient) Start(ctx context.Context) {
	go func() {
		defer close(c.done)

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		batch := make([]string, 0, 512)

		flush := func() {
			if len(batch) == 0 {
				return
			}
			if err := c.push(batch); err != nil {
				c.logger.Warn("victoriametrics write failed", "error", err, "lines", len(batch))
			}
			batch = batch[:0]
		}

		for {
			select {
			case <-ctx.Done():
				flush()
				return
			case line, ok := <-c.lines:
				if !ok {
					flush()
					return
				}
				batch = append(batch, line)
				if len(batch) >= 500 {
					flush()
				}
			case <-ticker.C:
				flush()
			}
		}
	}()
}

func (c *VictoriaClient) push(lines []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v1/import/prometheus",
		strings.NewReader(strings.Join(lines, "\n")),
	)
	if err != nil {
		return err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return fmt.Errorf("VictoriaMetrics returned %s", response.Status)
	}

	return nil
}

// Enqueue converts a probe result to metric lines. It never blocks: when the
// buffer is full, lines are dropped and a warning is logged.
func (c *VictoriaClient) Enqueue(result *domain.ProbeResult, monitorType, locationCode string) {
	if c == nil {
		return
	}
	for _, line := range buildLines(result, monitorType, locationCode) {
		select {
		case c.lines <- line:
		default:
			c.logger.Warn("victoriametrics buffer full, dropping metric line")
			return
		}
	}
}

func (c *VictoriaClient) Health(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return fmt.Errorf("VictoriaMetrics health returned %s", response.Status)
	}

	return nil
}

func buildLines(result *domain.ProbeResult, monitorType, locationCode string) []string {
	labels := fmt.Sprintf(
		`monitor_id=%q,monitor_type=%q,probe_location=%q`,
		result.MonitorID, monitorType, locationCode,
	)
	timestamp := result.FinishedAt.UnixMilli()

	successValue := 0
	if result.Success {
		successValue = 1
	}

	lines := []string{
		fmt.Sprintf(`monitor_probe_success{%s} %d %d`, labels, successValue, timestamp),
		fmt.Sprintf(
			`monitor_probe_duration_seconds{%s} %f %d`,
			labels, float64(result.DurationMillis)/1000, timestamp,
		),
	}

	if monitorType == string(domain.MonitorHTTP) {
		if statusCode, ok := numericValue(result.Attributes["status_code"]); ok {
			lines = append(lines, fmt.Sprintf(`monitor_http_status_code{%s} %g %d`, labels, statusCode, timestamp))
		}
	}

	for key, raw := range result.Metrics {
		if key == "total_duration_ms" {
			continue
		}

		value, ok := numericValue(raw)
		if !ok {
			continue
		}

		metricName := sanitizeMetricName(fmt.Sprintf("monitor_%s_%s", monitorType, key))
		lines = append(lines, fmt.Sprintf(`%s{%s} %g %d`, metricName, labels, value, timestamp))
	}

	return lines
}

func numericValue(raw any) (float64, bool) {
	switch value := raw.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	default:
		return 0, false
	}
}

func sanitizeMetricName(name string) string {
	var builder strings.Builder
	builder.Grow(len(name))

	for index, char := range name {
		isValid := char == '_' ||
			(char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9' && index > 0)

		if isValid {
			builder.WriteRune(char)
		} else {
			builder.WriteByte('_')
		}
	}

	return builder.String()
}
