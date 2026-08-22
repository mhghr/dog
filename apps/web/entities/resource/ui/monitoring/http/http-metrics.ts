// HTTP metric normalization. This is the data layer that turns raw
// ProbeResult[] / ProbeSeries[] payloads into stable, UI-friendly structures,
// mirroring the Ping pattern (see monitoring/ping/ping-metrics.ts).

import type { ProbeResult, MetricPoint } from "@/entities/monitor/model/result";
import type { MetricThreshold } from "./http-config";

export interface HttpSeriesPoint {
  time: string;
  value: number;
}

export interface HttpChartSeries {
  /** metric key, e.g. "response_time_ms", "ttfb_ms" */
  metric: string;
  /** probe location id (empty/absent for pooled/aggregate series). */
  probeId?: string;
  location: string;
  probeName: string;
  points: HttpSeriesPoint[];
}

export type ProbeHealth = "healthy" | "warning" | "critical" | "down" | "unknown";

export interface HttpBreakdown {
  dns: number | null;
  connect: number | null;
  tls: number | null;
  ttfb: number | null;
  download: number | null;
}

export interface HttpProbeStat {
  probeId: string;
  location: string;
  success: boolean;
  statusCode: number | null;
  responseTimeMs: number | null;
  ttfbMs: number | null;
  errorCode: string | null;
  errorMessage: string | null;
  lastCheckedAt: string | null;
}

export interface HttpProbeHealth extends HttpProbeStat {
  health: ProbeHealth;
  availability: number | null;
  breakdown: HttpBreakdown;
  responseSize: number | null;
}

export interface HttpSummary {
  availability: number | null;
  totalChecks: number;
  successChecks: number;
  failedChecks: number;
  responseTimeMs: number | null;
  ttfbMs: number | null;
  minLatencyMs: number | null;
  maxLatencyMs: number | null;
  p95LatencyMs: number | null;
}

/** Metric keys the HTTP probe executor actually writes into `metrics`. */
export const HTTP_METRIC_KEYS = {
  reachability: ["reachability"],
  responseTime: ["response_time_ms", "total_duration_ms"],
  ttfb: ["ttfb_ms", "time_to_first_byte_ms"],
  dns: ["dns_duration_ms"],
  connect: ["connect_duration_ms"],
  tls: ["tls_duration_ms"],
  requestWrite: ["request_write_ms"],
  download: ["download_time_ms"],
  responseSize: ["response_size_bytes"],
  contentAssertion: ["content_assertion"],
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

export function toHttpProbeStats(latest: ProbeResult[]): HttpProbeStat[] {
  return latest.map((result) => {
    const location =
      (result.attributes?.probe_name as string) ||
      (result.attributes?.probe_code as string) ||
      result.probe_location_id ||
      "—";
    const statusCodeRaw = result.attributes?.status_code;
    return {
      probeId: result.probe_location_id || result.id,
      location,
      success: Boolean(result.success),
      statusCode: typeof statusCodeRaw === "number" ? statusCodeRaw : null,
      responseTimeMs: numberValue(result, HTTP_METRIC_KEYS.responseTime),
      ttfbMs: numberValue(result, HTTP_METRIC_KEYS.ttfb),
      errorCode: result.error_code ?? null,
      errorMessage: result.error_message ?? null,
      lastCheckedAt: result.finished_at ?? result.started_at ?? null,
    };
  });
}

// Classifies a probe's health from its latest result and the configured
// latency thresholds. A failed check is critical; a slow-but-successful check
// is warning/critical per the thresholds.
export function probeHealthOf(
  result: ProbeResult,
  thresholds: MetricThreshold,
): ProbeHealth {
  if (!result.success) return "critical";
  const responseTime = numberValue(result, HTTP_METRIC_KEYS.responseTime);
  if (responseTime == null) return "healthy";
  if (thresholds.critical != null && responseTime >= thresholds.critical) return "critical";
  if (thresholds.warning != null && responseTime >= thresholds.warning) return "warning";
  return "healthy";
}

// Derives the global status from every probe's health: the worst state wins.
export function globalHealthOf(probeHealth: Array<{ health: ProbeHealth }>): ProbeHealth {
  if (probeHealth.length === 0) return "unknown";
  const rank: Record<ProbeHealth, number> = {
    unknown: 0,
    healthy: 1,
    warning: 2,
    critical: 3,
    down: 4,
  };
  let worst: ProbeHealth = "unknown";
  for (const p of probeHealth) {
    if (rank[p.health] > rank[worst]) worst = p.health;
  }
  return worst;
}

const STATUS_TEXT: Record<number, string> = {
  200: "OK",
  301: "Moved Permanently",
  302: "Found",
  400: "Bad Request",
  401: "Unauthorized",
  403: "Forbidden",
  404: "Not Found",
  500: "Internal Server Error",
  502: "Bad Gateway",
  503: "Service Unavailable",
  504: "Gateway Timeout",
};

// Human-readable HTTP status label, e.g. "200 OK" or "503 Service Unavailable".
export function statusLabelOf(code: number | null, fallback: string | null): string {
  if (code != null) {
    const text = STATUS_TEXT[code];
    return text ? `${code} ${text}` : String(code);
  }
  return fallback ?? "—";
}

// The most recent error code across failing probes.
export function lastErrorOf(probeHealth: Array<{ errorCode: string | null; lastCheckedAt: string | null }>): string | null {
  const failed = probeHealth
    .filter((p) => p.errorCode)
    .sort((a, b) => String(b.lastCheckedAt ?? "").localeCompare(String(a.lastCheckedAt ?? "")));
  return failed[0]?.errorCode ?? null;
}

function breakdownOf(result: ProbeResult): HttpBreakdown {
  return {
    dns: numberValue(result, HTTP_METRIC_KEYS.dns),
    connect: numberValue(result, HTTP_METRIC_KEYS.connect),
    tls: numberValue(result, HTTP_METRIC_KEYS.tls),
    ttfb: numberValue(result, HTTP_METRIC_KEYS.ttfb),
    download: numberValue(result, HTTP_METRIC_KEYS.download),
  };
}

// Per-probe health enriched with availability (from the status series over the
// selected range) and the request breakdown available in the latest result.
export function toHttpProbeHealth(
  latest: ProbeResult[],
  statusSeries: HttpChartSeries[],
  thresholds: MetricThreshold,
): HttpProbeHealth[] {
  const availabilityByLocation = new Map<string, number>();
  for (const series of statusSeries) {
    const values = series.points.map((p) => p.value).filter((v) => v != null);
    if (values.length > 0) {
      availabilityByLocation.set(
        series.location || series.probeName,
        (values.reduce((a, b) => a + b, 0) / values.length) * 100,
      );
    }
  }

  return latest.map((result) => {
    const location =
      (result.attributes?.probe_name as string) ||
      (result.attributes?.probe_code as string) ||
      result.probe_location_id ||
      "—";
    const statusCodeRaw = result.attributes?.status_code;
    const stat: HttpProbeHealth = {
      ...toHttpProbeStats([result])[0],
      health: probeHealthOf(result, thresholds),
      availability: availabilityByLocation.get(location) ?? null,
      breakdown: breakdownOf(result),
      responseSize: numberValue(result, HTTP_METRIC_KEYS.responseSize),
    };
    stat.statusCode = typeof statusCodeRaw === "number" ? statusCodeRaw : null;
    return stat;
  });
}

export function summarizeHttp(
  latest: ProbeResult[],
  responseSeries: HttpChartSeries[] = [],
): HttpSummary {
  const stats = toHttpProbeStats(latest);
  const total = stats.length;
  const success = stats.filter((s) => s.success).length;

  const responseTimes = stats
    .map((s) => s.responseTimeMs)
    .filter((v): v is number => v != null);
  const ttfbValues = stats
    .map((s) => s.ttfbMs)
    .filter((v): v is number => v != null);

  // Pooled latency statistics across all probes from the series points.
  const pooled = responseSeries.flatMap((s) =>
    s.points.map((p) => p.value).filter((v): v is number => v != null && v >= 0),
  );

  const average = (values: number[]): number | null =>
    values.length ? values.reduce((a, b) => a + b, 0) / values.length : null;

  const percentile = (values: number[], p: number): number | null => {
    if (values.length === 0) return null;
    const sorted = [...values].sort((a, b) => a - b);
    const index = Math.min(sorted.length - 1, Math.max(0, Math.ceil((p / 100) * sorted.length) - 1));
    return sorted[index];
  };

  return {
    availability: total > 0 ? (success / total) * 100 : null,
    totalChecks: total,
    successChecks: success,
    failedChecks: total - success,
    responseTimeMs: average(responseTimes),
    ttfbMs: average(ttfbValues),
    minLatencyMs: pooled.length ? Math.min(...pooled) : null,
    maxLatencyMs: pooled.length ? Math.max(...pooled) : null,
    p95LatencyMs: percentile(pooled, 95),
  };
}

export function toHttpChartSeries(
  series: Array<{
    probe_id: string;
    probe_name: string;
    location: string;
    metric_key?: string;
    points: MetricPoint[];
  }>,
  metric: string,
): HttpChartSeries[] {
  return series.map((s) => ({
    metric,
    probeId: s.probe_id ?? "",
    location: s.location || s.probe_name,
    probeName: s.probe_name || s.location || s.probe_id,
    points: (s.points ?? []).map((p) => ({
      time: p.timestamp,
      value: p.value,
    })),
  }));
}

export interface HttpDownInterval {
  start: string;
  end: string;
}

// Builds contiguous downtime ranges from the explicit status series (value 0 =
// failed check), mirroring the Ping pattern so the timeline can shade failure
// windows instead of relying on data gaps.
export function buildDownIntervals(series: HttpChartSeries[]): HttpDownInterval[] {
  const intervals: HttpDownInterval[] = [];

  for (const s of series) {
    let start: string | null = null;
    for (const p of s.points) {
      if (p.value === 0) {
        if (start === null) start = p.time;
      } else if (start !== null) {
        intervals.push({ start, end: p.time });
        start = null;
      }
    }
    if (start !== null && s.points.length > 0) {
      intervals.push({ start, end: s.points[s.points.length - 1].time });
    }
  }
  return intervals;
}

/** Format a KPI value, keeping integers whole and embedding no unit. */
export function formatHttpKpiValue(
  value: number | null,
  format: "ms" | "bytes" | "percent",
  down: boolean,
): string {
  if (value != null) {
    if (format === "bytes") return String(Math.round(value));
    if (format === "percent") return String(Math.round(value));
    return String(Math.round(value));
  }
  if (down) return format === "bytes" ? "—" : "∞";
  return "N/A";
}

// ── Enterprise HTTP helpers ────────────────────────────────────

// Timing phases a single HTTP check records, in execution order. Each phase
// reads a dedicated executor metric so the waterfall never parses error text.
export interface HttpTimingPhase {
  key: string;
  label: { en: string; fa: string };
  /** Duration in ms, null when the phase did not happen (e.g. no DNS). */
  ms: number | null;
  color: string;
}

export function breakdownOfResult(result: ProbeResult): HttpBreakdown {
  return {
    dns: numberValue(result, HTTP_METRIC_KEYS.dns),
    connect: numberValue(result, HTTP_METRIC_KEYS.connect),
    tls: numberValue(result, HTTP_METRIC_KEYS.tls),
    ttfb: numberValue(result, HTTP_METRIC_KEYS.ttfb),
    download: numberValue(result, HTTP_METRIC_KEYS.download),
  };
}

export interface HttpWaterfallPhase {
  key: string;
  label: { en: string; fa: string };
  ms: number | null;
  color: string;
}

// Phases ordered by execution: DNS → TCP → TLS → TTFB → Download. Total is the
// full response time. Colors match the chart palette roles (sky/cyan/violet/
// primary/success/muted).
export function waterfallPhasesOf(result: ProbeResult): HttpWaterfallPhase[] {
  const b = breakdownOfResult(result);
  return [
    { key: "dns", label: { en: "DNS", fa: "DNS" }, ms: b.dns, color: "#0EA5E9" },
    { key: "connect", label: { en: "Connect", fa: "اتصال" }, ms: b.connect, color: "#06B6D4" },
    { key: "tls", label: { en: "TLS", fa: "TLS" }, ms: b.tls, color: "#8B5CF6" },
    { key: "ttfb", label: { en: "TTFB", fa: "TTFB" }, ms: b.ttfb, color: "#3B82F6" },
    { key: "download", label: { en: "Download", fa: "دریافت" }, ms: b.download, color: "#10B981" },
  ];
}

// Human-readable, machine-label-free rendering of a failure stage. Stages come
// straight from the probe's `attributes.failure_stage` (dns/tcp/tls/timeout/
// http/assertion), never parsed from error strings.
export function failureStageLabel(stage: string | null, isFa: boolean): string {
  if (!stage) return isFa ? "نامشخص" : "Unknown";
  const map: Record<string, { en: string; fa: string }> = {
    dns: { en: "DNS Resolution", fa: "تفکیک DNS" },
    tcp: { en: "TCP Connection", fa: "اتصال TCP" },
    tls: { en: "TLS Handshake", fa: "دست‌داد TLS" },
    timeout: { en: "Timeout", fa: "مهلت" },
    http: { en: "HTTP Response", fa: "پاسخ HTTP" },
    assertion: { en: "Content Assertion", fa: "تطابق محتوا" },
  };
  const entry = map[stage];
  return entry ? (isFa ? entry.fa : entry.en) : stage;
}

// Failure-stages mapped to their user-facing explanation and remediation hint.
export interface FailureStageInfo {
  title: { en: string; fa: string };
  detail: { en: string; fa: string };
  hint: { en: string; fa: string };
}

export const FAILURE_STAGE_INFO: Record<string, FailureStageInfo> = {
  dns: {
    title: { en: "DNS resolution failed", fa: "خطا در تفکیک DNS" },
    detail: { en: "The hostname could not be resolved to an IP address.", fa: "نام دامنه به آدرس IP تبدیل نشد." },
    hint: { en: "Check the domain spelling and the upstream DNS servers.", fa: "املای دامنه و سرورهای DNS بالادستی را بررسی کنید." },
  },
  tcp: {
    title: { en: "Connection refused or failed", fa: "اتصال برقرار نشد" },
    detail: { en: "The TCP connection to the target could not be established.", fa: "اتصال TCP به مقصد برقرار نشد." },
    hint: { en: "Verify the host and port are reachable and the firewall allows it.", fa: "میزان دسترس‌پذیری هاست و پورت و فایروال را بررسی کنید." },
  },
  tls: {
    title: { en: "TLS handshake failed", fa: "خطا در دست‌داد TLS" },
    detail: { en: "The TLS handshake with the server failed or the certificate is invalid.", fa: "دست‌داد TLS با سرور ناموفق بود یا گواهی نامعتبر است." },
    hint: { en: "Check the certificate validity, chain and hostname match.", fa: "اعتبار، زنجیره و تطابق نام میزبان گواهی را بررسی کنید." },
  },
  timeout: {
    title: { en: "Request timed out", fa: "مهلت درخواست به پایان رسید" },
    detail: { en: "The request exceeded the configured timeout.", fa: "درخواست از مهلت تنظیم‌شده تجاوز کرد." },
    hint: { en: "Check server load or raise the timeout in the monitor settings.", fa: "بار سرور را بررسی یا مهلت را در تنظیمات افزایش دهید." },
  },
  http: {
    title: { en: "Unexpected HTTP response", fa: "پاسخ HTTP غیرمنتظره" },
    detail: { en: "The response did not match the expected status codes.", fa: "پاسخ با کدهای وضعیت مورد انتظار مطابقت نداشت." },
    hint: { en: "Review the expected status codes and the endpoint behavior.", fa: "کدهای وضعیت مورد انتظار و رفتار نقطه پایانی را بررسی کنید." },
  },
  assertion: {
    title: { en: "Content assertion failed", fa: "تطابق محتوا ناموفق بود" },
    detail: { en: "The response body did not contain the expected text.", fa: "بدنه پاسخ متن مورد انتظار را نداشت." },
    hint: { en: "Confirm the expected text still appears in the page content.", fa: "مطمئن شوید متن مورد انتظار همچنان در محتوای صفحه است." },
  },
};

// Maps a probe error code to a stable tone for the failure analysis card.
export function failureToneOf(errorCode: string | null): "warning" | "destructive" | "info" {
  if (!errorCode) return "info";
  if (errorCode === "timeout" || errorCode === "body_assertion_failed" || errorCode === "unexpected_status_code") {
    return "warning";
  }
  return "destructive";
}
