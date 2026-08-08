"use client";

import { useTranslations } from "next-intl";

import { MonitorSummaryCard } from "@/entities/monitor/ui/summary-card";
import { formatDuration, formatInterval, formatPercent } from "@/shared/utils/formatters";
import type { MonitorSummaryProps } from "@/plugins/monitoring/core/definition";

export function GenericMonitorSummary({ monitor, metrics, locale, rangeLabel }: MonitorSummaryProps) {
  const t = useTranslations("monitorDetail");
  const tMonitors = useTranslations("monitors");
  const summary = metrics?.summary;

  return (
    <div className="grid grid-cols-2 gap-2.5 lg:grid-cols-4">
      <MonitorSummaryCard label={`${t("uptime")} (${rangeLabel})`} value={formatPercent(summary?.uptime_percent, locale)} />
      <MonitorSummaryCard label={t("currentLatency")} value={formatDuration(monitor.last_result?.duration_millis, locale)} />
      <MonitorSummaryCard label={t("p95Latency")} value={formatDuration(summary?.p95_latency_ms, locale)} />
      <MonitorSummaryCard label={tMonitors("interval")} value={formatInterval(monitor.interval_seconds, locale)} />
    </div>
  );
}
