"use client";

import { useMemo, useState } from "react";
import { useLocale } from "next-intl";

import { Skeleton } from "@/shared/ui/skeleton";
import { StatusBadge } from "@/design-system/components/status-badge";
import {
  useResourceMonitorMetrics,
  useResourceMonitorResultAt,
  useResourceMonitorStatus,
  type MetricsRange,
} from "@/entities/resource/hooks/use-resource";
import type { Monitor } from "@/entities/resource/hooks/types";
import { readHttpConfig } from "./http-config";
import { toHttpChartSeries, statusLabelOf, type HttpChartSeries, type ProbeHealth } from "./http-metrics";
import { HttpKpiGrid } from "./HttpKpiGrid";
import {
  HttpPerformanceChart,
  HTTP_CHART_METRICS,
  type HttpChartMetric,
} from "./HttpPerformanceChart";
import { HttpWaterfall } from "./HttpWaterfall";
import { HttpResponseDetails } from "./HttpResponseDetails";
import { HttpFailureAnalysis } from "./HttpFailureAnalysis";
import { HttpProbeTable } from "./HttpProbeTable";
import { HttpResponsesCard } from "./HttpResponsesCard";
import { HttpAlertRules } from "./HttpAlertRules";
import { PingTimeRangeSelector } from "../ping/PingTimeRangeSelector";

const EMPTY_AGGREGATE = {
  checks: { total_requests: 0, successful_requests: 0, failed_requests: 0 },
  availability: null,
  avg_response_time_ms: null,
  p95_response_time_ms: null,
  avg_ttfb_ms: null,
  error_rate: null,
  codes_4xx: 0,
  rate_4xx: null,
  codes_5xx: 0,
  rate_5xx: null,
};

// Executor metric key backing each chart metric. response_time is the default
// series (no `metric` param) and error_rate derives from the status series.
const METRIC_EXECUTOR_KEY: Record<Exclude<HttpChartMetric, "response_time" | "error_rate">, string> = {
  ttfb: "ttfb_ms",
  dns: "dns_duration_ms",
  connect: "connect_duration_ms",
  tls: "tls_duration_ms",
};

const HEALTH_TONE: Record<ProbeHealth, "success" | "warning" | "destructive" | "muted"> = {
  healthy: "success",
  warning: "warning",
  critical: "destructive",
  down: "destructive",
  unknown: "muted",
};

// Pools per-probe series into one mean-per-timestamp series for the aggregate
// view. Buckets are aligned (date_bin), so timestamps merge cleanly.
function aggregateSeries(series: HttpChartSeries[]): HttpChartSeries {
  const byTime = new Map<string, number[]>();
  for (const s of series) {
    for (const p of s.points) {
      const bucket = byTime.get(p.time) ?? [];
      bucket.push(p.value);
      byTime.set(p.time, bucket);
    }
  }
  const points = Array.from(byTime.entries())
    .sort((a, b) => a[0].localeCompare(b[0]))
    .map(([time, values]) => ({ time, value: values.reduce((a, b) => a + b, 0) / values.length }));
  return {
    metric: series[0]?.metric ?? "",
    probeId: "",
    location: "",
    probeName: "",
    points,
  };
}

export function HttpMonitoringView({
  resourceId,
  monitor,
}: {
  resourceId: string;
  monitor: Monitor;
}) {
  const locale = useLocale();
  const isFa = locale === "fa";
  const [range, setRange] = useState<MetricsRange>("1h");
  const [metric, setMetric] = useState<HttpChartMetric>("response_time");
  const [mode, setMode] = useState<"aggregate" | "probes">("aggregate");
  const [probeId, setProbeId] = useState<string | null>(null);
  const [selectedAt, setSelectedAt] = useState<string | null>(null);

  const t = (en: string, fa: string) => (isFa ? fa : en);

  const config = useMemo(() => readHttpConfig(monitor.configuration), [monitor.configuration]);

  // Base payload: response_time series + latest results + range aggregates.
  const baseQuery = useResourceMonitorMetrics(resourceId, monitor.id, range);
  const statusQuery = useResourceMonitorStatus(resourceId, monitor.id, range);

  // Series for the active chart metric (cached per metric; the response_time /
  // error_rate variants resolve to the base / status queries above).
  const metricKey =
    metric === "response_time" ? undefined : metric === "error_rate" ? "status" : METRIC_EXECUTOR_KEY[metric];
  const metricQuery = useResourceMonitorMetrics(resourceId, monitor.id, range, metricKey);

  const drillQuery = useResourceMonitorResultAt(resourceId, monitor.id, range, selectedAt, metricKey);

  const latest = useMemo(() => baseQuery.data?.latest ?? [], [baseQuery.data?.latest]);
  const aggregate = baseQuery.data?.aggregate;
  const probes = baseQuery.data?.probes ?? [];

  const responseSeries = useMemo(
    () => toHttpChartSeries(baseQuery.data?.series ?? [], "response_time_ms"),
    [baseQuery.data?.series],
  );
  const statusSeries = useMemo(
    () => toHttpChartSeries(statusQuery.data?.series ?? [], "status"),
    [statusQuery.data?.series],
  );

  const rawMetricSeries = useMemo(() => {
    const raw =
      metric === "response_time"
        ? (baseQuery.data?.series ?? [])
        : metric === "error_rate"
          ? (statusQuery.data?.series ?? [])
          : (metricQuery.data?.series ?? []);
    return toHttpChartSeries(raw, metricKey ?? "response_time_ms").map((s) => ({
      ...s,
      points:
        metric === "error_rate"
          ? s.points.map((p) => ({ ...p, value: (1 - p.value) * 100 }))
          : s.points,
    }));
  }, [metric, metricKey, baseQuery.data?.series, statusQuery.data?.series, metricQuery.data?.series]);

  // Probe filter → aggregate (pooled) or per-probe lines.
  const chartSeries = useMemo(() => {
    const filtered = probeId ? rawMetricSeries.filter((s) => s.probeId === probeId) : rawMetricSeries;
    if (mode === "aggregate" && filtered.length > 0) {
      return [aggregateSeries(filtered)];
    }
    return filtered;
  }, [rawMetricSeries, probeId, mode]);

  const activeMetric = HTTP_CHART_METRICS.find((m) => m.key === metric) ?? HTTP_CHART_METRICS[0];

  // warn/crit marklines per latency metric come from the saved config only.
  const chartThresholds =
    metric === "response_time"
      ? config.thresholds.responseTime
      : metric === "ttfb"
        ? config.thresholds.ttfb
        : metric === "dns"
          ? config.thresholds.dnsDuration
          : metric === "connect"
            ? config.thresholds.connectDuration
            : metric === "tls"
              ? config.thresholds.tlsDuration
              : undefined;

  // Detail result: the drilled-down check, else the latest check of the
  // selected probe (or the most recent overall).
  const detailResult = useMemo(() => {
    if (drillQuery.data) return drillQuery.data;
    const pool = probeId ? latest.filter((l) => l.probe_location_id === probeId) : latest;
    return pool.length > 0 ? pool[0] : null;
  }, [drillQuery.data, probeId, latest]);

  const latestStatusCode =
    detailResult && typeof detailResult.attributes?.status_code === "number"
      ? (detailResult.attributes.status_code as number)
      : null;

  const hasData = (aggregate?.checks.total_requests ?? 0) > 0;
  const isLoading = baseQuery.isPending || statusQuery.isPending;
  const isError = (baseQuery.isError && !baseQuery.isFetching) || (statusQuery.isError && !statusQuery.isFetching);

  const targetUrl = config.url || monitor.resource_target || "";
  const lastCheckedAt = baseQuery.data?.last_success_at ?? detailResult?.finished_at ?? detailResult?.started_at ?? null;

  const health: ProbeHealth = useMemo(() => {
    if (!detailResult) return "unknown";
    if (!detailResult.success) return "critical";
    const rt = detailResult.metrics?.response_time_ms;
    if (typeof rt === "number") {
      if (config.thresholds.responseTime.warning != null && rt >= config.thresholds.responseTime.warning) {
        return "warning";
      }
    }
    return "healthy";
  }, [detailResult, config.thresholds.responseTime.warning]);

  const switchMetric = (next: HttpChartMetric) => {
    if (next !== metric) {
      setMetric(next);
      setSelectedAt(null);
    }
  };

  const selectPoint = (timestamp: string) => {
    setSelectedAt(timestamp);
  };

  return (
    <section className="flex flex-col gap-6">
      {/* Header: monitor name + URL, global status, last check, time range. */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          <h2 className="truncate text-base font-semibold tracking-tight text-foreground">{monitor.name}</h2>
          {targetUrl && (
            <code className="max-w-[55%] truncate rounded-md bg-muted/50 px-2 py-1 font-mono text-xs text-muted-foreground" dir="ltr">
              {targetUrl}
            </code>
          )}
          <StatusBadge tone={HEALTH_TONE[health]} label={statusLabelOf(latestStatusCode, lastErrorLabel(latest, isFa))} />
          {lastCheckedAt && (
            <span className="hidden text-xs text-muted-foreground sm:block">
              {t("Last check", "آخرین بررسی")} <span className="tabular-nums" dir="ltr">{formatCheckTime(lastCheckedAt)}</span>
            </span>
          )}
        </div>
        <PingTimeRangeSelector range={range} onChange={setRange} />
      </div>

      {isError ? (
        <div className="flex flex-col items-center justify-center gap-2 rounded-xl border border-border/60 bg-card px-6 py-16 text-sm text-muted-foreground shadow-subtle">
          <span>{t("Unable to load data", "خطا در دریافت داده")}</span>
        </div>
      ) : isLoading ? (
        <div className="space-y-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-6">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-36 rounded-xl" />
            ))}
          </div>
          <Skeleton className="h-80 rounded-xl" />
          <Skeleton className="h-64 rounded-xl" />
        </div>
      ) : !hasData ? (
        <div className="flex flex-col items-center justify-center gap-2 rounded-xl border border-border/60 bg-card px-6 py-16 shadow-subtle">
          <p className="text-sm font-medium text-foreground/80">
            {t("No monitoring data yet", "هنوز داده مانیتورینگ وجود ندارد")}
          </p>
          <p className="text-xs text-muted-foreground">
            {t("The monitor is active but has not produced results in this range.", "مانیتور فعال است اما در این بازه نتیجه‌ای تولید نکرده است.")}
          </p>
        </div>
      ) : (
        <>
          <HttpKpiGrid
            isFa={isFa}
            statusCode={latestStatusCode}
            statusText={statusLabelOf(latestStatusCode, lastErrorLabel(latest, isFa))}
            aggregate={aggregate ?? EMPTY_AGGREGATE}
            thresholds={config.thresholds.responseTime}
            totalProbes={probes.length}
            responseSeries={responseSeries}
            statusSeries={statusSeries}
            rangeLabel={range}
          />

          <HttpPerformanceChart
            series={chartSeries}
            metric={metric}
            unit={activeMetric.unit}
            thresholds={chartThresholds}
            mode={mode}
            isLoading={metricQuery.isPending || (metricQuery.isFetching && !metricQuery.data)}
            isError={isError}
            isFa={isFa}
            onChangeMetric={switchMetric}
            onChangeMode={setMode}
            onSelectPoint={selectPoint}
          />

          <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
            <HttpWaterfall result={detailResult} isFa={isFa} />
            <HttpResponseDetails result={detailResult} isFa={isFa} />
            <HttpFailureAnalysis result={detailResult} isFa={isFa} />
          </div>

          <HttpProbeTable probes={probes} selectedProbeId={probeId} isFa={isFa} onSelect={setProbeId} />

          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            <HttpResponsesCard buckets={baseQuery.data?.status_codes ?? []} rangeLabel={range} isFa={isFa} />
            <HttpAlertRules isFa={isFa} />
          </div>
        </>
      )}
    </section>
  );
}

function lastErrorLabel(
  latest: Array<{ success: boolean; error_code?: string }>,
  isFa: boolean,
): string {
  const failed = latest.find((r) => !r.success);
  if (!failed) return isFa ? "در حال بررسی" : "—";
  return failed.error_code ?? (isFa ? "خطا" : "Error");
}

function formatCheckTime(iso: string): string {
  const then = new Date(iso).getTime();
  const diff = Math.max(0, Date.now() - then);
  const min = Math.floor(diff / 60000);
  const sec = Math.floor((diff % 60000) / 1000);
  if (min >= 60) {
    const h = Math.floor(min / 60);
    const m = min % 60;
    return `${h}h ${m}m`;
  }
  return `${min}m ${sec}s`;
}
