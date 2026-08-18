"use client";

import { useMemo, useState } from "react";
import { useLocale } from "next-intl";

import { Skeleton } from "@/shared/ui/skeleton";
import { Warning } from "@/shared/ui/icons";
import { useChartPalette } from "@/shared/ui/charts/echart";
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
  formatPingKpiValue,
  type PingChartSeries,
} from "./ping-metrics";
import { PingMonitorHeader } from "./PingMonitorHeader";
import { PingKpiCard } from "./PingKpiCard";
import { PingMetricChart } from "./PingMetricChart";
import { PingAvailabilityChart } from "./PingAvailabilityChart";
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
  const palette = useChartPalette();

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
          <Skeleton className="h-48 rounded-xl" />
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
            colors={palette.series}
            availabilitySeries={statusSeries}
            latencySeries={latencySeries}
            lossSeries={packetLossSeries}
            jitterSeries={jitterSeries}
            lastSuccessAt={lastSuccessAt}
            failureReason={failureReason}
          />

          {/* Charts — equal width side by side */}
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
            <PingAvailabilityChart
              title={t("Availability over time", "دسترس‌پذیری در طول زمان")}
              series={statusSeries}
              isLoading={statusQuery.isPending}
              isError={statusQuery.isError}
            />
          </div>

          {/* Per-probe parameters table */}
          <PingProbeLocations
            stats={probeStats}
            thresholds={config.thresholds}
            isLoading={latencyQuery.isPending}
          />
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
  colors,
  availabilitySeries,
  latencySeries,
  lossSeries,
  jitterSeries,
  lastSuccessAt,
  failureReason,
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
  colors: string[];
  availabilitySeries: PingChartSeries[];
  latencySeries: PingChartSeries[];
  lossSeries: PingChartSeries[];
  jitterSeries: PingChartSeries[];
  lastSuccessAt: string | null;
  failureReason: string | null;
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

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <PingKpiCard
          label={t("Availability", "دسترس‌پذیری")}
          value={summary.availability == null
            ? down ? "0.00" : "N/A"
            : summary.availability.toFixed(2)}
          unit="%"
          state={states.availability}
          spark={availabilitySeries}
          colors={colors}
          sparkLabel={t("Availability trend", "روند دسترس‌پذیری")}
        />
        <PingKpiCard
          label={t("Latency", "تأخیر")}
          value={formatPingKpiValue(summary.latency, "ms", down)}
          unit={summary.latency != null || down ? "ms" : undefined}
          state={states.latency}
          spark={latencySeries}
          colors={colors}
          sparkLabel={t("Latency trend", "روند تأخیر")}
        />
        <PingKpiCard
          label={t("Packet loss", "افت بسته")}
          value={formatPingKpiValue(summary.packetLoss, "percent", down)}
          unit={summary.packetLoss != null || down ? "%" : undefined}
          state={states.packetLoss}
          spark={lossSeries}
          colors={colors}
          sparkLabel={t("Packet loss trend", "روند افت بسته")}
        />
        <PingKpiCard
          label={t("Jitter", "نوسان")}
          value={formatPingKpiValue(summary.jitter, "ms", down)}
          unit={summary.jitter != null || down ? "ms" : undefined}
          state={states.jitter}
          spark={jitterSeries}
          colors={colors}
          sparkLabel={t("Jitter trend", "روند نوسان")}
        />
      </div>
    </section>
  );
}
