"use client";

import { useMemo } from "react";
import { useLocale } from "next-intl";

import { EChart, useChartPalette } from "@/shared/ui/charts/echart";
import { makeGrid, makeTimeXAxis, makeTooltip } from "@/shared/ui/charts/chart-config";
import { Card, CardContent } from "@/shared/ui/card";
import { Skeleton } from "@/shared/ui/skeleton";
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
      grid: makeGrid({ top: 16, right: 16, bottom: 40, left: 40 }),
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
        name: isFa ? "دسترس‌پذیری" : "availability",
        axisLabel: {
          color: palette.text,
          fontFamily: CHART_FONT,
          formatter: (value: unknown) =>
            typeof value === "number" ? `${Math.round(value * 100)}%` : String(value),
        },
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { show: false },
      },
      series: series.map((s, i) => ({
        type: "line" as const,
        name: s.probeName || s.location || `probe-${i + 1}`,
        showSymbol: false,
        smooth: 0.2,
        step: "end" as const,
        lineStyle: { width: 2, color: i === 0 ? "#0D9464" : "#4F66F0" },
        itemStyle: { color: i === 0 ? "#0D9464" : "#4F66F0" },
        areaStyle: { color: "rgba(13,148,100,0.12)" },
        data: s.points.map((p) => [p.time, p.value]),
      })),
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [series, locale]);

  return (
    <Card variant="bordered" className="h-full">
      <CardContent className="pt-1">
        {isLoading ? (
          <Skeleton className="h-40 w-full rounded-lg" />
        ) : isError ? (
          <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
            {isFa ? "خطا در دریافت داده" : "Unable to load data"}
          </div>
        ) : series.length === 0 ? (
          <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
            {isFa ? "داده‌ای برای نمایش نیست" : "No data to display"}
          </div>
        ) : (
          <EChart option={option} className="h-40 w-full" ariaLabel={title} />
        )}
      </CardContent>
    </Card>
  );
}
