// DNS metric normalization. This is the data layer that turns raw
// ProbeResult[] / ProbeSeries[] payloads into stable, UI-friendly structures,
// mirroring the Ping/HTTP patterns.

import type { ProbeResult, MetricPoint } from "@/entities/monitor/model/result";

export interface DnsSeriesPoint {
  time: string;
  value: number;
}

export interface DnsChartSeries {
  /** metric key, e.g. "response_time_ms" */
  metric: string;
  location: string;
  probeName: string;
  points: DnsSeriesPoint[];
}

export interface DnsProbeStat {
  probeId: string;
  location: string;
  success: boolean;
  responseTimeMs: number | null;
  answerCount: number | null;
  expectedMatch: boolean | null;
  resolver: string | null;
  errorCode: string | null;
  errorMessage: string | null;
  lastCheckedAt: string | null;
}

export interface DnsSummary {
  availability: number | null;
  totalChecks: number;
  successChecks: number;
  failedChecks: number;
  responseTimeMs: number | null;
  answerCount: number | null;
}

/** Metric keys the DNS probe executor actually writes into `metrics`. */
export const DNS_METRIC_KEYS = {
  reachability: ["reachability"],
  responseTime: ["response_time_ms", "resolution_duration_ms", "dns_duration_ms"],
  answerCount: ["answer_count"],
  expectedRecordMatch: ["expected_record_match", "record_match"],
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

export function toDnsProbeStats(latest: ProbeResult[]): DnsProbeStat[] {
  return latest.map((result) => {
    const location =
      (result.attributes?.probe_name as string) ||
      (result.attributes?.probe_code as string) ||
      result.probe_location_id ||
      "—";
    const match = numberValue(result, DNS_METRIC_KEYS.expectedRecordMatch);
    return {
      probeId: result.probe_location_id || result.id,
      location,
      success: Boolean(result.success),
      responseTimeMs: numberValue(result, DNS_METRIC_KEYS.responseTime),
      answerCount: numberValue(result, DNS_METRIC_KEYS.answerCount),
      expectedMatch: match == null ? null : match === 1,
      resolver: (result.attributes?.resolver as string | undefined) ?? null,
      errorCode: result.error_code ?? null,
      errorMessage: result.error_message ?? null,
      lastCheckedAt: result.finished_at ?? result.started_at ?? null,
    };
  });
}

export function summarizeDns(latest: ProbeResult[]): DnsSummary {
  const stats = toDnsProbeStats(latest);
  const total = stats.length;
  const success = stats.filter((s) => s.success).length;

  const responseTimes = stats
    .map((s) => s.responseTimeMs)
    .filter((v): v is number => v != null);
  const answerCounts = stats
    .map((s) => s.answerCount)
    .filter((v): v is number => v != null);

  const average = (values: number[]): number | null =>
    values.length ? values.reduce((a, b) => a + b, 0) / values.length : null;

  return {
    availability: total > 0 ? (success / total) * 100 : null,
    totalChecks: total,
    successChecks: success,
    failedChecks: total - success,
    responseTimeMs: average(responseTimes),
    answerCount: average(answerCounts),
  };
}

export function toDnsChartSeries(
  series: Array<{
    probe_id: string;
    probe_name: string;
    location: string;
    metric_key?: string;
    points: MetricPoint[];
  }>,
  metric: string,
): DnsChartSeries[] {
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
