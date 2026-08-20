"use client";

import { useMemo, useState } from "react";
import { useLocale } from "next-intl";

import { Skeleton } from "@/shared/ui/skeleton";
import { Warning } from "@/shared/ui/icons";
import {
  useResourceMonitorMetrics,
  useResourceMonitorStatus,
  type MetricsRange,
} from "@/entities/resource/hooks/use-resource";
import type { Monitor } from "@/entities/resource/hooks/types";
import { readPingConfig } from "./ping-config";
import {
  evaluatePingHealth,
  evaluateMetric,
  evaluateAvailability,
  type PingHealthState,
} from "./ping-health";
import {
  summarize,
  toProbeStats,
  toChartSeries,
  type PingChartSeries,
} from "./ping-metrics";
import { PingMonitorHeader } from "./PingMonitorHeader";
import { PingKpiGrid } from "./PingKpiGrid";
import { PingMetricChart } from "./PingMetricChart";
import { PingProbeLocations } from "./PingProbeLocations";

function isDataStale(
  lastCheckedAt: string | null | undefined,
  intervalSeconds: number,
): boolean {
  if (!lastCheckedAt) return true;
  const last = new Date(lastCheckedAt).getTime();
  if (Number.isNaN(last)) return true;
  const maxAge = Math.max(intervalSeconds, 60) * 3 * 1000;
  return Date.now() - last > maxAge;
}

// Orders per-probe series by a stable key so the index-based chart palette
// maps the same color to the same probe across every card and chart.
function byProbe(a: PingChartSeries, b: PingChartSeries): number {
  const ka = a.probeName || a.location || "";
  const kb = b.probeName || b.location || "";
  return ka.localeCompare(kb);
}

export function PingMonitoringView({
  resourceId,
  monitor,
}: {
  resourceId: string;
  monitor: Monitor;
}) {
  const locale = useLocale();
  const isFa = locale === "fa";
  const [range, setRange] = useState<MetricsRange>("1h");

  const config = useMemo(() => readPingConfig(monitor.configuration), [monitor.configuration]);

  const latencyQuery = useResourceMonitorMetrics(resourceId, monitor.id, range);
  const statusQuery = useResourceMonitorStatus(resourceId, monitor.id, range);
  const packetLossQuery = useResourceMonitorMetrics(
    resourceId,
    monitor.id,
    range,
    "packet_loss_percent",
  );
  const jitterQuery = useResourceMonitorMetrics(resourceId, monitor.id, range, "jitter_ms");

  const statusSeries: PingChartSeries[] = useMemo(
    () => toChartSeries(statusQuery.data?.series ?? [], "status").sort(byProbe),
    [statusQuery.data?.series],
  );
  const lastSuccessAt = statusQuery.data?.last_success_at ?? null;

  const latest = useMemo(() => latencyQuery.data?.latest ?? [], [latencyQuery.data?.latest]);

  const failureReason = useMemo(() => {
    const failed = latest.find((r) => !r.success && (r.error_message || r.error_code));
    if (failed?.error_message) return failed.error_message;
    if (failed?.error_code) return failed.error_code;
    return null;
  }, [latest]);
  const summary = useMemo(() => summarize(latest), [latest]);
  const probeStats = useMemo(() => toProbeStats(latest), [latest]);

  const latencySeries = useMemo(
    () => toChartSeries(latencyQuery.data?.series ?? [], "rtt_ms").sort(byProbe),
    [latencyQuery.data?.series],
  );
  const packetLossSeries = useMemo(
    () => toChartSeries(packetLossQuery.data?.series ?? [], "packet_loss_percent").sort(byProbe),
    [packetLossQuery.data?.series],
  );
  const jitterSeries = useMemo(
    () => toChartSeries(jitterQuery.data?.series ?? [], "jitter_ms").sort(byProbe),
    [jitterQuery.data?.series],
  );

  const hasData = latest.length > 0;
  const isLoading =
    latencyQuery.isPending ||
    statusQuery.isPending ||
    packetLossQuery.isPending ||
    jitterQuery.isPending;
  const isError =
    latencyQuery.isError ||
    statusQuery.isError ||
    packetLossQuery.isError ||
    jitterQuery.isError;

  const overall = evaluatePingHealth({
    lastStatus: monitor.last_status,
    latency: summary.latency,
    packetLoss: summary.packetLoss,
    jitter: summary.jitter,
    thresholds: config.thresholds,
  });

  const kpiStates = useMemo(
    () => ({
      availability: evaluateAvailability(summary.availability),
      latency: evaluateMetric(summary.latency, config.thresholds.latency),
      packetLoss: evaluateMetric(summary.packetLoss, config.thresholds.packetLoss),
      jitter: evaluateMetric(summary.jitter, config.thresholds.jitter),
    }),
    [summary, config.thresholds],
  );

  const t = (en: string, fa: string) => (isFa ? fa : en);

  return (
    <section className="flex flex-col gap-6">
      {/* Monitor header (boxless) */}
      <PingMonitorHeader
        lastCheckedAt={monitor.last_checked_at}
        overallState={overall}
        isLive={Boolean(monitor.enabled) && !isDataStale(monitor.last_checked_at, monitor.interval_seconds)}
        range={range}
        onRangeChange={setRange}
      />

      {isError && !isLoading ? (
        <div className="flex flex-col items-center justify-center gap-2 rounded-xl border border-border/60 bg-card px-6 py-16 text-sm text-muted-foreground shadow-subtle">
          <span>{t("Unable to load data", "خطا در دریافت داده")}</span>
        </div>
      ) : isLoading ? (
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="h-36 rounded-xl border border-border/40 bg-card shadow-subtle">
                <Skeleton className="h-full w-full rounded-xl opacity-40" />
              </div>
            ))}
          </div>
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            <Skeleton className="h-60 rounded-xl" />
            <Skeleton className="h-60 rounded-xl" />
          </div>
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
          {/* Parameter cards with per-probe sparklines */}
          <KpiGrid
            isFa={isFa}
            down={overall === "down"}
            summary={summary}
            states={kpiStates}
            availabilitySeries={statusSeries}
            latencySeries={latencySeries}
            lossSeries={packetLossSeries}
            jitterSeries={jitterSeries}
            lastSuccessAt={lastSuccessAt}
            failureReason={failureReason}
            rangeLabel={range}
          />

          {/* Latency chart + per-probe table — equal width side by side */}
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            <PingMetricChart
              title={t("Latency over time", "تأخیر در طول زمان")}
              unit="ms"
              series={latencySeries}
              thresholds={config.thresholds.latency}
              statusSeries={statusSeries}
              isLoading={latencyQuery.isPending}
              isError={latencyQuery.isError}
            />
            <PingProbeLocations
              stats={probeStats}
              thresholds={config.thresholds}
              isLoading={latencyQuery.isPending}
            />
          </div>
        </>
      )}
    </section>
  );
}

function KpiGrid({
  isFa,
  down,
  summary,
  states,
  availabilitySeries,
  latencySeries,
  lossSeries,
  jitterSeries,
  lastSuccessAt,
  failureReason,
  rangeLabel,
}: {
  isFa: boolean;
  down: boolean;
  summary: ReturnType<typeof summarize>;
  states: {
    availability: PingHealthState;
    latency: PingHealthState;
    packetLoss: PingHealthState;
    jitter: PingHealthState;
  };
  availabilitySeries: PingChartSeries[];
  latencySeries: PingChartSeries[];
  lossSeries: PingChartSeries[];
  jitterSeries: PingChartSeries[];
  lastSuccessAt: string | null;
  failureReason: string | null;
  rangeLabel: string;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  return (
    <section className="space-y-3">
      {down && (
        <div
          role="alert"
          className="flex flex-wrap items-center gap-x-3 gap-y-1.5 rounded-xl border border-destructive/25 bg-destructive/[0.06] px-4 py-3 text-sm dark:shadow-[0_0_20px_-4px_var(--destructive)]"
        >
          <span className="flex items-center gap-2 font-semibold text-destructive">
            <Warning className="size-4" aria-hidden />
            {t("Target is down", "منبع قطع است")}
          </span>
          {failureReason && (
            <span className="text-muted-foreground">
              {t("Reason: ", "دلیل: ")}
              {failureReason}
            </span>
          )}
          {lastSuccessAt && (
            <span className="text-muted-foreground">
              {t("Last successful check: ", "آخرین بررسی موفق: ")}
              {lastSuccessAt}
            </span>
          )}
        </div>
      )}

      <PingKpiGrid
        isFa={isFa}
        summary={summary}
        states={states}
        availabilitySeries={availabilitySeries}
        latencySeries={latencySeries}
        lossSeries={lossSeries}
        jitterSeries={jitterSeries}
        rangeLabel={rangeLabel}
      />
    </section>
  );
}
