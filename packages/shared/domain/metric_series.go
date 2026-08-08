package domain

import (
	"encoding/json"
	"time"
)

// MetricSeriesRow represents a unique metric name + label combination
// stored in the metric_series table.
type MetricSeriesRow struct {
	ID         string          `json:"id"`
	ResourceID string          `json:"resource_id"`
	MetricName string          `json:"metric_name"`
	Labels     json.RawMessage `json:"labels"`
	Unit       string          `json:"unit"`
	CreatedAt  time.Time       `json:"created_at"`
}

// MetricPointDB represents a single time-series data point stored in
// the metric_points table (separate from the in-memory MetricPoint
// used for API responses).
type MetricPointDB struct {
	Time       time.Time       `json:"time"`
	SeriesID   string          `json:"series_id"`
	Value      float64         `json:"value"`
	Attributes json.RawMessage `json:"attributes"`
}

// MetricRollup holds pre-aggregated statistics for a time bucket.
type MetricRollup struct {
	ID                 string    `json:"id"`
	SeriesID           string    `json:"series_id"`
	Bucket             time.Time `json:"bucket"`
	BucketWidthSeconds int       `json:"bucket_width_seconds"`
	Count              int64     `json:"count"`
	Sum                float64   `json:"sum"`
	Min                *float64  `json:"min,omitempty"`
	Max                *float64  `json:"max,omitempty"`
	P50                *float64  `json:"p50,omitempty"`
	P95                *float64  `json:"p95,omitempty"`
	P99                *float64  `json:"p99,omitempty"`
	LastValue          *float64  `json:"last_value,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}
