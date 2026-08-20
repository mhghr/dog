"use client";

import { useMemo } from "react";

import type { SparklineSeries } from "@/shared/ui/charts/sparkline";
import { MonitorKpiCard, type KpiHealth } from "@/entities/resource/ui/components/monitor-kpi-card";
import type { HttpChartSeries } from "./http-metrics";

interface HttpKpiGridProps {
  isFa: boolean;
  currentStatusCode: number | null;
  statusText: string;
  currentLatencyMs: number | null;
  avgLatencyMs: number | null;
  p95LatencyMs: number | null;
  availability: number | null;
  errorRate: number | null;
  successRate: number | null;
  healthy: number;
  warning: number;
  critical: number;
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

// Six unified KPI cards for the HTTP section: HTTP Status, Response Time,
// Availability, Error Rate, Success Rate and Probe Health. The main value is
// always the latest real value; the secondary line and sparkline follow the
// selected time range.
export function HttpKpiGrid({
  isFa,
  currentStatusCode,
  statusText,
  currentLatencyMs,
  avgLatencyMs,
  p95LatencyMs,
  availability,
  errorRate,
  successRate,
  healthy,
  warning,
  critical,
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

  const statusOk = currentStatusCode != null && currentStatusCode >= 200 && currentStatusCode < 300;

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-6">
      <MonitorKpiCard
        title={t("HTTP Status", "وضعیت HTTP")}
        value={statusText}
        secondary={{
          value: successRate == null ? "—" : `${successRate.toFixed(1)}%`,
          label: t("Success rate", "نرخ موفقیت"),
        }}
        spark={spark.availability}
        health={statusOk ? "healthy" : currentStatusCode != null ? "critical" : "unknown"}
        reason={
          !statusOk && currentStatusCode != null
            ? t("Unexpected status code", "کد وضعیت غیرمنتظره")
            : undefined
        }
      />

      <MonitorKpiCard
        title={t("Response Time", "زمان پاسخ")}
        value={currentLatencyMs == null ? "—" : String(Math.round(currentLatencyMs))}
        unit="ms"
        secondary={{
          value: avgLatencyMs == null ? "—" : `${Math.round(avgLatencyMs)} ms`,
          label: t("Avg", "میانگین"),
        }}
        spark={spark.response}
        health={availabilityHealth(availability)}
        reason={p95LatencyMs != null && p95LatencyMs >= 5000 ? t("High latency", "تأخیر بالا") : undefined}
      />

      <MonitorKpiCard
        title={t("Availability", "دسترس‌پذیری")}
        value={availability == null ? "—" : availability.toFixed(2)}
        unit="%"
        secondary={{ value: rangeLabel, label: t("Range", "بازه") }}
        spark={spark.availability}
        health={availabilityHealth(availability)}
        reason={availability != null && availability < 99 ? `${t("Uptime", "آپ‌تایم")} ${availability.toFixed(2)}%` : undefined}
      />

      <MonitorKpiCard
        title={t("Error Rate", "نرخ خطا")}
        value={errorRate == null ? "—" : errorRate.toFixed(2)}
        unit="%"
        secondary={{ value: "4xx + 5xx", label: t("Rate", "نرخ") }}
        spark={spark.error}
        health={availabilityHealth(availability)}
        reason={errorRate != null && errorRate > 0 ? `${errorRate.toFixed(2)}% ${t("errors", "خطا")}` : undefined}
      />

      <MonitorKpiCard
        title={t("Success Rate", "نرخ موفقیت")}
        value={successRate == null ? "—" : successRate.toFixed(2)}
        unit="%"
        secondary={{ value: "200", label: t("Successful", "موفق") }}
        spark={spark.availability}
        health={availabilityHealth(successRate)}
        reason={successRate != null && successRate < 99 ? `${successRate.toFixed(2)}%` : undefined}
      />

      <MonitorKpiCard
        title={t("Probe Health", "سلامت پراب‌ها")}
        value={`${healthy}/${totalProbes || healthy}`}
        secondary={{ value: `${healthy}`, label: t("healthy probes", "پراب سالم") }}
        spark={spark.availability}
        health={critical > 0 ? "critical" : warning > 0 ? "warning" : totalProbes > 0 ? "healthy" : "unknown"}
        reason={
          critical > 0
            ? `${critical} ${t("critical", "بحرانی")}`
            : warning > 0
              ? `${warning} ${t("warning", "هشدار")}`
              : undefined
        }
      />
    </div>
  );
}
