"use client";

import { MonitorKpiCard } from "../../components/monitor-kpi-card";
import type { SparklineSeries } from "@/shared/ui/charts/sparkline";
import type { SnmpDeviceSummary } from "./snmp-metrics";
import type { SnmpThresholds } from "./snmp-config";
import { cpuHealth, utilHealth } from "./snmp-metrics";

function formatUptime(seconds: number | null): string {
  if (seconds == null) return "—";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  if (days > 0) return `${days}d`;
  if (hours > 0) return `${hours}h`;
  return `${Math.floor(seconds / 60)}m`;
}

interface SnmpKpiGridProps {
  summary: SnmpDeviceSummary;
  thresholds: SnmpThresholds;
  rangeLabel: string;
  cpuSpark: SparklineSeries[];
  memSpark: SparklineSeries[];
  statusSpark: SparklineSeries[];
  cpuAvg: number | null;
  memAvg: number | null;
  isFa: boolean;
}

export function SnmpKpiGrid({
  summary,
  thresholds,
  rangeLabel,
  cpuSpark,
  memSpark,
  statusSpark,
  cpuAvg,
  memAvg,
  isFa,
}: SnmpKpiGridProps) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  const cpuHealthState = cpuHealth(summary.cpuPercent, thresholds.cpuWarning, thresholds.cpuCritical);
  const memHealthState = cpuHealth(summary.memoryPercent, thresholds.memWarning, thresholds.memCritical);
  const utilState = utilHealth(summary.maxUtilization, thresholds.utilizationWarning, thresholds.utilizationCritical);
  const availability = summary.reachable ? 100 : 0;
  const temp = summary.temperatureCelsius;
  const tempHealth: "healthy" | "warning" | "critical" | "unknown" =
    temp == null
      ? "unknown"
      : temp >= thresholds.temperatureCritical
        ? "critical"
        : temp >= thresholds.temperatureWarning
          ? "warning"
          : "healthy";

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-6">
      <MonitorKpiCard
        title={t("CPU", "پردازنده")}
        value={summary.cpuPercent == null ? "—" : String(Math.round(summary.cpuPercent))}
        unit="%"
        secondary={{
          value: cpuAvg == null ? "—" : `${Math.round(cpuAvg)}%`,
          label: t("Avg", "میانگین"),
        }}
        spark={cpuSpark}
        health={cpuHealthState}
        reason={cpuHealthState !== "healthy" ? `${Math.round(summary.cpuPercent ?? 0)}%` : undefined}
      />

      <MonitorKpiCard
        title={t("Memory", "حافظه")}
        value={summary.memoryPercent == null ? "—" : String(Math.round(summary.memoryPercent))}
        unit="%"
        secondary={{
          value: memAvg == null ? "—" : `${Math.round(memAvg)}%`,
          label: t("Avg", "میانگین"),
        }}
        spark={memSpark}
        health={memHealthState}
        reason={memHealthState !== "healthy" ? `${Math.round(summary.memoryPercent ?? 0)}%` : undefined}
      />

      <MonitorKpiCard
        title={t("Availability", "دسترس‌پذیری")}
        value={String(availability)}
        unit="%"
        secondary={{ value: rangeLabel, label: t("Range", "بازه") }}
        spark={statusSpark}
        health={summary.reachable ? "healthy" : "critical"}
        reason={summary.state && summary.state !== "success" && summary.state !== "partial" ? summary.state : undefined}
      />

      <MonitorKpiCard
        title={t("Temperature", "دما")}
        value={temp == null ? "—" : String(Math.round(temp))}
        unit="°C"
        secondary={{
          value: temp == null ? "—" : `${Math.round(temp)}°C`,
          label: t("Max", "حداکثر"),
        }}
        health={tempHealth}
        reason={tempHealth !== "healthy" && temp != null ? `${Math.round(temp)}°C` : undefined}
      />

      <MonitorKpiCard
        title={t("Interfaces", "اینترفیس‌ها")}
        value={summary.interfacesDown > 0 ? `${summary.interfacesDown}/${summary.interfacesTotal}` : String(summary.interfacesTotal)}
        secondary={{
          value: summary.maxUtilization == null ? "—" : `${Math.round(summary.maxUtilization)}%`,
          label: t("Max util", "حداکثر مصرف"),
        }}
        spark={statusSpark}
        health={summary.interfacesDown > 0 ? "critical" : utilState}
        reason={summary.interfacesDown > 0 ? `${summary.interfacesDown} ${t("down", "قطعی")}` : undefined}
      />

      <MonitorKpiCard
        title={t("Uptime", "آپ‌تایم")}
        value={formatUptime(summary.uptimeSeconds)}
        secondary={{
          value: summary.uptimeSeconds == null ? "—" : `${Math.round(summary.uptimeSeconds / 86400)}d`,
          label: t("Uptime", "آپ‌تایم"),
        }}
        health="healthy"
      />
    </div>
  );
}
