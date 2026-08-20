// TCP metric normalization. This is the data layer that turns raw
// ProbeResult[] / ProbeSeries[] payloads into stable, UI-friendly structures,
// mirroring the Ping/HTTP patterns.

import type { ProbeResult, MetricPoint } from "@/entities/monitor/model/result";

export interface TcpSeriesPoint {
  time: string;
  value: number;
}

export interface TcpChartSeries {
  /** metric key, e.g. "connect_time_ms" */
  metric: string;
  location: string;
  probeName: string;
  points: TcpSeriesPoint[];
}

export interface TcpProbeStat {
  probeId: string;
  location: string;
  success: boolean;
  connectTimeMs: number | null;
  remoteAddress: string | null;
  errorCode: string | null;
  errorMessage: string | null;
  lastCheckedAt: string | null;
}

export interface TcpSummary {
  availability: number | null;
  totalChecks: number;
  successChecks: number;
  failedChecks: number;
  connectTimeMs: number | null;
}

/** Metric keys the TCP probe executor actually writes into `metrics`. */
export const TCP_METRIC_KEYS = {
  reachability: ["reachability"],
  connectTime: ["connect_time_ms", "connection_time_ms", "connect_duration_ms"],
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

export function toTcpProbeStats(latest: ProbeResult[]): TcpProbeStat[] {
  return latest.map((result) => {
    const location =
      (result.attributes?.probe_name as string) ||
      (result.attributes?.probe_code as string) ||
      result.probe_location_id ||
      "—";
    const remoteAddress = result.attributes?.remote_address as string | undefined;
    return {
      probeId: result.probe_location_id || result.id,
      location,
      success: Boolean(result.success),
      connectTimeMs: numberValue(result, TCP_METRIC_KEYS.connectTime),
      remoteAddress: remoteAddress ?? null,
      errorCode: result.error_code ?? null,
      errorMessage: result.error_message ?? null,
      lastCheckedAt: result.finished_at ?? result.started_at ?? null,
    };
  });
}

export function summarizeTcp(latest: ProbeResult[]): TcpSummary {
  const stats = toTcpProbeStats(latest);
  const total = stats.length;
  const success = stats.filter((s) => s.success).length;

  const connectTimes = stats
    .map((s) => s.connectTimeMs)
    .filter((v): v is number => v != null);

  const average = (values: number[]): number | null =>
    values.length ? values.reduce((a, b) => a + b, 0) / values.length : null;

  return {
    availability: total > 0 ? (success / total) * 100 : null,
    totalChecks: total,
    successChecks: success,
    failedChecks: total - success,
    connectTimeMs: average(connectTimes),
  };
}

export function toTcpChartSeries(
  series: Array<{
    probe_id: string;
    probe_name: string;
    location: string;
    metric_key?: string;
    points: MetricPoint[];
  }>,
  metric: string,
): TcpChartSeries[] {
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
