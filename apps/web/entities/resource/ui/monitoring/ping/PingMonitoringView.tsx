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
  buildDownIntervals,
  formatPingKpiValue,
  formatPingKpiValueWithUnit,
  type PingProbeStat,
  type PingChartSeries,
} from "./ping-metrics";
import { PingMonitorHeader } from "./PingMonitorHeader";
import { PingKpiCard, type PingKpiRow } from "./PingKpiCard";
import { PingMetricChart } from "./PingMetricChart";

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
  const statusSeries: PingChartSeries[] = useMemo(
    () => toChartSeries(statusQuery.data?.series ?? [], "status"),
    [statusQuery.data?.series],
  );
  const downIntervals = useMemo(() => buildDownIntervals(statusSeries), [statusSeries]);
  const lastSuccessAt = statusQuery.data?.last_success_at ?? null;

  const latest = useMemo(() => latencyQuery.data?.latest ?? [], [latencyQuery.data?.latest]);
  const summary = useMemo(() => summarize(latest), [latest]);
  const probeStats = useMemo(() => toProbeStats(latest), [latest]);

  const latencySeries = useMemo(
    () => toChartSeries(latencyQuery.data?.series ?? [], "rtt_ms"),
    [latencyQuery.data?.series],
  );

  const hasData = latest.length > 0;
  const isLoading = latencyQuery.isPending;
  const isError = latencyQuery.isError;

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
        <div className="flex flex-col items-center justify-center gap-2 rounded-2xl border border-border/50 bg-white/50 py-16 text-sm text-muted-foreground dark:bg-white/[0.02]">
          <span>{t("Unable to load data", "خطا در دریافت داده")}</span>
        </div>
      ) : isLoading ? (
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <div
                key={i}
                className="h-36 rounded-2xl border border-border/70 bg-white/70 shadow-[var(--shadow-panel)] dark:bg-white/[0.03]"
              >
                <Skeleton className="h-full w-full rounded-2xl opacity-40" />
              </div>
            ))}
          </div>
          <Skeleton className="h-72 rounded-2xl" />
        </div>
      ) : !hasData ? (
        <div className="flex flex-col items-center justify-center gap-2 rounded-2xl border border-border/50 bg-white/50 py-16 dark:bg-white/[0.02]">
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
          {/* Parameter cards */}
          <KpiGrid
            isFa={isFa}
            down={overall === "down"}
            summary={summary}
            states={kpiStates}
            probeStats={probeStats}
            lastSuccessAt={lastSuccessAt}
          />

          {/* Latency chart */}
          <PingMetricChart
            title={t("Latency over time", "تأخیر در طول زمان")}
            unit="ms"
            series={latencySeries}
            thresholds={config.thresholds.latency}
            downIntervals={downIntervals}
            isLoading={latencyQuery.isPending}
            isError={latencyQuery.isError}
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
  probeStats,
  lastSuccessAt,
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
  probeStats: PingProbeStat[];
  lastSuccessAt: string | null;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  const availabilityRows: PingKpiRow[] = probeStats.map((s) => ({
    label: s.location,
    value: s.success ? t("Up", "بالا") : t("Down", "پایین"),
    tone: s.success ? "success" : "destructive",
  }));

  const latencyRows: PingKpiRow[] = probeStats.map((s) => ({
    label: s.location,
    value: formatPingKpiValueWithUnit(s.latency, "ms", !s.success),
  }));

  const lossRows: PingKpiRow[] = probeStats.map((s) => ({
    label: s.location,
    value: formatPingKpiValueWithUnit(s.packetLoss, "percent", !s.success),
    tone: s.packetLoss != null && s.packetLoss > 0 ? "warning" : "muted",
  }));

  const jitterRows: PingKpiRow[] = probeStats.map((s) => ({
    label: s.location,
    value: formatPingKpiValueWithUnit(s.jitter, "ms", !s.success),
  }));

  return (
    <section className="space-y-3">
      {down && (
        <div
          role="alert"
          className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-2xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm"
        >
          <span className="font-semibold text-destructive">
            {t("Target is down", "منبع قطع است")}
          </span>
          {lastSuccessAt && (
            <span className="text-muted-foreground">
              {t("Last successful check: ", "آخرین بررسی موفق: ")}
              {lastSuccessAt}
            </span>
          )}
        </div>
      )}

      <div className="grid max-w-4xl grid-cols-2 gap-3 sm:grid-cols-4">
        <PingKpiCard
          label={t("Availability", "دسترس‌پذیری")}
          value={summary.availability == null
            ? down ? "0.00" : "N/A"
            : summary.availability.toFixed(2)}
          unit="%"
          state={states.availability}
          rows={availabilityRows}
        />
        <PingKpiCard
          label={t("Latency", "تأخیر")}
          value={formatPingKpiValue(summary.latency, "ms", down)}
          unit={summary.latency != null || down ? "ms" : undefined}
          state={states.latency}
          rows={latencyRows}
        />
        <PingKpiCard
          label={t("Packet loss", "افت بسته")}
          value={formatPingKpiValue(summary.packetLoss, "percent", down)}
          unit={summary.packetLoss != null || down ? "%" : undefined}
          state={states.packetLoss}
          rows={lossRows}
        />
        <PingKpiCard
          label={t("Jitter", "نوسان")}
          value={formatPingKpiValue(summary.jitter, "ms", down)}
          unit={summary.jitter != null || down ? "ms" : undefined}
          state={states.jitter}
          rows={jitterRows}
        />
      </div>
    </section>
  );
}
