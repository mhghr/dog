package domain

import "time"

type MetricType string

const (
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeSum       MetricType = "sum"
	MetricTypeHistogram MetricType = "histogram"
)

type MetricSample struct {
	Name      string
	Type      MetricType
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
	TenantID  string
	AgentID   string
	Hostname  string
}

type MetricBatch struct {
	Samples    []MetricSample
	AgentID    string
	TenantID   string
	ReceivedAt time.Time
}
