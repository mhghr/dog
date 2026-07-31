package config

// AgentConfig is the agent configuration downloaded from Core.
type AgentConfig struct {
	Version                   int             `json:"version"`
	CollectionIntervalSeconds int             `json:"collection_interval_seconds"`
	BatchSize                 int             `json:"batch_size"`
	ExportIntervalSeconds     int             `json:"export_interval_seconds"`
	EnabledReceivers          []string        `json:"enabled_receivers"`
	MaxMetricsPerBatch        int             `json:"max_metrics_per_batch"`
	MaxLabelCount             int             `json:"max_label_count"`
	MaxLabelLength            int             `json:"max_label_length"`
	FeatureFlags              map[string]bool `json:"feature_flags"`
	OTLPEndpoint              string          `json:"otlp_endpoint"`
	Compress                  bool            `json:"compress"`
	RetryInitialIntervalMs    int             `json:"retry_initial_interval_ms"`
	RetryMaxIntervalMs        int             `json:"retry_max_interval_ms"`
	RetryMaxElapsedMs         int             `json:"retry_max_elapsed_ms"`
	LogLevel                  string          `json:"log_level"`
}

// DefaultConfig returns a sane default agent configuration.
func DefaultConfig() *AgentConfig {
	return &AgentConfig{
		Version:                   1,
		CollectionIntervalSeconds: 60,
		BatchSize:                 500,
		ExportIntervalSeconds:     60,
		EnabledReceivers:          []string{"cpu", "memory", "filesystem", "disk", "network", "load"},
		MaxMetricsPerBatch:        2000,
		MaxLabelCount:             40,
		MaxLabelLength:            256,
		FeatureFlags:              map[string]bool{},
		Compress:                  true,
		RetryInitialIntervalMs:    1000,
		RetryMaxIntervalMs:        60000,
		RetryMaxElapsedMs:         300000,
		LogLevel:                  "info",
	}
}

// SanitizeConfig clamps invalid values to safe defaults.
func SanitizeConfig(c *AgentConfig) {
	if c.CollectionIntervalSeconds < 10 {
		c.CollectionIntervalSeconds = 10
	}
	if c.BatchSize < 1 {
		c.BatchSize = 100
	}
	if c.MaxMetricsPerBatch < 100 {
		c.MaxMetricsPerBatch = 100
	}
	if c.MaxLabelCount < 1 {
		c.MaxLabelCount = 10
	}
	if len(c.EnabledReceivers) == 0 {
		c.EnabledReceivers = []string{"cpu", "memory"}
	}
	if c.FeatureFlags == nil {
		c.FeatureFlags = map[string]bool{}
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
}
