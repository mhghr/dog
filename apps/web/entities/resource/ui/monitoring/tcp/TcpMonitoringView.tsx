"use client";

import { useMemo, useState } from "react";
import { useLocale } from "next-intl";

import { Skeleton } from "@/shared/ui/skeleton";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { StatusBadge } from "@/design-system/components";
import { EChart, useChartPalette } from "@/shared/ui/charts/echart";
import { makeGrid, makeTimeXAxis, makeTooltip, hexToRgba } from "@/shared/ui/charts/chart-config";
import { cn } from "@/shared/utils/cn";
import { formatRelativeTime } from "@/shared/utils/formatters";
import {
  useResourceMonitorMetrics,
  useResourceMonitorStatus,
  type MetricsRange,
} from "@/entities/resource/hooks/use-resource";
import type { Monitor } from "@/entities/resource/hooks/types";
import { readTcpConfig } from "./tcp-config";
import {
  evaluateMetric,
  evaluateTcpHealth,
  tcpHealthTone,
  type TcpHealthState,
} from "./tcp-health";
import {
  summarizeTcp,
  toTcpProbeStats,
  toTcpChartSeries,
  type TcpChartSeries,
} from "./tcp-metrics";
import { TcpKpiGrid } from "./TcpKpiGrid";
import { PingTimeRangeSelector } from "../ping/PingTimeRangeSelector";

function byProbe(a: TcpChartSeries, b: TcpChartSeries): number {
  const ka = a.probeName || a.location || "";
  const kb = b.probeName || b.location || "";
  return ka.localeCompare(kb);
}

export function TcpMonitoringView({
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

  const config = useMemo(() => readTcpConfig(monitor.configuration), [monitor.configuration]);

  const metricsQuery = useResourceMonitorMetrics(resourceId, monitor.id, range);
  const statusQuery = useResourceMonitorStatus(resourceId, monitor.id, range);

  const latest = useMemo(() => metricsQuery.data?.latest ?? [], [metricsQuery.data?.latest]);

  const summary = useMemo(() => summarizeTcp(latest), [latest]);
  const probeStats = useMemo(() => toTcpProbeStats(latest), [latest]);

  const connectSeries = useMemo(
    () => toTcpChartSeries(metricsQuery.data?.series ?? [], "connect_time_ms").sort(byProbe),
    [metricsQuery.data?.series],
  );
  const statusSeries = useMemo(
    () => toTcpChartSeries(statusQuery.data?.series ?? [], "status").sort(byProbe),
    [statusQuery.data?.series],
  );
  const avgConnectMs = useMemo(() => {
    const pooled = connectSeries.flatMap((s) => s.points.map((p) => p.value).filter((v): v is number => v != null && v >= 0));
    return pooled.length ? pooled.reduce((a, b) => a + b, 0) / pooled.length : null;
  }, [connectSeries]);

  const hasData = latest.length > 0;
  const isLoading = metricsQuery.isPending || statusQuery.isPending;
  const isError = metricsQuery.isError || statusQuery.isError;

  const overall = evaluateTcpHealth({
    lastStatus: monitor.last_status,
    success: latest.some((r) => r.success),
    connectTimeMs: summary.connectTimeMs,
    thresholds: config.thresholds,
  });

  const kpiStates = useMemo(
    () => ({
      availability: (hasData
        ? summary.availability === 100
          ? "healthy"
          : summary.availability != null
            ? "warning"
            : "unknown"
        : "unknown") as TcpHealthState,
      connectTime: evaluateMetric(summary.connectTimeMs, config.thresholds.connectTime),
    }),
    [summary, config.thresholds, hasData],
  );

  const option = useMemo(() => {
    const formatter = (value: unknown) =>
      typeof value === "number" ? `${Math.round(value)} ms` : String(value);
    return {
      animation: false,
      grid: makeGrid({ top: 16, right: 16, bottom: 40, left: 48 }),
      tooltip: { ...makeTooltip(palette, formatter) },
      xAxis: makeTimeXAxis(locale, palette),
      yAxis: {
        type: "value" as const,
        axisLabel: {
          color: palette.text,
          formatter: (value: number) => `${Math.round(value)}`,
        },
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { lineStyle: { color: palette.axis, opacity: 0.35 } },
      },
      legend: {
        type: "scroll" as const,
        bottom: 0,
        left: 0,
        right: 0,
        icon: "roundRect",
        itemWidth: 14,
        itemHeight: 7,
        itemGap: 14,
        textStyle: { color: palette.text, fontSize: 11 },
      },
      series: connectSeries.map((s, i) => {
        const color = palette.series[i % palette.series.length];
        return {
          type: "line" as const,
          name: s.probeName || s.location,
          showSymbol: false,
          sampling: "lttb" as const,
          lineStyle: { width: 1.5, color },
          itemStyle: { color },
          areaStyle: {
            color: {
              type: "linear" as const,
              x: 0, y: 0, x2: 0, y2: 1,
              colorStops: [
                { offset: 0, color: hexToRgba(color, 0.22) },
                { offset: 1, color: hexToRgba(color, 0) },
              ],
            },
          },
          data: s.points.map((p) => [p.time, p.value]),
        };
      }),
    };
  }, [connectSeries, locale, palette]);

  const t = (en: string, fa: string) => (isFa ? fa : en);

  const STATUS_LABEL: Record<TcpHealthState, { en: string; fa: string }> = {
    healthy: { en: "Healthy", fa: "سالم" },
    warning: { en: "Warning", fa: "هشدار" },
    critical: { en: "Critical", fa: "بحرانی" },
    down: { en: "Down", fa: "قطع" },
    unknown: { en: "Unknown", fa: "نامشخص" },
  };

  return (
    <section className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center gap-3">
        <div className="ms-auto flex shrink-0 items-center gap-3">
          <StatusBadge tone={tcpHealthTone(overall)} label={isFa ? STATUS_LABEL[overall].fa : STATUS_LABEL[overall].en} />
          {monitor.last_checked_at && (
            <span className="hidden text-xs tabular-nums text-muted-foreground sm:inline">
              {t("Last check: ", "آخرین بررسی: ")}
              {formatRelativeTime(monitor.last_checked_at, locale)}
            </span>
          )}
          <PingTimeRangeSelector range={range} onChange={setRange} />
        </div>
      </div>

      {isError && !isLoading ? (
        <div className="flex flex-col items-center justify-center gap-2 rounded-xl border border-border/60 bg-card px-6 py-16 text-sm text-muted-foreground shadow-subtle">
          <span>{t("Unable to load data", "خطا در دریافت داده")}</span>
        </div>
      ) : isLoading ? (
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="h-28 rounded-xl border border-border/40 bg-card shadow-subtle">
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
          <TcpKpiGrid
            isFa={isFa}
            connectTimeMs={summary.connectTimeMs}
            avgConnectMs={avgConnectMs}
            availability={summary.availability}
            totalChecks={summary.totalChecks}
            successChecks={summary.successChecks}
            connectSeries={connectSeries}
            statusSeries={statusSeries}
            rangeLabel={range}
          />

          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            <Card variant="bordered" className="h-full shadow-subtle">
              <CardHeader className="px-5 pt-4">
                <CardTitle className="text-sm font-semibold text-foreground">
                  {t("Connect time over time", "زمان اتصال در طول زمان")}
                </CardTitle>
              </CardHeader>
              <CardContent className="px-1 pb-3 pt-1 sm:px-2">
                {connectSeries.length === 0 ? (
                  <div className="flex h-60 items-center justify-center text-sm text-muted-foreground">
                    {t("No data to display", "داده‌ای برای نمایش وجود ندارد")}
                  </div>
                ) : (
                  <EChart option={option} className="h-60 w-full" ariaLabel="Connect time" />
                )}
              </CardContent>
            </Card>

            <TcpProbeLocations stats={probeStats} isLoading={isLoading} isFa={isFa} />
          </div>
        </>
      )}
    </section>
  );
}

function TcpKpiCard({
  label,
  value,
  unit,
  state,
  detail,
}: {
  label: string;
  value: string;
  unit?: string;
  state: TcpHealthState;
  detail?: string;
}) {
  const toneText: Record<TcpHealthState, string> = {
    healthy: "text-success dark:neon-text-current",
    warning: "text-warning dark:neon-text-current",
    critical: "text-destructive dark:neon-text-current",
    down: "text-destructive dark:neon-text-current",
    unknown: "text-muted-foreground",
  };

  return (
    <Card
      variant="bordered"
      className="relative h-full overflow-hidden bg-card shadow-subtle transition-[border-color,box-shadow] duration-300 hover:border-border/70 hover:shadow-card-hover"
    >
      <CardContent className="flex h-full flex-col gap-2 p-3.5">
        <div className="flex items-center justify-between gap-2">
          <p className="truncate text-[10px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
            {label}
          </p>
          <span className={cn("size-1.5 shrink-0 rounded-full bg-current", toneText[state])} aria-hidden />
        </div>
        <p className={cn("text-2xl font-bold leading-none tracking-tight tabular-nums", toneText[state])} dir="ltr">
          {value}
          {unit && <span className="ms-1 text-[11px] font-medium text-muted-foreground">{unit}</span>}
        </p>
        {detail && <p className="mt-auto truncate text-[11px] text-muted-foreground">{detail}</p>}
      </CardContent>
    </Card>
  );
}

function TcpProbeLocations({
  stats,
  isLoading,
  isFa,
}: {
  stats: ReturnType<typeof toTcpProbeStats>;
  isLoading: boolean;
  isFa: boolean;
}) {
  return (
    <Card variant="bordered" className="h-full shadow-subtle">
      <CardHeader className="px-5 pt-4">
        <CardTitle className="text-sm font-semibold text-foreground">
          {isFa ? "موقعیت پراب‌ها" : "Probe locations"}
        </CardTitle>
      </CardHeader>
      <CardContent className="overflow-x-auto px-4 pb-4">
        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-11 w-full rounded-lg" />
            ))}
          </div>
        ) : stats.length === 0 ? (
          <p className="py-4 text-sm text-muted-foreground">
            {isFa ? "هیچ پرابی داده اخیر ندارد" : "No probe has recent data"}
          </p>
        ) : (
          <table className="w-full min-w-[480px] border-collapse text-left">
            <thead>
              <tr className="border-b border-border/60 text-[11px] font-semibold uppercase tracking-[0.05em] text-muted-foreground">
                <th className="px-1 pb-2.5 font-semibold">{isFa ? "موقعیت" : "Location"}</th>
                <th className="px-1 pb-2.5 font-semibold">{isFa ? "وضعیت" : "Status"}</th>
                <th className="px-1 pb-2.5 text-right font-semibold">{isFa ? "خطا" : "Error"}</th>
                <th className="px-1 pb-2.5 text-right font-semibold">{isFa ? "زمان اتصال" : "Connect"}</th>
                <th className="px-1 pb-2.5 text-right font-semibold">{isFa ? "آخرین بررسی" : "Last check"}</th>
              </tr>
            </thead>
            <tbody>
              {stats.map((stat) => (
                <tr key={stat.probeId} className="border-b border-border/40 transition-colors last:border-0 hover:bg-muted/30">
                  <td className="px-1 py-2.5 text-sm font-medium" dir="auto">
                    {stat.location}
                  </td>
                  <td className="px-1 py-2.5">
                    <StatusBadge
                      tone={stat.success ? "success" : "destructive"}
                      label={stat.success ? (isFa ? "موفق" : "OK") : (isFa ? "ناموفق" : "FAIL")}
                    />
                  </td>
                  <td className="shrink-0 px-1 py-2.5 text-right tabular-nums text-[13px] text-muted-foreground" dir="ltr">
                    {stat.errorCode ?? "—"}
                  </td>
                  <td className="shrink-0 px-1 py-2.5 text-right tabular-nums text-[13px] text-muted-foreground" dir="ltr">
                    {stat.connectTimeMs == null ? "—" : `${Math.round(stat.connectTimeMs)} ms`}
                  </td>
                  <td className="px-1 py-2.5 text-right text-xs text-muted-foreground">
                    {formatRelativeTime(stat.lastCheckedAt, "en")}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </CardContent>
    </Card>
  );
}
