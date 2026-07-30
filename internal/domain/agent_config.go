package domain

import "time"

// AgentConfig represents the configuration for a monitoring agent.
type AgentConfig struct {
	// ID is the unique identifier for this config record.
	ID string
	// AgentID is the agent this configuration belongs to.
	AgentID string
	// TenantID is the tenant that owns this configuration.
	TenantID string
	// Version is the monotonically increasing version number of this config.
	Version int
	// CollectionIntervalSeconds is how often the agent collects metrics.
	CollectionIntervalSeconds int
	// BatchSize is the number of metrics to batch before exporting.
	BatchSize int
	// ExportIntervalSeconds is how often the agent exports metrics.
	ExportIntervalSeconds int
	// EnabledReceivers lists the receiver plugins currently active.
	EnabledReceivers []string
	// MaxMetricsPerBatch is the maximum number of metrics in a single export batch.
	MaxMetricsPerBatch int
	// MaxLabelCount is the maximum number of labels allowed per metric.
	MaxLabelCount int
	// MaxLabelLength is the maximum length of a single label value.
	MaxLabelLength int
	// FeatureFlags is a set of boolean feature toggles.
	FeatureFlags map[string]bool
	// OTLPEndpoint is the target OTLP/gRPC endpoint for metric export.
	OTLPEndpoint string
	// Compress enables gzip compression on exported payloads.
	Compress bool
	// RetryInitialIntervalMs is the initial back-off interval for retries.
	RetryInitialIntervalMs int
	// RetryMaxIntervalMs is the maximum back-off interval for retries.
	RetryMaxIntervalMs int
	// RetryMaxElapsedMs is the maximum total time spent on retries.
	RetryMaxElapsedMs int
	// LogLevel controls the verbosity of agent logging.
	LogLevel string
	// IsActive indicates whether this configuration is the active one.
	IsActive bool
	// CreatedAt is when this configuration was created.
	CreatedAt time.Time
}

// AgentConfigUpdate wraps a new AgentConfig for atomic configuration updates.
type AgentConfigUpdate struct {
	Config AgentConfig
}
