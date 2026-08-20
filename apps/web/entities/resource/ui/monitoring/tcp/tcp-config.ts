// TCP monitor configuration helpers.
//
// This is the single source of truth for reading TCP execution parameters and
// health thresholds out of a resource monitor's `configuration` payload,
// mirroring the HTTP pattern (see monitoring/http/http-config.ts).
//
// The saved configuration stores thresholds under
// `configuration.health_rules.<metricKey> = { warning, critical }` and flat
// execution fields matching the probe executor keys (port, timeout_ms,
// ip_version).

export interface MetricThreshold {
  warning?: number;
  critical?: number;
}

export interface TcpThresholds {
  /** TCP connect duration, in milliseconds */
  connectTime: MetricThreshold;
}

export interface TcpConfig {
  port: number;
  timeoutMs: number;
  ipVersion: "auto" | "ipv4" | "ipv6";
  thresholds: TcpThresholds;
}

// Key aliases: the probe executor writes `connect_time_ms`; legacy names are
// accepted so previously-saved configurations keep working.
const CONNECT_TIME_KEYS = ["connect_time_ms", "connection_time_ms", "connect_duration_ms"] as const;

// Default thresholds (single source). These match the backend catalog
// defaults (packages/shared/health/catalog.go TCPParameters) so a fresh
// monitor behaves identically before it is saved.
const DEFAULT_THRESHOLDS: TcpThresholds = {
  connectTime: { warning: 500, critical: 2000 },
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

export function readTcpThresholds(
  configuration: Record<string, unknown> | undefined,
): TcpThresholds {
  const healthRules = (configuration?.health_rules ?? {}) as HealthRules;
  return {
    connectTime: readRule(healthRules, CONNECT_TIME_KEYS, DEFAULT_THRESHOLDS.connectTime),
  };
}

export function readTcpConfig(
  configuration: Record<string, unknown> | undefined,
): TcpConfig {
  const cfg = configuration ?? {};
  const ipVersion = cfg.ip_version === "ipv4" || cfg.ip_version === "ipv6" ? cfg.ip_version : "auto";
  return {
    port: asNumber(cfg.port) ?? 0,
    timeoutMs: asNumber(cfg.timeout_ms) ?? 5000,
    ipVersion,
    thresholds: readTcpThresholds(cfg),
  };
}
