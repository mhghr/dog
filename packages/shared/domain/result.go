package domain

import "time"

type ProbeResult struct {
	ID              string         `json:"id"`
	JobID           string         `json:"job_id"`
	MonitorID       string         `json:"monitor_id"`
	MonitorName     string         `json:"monitor_name"`
	ProbeLocationID string         `json:"probe_location_id"`
	Status          MonitorStatus  `json:"status"`
	Success         bool           `json:"success"`
	ErrorCode       string         `json:"error_code,omitempty"`
	ErrorMessage    string         `json:"error_message,omitempty"`
	DurationMillis  int64          `json:"duration_millis"`
	Metrics         map[string]any `json:"metrics"`
	Attributes      map[string]any `json:"attributes"`
	StartedAt       time.Time      `json:"started_at"`
	FinishedAt      time.Time      `json:"finished_at"`
}

type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type MetricsSummary struct {
	UptimePercent *float64 `json:"uptime_percent"`
	P50LatencyMS  *float64 `json:"p50_latency_ms"`
	P95LatencyMS  *float64 `json:"p95_latency_ms"`
	P99LatencyMS  *float64 `json:"p99_latency_ms"`
}

type MetricSeries struct {
	Latency []MetricPoint  `json:"latency"`
	Success []MetricPoint  `json:"success"`
	Summary MetricsSummary `json:"-"`
}

// ProbeSeries is a time-bucketed series for a single probe location.
type ProbeSeries struct {
	ProbeID   string        `json:"probe_id"`
	ProbeName string        `json:"probe_name"`
	Location  string        `json:"location"`
	MetricKey string        `json:"metric_key"`
	Points    []MetricPoint `json:"points"`
	Values    []MetricPoint `json:"values"`
}

type RecentFailure struct {
	MonitorID    string    `json:"monitor_id"`
	MonitorName  string    `json:"monitor_name"`
	MonitorType  string    `json:"monitor_type"`
	ErrorCode    *string   `json:"error_code"`
	ErrorMessage *string   `json:"error_message"`
	StartedAt    time.Time `json:"started_at"`
}

type SlowMonitor struct {
	MonitorID      string `json:"monitor_id"`
	MonitorName    string `json:"monitor_name"`
	MonitorType    string `json:"monitor_type"`
	DurationMillis int64  `json:"duration_millis"`
}

type AttentionRequired struct {
	CertificatesExpiring30d int `json:"certificates_expiring_30d"`
	DomainsExpiring45d      int `json:"domains_expiring_45d"`
	SMTPStartTLSFailures    int `json:"smtp_starttls_failures"`
	NTPHighOffset           int `json:"ntp_high_offset"`
}

type DashboardSummary struct {
	TotalMonitors   int                `json:"total_monitors"`
	StatusCounts    map[string]int     `json:"status_counts"`
	Availability24h *float64           `json:"availability_24h"`
	Checks24h       Checks24h          `json:"checks_24h"`
	AvailabilitySeries []AvailabilityPoint `json:"availability_series"`
	RecentFailures  []RecentFailure    `json:"recent_failures"`
	SlowestMonitors []SlowMonitor      `json:"slowest_monitors"`
	Attention       AttentionRequired  `json:"attention_required"`
}

// AvailabilityPoint is one time-bucketed sample of probe success over a
// monitoring window (used by the dashboard trend chart).
type AvailabilityPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	Successful int64     `json:"successful"`
	Total      int64     `json:"total"`
	Rate       float64   `json:"rate"`
}

// AggregateChecks counts successful/failed checks over a range.
type AggregateChecks struct {
	Total      int64 `json:"total_requests"`
	Successful int64 `json:"successful_requests"`
	Failed     int64 `json:"failed_requests"`
}

// MonitorAggregateMetrics is the range-scoped KPI set for an HTTP monitor
// (optionally filtered to one probe), computed by the metric layer so the
// frontend never derives P95 or error rates from raw data.
type MonitorAggregateMetrics struct {
	Checks            AggregateChecks `json:"checks"`
	Availability      *float64        `json:"availability"`
	AvgResponseTimeMS *float64        `json:"avg_response_time_ms"`
	P95ResponseTimeMS *float64        `json:"p95_response_time_ms"`
	AvgTTFBMS         *float64        `json:"avg_ttfb_ms"`
	ErrorRate         *float64        `json:"error_rate"`
	Codes4xx          int64           `json:"codes_4xx"`
	Rate4xx           *float64        `json:"rate_4xx"`
	Codes5xx          int64           `json:"codes_5xx"`
	Rate5xx           *float64        `json:"rate_5xx"`
}

// ProbeAggregateMetrics is the per-probe KPI set over a range. Last-status
// facts are merged by the handler from the latest result per probe.
type ProbeAggregateMetrics struct {
	ProbeID           string         `json:"probe_id"`
	ProbeName         string         `json:"probe_name"`
	Location          string         `json:"location"`
	Checks            AggregateChecks `json:"checks"`
	Availability      *float64       `json:"availability"`
	AvgResponseTimeMS *float64       `json:"avg_response_time_ms"`
	P95ResponseTimeMS *float64       `json:"p95_response_time_ms"`
	AvgTTFBMS         *float64       `json:"avg_ttfb_ms"`
	ErrorRate         *float64       `json:"error_rate"`
	LastCheckedAt     *time.Time     `json:"last_checked_at"`
	LastStatusCode    *int           `json:"last_status_code"`
	LastSuccess       bool           `json:"last_success"`
}

type Checks24h struct {
	Successful int64 `json:"successful"`
	Failed     int64 `json:"failed"`
}

// StatusCodeCount is one bucket of the HTTP status-code distribution for a
// monitor over a time window.
type StatusCodeCount struct {
	Code  int   `json:"code"`
	Count int64 `json:"count"`
}
