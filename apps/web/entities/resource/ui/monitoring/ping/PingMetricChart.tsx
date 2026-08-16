"use client";

import { useMemo, useState } from "react";
import { useLocale } from "next-intl";

import { EChart, useChartPalette } from "@/shared/ui/charts/echart";
import { makeGrid, makeTimeXAxis, makeTooltip } from "@/shared/ui/charts/chart-config";
import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardHeader } from "@/shared/ui/card";
import { Skeleton } from "@/shared/ui/skeleton";
import type { PingChartSeries, DownInterval } from "./ping-metrics";
import type { MetricThreshold } from "./ping-config";

const PALETTE = ["#4F66F0", "#0D9464", "#DC3035", "#F59E0B", "#8B5CF6", "#06B6D4", "#EC4899", "#84CC16"];

// Font used for the canvas-rendered chart text (axis numbers, legend).
const CHART_FONT = "'bakh', 'estedad', ui-sans-serif, system-ui, sans-serif";

export interface PingMetricChartProps {
  title: string;
  unit: "ms" | "%";
  series: PingChartSeries[];
  thresholds: MetricThreshold;
  downIntervals?: DownInterval[];
  isLoading: boolean;
  isError: boolean;
  onRetry?: () => void;
}

export function PingMetricChart({
  title,
  unit,
  series,
  thresholds,
  downIntervals = [],
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

  const formatter = (value: unknown) =>
    typeof value === "number"
      ? unit === "ms"
        ? `${Math.round(value)} ms`
        : `${value.toFixed(2)}%`
      : String(value);

  // Axis ticks show plain numbers — the unit lives in the axis name.
  const axisFormatter = (value: unknown) =>
    typeof value === "number"
      ? Number.isInteger(value)
        ? String(value)
        : value.toFixed(1)
      : String(value);

  const unitLabel = unit === "ms" ? (isFa ? "میلی‌ثانیه" : "milliseconds") : (isFa ? "درصد" : "percent");
  const timeLabel = isFa ? "زمان" : "time";

  const option = useMemo(() => {
    const markLine = {
      silent: true,
      symbol: "none",
      label: { show: true, position: "insideEndTop", color: palette.text, fontSize: 11 },
      data: [] as Array<{ yAxis: number; name?: string; lineStyle?: { color: string; type: "dashed" } }>,
    };

    if (thresholds.warning != null) {
      markLine.data.push({
        yAxis: thresholds.warning,
        name: `warn ${thresholds.warning}`,
        lineStyle: { color: "#F59E0B", type: "dashed" },
      });
    }
    if (thresholds.critical != null) {
      markLine.data.push({
        yAxis: thresholds.critical,
        name: `crit ${thresholds.critical}`,
        lineStyle: { color: "#DC3035", type: "dashed" },
      });
    }

    const markArea = {
      silent: true,
      data: toDownMarkArea(downIntervals),
    };

    return {
      animation: false,
      grid: makeGrid({ top: 16, right: 16, bottom: 56, left: 48 }),
      tooltip: { ...makeTooltip(palette, formatter), textStyle: { color: palette.tooltipText, fontSize: 12, fontFamily: CHART_FONT } },
      xAxis: { ...makeTimeXAxis(locale, palette, CHART_FONT), name: timeLabel, nameTextStyle: { color: palette.axis, fontFamily: CHART_FONT, fontSize: 11, padding: [0, 0, 0, 8] } },
      yAxis: {
        type: "value" as const,
        name: unitLabel,
        nameTextStyle: { color: palette.axis, fontFamily: CHART_FONT, fontSize: 11, padding: [0, 0, 6, 0] },
        axisLabel: { color: palette.text, fontFamily: CHART_FONT, formatter: axisFormatter },
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { show: false },
      },
      legend: {
        type: "scroll" as const,
        bottom: 0,
        itemWidth: 16,
        itemHeight: 8,
        icon: "roundRect",
        textStyle: { color: palette.text, fontSize: 11, fontFamily: CHART_FONT },
      },
      series: visible.map((s, i) => ({
        type: "line" as const,
        name: s.probeName || s.location || `probe-${i + 1}`,
        showSymbol: false,
        smooth: 0.2,
        lineStyle: { width: 2, color: PALETTE[i % PALETTE.length] },
        itemStyle: { color: PALETTE[i % PALETTE.length] },
        areaStyle: { color: "transparent" },
        data: s.points.map((p) => [p.time, p.value]),
        markArea: markArea.data.length > 0 ? markArea : undefined,
        markLine: i === 0 && markLine.data.length > 0 ? markLine : undefined,
      })),
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible, locale, thresholds.warning, thresholds.critical, downIntervals]);

  return (
    <Card variant="bordered" className="h-full">
      {locations.length > 1 && (
        <CardHeader className="flex-row items-center justify-end gap-3 space-y-0">
          <div className="flex flex-wrap items-center gap-1">
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
        </CardHeader>
      )}
      <CardContent className="pt-1">
        {isLoading ? (
          <Skeleton className="h-64 w-full rounded-lg" />
        ) : isError ? (
          <div className="flex h-64 flex-col items-center justify-center gap-2 text-sm text-muted-foreground">
            <span>{isFa ? "خطا در دریافت داده" : "Unable to load data"}</span>
            {onRetry && (
              <Button type="button" variant="outline" size="sm" onClick={onRetry}>
                {isFa ? "تلاش مجدد" : "Retry"}
              </Button>
            )}
          </div>
        ) : visible.length === 0 ? (
          <div className="flex h-64 items-center justify-center text-sm text-muted-foreground">
            {isFa ? "داده‌ای برای نمایش نیست" : "No data to display"}
          </div>
        ) : (
          <EChart
            option={option}
            className="h-64 w-full"
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
): Array<{ name: string; xAxis: [string, string]; itemStyle: { color: string } }> {
  return downIntervals.map((interval) => ({
    name: "Down",
    xAxis: [interval.start, interval.end],
    itemStyle: { color: "rgba(220,48,53,0.08)" },
  }));
}
