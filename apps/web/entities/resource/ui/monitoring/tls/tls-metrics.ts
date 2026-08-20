// TLS metric normalization. This is the data layer that turns raw
// ProbeResult[] / ProbeSeries[] payloads into stable, UI-friendly structures,
// mirroring the Ping/HTTP patterns.

import type { ProbeResult, MetricPoint } from "@/entities/monitor/model/result";

export interface TlsSeriesPoint {
  time: string;
  value: number;
}

export interface TlsChartSeries {
  /** metric key, e.g. "handshake_time_ms" or "certificate_expiry_days" */
  metric: string;
  location: string;
  probeName: string;
  points: TlsSeriesPoint[];
}

export interface TlsProbeStat {
  probeId: string;
  location: string;
  success: boolean;
  handshakeTimeMs: number | null;
  certificateExpiryDays: number | null;
  certificateValid: boolean | null;
  hostnameMatch: boolean | null;
  chainValid: boolean | null;
  verified: boolean;
  tlsVersion: string | null;
  issuer: string | null;
  errorCode: string | null;
  errorMessage: string | null;
  lastCheckedAt: string | null;
}

export interface TlsSummary {
  availability: number | null;
  totalChecks: number;
  successChecks: number;
  failedChecks: number;
  handshakeTimeMs: number | null;
  certificateExpiryDays: number | null;
}

/** Metric keys the TLS probe executor actually writes into `metrics`. */
export const TLS_METRIC_KEYS = {
  reachability: ["reachability"],
  handshakeTime: ["handshake_time_ms", "handshake_duration_ms"],
  certificateExpiryDays: ["certificate_expiry_days", "days_remaining", "cert_days_remaining"],
  certificateValid: ["certificate_valid", "cert_valid"],
  hostnameMatch: ["hostname_match"],
  chainValid: ["chain_valid", "chain_trusted"],
} as const;

function numberValue(result: ProbeResult, keys: readonly string[]): number | null {
  const metrics = result.metrics ?? {};
  for (const key of keys) {
    const raw = metrics[key];
    if (typeof raw === "number" && !Number.isNaN(raw)) return raw;
    if (typeof raw === "string" && raw.trim() !== "" && !Number.isNaN(Number(raw))) {
      return Number(raw);
    }
  }
  return null;
}

export function toTlsProbeStats(latest: ProbeResult[]): TlsProbeStat[] {
  return latest.map((result) => {
    const location =
      (result.attributes?.probe_name as string) ||
      (result.attributes?.probe_code as string) ||
      result.probe_location_id ||
      "—";
    const boolOf = (value: number | null): boolean | null =>
      value == null ? null : value === 1;
    return {
      probeId: result.probe_location_id || result.id,
      location,
      success: Boolean(result.success),
      handshakeTimeMs: numberValue(result, TLS_METRIC_KEYS.handshakeTime),
      certificateExpiryDays: numberValue(result, TLS_METRIC_KEYS.certificateExpiryDays),
      certificateValid: boolOf(numberValue(result, TLS_METRIC_KEYS.certificateValid)),
      hostnameMatch: boolOf(numberValue(result, TLS_METRIC_KEYS.hostnameMatch)),
      chainValid: boolOf(numberValue(result, TLS_METRIC_KEYS.chainValid)),
      verified: result.attributes?.verified === true,
      tlsVersion: (result.attributes?.tls_version as string | undefined) ?? null,
      issuer: (result.attributes?.certificate_issuer as string | undefined) ?? null,
      errorCode: result.error_code ?? null,
      errorMessage: result.error_message ?? null,
      lastCheckedAt: result.finished_at ?? result.started_at ?? null,
    };
  });
}

export function summarizeTls(latest: ProbeResult[]): TlsSummary {
  const stats = toTlsProbeStats(latest);
  const total = stats.length;
  const success = stats.filter((s) => s.success).length;

  const handshakes = stats
    .map((s) => s.handshakeTimeMs)
    .filter((v): v is number => v != null);
  const expiries = stats
    .map((s) => s.certificateExpiryDays)
    .filter((v): v is number => v != null);

  const average = (values: number[]): number | null =>
    values.length ? values.reduce((a, b) => a + b, 0) / values.length : null;

  return {
    availability: total > 0 ? (success / total) * 100 : null,
    totalChecks: total,
    successChecks: success,
    failedChecks: total - success,
    handshakeTimeMs: average(handshakes),
    certificateExpiryDays: average(expiries),
  };
}

export function toTlsChartSeries(
  series: Array<{
    probe_id: string;
    probe_name: string;
    location: string;
    metric_key?: string;
    points: MetricPoint[];
  }>,
  metric: string,
): TlsChartSeries[] {
  return series.map((s) => ({
    metric,
    location: s.location || s.probe_name,
    probeName: s.probe_name || s.location || s.probe_id,
    points: (s.points ?? []).map((p) => ({
      time: p.timestamp,
      value: p.value,
    })),
  }));
}
