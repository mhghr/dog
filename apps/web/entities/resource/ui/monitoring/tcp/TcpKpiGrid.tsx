"use client";

import { MonitorKpiCard, type KpiHealth } from "@/entities/resource/ui/components/monitor-kpi-card";

export interface TcpKpiSeries {
  probeName: string;
  location: string;
  points: Array<{ time: string; value: number }>;
}

interface TcpKpiGridProps {
  isFa: boolean;
  connectTimeMs: number | null;
  avgConnectMs: number | null;
  availability: number | null;
  totalChecks: number;
  successChecks: number;
  connectSeries: TcpKpiSeries[];
  statusSeries: TcpKpiSeries[];
  rangeLabel: string;
}

function sparkOf(series: TcpKpiSeries[], transform?: (v: number) => number) {
  return series
    .map((s) => ({
      name: s.probeName || s.location,
      points: transform ? s.points.map((p) => ({ ...p, value: transform(p.value) })) : s.points,
    }))
    .filter((s) => s.points.length > 0);
}

function healthOf(availability: number | null): KpiHealth {
  if (availability == null) return "unknown";
  if (availability >= 99) return "healthy";
  if (availability >= 95) return "warning";
  return "critical";
}

export function TcpKpiGrid({
  isFa,
  connectTimeMs,
  avgConnectMs,
  availability,
  totalChecks,
  successChecks,
  connectSeries,
  statusSeries,
  rangeLabel,
}: TcpKpiGridProps) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-2">
      <MonitorKpiCard
        title={t("Availability", "دسترس‌پذیری")}
        value={availability == null ? "—" : String(Math.round(availability))}
        unit="%"
        secondary={{ value: `${successChecks}/${totalChecks || 1}`, label: t("healthy", "موفق") }}
        spark={sparkOf(statusSeries, (v) => v * 100)}
        health={healthOf(availability)}
        reason={availability != null && availability < 99 ? `${Math.round(availability)}%` : undefined}
      />

      <MonitorKpiCard
        title={t("Connect Time", "زمان اتصال")}
        value={connectTimeMs == null ? "—" : String(Math.round(connectTimeMs))}
        unit="ms"
        secondary={{
          value: avgConnectMs == null ? "—" : `${Math.round(avgConnectMs)} ms`,
          label: t("Avg", "میانگین"),
        }}
        spark={sparkOf(connectSeries)}
        health={healthOf(availability)}
      />
    </div>
  );
}
