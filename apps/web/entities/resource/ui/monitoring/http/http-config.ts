// HTTP monitor configuration helpers.
//
// This is the single source of truth for reading HTTP execution parameters
// and health thresholds out of a resource monitor's `configuration` payload,
// mirroring the Ping pattern (see monitoring/ping/ping-config.ts).
//
// The saved configuration is produced by `MonitorConfig` (entities/resource)
// and stores thresholds under `configuration.health_rules.<metricKey> =
// { warning, critical }`. Execution parameters are flat fields matching the
// probe executor keys (method, follow_redirects, verify_tls,
// expected_status_codes, body_contains, request_body, headers).
//
// NOTE: these defaults only apply when the monitor has no saved values yet.
// They mirror the backend health catalog (packages/shared/health/catalog.go
// HTTPParameters) and the monitor-type seed health_parameters so that the
// pre-save UI and the monitoring view agree. UI components must never invent
// their own thresholds — always read them via `readHttpConfig`.

export interface MetricThreshold {
  warning?: number;
  critical?: number;
}

export interface HttpThresholds {
  /** Total response time, in milliseconds */
  responseTime: MetricThreshold;
  /** Time to first byte, in milliseconds */
  ttfb: MetricThreshold;
  /** DNS resolution duration, in milliseconds */
  dnsDuration: MetricThreshold;
  /** TCP connect duration, in milliseconds */
  connectDuration: MetricThreshold;
  /** TLS handshake duration, in milliseconds */
  tlsDuration: MetricThreshold;
}

export interface HttpConfig {
  url: string;
  method: string;
  followRedirects: boolean;
  verifyTls: boolean;
  maxRedirects: number;
  expectedStatusCodes: number[];
  bodyContains: string;
  requestBody: string;
  headers: Record<string, string>;
  ipVersion: "auto" | "ipv4" | "ipv6";
  maxResponseSizeBytes: number;
  thresholds: HttpThresholds;
}

// Key aliases: the health_parameters seed historically used `response_time_ms`
// while the probe executor writes `response_time_ms`/`ttfb_ms`/`dns_duration_ms`.
// Accept both the executor keys and legacy aliases.
const RESPONSE_TIME_KEYS = ["response_time_ms", "total_duration_ms"] as const;
const TTFB_KEYS = ["ttfb_ms", "time_to_first_byte_ms"] as const;
const DNS_KEYS = ["dns_duration_ms", "dns_time_ms"] as const;
const CONNECT_KEYS = ["connect_duration_ms", "connect_time_ms"] as const;
const TLS_KEYS = ["tls_duration_ms", "tls_time_ms"] as const;

// Default thresholds (single source). These match the backend catalog
// defaults so that a fresh monitor behaves identically before it is saved.
const DEFAULT_THRESHOLDS: HttpThresholds = {
  responseTime: { warning: 2000, critical: 5000 },
  ttfb: { warning: 1000, critical: 3000 },
  dnsDuration: { warning: 500, critical: 2000 },
  connectDuration: { warning: 500, critical: 2000 },
  tlsDuration: { warning: 500, critical: 2000 },
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

function asStringList(value: unknown): number[] {
  if (typeof value === "number" && !Number.isNaN(value)) return [value];
  if (typeof value === "string" && value.trim() !== "") {
    return value
      .split(",")
      .map((part) => Number(part.trim()))
      .filter((n) => !Number.isNaN(n));
  }
  if (Array.isArray(value)) {
    return value
      .map((item) => asNumber(item))
      .filter((n): n is number => n != null);
  }
  return [];
}

function asRecord(value: unknown): Record<string, string> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return {};
  }
  const result: Record<string, string> = {};
  for (const [key, raw] of Object.entries(value)) {
    result[key] = String(raw);
  }
  return result;
}

export function readHttpThresholds(
  configuration: Record<string, unknown> | undefined,
): HttpThresholds {
  const healthRules = (configuration?.health_rules ?? {}) as HealthRules;
  return {
    responseTime: readRule(healthRules, RESPONSE_TIME_KEYS, DEFAULT_THRESHOLDS.responseTime),
    ttfb: readRule(healthRules, TTFB_KEYS, DEFAULT_THRESHOLDS.ttfb),
    dnsDuration: readRule(healthRules, DNS_KEYS, DEFAULT_THRESHOLDS.dnsDuration),
    connectDuration: readRule(healthRules, CONNECT_KEYS, DEFAULT_THRESHOLDS.connectDuration),
    tlsDuration: readRule(healthRules, TLS_KEYS, DEFAULT_THRESHOLDS.tlsDuration),
  };
}

export function readHttpConfig(
  configuration: Record<string, unknown> | undefined,
): HttpConfig {
  const cfg = configuration ?? {};
  const expected = asStringList(cfg.expected_status_codes);
  const ipVersion = cfg.ip_version === "ipv4" || cfg.ip_version === "ipv6" ? cfg.ip_version : "auto";
  return {
    url: typeof cfg.url === "string" ? cfg.url : "",
    method: typeof cfg.method === "string" ? cfg.method.toUpperCase() : "GET",
    followRedirects: typeof cfg.follow_redirects === "boolean" ? cfg.follow_redirects : true,
    verifyTls: typeof cfg.verify_tls === "boolean"
      ? cfg.verify_tls
      : typeof cfg.verify_ssl === "boolean" ? cfg.verify_ssl : true,
    maxRedirects: asNumber(cfg.max_redirects) ?? 5,
    expectedStatusCodes: expected.length > 0 ? expected : [200],
    bodyContains: typeof cfg.body_contains === "string" ? cfg.body_contains : "",
    requestBody: typeof cfg.request_body === "string"
      ? cfg.request_body
      : typeof cfg.body === "string" ? cfg.body : "",
    headers: asRecord(cfg.headers),
    ipVersion,
    maxResponseSizeBytes: asNumber(cfg.max_response_size_bytes) ?? 10 * 1024 * 1024,
    thresholds: readHttpThresholds(cfg),
  };
}
