package domain

import "time"

type AgentConfig struct {
	ID                       string
	AgentID                  string
	TenantID                 string
	Version                  int
	CollectionIntervalSeconds int
	BatchSize                int
	ExportIntervalSeconds    int
	EnabledReceivers         []string
	MaxMetricsPerBatch       int
	MaxLabelCount            int
	MaxLabelLength           int
	FeatureFlags             map[string]bool
	OTLPEndpoint             string
	Compress                 bool
	RetryInitialIntervalMs   int
	RetryMaxIntervalMs       int
	RetryMaxElapsedMs        int
	LogLevel                 string
	IsActive                 bool
	CreatedAt                time.Time
}

type AgentConfigUpdate struct {
	Version int
	Config  AgentConfig
}
