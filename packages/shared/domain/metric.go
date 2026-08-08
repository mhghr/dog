package domain

import "time"

// MetricType enumerates the supported metric kinds.
type MetricType string

const (
	// MetricTypeGauge represents a value that can go up and down.
	MetricTypeGauge MetricType = "gauge"
	// MetricTypeSum represents a monotonically increasing counter.
	MetricTypeSum MetricType = "sum"
	// MetricTypeHistogram represents a distribution of values.
	MetricTypeHistogram MetricType = "histogram"
)

// MetricSample is a single data point collected by an agent.
type MetricSample struct {
	// Name is the metric name (e.g. "cpu.usage.user").
	Name string
	// Type is the kind of metric (gauge, sum, or histogram).
	Type MetricType
	// Value is the numeric measurement.
	Value float64
	// Labels are key/value pairs that add dimensionality to the metric.
	Labels map[string]string
	// Timestamp is when the measurement was taken.
	Timestamp time.Time
	// TenantID is the tenant that owns this metric.
	TenantID string
	// AgentID is the agent that collected this metric.
	AgentID string
	// Hostname is the host where the metric was collected.
	Hostname string
}

// MetricBatch is a collection of samples grouped for export.
type MetricBatch struct {
	// Samples is the list of metric data points in this batch.
	Samples []MetricSample
	// AgentID is the agent that produced the batch.
	AgentID string
	// TenantID is the tenant that owns the batch.
	TenantID string
	// ReceivedAt is when the control plane received the batch.
	ReceivedAt time.Time
}
