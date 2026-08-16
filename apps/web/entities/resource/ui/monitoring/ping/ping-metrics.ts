// Ping metric normalization. This is the data layer that turns raw
// ProbeResult[] / ProbeSeries[] payloads into stable, UI-friendly structures.
// Presentation components must consume these helpers, not inspect raw
// `metrics` maps or VictoriaMetrics/PromQL response shapes.

import type { ProbeResult, MetricPoint } from "@/entities/monitor/model/result";

export interface PingSeriesPoint {
  time: string;
  value: number;
}

export interface PingChartSeries {
  /** metric key, e.g. "rtt_ms", "packet_loss_percent", "jitter_ms" */
  metric: string;
  location: string;
  probeName: string;
  points: PingSeriesPoint[];
}

export interface PingProbeStat {
  probeId: string;
  location: string;
  success: boolean;
  latency: number | null;
  packetLoss: number | null;
  jitter: number | null;
  lastCheckedAt: string | null;
  durationMillis: number;
}

export interface PingSummary {
  availability: number | null;
  totalChecks: number;
  successChecks: number;
  failedChecks: number;
  latency: number | null;
  latencyMin: number | null;
  latencyMax: number | null;
  packetLoss: number | null;
  jitter: number | null;
  jitterMax: number | null;
  packetsSent: number;
  packetsReceived: number;
  packetsLost: number;
}

/** Metric keys the ping probe executor actually writes into `metrics`. */
export const PING_METRIC_KEYS = {
  rtt: ["rtt_ms", "latency_ms", "avg_rtt_ms"],
  minRtt: ["min_rtt_ms"],
  maxRtt: ["max_rtt_ms"],
  packetLoss: ["packet_loss_percent", "packet_loss"],
  jitter: ["jitter_ms"],
} as const;

export function getMetricValue(
  result: ProbeResult,
  keys: readonly string[],
): number | null {
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

function attrNumber(result: ProbeResult, keys: string[]): number {
  const attrs = (result.attributes ?? {}) as Record<string, unknown>;
  for (const key of keys) {
    const raw = attrs[key];
    if (typeof raw === "number" && !Number.isNaN(raw)) return raw;
    if (typeof raw === "string" && raw.trim() !== "" && !Number.isNaN(Number(raw))) {
      return Number(raw);
    }
  }
  return 0;
}

function average(values: number[]): number | null {
  if (values.length === 0) return null;
  return values.reduce((a, b) => a + b, 0) / values.length;
}

export function toProbeStats(latest: ProbeResult[]): PingProbeStat[] {
  return latest.map((result) => {
    return {
      probeId: result.probe_location_id ?? result.id,
      location:
        (result.attributes?.probe_name as string) ||
        (result.attributes?.probe_code as string) ||
        result.probe_location_id ||
        "—",
      success: Boolean(result.success),
      latency: getMetricValue(result, PING_METRIC_KEYS.rtt),
      packetLoss: getMetricValue(result, PING_METRIC_KEYS.packetLoss),
      jitter: getMetricValue(result, PING_METRIC_KEYS.jitter),
      lastCheckedAt: result.finished_at ?? result.started_at ?? null,
      durationMillis:
        typeof result.duration_millis === "number" ? result.duration_millis : 0,
    };
  });
}

export function summarize(latest: ProbeResult[]): PingSummary {
  const stats = toProbeStats(latest);
  const total = stats.length;
  const success = stats.filter((s) => s.success).length;

  const latencies = stats
    .map((s) => s.latency)
    .filter((v): v is number => v != null);
  const packetLosses = stats
    .map((s) => s.packetLoss)
    .filter((v): v is number => v != null);
  const jitters = stats
    .map((s) => s.jitter)
    .filter((v): v is number => v != null);

  const packetsSent = latest.reduce(
    (sum, r) => sum + attrNumber(r, ["packets_sent"]),
    0,
  );
  const packetsReceived = latest.reduce(
    (sum, r) => sum + attrNumber(r, ["packets_received"]),
    0,
  );

  return {
    availability: total > 0 ? (success / total) * 100 : null,
    totalChecks: total,
    successChecks: success,
    failedChecks: total - success,
    latency: average(latencies),
    latencyMin: latencies.length ? Math.min(...latencies) : null,
    latencyMax: latencies.length ? Math.max(...latencies) : null,
    packetLoss: average(packetLosses),
    jitter: average(jitters),
    jitterMax: jitters.length ? Math.max(...jitters) : null,
    packetsSent,
    packetsReceived,
    packetsLost: Math.max(0, packetsSent - packetsReceived),
  };
}

export function toChartSeries(
  series: Array<{
    probe_id: string;
    probe_name: string;
    location: string;
    metric_key?: string;
    points: MetricPoint[];
  }>,
  metric: string,
): PingChartSeries[] {
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

// KPI display formatter. When a metric is missing and the monitor is down,
// latency/jitter read as "∞" (unreachable) and packet loss as 100. Zero is
// never shown for a down target — absence of data is a distinct state.
//
// These return the BARE value; the KpiCard renders the unit in its own span.
// Use `formatPingKpiValueWithUnit` for inline row values.
export function formatPingKpiValue(
  value: number | null,
  format: "ms" | "percent",
  down: boolean,
): string {
  if (value != null) {
    return format === "ms" ? String(Math.round(value)) : value.toFixed(2);
  }
  if (down) {
    return format === "ms" ? "∞" : "100";
  }
  return "N/A";
}

// Same as `formatPingKpiValue` but embeds the unit for standalone rendering.
export function formatPingKpiValueWithUnit(
  value: number | null,
  format: "ms" | "percent",
  down: boolean,
): string {
  const bare = formatPingKpiValue(value, format, down);
  if (bare === "N/A") return bare;
  return `${bare}${format === "ms" ? " ms" : "%"}`;
}

// Computes [start, end) half-open windows where a status series reports fully
// down (value === 0). The end of the last window is the timestamp of the last
// point. Series points must be ordered ascending by time.
export interface DownInterval {
  start: string;
  end: string;
}

export function buildDownIntervals(series: PingChartSeries[]): DownInterval[] {
  const intervals: DownInterval[] = [];
  let start: string | null = null;

  for (const s of series) {
    for (const p of s.points) {
      if (p.value === 0) {
        if (start === null) start = p.time;
      } else if (start !== null) {
        intervals.push({ start, end: p.time });
        start = null;
      }
    }
  }
  if (start !== null) {
    intervals.push({ start, end: start });
  }
  return intervals;
}
