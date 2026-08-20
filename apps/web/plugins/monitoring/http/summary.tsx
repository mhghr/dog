"use client";

import { Gauge } from "lucide-react";
import { useTranslations } from "next-intl";

import { HttpMetricCard } from "@/plugins/monitoring/http/http-metric-card";
import type { MonitorSummaryProps } from "@/plugins/monitoring/core/definition";
import type { ProbeResult } from "@/entities/monitor/model/result";

function latestByMetric(results: ProbeResult[]): ProbeResult | null {
  return results.length > 0 ? results[0] : null;
}

export function HttpMonitorSummary({ monitor, recentResults, locale }: MonitorSummaryProps) {
  const t = useTranslations("monitorDetail");
  const latest = latestByMetric(recentResults);

  const statusCode =
    typeof latest?.attributes?.status_code === "number" ? latest.attributes.status_code : null;

  return (
    <section className="grid min-w-0 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <HttpMetricCard
        icon={Gauge}
        label={t("statusCode")}
        value={statusCode != null ? String(statusCode) : "—"}
        latestResult={latest}
        locale={locale}
        recentResults={recentResults}
      />

      <HttpMetricCard
        icon={Gauge}
        label={t("currentLatency")}
        metricKey="response_time_ms"
        latestResult={latest}
        locale={locale}
        unit="ms"
        warningThreshold={monitor.config.warning_response_time_ms as number}
        criticalThreshold={monitor.config.critical_response_time_ms as number}
        recentResults={recentResults}
      />

      <HttpMetricCard
        icon={Gauge}
        label={t("timeToFirstByte")}
        metricKey="ttfb_ms"
        latestResult={latest}
        locale={locale}
        unit="ms"
        warningThreshold={monitor.config.warning_ttfb_ms as number}
        criticalThreshold={monitor.config.critical_ttfb_ms as number}
        recentResults={recentResults}
      />

      <HttpMetricCard
        icon={Gauge}
        label={t("responseSize")}
        metricKey="response_size_bytes"
        latestResult={latest}
        locale={locale}
        unit="bytes"
        recentResults={recentResults}
      />
    </section>
  );
}
