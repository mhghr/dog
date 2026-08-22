"use client";

import { useMemo } from "react";

import type { SparklineSeries } from "@/shared/ui/charts/sparkline";
import { MonitorKpiCard, type KpiHealth } from "@/entities/resource/ui/components/monitor-kpi-card";
import type { MonitorAggregateMetrics } from "@/entities/resource/api/resource.api";
import type { HttpChartSeries } from "./http-metrics";
import type { MetricThreshold } from "./http-config";

interface HttpKpiGridProps {
  isFa: boolean;
  statusCode: number | null;
  statusText: string;
  aggregate: MonitorAggregateMetrics;
  thresholds: MetricThreshold;
  totalProbes: number;
  responseSeries: HttpChartSeries[];
  statusSeries: HttpChartSeries[];
  rangeLabel: string;
}

function sparkOf(series: HttpChartSeries[], transform?: (v: number) => number): SparklineSeries[] {
  return series
    .map((s) => ({
      name: s.probeName || s.location,
      points: transform ? s.points.map((p) => ({ ...p, value: transform(p.value) })) : s.points,
    }))
    .filter((s) => s.points.length > 0);
}

function availabilityHealth(v: number | null): KpiHealth {
  if (v == null) return "unknown";
  if (v >= 99) return "healthy";
  if (v >= 95) return "warning";
  return "critical";
}

function latencyHealth(v: number | null, thresholds: MetricThreshold): KpiHealth {
  if (v == null) return "unknown";
  if (thresholds.critical != null && v >= thresholds.critical) return "critical";
  if (thresholds.warning != null && v >= thresholds.warning) return "warning";
  return "healthy";
}

// Six unified KPI cards for the HTTP section, all fed by the metric layer's
// range-scoped aggregates (P95, error rates) — never recomputed from raw data.
export function HttpKpiGrid({
  isFa,
  statusCode,
  statusText,
  aggregate,
  thresholds,
  totalProbes,
  responseSeries,
  statusSeries,
  rangeLabel,
}: HttpKpiGridProps) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  const spark = useMemo(
    () => ({
      response: sparkOf(responseSeries),
      availability: sparkOf(statusSeries, (v) => v * 100),
      error: sparkOf(statusSeries, (v) => (1 - v) * 100),
    }),
    [responseSeries, statusSeries],
  );

  const statusOk = statusCode != null && statusCode >= 200 && statusCode < 300;
  const { availability, avg_response_time_ms: avg, p95_response_time_ms: p95 } = aggregate;

  const errorRate = aggregate.error_rate;
  const rate4xx = aggregate.rate_4xx ?? 0;
  const rate5xx = aggregate.rate_5xx ?? 0;

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-6">
      <MonitorKpiCard
        title={t("HTTP Status", "وضعیت HTTP")}
        value={statusText}
        secondary={{
          value: aggregate.checks.total_requests != null ? String(aggregate.checks.total_requests) : "—",
          label: t("Checks", "بررسی‌ها"),
        }}
        spark={spark.availability}
        health={statusOk ? "healthy" : statusCode != null ? "critical" : "unknown"}
        reason={
          !statusOk && statusCode != null
            ? t("Unexpected status code", "کد وضعیت غیرمنتظره")
            : undefined
        }
      />

      <MonitorKpiCard
        title={t("Availability", "دسترس‌پذیری")}
        value={availability == null ? "—" : String(Math.round(availability))}
        unit="%"
        secondary={{ value: rangeLabel, label: t("Range", "بازه") }}
        spark={spark.availability}
        health={availabilityHealth(availability)}
        reason={availability != null && availability < 99 ? `${t("Uptime", "آپ‌تایم")} ${Math.round(availability)}%` : undefined}
      />

      <MonitorKpiCard
        title={t("Avg Latency", "میانگین تأخیر")}
        value={avg == null ? "—" : String(Math.round(avg))}
        unit="ms"
        secondary={{ value: rangeLabel, label: t("Range", "بازه") }}
        spark={spark.response}
        health={latencyHealth(avg, thresholds)}
        reason={avg != null && latencyHealth(avg, thresholds) !== "healthy" ? `${Math.round(avg)} ms` : undefined}
      />

      <MonitorKpiCard
        title={t("P95 Latency", "تأخیر P95")}
        value={p95 == null ? "—" : String(Math.round(p95))}
        unit="ms"
        secondary={{ value: rangeLabel, label: t("Range", "بازه") }}
        spark={spark.response}
        health={latencyHealth(p95, thresholds)}
        reason={p95 != null && latencyHealth(p95, thresholds) !== "healthy" ? `${Math.round(p95)} ms` : undefined}
      />

      <MonitorKpiCard
        title={t("Error Rate", "نرخ خطا")}
        value={errorRate == null ? "—" : String(Math.round(errorRate))}
        unit="%"
        secondary={{
          value: errorRate == null ? "—" : `${Math.round(rate4xx)}% / ${Math.round(rate5xx)}%`,
          label: t("4xx / 5xx", "4xx / 5xx"),
        }}
        spark={spark.error}
        health={availabilityHealth(availability)}
        reason={errorRate != null && errorRate > 0 ? `${Math.round(errorRate)}% ${t("errors", "خطا")}` : undefined}
      />

      <MonitorKpiCard
        title={t("Probes", "پراب‌ها")}
        value={`${aggregate.checks.total_requests}`}
        secondary={{ value: `${totalProbes}`, label: t("active", "فعال") }}
        spark={spark.availability}
        health={totalProbes > 0 ? "healthy" : "unknown"}
        reason={totalProbes === 0 ? t("No probes", "بدون پراب") : undefined}
      />
    </div>
  );
}
