"use client";

import { useMemo } from "react";
import { useLocale } from "next-intl";

import { EChart, useChartPalette } from "@/shared/ui/charts/echart";
import {
  hexToRgba,
  makeGrid,
  makeTimeXAxis,
  makeTooltip,
} from "@/shared/ui/charts/chart-config";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { Skeleton } from "@/shared/ui/skeleton";
import { cn } from "@/shared/utils/cn";
import type { PingChartSeries } from "./ping-metrics";

// Font used for the canvas-rendered chart text.
const CHART_FONT = "'bakh', 'estedad', ui-sans-serif, system-ui, sans-serif";

export function PingAvailabilityChart({
  title,
  series,
  isLoading,
  isError,
}: {
  title: string;
  series: PingChartSeries[];
  isLoading: boolean;
  isError: boolean;
}) {
  const locale = useLocale();
  const isFa = locale === "fa";
  const palette = useChartPalette();

  const option = useMemo(() => {
    return {
      animation: false,
      grid: makeGrid({ top: 24, right: 16, bottom: 40, left: 48 }),
      tooltip: {
        ...makeTooltip(palette, (value: unknown) =>
          typeof value === "number" ? `${Math.round(value * 100)}%` : String(value),
        ),
        textStyle: { color: palette.tooltipText, fontSize: 12, fontFamily: CHART_FONT },
      },
      xAxis: { ...makeTimeXAxis(locale, palette, CHART_FONT) },
      yAxis: {
        type: "value" as const,
        min: 0,
        max: 1,
        axisLabel: {
          color: palette.text,
          fontFamily: CHART_FONT,
          formatter: (value: unknown) =>
            typeof value === "number" ? `${Math.round(value * 100)}%` : String(value),
        },
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { lineStyle: { color: palette.axis, opacity: 0.35 } },
      },
      series: series.map((s, i) => {
        const color = palette.series[i % palette.series.length];
        const area =
          series.length <= 8
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
        };
      }),
    };
  }, [series, locale, palette]);

  return (
    <Card
      variant="bordered"
      className="h-full shadow-subtle transition-[border-color,box-shadow] duration-300 dark:hover:border-primary/40 dark:hover:shadow-glow"
    >
      <CardHeader className="px-5 pt-4">
        <CardTitle className="text-sm font-semibold text-foreground">{title}</CardTitle>
      </CardHeader>
      <CardContent className={cn("px-1 pb-3 pt-1 sm:px-2")}>
        {isLoading ? (
          <Skeleton className="h-60 w-full rounded-lg" />
        ) : isError ? (
          <div className="flex h-60 items-center justify-center text-sm text-muted-foreground">
            {isFa ? "امکان بارگذاری داده وجود ندارد" : "Unable to load data"}
          </div>
        ) : series.length === 0 ? (
          <div className="flex h-60 items-center justify-center text-sm text-muted-foreground">
            {isFa ? "داده‌ای برای نمایش وجود ندارد" : "No data to display"}
          </div>
        ) : (
          <EChart option={option} className="h-60 w-full" ariaLabel={title} />
        )}
      </CardContent>
    </Card>
  );
}
