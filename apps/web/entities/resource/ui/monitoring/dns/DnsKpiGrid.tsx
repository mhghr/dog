"use client";

import { MonitorKpiCard, type KpiHealth } from "@/entities/resource/ui/components/monitor-kpi-card";

export interface DnsKpiSeries {
  probeName: string;
  location: string;
  points: Array<{ time: string; value: number }>;
}

interface DnsKpiGridProps {
  isFa: boolean;
  responseTimeMs: number | null;
  avgResponseMs: number | null;
  availability: number | null;
  answerCount: number | null;
  rcode: string | null;
  totalChecks: number;
  successChecks: number;
  responseSeries: DnsKpiSeries[];
  statusSeries: DnsKpiSeries[];
  rangeLabel: string;
}

function sparkOf(series: DnsKpiSeries[], transform?: (v: number) => number) {
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

export function DnsKpiGrid({
  isFa,
  responseTimeMs,
  avgResponseMs,
  availability,
  answerCount,
  rcode,
  totalChecks,
  successChecks,
  responseSeries,
  statusSeries,
  rangeLabel,
}: DnsKpiGridProps) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-3">
      <MonitorKpiCard
        title={t("Query Time", "زمان Query")}
        value={responseTimeMs == null ? "—" : String(Math.round(responseTimeMs))}
        unit="ms"
        secondary={{
          value: avgResponseMs == null ? "—" : `${Math.round(avgResponseMs)} ms`,
          label: t("Avg", "میانگین"),
        }}
        spark={sparkOf(responseSeries)}
        health={healthOf(availability)}
      />

      <MonitorKpiCard
        title={t("Resolution Success", "موفقیت تفکیک")}
        value={availability == null ? "—" : String(Math.round(availability))}
        unit="%"
        secondary={{ value: `${successChecks}/${totalChecks || 1}`, label: t("successful", "موفق") }}
        spark={sparkOf(statusSeries, (v) => v * 100)}
        health={healthOf(availability)}
        reason={availability != null && availability < 99 ? `${Math.round(availability)}%` : undefined}
      />

      <MonitorKpiCard
        title={t("DNS Response Code", "کد پاسخ DNS")}
        value={rcode ?? "—"}
        secondary={{ value: answerCount == null ? "—" : String(answerCount), label: t("Answers", "پاسخ‌ها") }}
        spark={sparkOf(responseSeries)}
        health={healthOf(availability)}
        reason={rcode != null && rcode !== "NOERROR" ? rcode : undefined}
      />
    </div>
  );
}
