"use client";

import { useMemo, useState } from "react";
import { useLocale } from "next-intl";

import { EChart, useChartPalette } from "@/shared/ui/charts/echart";
import {
  hexToRgba,
  makeGrid,
  makeTimeXAxis,
  makeTooltip,
} from "@/shared/ui/charts/chart-config";
import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { Skeleton } from "@/shared/ui/skeleton";
import { cn } from "@/shared/utils/cn";
import { buildDownIntervals, type PingChartSeries, type DownInterval } from "./ping-metrics";
import type { MetricThreshold } from "./ping-config";

// Font used for the canvas-rendered chart text (axis numbers, legend).
const CHART_FONT = "'bakh', 'estedad', ui-sans-serif, system-ui, sans-serif";

export interface PingMetricChartProps {
  title: string;
  unit: "ms" | "%";
  series: PingChartSeries[];
  thresholds: MetricThreshold;
  statusSeries?: PingChartSeries[];
  isLoading: boolean;
  isError: boolean;
  onRetry?: () => void;
}

export function PingMetricChart({
  title,
  unit,
  series,
  thresholds,
  statusSeries = [],
  isLoading,
  isError,
  onRetry,
}: PingMetricChartProps) {
  const locale = useLocale();
  const isFa = locale === "fa";
  const palette = useChartPalette();
  const [selected, setSelected] = useState<string>("all");

  const locations = useMemo(() => {
    const set = new Set<string>();
    for (const s of series) if (s.location) set.add(s.location);
    return Array.from(set);
  }, [series]);

  const visible = useMemo(
    () => (selected === "all" ? series : series.filter((s) => s.location === selected)),
    [series, selected],
  );

  const visibleStatus = useMemo(
    () =>
      selected === "all"
        ? statusSeries
        : statusSeries.filter((s) => s.location === selected),
    [statusSeries, selected],
  );

  const downIntervals = useMemo(() => buildDownIntervals(visibleStatus), [visibleStatus]);

  const formatter = (value: unknown) =>
    typeof value === "number"
      ? unit === "ms"
        ? `${Math.round(value)} ms`
        : `${Math.round(value)}%`
      : String(value);

  // Axis ticks show plain numbers — the unit lives in the tooltip.
  const axisFormatter = (value: unknown) =>
    typeof value === "number"
      ? Number.isInteger(value)
        ? String(value)
        : value.toFixed(1)
      : String(value);

  // X-axis time labels only at multiples of ten minutes (e.g. 14:00, 14:10,
  // 14:20) so the axis stays sparse instead of a wall of overlapping times.
  const timeLabelFormatter = (value: unknown) => {
    if (typeof value !== "number") return "";
    const date = new Date(value);
    if (date.getMinutes() % 10 !== 0) return "";
    return new Intl.DateTimeFormat(locale, {
      hour: "2-digit",
      minute: "2-digit",
    }).format(date);
  };

  const option = useMemo(() => {
    const markLine = {
      silent: true,
      symbol: "none",
      label: { show: true, position: "insideEndTop" as const, color: palette.text, fontSize: 11 },
      data: [] as Array<{ yAxis: number; name?: string; lineStyle?: { color: string; type: "dashed" } }>,
    };

    if (thresholds.warning != null) {
      markLine.data.push({
        yAxis: thresholds.warning,
        name: `warn ${thresholds.warning}`,
        lineStyle: { color: palette.warning, type: "dashed" },
      });
    }
    if (thresholds.critical != null) {
      markLine.data.push({
        yAxis: thresholds.critical,
        name: `crit ${thresholds.critical}`,
        lineStyle: { color: palette.danger, type: "dashed" },
      });
    }

    const markArea = {
      silent: true,
      data: toDownMarkArea(downIntervals, palette.danger),
    };

    return {
      animation: false,
      grid: makeGrid({ top: 16, right: 16, bottom: 56, left: 48 }),
      tooltip: { ...makeTooltip(palette, formatter), textStyle: { color: palette.tooltipText, fontSize: 12, fontFamily: CHART_FONT } },
      legend: {
        type: "scroll" as const,
        bottom: 0,
        left: 0,
        right: 0,
        icon: "roundRect",
        itemWidth: 14,
        itemHeight: 7,
        itemGap: 14,
        textStyle: { color: palette.text, fontFamily: CHART_FONT, fontSize: 11 },
      },
      xAxis: {
        ...makeTimeXAxis(locale, palette, CHART_FONT),
        axisLabel: {
          color: palette.text,
          fontFamily: CHART_FONT,
          hideOverlap: true,
          interval: 0,
          formatter: timeLabelFormatter,
        },
      },
      yAxis: {
        type: "value" as const,
        name: unit === "ms" ? "ms" : "%",
        nameTextStyle: { color: palette.text, fontFamily: CHART_FONT, align: "left", verticalAlign: "bottom" },
        axisLabel: { color: palette.text, fontFamily: CHART_FONT, formatter: axisFormatter },
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { lineStyle: { color: palette.axis, opacity: 0.35 } },
      },
      series: visible.map((s, i) => {
        const color = palette.series[i % palette.series.length];
        const area =
          visible.length <= 8
            ? {
                color: {
                  type: "linear" as const,
                  x: 0,
                  y: 0,
                  x2: 0,
                  y2: 1,
                  colorStops: [
                    { offset: 0, color: hexToRgba(color, 0.25) },
                    { offset: 0.35, color: hexToRgba(color, 0.08) },
                    { offset: 1, color: hexToRgba(color, 0) },
                  ],
                },
              }
            : undefined;
        return {
          type: "line" as const,
          name: s.probeName || s.location || `probe-${i + 1}`,
          showSymbol: false,
          sampling: "lttb" as const,
          progressive: 500,
          progressiveThreshold: 2000,
          lineStyle: {
            width: 1.5,
            color,
          },
          itemStyle: { color },
          // Explicit emphasis keeps the line visible and unchanged on hover —
          // without it ECharts re-renders the emphasized series with default
          // styles and the line can vanish.
          emphasis: {
            focus: "none",
            lineStyle: {
              width: 1.5,
              color,
            },
            areaStyle: area,
          },
          areaStyle: area,
          data: s.points.map((p) => [p.time, p.value]),
          markArea: markArea.data.length > 0 ? markArea : undefined,
          markLine: i === 0 && markLine.data.length > 0 ? markLine : undefined,
        };
      }),
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible, locale, thresholds.warning, thresholds.critical, downIntervals]);

  return (
    <Card
      variant="bordered"
      className="h-full shadow-subtle transition-[border-color,box-shadow] duration-300 dark:hover:border-primary/40 dark:hover:shadow-glow"
    >
      <CardHeader className="flex-row items-center justify-between gap-3 space-y-0 px-5 pt-4">
        <div className="min-w-0">
          <CardTitle className="text-sm font-semibold text-foreground">{title}</CardTitle>
        </div>
        {locations.length > 1 && (
          <div className="flex shrink-0 flex-wrap items-center gap-1">
            <Button
              type="button"
              variant={selected === "all" ? "secondary" : "ghost"}
              size="sm"
              className="h-6 px-2 text-xs"
              onClick={() => setSelected("all")}
            >
              {isFa ? "همه" : "All"}
            </Button>
            {locations.map((loc) => (
              <Button
                key={loc}
                type="button"
                variant={selected === loc ? "secondary" : "ghost"}
                size="sm"
                className="h-6 px-2 text-xs"
                onClick={() => setSelected(loc)}
              >
                {loc}
              </Button>
            ))}
          </div>
        )}
      </CardHeader>
      <CardContent className={cn("px-1 pb-3 pt-1 sm:px-2")}>
        {isLoading ? (
          <Skeleton className="h-60 w-full rounded-lg" />
        ) : isError ? (
          <div className="flex h-60 flex-col items-center justify-center gap-2 text-sm text-muted-foreground">
            <span>{isFa ? "امکان بارگذاری داده وجود ندارد" : "Unable to load data"}</span>
            {onRetry && (
              <Button type="button" variant="outline" size="sm" onClick={onRetry}>
                {isFa ? "تلاش مجدد" : "Retry"}
              </Button>
            )}
          </div>
        ) : visible.length === 0 ? (
          <div className="flex h-60 items-center justify-center text-sm text-muted-foreground">
            {isFa ? "داده‌ای برای نمایش وجود ندارد" : "No data to display"}
          </div>
        ) : (
          <EChart
            option={option}
            className="h-60 w-full"
            ariaLabel={title}
          />
        )}
      </CardContent>
    </Card>
  );
}

// Converts down intervals into the echarts markArea data shape so the latency
// chart can shade downtime from the explicit status signal (not from gaps).
export function toDownMarkArea(
  downIntervals: DownInterval[],
  danger: string,
): Array<{ name: string; xAxis: [string, string]; itemStyle: { color: string } }> {
  return downIntervals.map((interval) => ({
    name: "Down",
    xAxis: [interval.start, interval.end],
    itemStyle: { color: hexToRgba(danger, 0.1) },
  }));
}
