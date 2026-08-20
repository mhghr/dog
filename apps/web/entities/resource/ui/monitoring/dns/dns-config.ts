// DNS monitor configuration helpers.
//
// This is the single source of truth for reading DNS execution parameters and
// health thresholds out of a resource monitor's `configuration` payload,
// mirroring the HTTP pattern.

export interface MetricThreshold {
  warning?: number;
  critical?: number;
}

export interface DnsThresholds {
  /** DNS query response time, in milliseconds */
  responseTime: MetricThreshold;
}

export interface DnsConfig {
  recordType: string;
  resolver: string;
  expectedValues: string[];
  timeoutMs: number;
  ipVersion: "auto" | "ipv4" | "ipv6";
  thresholds: DnsThresholds;
}

// Key aliases: the probe executor writes `response_time_ms`; legacy names are
// accepted so previously-saved configurations keep working.
const RESPONSE_TIME_KEYS = ["response_time_ms", "resolution_duration_ms", "dns_duration_ms"] as const;

// Default thresholds (single source). These match the backend catalog
// defaults (packages/shared/health/catalog.go DNSParameters).
const DEFAULT_THRESHOLDS: DnsThresholds = {
  responseTime: { warning: 500, critical: 2000 },
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

function asStringList(value: unknown): string[] {
  if (typeof value === "string" && value.trim() !== "") {
    return value
      .split(",")
      .map((part) => part.trim())
      .filter(Boolean);
  }
  if (Array.isArray(value)) {
    return value
      .map((item) => (typeof item === "string" ? item.trim() : String(item)))
      .filter(Boolean);
  }
  return [];
}

export function readDnsThresholds(
  configuration: Record<string, unknown> | undefined,
): DnsThresholds {
  const healthRules = (configuration?.health_rules ?? {}) as HealthRules;
  return {
    responseTime: readRule(healthRules, RESPONSE_TIME_KEYS, DEFAULT_THRESHOLDS.responseTime),
  };
}

export function readDnsConfig(
  configuration: Record<string, unknown> | undefined,
): DnsConfig {
  const cfg = configuration ?? {};
  const ipVersion = cfg.ip_version === "ipv4" || cfg.ip_version === "ipv6" ? cfg.ip_version : "auto";
  const resolver =
    typeof cfg.resolver === "string" && cfg.resolver.trim() !== ""
      ? cfg.resolver
      : typeof cfg.server === "string" && cfg.server.trim() !== ""
        ? cfg.server
        : typeof cfg.nameserver === "string" && cfg.nameserver.trim() !== ""
          ? cfg.nameserver
          : "";
  const expectedValues = asStringList(cfg.expected_values);
  return {
    recordType: typeof cfg.record_type === "string" && cfg.record_type !== "" ? cfg.record_type.toUpperCase() : "A",
    resolver,
    expectedValues,
    timeoutMs: asNumber(cfg.timeout_ms) ?? 5000,
    ipVersion,
    thresholds: readDnsThresholds(cfg),
  };
}
