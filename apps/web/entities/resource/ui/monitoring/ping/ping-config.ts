// Ping monitor configuration helpers.
//
// This is the single source of truth for reading Ping execution parameters
// and health thresholds out of a resource monitor's `configuration` payload.
// The saved configuration is produced by `MonitorConfig` (entities/resource)
// and stores thresholds under `configuration.health_rules.<metricKey> =
// { warning, critical }`. Execution parameters are flat fields
// (`packet_count`, `packet_interval_millis`) matching the probe executor.
//
// NOTE: these defaults only apply when the monitor has no saved values yet.
// They mirror the backend health catalog (packages/shared/health/catalog.go
// PingParameters) and the monitor-type seed health_parameters so that the
// pre-save UI and the monitoring view agree. UI components must never invent
// their own thresholds — always read them via `readPingConfig`.

export interface MetricThreshold {
  warning?: number;
  critical?: number;
}

export interface PingThresholds {
  /** RTT latency, in milliseconds */
  latency: MetricThreshold;
  /** Packet loss, in percent */
  packetLoss: MetricThreshold;
  /** Jitter, in milliseconds */
  jitter: MetricThreshold;
}

export interface PingConfig {
  packetCount: number;
  packetIntervalMillis: number;
  thresholds: PingThresholds;
}

// Key aliases: the health_parameters seed uses `latency_ms`/`packet_loss`
// while the probe executor writes `rtt_ms`/`packet_loss_percent`. Accept both.
const LATENCY_KEYS = ["latency_ms", "rtt_ms"] as const;
const PACKET_LOSS_KEYS = ["packet_loss", "packet_loss_percent"] as const;
const JITTER_KEYS = ["jitter_ms"] as const;

// Default thresholds (single source). These match the backend catalog
// defaults so that a fresh monitor behaves identically before it is saved.
const DEFAULT_THRESHOLDS: PingThresholds = {
  latency: { warning: 200, critical: 500 },
  packetLoss: { warning: 5, critical: 20 },
  jitter: { warning: 30, critical: 80 },
};

type HealthRules = Record<string, { warning?: number; critical?: number }>;

function asNumber(value: unknown): number | undefined {
  if (typeof value === "number" && !Number.isNaN(value)) return value;
  if (typeof value === "string" && value.trim() !== "") {
    const n = Number(value);
    return Number.isNaN(n) ? undefined : n;
  }
  return undefined;
}

function readRule(
  healthRules: HealthRules | undefined,
  keys: readonly string[],
  fallback: MetricThreshold,
): MetricThreshold {
  if (!healthRules) return fallback;
  for (const key of keys) {
    const rule = healthRules[key];
    if (rule) {
      return {
        warning: asNumber(rule.warning) ?? fallback.warning,
        critical: asNumber(rule.critical) ?? fallback.critical,
      };
    }
  }
  return fallback;
}

export function readPingThresholds(
  configuration: Record<string, unknown> | undefined,
): PingThresholds {
  const healthRules = (configuration?.health_rules ?? {}) as HealthRules;
  return {
    latency: readRule(healthRules, LATENCY_KEYS, DEFAULT_THRESHOLDS.latency),
    packetLoss: readRule(healthRules, PACKET_LOSS_KEYS, DEFAULT_THRESHOLDS.packetLoss),
    jitter: readRule(healthRules, JITTER_KEYS, DEFAULT_THRESHOLDS.jitter),
  };
}

export function readPingConfig(
  configuration: Record<string, unknown> | undefined,
): PingConfig {
  const cfg = configuration ?? {};
  return {
    packetCount: asNumber(cfg.packet_count ?? cfg.count) ?? 4,
    packetIntervalMillis: asNumber(cfg.packet_interval_millis) ?? 200,
    thresholds: readPingThresholds(cfg),
  };
}
