"use client";

import { Activity, Radio, Waves } from "lucide-react";
import { useTranslations } from "next-intl";

import { PingMetricCard } from "@/plugins/monitoring/ping/ping-metric-card";
import type { MonitorSummaryProps } from "@/plugins/monitoring/core/definition";
import type { ProbeResult } from "@/entities/monitor/model/result";

function latestByMetric(results: ProbeResult[]): ProbeResult | null {
  return results.length > 0 ? results[0] : null;
}

export function PingMonitorSummary({ monitor, recentResults, locale }: MonitorSummaryProps) {
  const t = useTranslations("monitorDetail");
  const latest = latestByMetric(recentResults);

  return (
    <section className="grid min-w-0 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <PingMetricCard
        icon={Activity}
        label={t("rttRange")}
        metricKey="rtt_ms"
        latestResult={latest}
        locale={locale}
        format="ms"
        warningThreshold={monitor.config.warning_latency_millis as number}
        criticalThreshold={monitor.config.critical_latency_millis as number}
        recentResults={recentResults}
      />

      <PingMetricCard
        icon={Radio}
        label={t("packetLoss")}
        metricKey="packet_loss_percent"
        latestResult={latest}
        locale={locale}
        format="percent"
        warningThreshold={monitor.config.warning_packet_loss_percent as number}
        criticalThreshold={monitor.config.critical_packet_loss_percent as number}
        recentResults={recentResults}
      />

      <PingMetricCard
        icon={Waves}
        label={t("jitter")}
        metricKey="jitter_ms"
        latestResult={latest}
        locale={locale}
        format="ms"
        warningThreshold={monitor.config.warning_jitter_millis as number}
        criticalThreshold={monitor.config.critical_jitter_millis as number}
        recentResults={recentResults}
      />

      <PingMetricCard
        icon={Activity}
        label={t("currentLatency")}
        metricKey="rtt_ms"
        latestResult={latest}
        locale={locale}
        format="ms"
        recentResults={recentResults}
      />
    </section>
  );
}
