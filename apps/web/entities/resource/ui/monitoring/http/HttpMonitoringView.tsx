"use client";

import { useMemo, useState } from "react";
import { useLocale } from "next-intl";

import { Skeleton } from "@/shared/ui/skeleton";
import {
  useResourceMonitorMetrics,
  useResourceMonitorStatus,
  type MetricsRange,
} from "@/entities/resource/hooks/use-resource";
import type { Monitor } from "@/entities/resource/hooks/types";
import { readHttpConfig } from "./http-config";
import {
  toHttpChartSeries,
  toHttpProbeHealth,
  summarizeHttp,
  globalHealthOf,
  statusLabelOf,
  lastErrorOf,
  type HttpChartSeries,
  type HttpProbeHealth,
} from "./http-metrics";
import { HttpKpiGrid } from "./HttpKpiGrid";
import { HttpPerformanceChart } from "./HttpPerformanceChart";
import { HttpResponsesCard } from "./HttpResponsesCard";
import { HttpProbeLocations } from "./HttpProbeLocations";
import { HttpAvailabilityHistory } from "./HttpAvailabilityHistory";
import { HttpRecentFailures } from "./HttpRecentFailures";
import { HttpAlertRules } from "./HttpAlertRules";
import { PingTimeRangeSelector } from "../ping/PingTimeRangeSelector";

function byProbe(a: HttpChartSeries, b: HttpChartSeries): number {
  const ka = a.probeName || a.location || "";
  const kb = b.probeName || b.location || "";
  return ka.localeCompare(kb);
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

  const config = useMemo(() => readHttpConfig(monitor.configuration), [monitor.configuration]);

  const metricsQuery = useResourceMonitorMetrics(resourceId, monitor.id, range);
  const statusQuery = useResourceMonitorStatus(resourceId, monitor.id, range);

  const latest = useMemo(() => metricsQuery.data?.latest ?? [], [metricsQuery.data?.latest]);

  const responseSeries = useMemo(
    () => toHttpChartSeries(metricsQuery.data?.series ?? [], "response_time_ms").sort(byProbe),
    [metricsQuery.data?.series],
  );
  const statusSeries = useMemo(
    () => toHttpChartSeries(statusQuery.data?.series ?? [], "status").sort(byProbe),
    [statusQuery.data?.series],
  );

  const summary = useMemo(() => summarizeHttp(latest, responseSeries), [latest, responseSeries]);
  const probeHealth: HttpProbeHealth[] = useMemo(
    () => toHttpProbeHealth(latest, statusSeries, config.thresholds.responseTime),
    [latest, statusSeries, config.thresholds],
  );

  const globalHealth = globalHealthOf(probeHealth);
  const healthyCount = probeHealth.filter((p) => p.health === "healthy").length;
  const warningCount = probeHealth.filter((p) => p.health === "warning").length;
  const criticalCount = probeHealth.filter((p) => p.health === "critical" || p.health === "down").length;

  const currentStatus = probeHealth.find((p) => p.statusCode != null)?.statusCode ?? null;
  const statusText = statusLabelOf(currentStatus, lastErrorOf(probeHealth));

  // Range average from the pooled series points (all probes).
  const rangeAvgMs = useMemo(() => {
    const pooled = responseSeries.flatMap((s) =>
      s.points.map((p) => p.value).filter((v): v is number => v != null && v >= 0),
    );
    return pooled.length ? pooled.reduce((a, b) => a + b, 0) / pooled.length : null;
  }, [responseSeries]);

  const targetUrl = config.url || monitor.resource_target || "";

  const hasData = latest.length > 0;
  const isLoading = metricsQuery.isPending || statusQuery.isPending;
  const isError = metricsQuery.isError || statusQuery.isError;

  const t = (en: string, fa: string) => (isFa ? fa : en);

  return (
    <section className="flex flex-col gap-6">
      {/* Header row: time range on the left, monitor name + target URL on the right. */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <PingTimeRangeSelector range={range} onChange={setRange} />
        <div className="ms-auto flex min-w-0 items-center gap-2">
          <h2 className="truncate text-base font-semibold tracking-tight text-foreground">
            {monitor.name}
          </h2>
          <code className="max-w-[60%] truncate rounded-md bg-muted/50 px-2 py-1 font-mono text-xs text-muted-foreground" dir="ltr">
            {targetUrl}
          </code>
        </div>
      </div>

      {isError && !isLoading ? (
        <div className="flex flex-col items-center justify-center gap-2 rounded-xl border border-border/60 bg-card px-6 py-16 text-sm text-muted-foreground shadow-subtle">
          <span>{t("Unable to load data", "خطا در دریافت داده")}</span>
        </div>
      ) : isLoading ? (
        <div className="space-y-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-36 rounded-xl" />
            ))}
          </div>
          <Skeleton className="h-72 rounded-xl" />
          <Skeleton className="h-80 rounded-xl" />
        </div>
      ) : !hasData ? (
        <div className="flex flex-col items-center justify-center gap-2 rounded-xl border border-border/60 bg-card px-6 py-16 shadow-subtle">
          <p className="text-sm font-medium text-foreground/80">
            {t("No monitoring data yet", "هنوز داده مانیتورینگ وجود ندارد")}
          </p>
          <p className="text-xs text-muted-foreground">
            {t(
              "The monitor is active but has not produced results.",
              "مانیتور فعال است اما هنوز نتیجه‌ای تولید نکرده است.",
            )}
          </p>
        </div>
      ) : (
        <>
          <HttpKpiGrid
            isFa={isFa}
            currentStatusCode={currentStatus}
            statusText={statusText}
            currentLatencyMs={summary.responseTimeMs}
            avgLatencyMs={rangeAvgMs}
            p95LatencyMs={summary.p95LatencyMs}
            availability={summary.availability}
            errorRate={summary.availability != null ? 100 - summary.availability : null}
            successRate={summary.availability}
            healthy={healthyCount}
            warning={warningCount}
            critical={criticalCount}
            totalProbes={probeHealth.length}
            responseSeries={responseSeries}
            statusSeries={statusSeries}
            rangeLabel={range}
          />

          <HttpRecentFailures probeHealth={probeHealth} isFa={isFa} />

          {/* Status-code distribution + availability history side by side. */}
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            <HttpResponsesCard
              buckets={metricsQuery.data?.status_codes ?? []}
              rangeLabel={range}
              isFa={isFa}
            />
            <HttpAvailabilityHistory probeHealth={probeHealth} isFa={isFa} />
          </div>

          {/* Timeline chart + probe table side by side (not full-width). */}
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            <HttpPerformanceChart
              responseSeries={responseSeries}
              statusSeries={statusSeries}
              isLoading={false}
              isError={false}
              isFa={isFa}
            />
            <HttpProbeLocations probeHealth={probeHealth} isFa={isFa} isLoading={false} />
          </div>

          <HttpAlertRules isFa={isFa} />
        </>
      )}
    </section>
  );
}
