package config

import (
	"fmt"
	"strings"
)

// GenerateOTelConfig renders an OTel Collector YAML config from the agent config.
func GenerateOTelConfig(c *AgentConfig, otlpEndpoint, agentID string) string {
	scrapers := make([]string, 0, len(c.EnabledReceivers))
	for _, r := range c.EnabledReceivers {
		scrapers = append(scrapers, fmt.Sprintf("    %s: {}", r))
	}

	compression := "gzip"
	if !c.Compress {
		compression = "none"
	}

	return fmt.Sprintf(`receivers:
  hostmetrics:
    collection_interval: %ds
    scrapers:
%s

processors:
  batch:
    timeout: %ds
    send_batch_size: %d
  memory_limiter:
    check_interval: 5s
    limit_mib: 512
    spike_limit_mib: 128

exporters:
  otlp:
    endpoint: %s
    tls:
      insecure: true
    compression: "%s"
    headers:
      x-agent-id: "%s"
    retry_on_failure:
      enabled: true
      initial_interval: %dms
      max_interval: %dms
      max_elapsed: %dms
    sending_queue:
      enabled: true
      queue_size: %d

service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [memory_limiter, batch]
      exporters: [otlp]
`,
		c.CollectionIntervalSeconds,
		strings.Join(scrapers, "\n"),
		c.ExportIntervalSeconds,
		c.BatchSize,
		otlpEndpoint,
		compression,
		agentID,
		c.RetryInitialIntervalMs,
		c.RetryMaxIntervalMs,
		c.RetryMaxElapsedMs,
		c.MaxMetricsPerBatch,
	)
}
