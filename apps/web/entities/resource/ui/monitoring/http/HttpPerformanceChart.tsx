"use client";

import { useMemo, useState } from "react";
import { useLocale } from "next-intl";

import { EChart, useChartPalette } from "@/shared/ui/charts/echart";
import { makeGrid, makeTimeXAxis, makeTooltip, hexToRgba } from "@/shared/ui/charts/chart-config";
import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { Skeleton } from "@/shared/ui/skeleton";
import { cn } from "@/shared/utils/cn";
import type { HttpChartSeries } from "./http-metrics";

// Font used for canvas-rendered chart text (axis numbers, legend) — the same
// stack as the Ping charts so the numbers stay clean in both languages.
const CHART_FONT = "'bakh', 'estedad', ui-sans-serif, system-ui, sans-serif";

type ChartMetric = "response_time" | "availability" | "error_rate";

const METRICS: Array<{ key: ChartMetric; en: string; fa: string; unit: "ms" | "%" }> = [
  { key: "response_time", en: "Response Time", fa: "زمان پاسخ", unit: "ms" },
  { key: "availability", en: "Availability", fa: "دسترس‌پذیری", unit: "%" },
  { key: "error_rate", en: "Error Rate", fa: "نرخ خطا", unit: "%" },
];

interface HttpPerformanceChartProps {
  responseSeries: HttpChartSeries[];
  statusSeries: HttpChartSeries[];
  isLoading: boolean;
  isError: boolean;
  isFa: boolean;
}

// Response time / availability / error-rate timeline with one line per probe.
// Every probe can be hidden/shown independently to isolate regional issues.
export function HttpPerformanceChart({
  responseSeries,
  statusSeries,
  isLoading,
  isError,
  isFa,
}: HttpPerformanceChartProps) {
  const locale = useLocale();
  const palette = useChartPalette();
  const [metric, setMetric] = useState<ChartMetric>("response_time");
  const [hidden, setHidden] = useState<Set<string>>(() => new Set());

  const t = (en: string, fa: string) => (isFa ? fa : en);
  const active = METRICS.find((m) => m.key === metric)!;

  const probes = useMemo(() => {
    const base = metric === "response_time" ? responseSeries : statusSeries;
    const seen = new Map<string, HttpChartSeries>();
    for (const s of base) {
      const mapped = { ...s, points: s.points.map((p) => ({ ...p, value: p.value })) };
      if (metric === "availability") mapped.points = mapped.points.map((p) => ({ ...p, value: p.value * 100 }));
      if (metric === "error_rate") mapped.points = mapped.points.map((p) => ({ ...p, value: (1 - p.value) * 100 }));
      seen.set(mapped.probeName || mapped.location, mapped);
    }
    return Array.from(seen.values());
  }, [metric, responseSeries, statusSeries]);

  const visible = probes.filter((s) => !hidden.has(s.probeName || s.location));

  const toggle = (name: string) => {
    setHidden((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  const option = useMemo(() => {
    const unit = active.unit;
    const formatter = (value: unknown) =>
      typeof value === "number" ? `${Math.round(value)} ${unit}` : String(value);
    // Show every other time tick (skip :05, :15, :25, ...) so the axis stays
    // sparse and readable, matching the Ping charts.
    const timeLabelFormatter = (value: unknown) => {
      if (typeof value !== "number") return "";
      const date = new Date(value);
      if (date.getMinutes() % 10 !== 0) return "";
      return new Intl.DateTimeFormat(locale, {
        hour: "2-digit",
        minute: "2-digit",
      }).format(date);
    };
    return {
      animation: false,
      grid: makeGrid({ top: 16, right: 16, bottom: 40, left: 48 }),
      tooltip: {
        ...makeTooltip(palette, formatter),
        textStyle: { color: palette.tooltipText, fontSize: 12, fontFamily: CHART_FONT },
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
        axisLabel: { color: palette.text, fontFamily: CHART_FONT, formatter: (v: number) => `${Math.round(v)}` },
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
        textStyle: { color: palette.text, fontFamily: CHART_FONT, fontSize: 11 },
      },
      series: visible.map((s, i) => {
        const color = palette.series[i % palette.series.length];
        const glowColor = hexToRgba(color, 0.55);
        return {
          type: "line" as const,
          name: s.probeName || s.location,
          showSymbol: false,
          sampling: "lttb" as const,
          lineStyle: {
            width: 2.5,
            color,
            shadowBlur: 16,
            shadowColor: glowColor,
            shadowOffsetY: 2,
          },
          itemStyle: { color },
          emphasis: {
            focus: "series" as const,
            lineStyle: { width: 3.5, color, shadowBlur: 24, shadowColor: glowColor },
          },
          areaStyle: {
            color: {
              type: "linear" as const,
              x: 0, y: 0, x2: 0, y2: 1,
              colorStops: [
                { offset: 0, color: hexToRgba(color, 0.42) },
                { offset: 0.5, color: hexToRgba(color, 0.16) },
                { offset: 1, color: hexToRgba(color, 0.02) },
              ],
            },
          },
          data: s.points.map((p) => [p.time, p.value]),
        };
      }),
    };
  }, [visible, locale, palette, active.unit]);

  return (
    <Card variant="bordered" className="h-full shadow-subtle">
      <CardHeader className="flex-row flex-wrap items-center justify-between gap-3 space-y-0 px-5 pt-4">
        <div className="min-w-0">
          <CardTitle className="text-sm font-semibold text-foreground">
            {isFa ? "روند زمانی" : "Timeline"}
          </CardTitle>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="inline-flex items-center gap-0.5 rounded-lg border border-border/60 bg-muted/25 p-0.5">
            {METRICS.map((m) => (
              <button
                key={m.key}
                type="button"
                onClick={() => setMetric(m.key)}
                className={cn(
                  "inline-flex h-7 items-center rounded-md px-2.5 text-xs font-medium transition-colors",
                  metric === m.key ? "bg-card text-foreground shadow-sm" : "text-muted-foreground",
                )}
              >
                {isFa ? m.fa : m.en}
              </button>
            ))}
          </div>
          {probes.length > 1 && (
            <div className="flex max-w-[50%] flex-wrap justify-end gap-1">
              {probes.map((s) => {
                const name = s.probeName || s.location;
                const isHidden = hidden.has(name);
                return (
                  <Button
                    key={name}
                    type="button"
                    size="sm"
                    className={cn("h-6 px-2 text-xs", isHidden && "opacity-40")}
                    variant={isHidden ? "outline" : "secondary"}
                    onClick={() => toggle(name)}
                  >
                    {name}
                  </Button>
                );
              })}
            </div>
          )}
        </div>
      </CardHeader>
      <CardContent className="px-1 pb-3 pt-1 sm:px-2">
        {isLoading ? (
          <Skeleton className="h-72 w-full rounded-lg" />
        ) : isError ? (
          <div className="flex h-72 items-center justify-center text-sm text-muted-foreground">
            {t("Unable to load data", "خطا در بارگذاری داده")}
          </div>
        ) : visible.length === 0 ? (
          <div className="flex h-72 items-center justify-center text-sm text-muted-foreground">
            {t("No data to display", "داده‌ای برای نمایش وجود ندارد")}
          </div>
        ) : (
          <EChart option={option} className="h-72 w-full" ariaLabel={active.en} />
        )}
      </CardContent>
    </Card>
  );
}
