package domain

import "time"

// AgentConfig represents the configuration for a monitoring agent.
type AgentConfig struct {
	// ID is the unique identifier for this config record.
	ID string `json:"id"`
	// AgentID is the agent this configuration belongs to.
	AgentID string `json:"agent_id"`
	// TenantID is the tenant that owns this configuration.
	TenantID string `json:"tenant_id"`
	// Version is the monotonically increasing version number of this config.
	Version int `json:"version"`
	// CollectionIntervalSeconds is how often the agent collects metrics.
	CollectionIntervalSeconds int `json:"collection_interval_seconds"`
	// BatchSize is the number of metrics to batch before exporting.
	BatchSize int `json:"batch_size"`
	// ExportIntervalSeconds is how often the agent exports metrics.
	ExportIntervalSeconds int `json:"export_interval_seconds"`
	// EnabledReceivers lists the receiver plugins currently active.
	EnabledReceivers []string `json:"enabled_receivers"`
	// MaxMetricsPerBatch is the maximum number of metrics in a single export batch.
	MaxMetricsPerBatch int `json:"max_metrics_per_batch"`
	// MaxLabelCount is the maximum number of labels allowed per metric.
	MaxLabelCount int `json:"max_label_count"`
	// MaxLabelLength is the maximum length of a single label value.
	MaxLabelLength int `json:"max_label_length"`
	// FeatureFlags is a set of boolean feature toggles.
	FeatureFlags map[string]bool `json:"feature_flags"`
	// OTLPEndpoint is the target OTLP/gRPC endpoint for metric export.
	OTLPEndpoint string `json:"otlp_endpoint"`
	// Compress enables gzip compression on exported payloads.
	Compress bool `json:"compress"`
	// RetryInitialIntervalMs is the initial back-off interval for retries.
	RetryInitialIntervalMs int `json:"retry_initial_interval_ms"`
	// RetryMaxIntervalMs is the maximum back-off interval for retries.
	RetryMaxIntervalMs int `json:"retry_max_interval_ms"`
	// RetryMaxElapsedMs is the maximum total time spent on retries.
	RetryMaxElapsedMs int `json:"retry_max_elapsed_ms"`
	// LogLevel controls the verbosity of agent logging.
	LogLevel string `json:"log_level"`
	// IsActive indicates whether this configuration is the active one.
	IsActive bool `json:"is_active"`
	// CreatedAt is when this configuration was created.
	CreatedAt time.Time `json:"created_at"`
}

// AgentConfigUpdate wraps a new AgentConfig for atomic configuration updates.
type AgentConfigUpdate struct {
	Config AgentConfig
}
