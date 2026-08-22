"use client";

import { useMemo } from "react";
import { Globe } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { EChart, useChartPalette } from "@/shared/ui/charts/echart";
import {
  makeGrid,
  makeTimeXAxis,
  makeTooltip,
  hexToRgba,
} from "@/shared/ui/charts/chart-config";
import { WorldMonitoringMap } from "@/widgets/monitoring-map/world-monitoring-map";
import { cn } from "@/shared/utils/cn";
import type { DashboardSummary } from "@/entities/dashboard/model/types";

const FAILURE_THRESHOLD = 99.5;

function TrendChart({ summary, locale }: { summary: DashboardSummary; locale: string }) {
  const t = useTranslations("dashboard");
  const palette = useChartPalette();

  const option = useMemo(() => {
    const series = summary.availability_series ?? [];
    const downRanges: [string, string][] = [];
    let start: string | null = null;
    for (const point of series) {
      if (point.total > 0 && point.rate < FAILURE_THRESHOLD) {
        if (!start) start = point.timestamp;
      } else if (start) {
        downRanges.push([start, point.timestamp]);
        start = null;
      }
    }
    if (start && series.length > 0) {
      downRanges.push([start, series[series.length - 1]!.timestamp]);
    }

    return {
      grid: makeGrid({ left: 44, right: 12, top: 20, bottom: 28 }),
      xAxis: makeTimeXAxis(locale, palette),
      yAxis: {
        type: "value" as const,
        min: 0,
        max: 100,
        interval: 25,
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          color: palette.text,
          formatter: (value: number) => `${value}%`,
        },
        splitLine: { lineStyle: { color: palette.axis, opacity: 0.35 } },
      },
      tooltip: makeTooltip(palette, (value) =>
        typeof value === "number" ? `${value.toFixed(2)}%` : `${value}`,
      ),
      series: [
        {
          name: t("availability24h"),
          type: "line" as const,
          data: series.map((p) => [p.timestamp, p.rate]),
          smooth: true,
          showSymbol: false,
          symbol: "circle",
          symbolSize: 6,
          lineStyle: { width: 2, color: palette.primary },
          itemStyle: { color: palette.primary },
          areaStyle: {
            color: {
              type: "linear" as const,
              x: 0,
              y: 0,
              x2: 0,
              y2: 1,
              colorStops: [
                { offset: 0, color: hexToRgba(palette.primary, 0.3) },
                { offset: 1, color: hexToRgba(palette.primary, 0.02) },
              ],
            },
          },
          markArea: {
            silent: true,
            itemStyle: { color: hexToRgba(palette.danger, 0.1) },
            data: downRanges.map(([a, b]) => [{ xAxis: a }, { xAxis: b }]),
          },
        },
      ],
    };
  }, [summary, palette, locale, t]);

  return (
    <EChart
      option={option}
      ariaLabel={t("trend24h")}
      className="h-72 w-full"
    />
  );
}

function StatusDonut({ summary }: { summary: DashboardSummary }) {
  const t = useTranslations("dashboard");
  const palette = useChartPalette();

  const counts = summary.status_counts ?? {};
  const entries = [
    { key: "up", label: t("statusUp"), value: counts.up ?? 0, color: palette.success },
    { key: "down", label: t("statusDown"), value: counts.down ?? 0, color: palette.danger },
    { key: "unknown", label: t("statusUnknown"), value: counts.unknown ?? 0, color: palette.text },
    { key: "paused", label: t("statusPaused"), value: counts.paused ?? 0, color: palette.warning },
  ].filter((e) => e.value > 0);

  const option = useMemo(
    () => ({
      tooltip: {
        trigger: "item" as const,
        backgroundColor: palette.tooltipBg,
        borderColor: palette.axis,
        textStyle: { color: palette.tooltipText, fontSize: 12 },
        formatter: (params: { name: string; value: number; percent: number }) =>
          `${params.name}: ${params.value} (${params.percent}%)`,
      },
      series: [
        {
          type: "pie" as const,
          radius: ["62%", "82%"],
          center: ["50%", "50%"],
          avoidLabelOverlap: false,
          itemStyle: { borderRadius: 6, borderColor: "var(--card)", borderWidth: 2 },
          label: { show: false },
          emphasis: { scaleSize: 6 },
          data: entries.map((e) => ({ name: e.label, value: e.value, itemStyle: { color: e.color } })),
        },
      ],
    }),
    [entries, palette],
  );

  return (
    <div className="relative">
      <EChart option={option} ariaLabel={t("statusDistribution")} className="h-44 w-full" />
      <div className="pointer-events-none absolute inset-0 grid place-items-center">
        <div className="flex flex-col items-center leading-none">
          <span className="text-2xl font-semibold tabular-nums" dir="ltr">
            {summary.total_monitors}
          </span>
          <span className="mt-1 text-[11px] text-muted-foreground">{t("totalMonitors")}</span>
        </div>
      </div>
      <div className="mt-1 grid grid-cols-2 gap-x-4 gap-y-1.5">
        {entries.map((e) => (
          <div key={e.key} className="flex items-center gap-2 text-xs">
            <span className="size-2 shrink-0 rounded-full" style={{ background: e.color }} />
            <span className="truncate text-muted-foreground">{e.label}</span>
            <span className="ml-auto font-medium tabular-nums" dir="ltr">
              {e.value}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function SkeletonPanel({ className }: { className?: string }) {
  return (
    <Card variant="bordered" className={cn("animate-pulse shadow-subtle", className)}>
      <CardHeader>
        <div className="h-3 w-28 rounded-full bg-muted" />
      </CardHeader>
      <CardContent>
        <div className="h-56 w-full rounded-lg bg-muted/50" />
      </CardContent>
    </Card>
  );
}

export function DashboardCharts({ summary }: { summary?: DashboardSummary }) {
  const t = useTranslations("dashboard");
  const locale = useLocale();

  if (!summary) {
    return (
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-12">
        <SkeletonPanel className="lg:col-span-7" />
        <SkeletonPanel className="lg:col-span-5" />
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-3 lg:grid-cols-12">
      <Card
        variant="bordered"
        className="animate-in fade-in slide-in-from-bottom-2 shadow-subtle transition-all duration-[250ms] ease-[cubic-bezier(0.23,1,0.32,1)] hover:shadow-md lg:col-span-7"
        style={{ animationDelay: "450ms", animationFillMode: "backwards" }}
      >
        <CardHeader>
          <CardTitle>{t("trend24h")}</CardTitle>
        </CardHeader>
        <CardContent>
          <TrendChart summary={summary} locale={locale} />
        </CardContent>
      </Card>

      <Card
        variant="bordered"
        className="animate-in fade-in slide-in-from-bottom-2 shadow-subtle transition-all duration-[250ms] ease-[cubic-bezier(0.23,1,0.32,1)] hover:shadow-md lg:col-span-5"
        style={{ animationDelay: "540ms", animationFillMode: "backwards" }}
      >
        <CardHeader>
          <CardTitle>{t("statusDistribution")}</CardTitle>
        </CardHeader>
        <CardContent>
          <StatusDonut summary={summary} />
        </CardContent>
      </Card>

      <Card
        variant="bordered"
        className="animate-in fade-in slide-in-from-bottom-2 shadow-subtle transition-all duration-[250ms] ease-[cubic-bezier(0.23,1,0.32,1)] hover:shadow-md lg:col-span-12"
        style={{ animationDelay: "630ms", animationFillMode: "backwards" }}
      >
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="size-4 text-primary" aria-hidden />
            {t("mapTitle")}
          </CardTitle>
        </CardHeader>
        <CardContent className="px-2 pb-2 pt-0 sm:px-4">
          <WorldMonitoringMap className="aspect-[2/1] max-h-[26rem] w-full" />
        </CardContent>
      </Card>
    </div>
  );
}
