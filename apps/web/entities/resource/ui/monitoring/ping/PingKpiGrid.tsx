"use client";

import { MonitorKpiCard, type KpiHealth } from "@/entities/resource/ui/components/monitor-kpi-card";
import type { PingHealthState } from "./ping-health";
import type { PingSummary } from "./ping-metrics";
import type { HttpChartSeries } from "../http/http-metrics";

function toKpi(state: PingHealthState): KpiHealth {
  switch (state) {
    case "warning":
      return "warning";
    case "critical":
    case "down":
      return "critical";
    default:
      return "unknown";
  }
}

function sparkOf(series: HttpChartSeries[]) {
  return series.map((s) => ({
    name: s.probeName || s.location,
    points: s.points,
  }));
}

interface PingKpiGridProps {
  isFa: boolean;
  summary: PingSummary;
  states: {
    availability: PingHealthState;
    latency: PingHealthState;
    packetLoss: PingHealthState;
    jitter: PingHealthState;
  };
  availabilitySeries: HttpChartSeries[];
  latencySeries: HttpChartSeries[];
  lossSeries: HttpChartSeries[];
  jitterSeries: HttpChartSeries[];
  rangeLabel: string;
}

export function PingKpiGrid({
  isFa,
  summary,
  states,
  availabilitySeries,
  latencySeries,
  lossSeries,
  jitterSeries,
  rangeLabel,
}: PingKpiGridProps) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-5">
      <MonitorKpiCard
        title={t("Latency", "تأخیر")}
        value={summary.latency == null ? "—" : String(Math.round(summary.latency))}
        unit="ms"
        secondary={{
          value: summary.latency == null ? "—" : `${Math.round(summary.latency)} ms`,
          label: t("Avg", "میانگین"),
        }}
        spark={sparkOf(latencySeries)}
        health={toKpi(states.latency)}
      />

      <MonitorKpiCard
        title={t("Packet Loss", "افت بسته")}
        value={summary.packetLoss == null ? "—" : summary.packetLoss.toFixed(1)}
        unit="%"
        secondary={{
          value: summary.packetLoss == null ? "—" : `${summary.packetLoss.toFixed(2)}%`,
          label: t("Avg", "میانگین"),
        }}
        spark={sparkOf(lossSeries)}
        health={toKpi(states.packetLoss)}
        reason={summary.packetLoss != null && summary.packetLoss > 0 ? `${summary.packetLoss.toFixed(1)}%` : undefined}
      />

      <MonitorKpiCard
        title={t("Jitter", "نوسان")}
        value={summary.jitter == null ? "—" : String(Math.round(summary.jitter))}
        unit="ms"
        secondary={{
          value: summary.jitter == null ? "—" : `${Math.round(summary.jitter)} ms`,
          label: t("Avg", "میانگین"),
        }}
        spark={sparkOf(jitterSeries)}
        health={toKpi(states.jitter)}
      />

      <MonitorKpiCard
        title={t("Availability", "دسترس‌پذیری")}
        value={summary.availability == null ? "—" : summary.availability.toFixed(2)}
        unit="%"
        secondary={{ value: rangeLabel, label: t("Range", "بازه") }}
        spark={sparkOf(availabilitySeries)}
        health={toKpi(states.availability)}
        reason={summary.availability != null && summary.availability < 100 ? `${summary.availability.toFixed(2)}%` : undefined}
      />

      <MonitorKpiCard
        title={t("Probe Health", "سلامت پراب‌ها")}
        value={`${summary.successChecks}/${summary.totalChecks || 1}`}
        secondary={{ value: `${summary.successChecks}`, label: t("healthy probes", "پراب سالم") }}
        spark={sparkOf(availabilitySeries)}
        health={summary.failedChecks > 0 ? "critical" : summary.totalChecks > 0 ? "healthy" : "unknown"}
        reason={summary.failedChecks > 0 ? `${summary.failedChecks} ${t("failed", "ناموفق")}` : undefined}
      />
    </div>
  );
}
