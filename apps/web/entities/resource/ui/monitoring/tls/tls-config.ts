// TLS monitor configuration helpers.
//
// This is the single source of truth for reading TLS execution parameters and
// health thresholds out of a resource monitor's `configuration` payload,
// mirroring the HTTP pattern.

export interface MetricThreshold {
  warning?: number;
  critical?: number;
}

export interface TlsThresholds {
  /** TLS handshake duration, in milliseconds (higher is worse) */
  handshakeTime: MetricThreshold;
  /** Days until certificate expiry (lower is worse) */
  certificateExpiryDays: MetricThreshold;
}

export interface TlsConfig {
  port: number;
  serverName: string;
  verifyTls: boolean;
  minTlsVersion: string;
  timeoutMs: number;
  ipVersion: "auto" | "ipv4" | "ipv6";
  thresholds: TlsThresholds;
}

// Key aliases: the probe executor writes `handshake_time_ms` and
// `certificate_expiry_days`; legacy names are accepted so previously-saved
// configurations keep working.
const HANDSHAKE_TIME_KEYS = ["handshake_time_ms", "handshake_duration_ms"] as const;
const EXPIRY_DAYS_KEYS = ["certificate_expiry_days", "days_remaining", "cert_days_remaining"] as const;

// Default thresholds (single source). These match the backend catalog
// defaults (packages/shared/health/catalog.go TLSParameters).
const DEFAULT_THRESHOLDS: TlsThresholds = {
  handshakeTime: { warning: 500, critical: 2000 },
  certificateExpiryDays: { warning: 30, critical: 14 },
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

export function readTlsThresholds(
  configuration: Record<string, unknown> | undefined,
): TlsThresholds {
  const healthRules = (configuration?.health_rules ?? {}) as HealthRules;
  return {
    handshakeTime: readRule(healthRules, HANDSHAKE_TIME_KEYS, DEFAULT_THRESHOLDS.handshakeTime),
    certificateExpiryDays: readRule(healthRules, EXPIRY_DAYS_KEYS, DEFAULT_THRESHOLDS.certificateExpiryDays),
  };
}

export function readTlsConfig(
  configuration: Record<string, unknown> | undefined,
): TlsConfig {
  const cfg = configuration ?? {};
  const ipVersion = cfg.ip_version === "ipv4" || cfg.ip_version === "ipv6" ? cfg.ip_version : "auto";
  const minTlsVersion =
    typeof cfg.min_tls_version === "string" &&
    ["1.0", "1.1", "1.2", "1.3"].includes(cfg.min_tls_version)
      ? cfg.min_tls_version
      : "1.2";
  return {
    port: asNumber(cfg.port) ?? 443,
    serverName: typeof cfg.server_name === "string" ? cfg.server_name : "",
    verifyTls: typeof cfg.verify_tls === "boolean"
      ? cfg.verify_tls
      : typeof cfg.verify_ssl === "boolean" ? cfg.verify_ssl : true,
    minTlsVersion,
    timeoutMs: asNumber(cfg.timeout_ms) ?? 10000,
    ipVersion,
    thresholds: readTlsThresholds(cfg),
  };
}
