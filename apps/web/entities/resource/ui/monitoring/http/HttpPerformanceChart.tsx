"use client";

import { useMemo } from "react";
import { useLocale } from "next-intl";

import { EChart, useChartPalette } from "@/shared/ui/charts/echart";
import { makeGrid, makeTimeXAxis, makeTooltip, hexToRgba } from "@/shared/ui/charts/chart-config";
import { Card, CardContent, CardHeader } from "@/shared/ui/card";
import { Skeleton } from "@/shared/ui/skeleton";
import { cn } from "@/shared/utils/cn";
import { buildDownIntervals, type HttpChartSeries } from "./http-metrics";
import type { MetricThreshold } from "./http-config";

const CHART_FONT = "'bakh', 'estedad', ui-sans-serif, system-ui, sans-serif";

export type HttpChartMetric = "response_time" | "ttfb" | "dns" | "connect" | "tls" | "error_rate";

export const HTTP_CHART_METRICS: Array<{ key: HttpChartMetric; en: string; fa: string; unit: "ms" | "%" }> = [
  { key: "response_time", en: "Response Time", fa: "زمان پاسخ", unit: "ms" },
  { key: "ttfb", en: "TTFB", fa: "TTFB", unit: "ms" },
  { key: "dns", en: "DNS", fa: "DNS", unit: "ms" },
  { key: "connect", en: "Connect", fa: "اتصال", unit: "ms" },
  { key: "tls", en: "TLS", fa: "TLS", unit: "ms" },
  { key: "error_rate", en: "Error Rate", fa: "نرخ خطا", unit: "%" },
];

interface HttpPerformanceChartProps {
  series: HttpChartSeries[];
  metric: HttpChartMetric;
  unit: "ms" | "%";
  /** warn/crit marklines only apply to latency metrics. */
  thresholds?: MetricThreshold;
  mode: "aggregate" | "probes";
  isLoading: boolean;
  isError: boolean;
  isFa: boolean;
  onChangeMetric?: (metric: HttpChartMetric) => void;
  onChangeMode?: (mode: "aggregate" | "probes") => void;
  /** Fired with the RFC3339 timestamp when a data point is clicked. */
  onSelectPoint?: (timestamp: string) => void;
}

// HTTP timeline with a metric selector (per-phase durations + error rate), an
// aggregate/per-probe view toggle, a zoom slider and click-to-drill-down.
export function HttpPerformanceChart({
  series,
  metric,
  unit,
  thresholds,
  mode,
  isLoading,
  isError,
  isFa,
  onChangeMetric,
  onChangeMode,
  onSelectPoint,
}: HttpPerformanceChartProps) {
  const locale = useLocale();
  const palette = useChartPalette();

  const t = (en: string, fa: string) => (isFa ? fa : en);
  const aggregateName = mode === "aggregate" ? (isFa ? "تجمیعی" : "Aggregate") : "";

  // Downtime shading is only meaningful on the error-rate view.
  const downIntervals = useMemo(() => {
    if (metric !== "error_rate") return [];
    const intervals = buildDownIntervals(series);
    return intervals.slice(0, 12);
  }, [metric, series]);

  const downMarkArea = useMemo(
    () => ({
      silent: true,
      data: downIntervals.map((interval) => ({
        name: "Down",
        xAxis: [interval.start, interval.end] as [string, string],
        itemStyle: { color: hexToRgba(palette.danger, 0.12) },
      })),
    }),
    [downIntervals, palette.danger],
  );

  // Dashed warn/crit threshold lines for latency metrics.
  const markLine = useMemo(() => {
    if (metric === "error_rate") return undefined;
    const data: Array<{ yAxis: number; name?: string; lineStyle?: { color: string; type: "dashed" } }> = [];
    if (thresholds?.warning != null) {
      data.push({ yAxis: thresholds.warning, name: `warn ${thresholds.warning}`, lineStyle: { color: palette.warning, type: "dashed" } });
    }
    if (thresholds?.critical != null) {
      data.push({ yAxis: thresholds.critical, name: `crit ${thresholds.critical}`, lineStyle: { color: palette.danger, type: "dashed" } });
    }
    if (data.length === 0) return undefined;
    return {
      silent: true,
      symbol: "none" as const,
      label: { show: true, position: "insideEndTop" as const, color: palette.text, fontSize: 11 },
      data,
    };
  }, [metric, thresholds, palette.warning, palette.danger, palette.text]);

  const option = useMemo(() => {
    const formatter = (value: unknown) =>
      typeof value === "number" ? `${Math.round(value)} ${unit}` : String(value);
    const timeLabelFormatter = (value: unknown) => {
      if (typeof value !== "number") return "";
      const date = new Date(value);
      if (date.getMinutes() % 10 !== 0) return "";
      return new Intl.DateTimeFormat(locale, { hour: "2-digit", minute: "2-digit" }).format(date);
    };

    return {
      animation: false,
      grid: makeGrid({ top: 16, right: 16, bottom: 72, left: 48 }),
      tooltip: {
        ...makeTooltip(palette, formatter),
        textStyle: { color: palette.tooltipText, fontSize: 12, fontFamily: CHART_FONT },
      },
      dataZoom: [
        { type: "inside" as const, start: 0, end: 100, zoomOnMouseWheel: true, moveOnMouseMove: true },
        { type: "slider" as const, height: 18, bottom: 30, borderColor: palette.axis, backgroundColor: "transparent", fillerColor: hexToRgba(palette.primary, 0.08), handleStyle: { color: palette.primary }, textStyle: { color: palette.text, fontFamily: CHART_FONT, fontSize: 10 } },
      ],
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
        bottom: 6,
        left: "center",
        right: "auto",
        icon: "roundRect",
        itemWidth: 14,
        itemHeight: 7,
        itemGap: 14,
        textStyle: { color: palette.text, fontFamily: CHART_FONT, fontSize: 11 },
      },
      series: series.map((s, i) => {
        const color = palette.series[i % palette.series.length];
        const glowColor = hexToRgba(color, 0.8);
        return {
          type: "line" as const,
          name: s.probeName || s.location || aggregateName,
          showSymbol: false,
          sampling: "lttb" as const,
          symbol: "circle" as const,
          symbolSize: 7,
          lineStyle: {
            width: mode === "aggregate" ? 3 : 2.5,
            color,
            shadowBlur: 26,
            shadowColor: glowColor,
            shadowOffsetY: 3,
          },
          itemStyle: { color },
          emphasis: {
            focus: "series" as const,
            lineStyle: { width: 4, color, shadowBlur: 38, shadowColor: hexToRgba(color, 0.95), shadowOffsetY: 4 },
          },
          areaStyle: {
            color: {
              type: "linear" as const,
              x: 0, y: 0, x2: 0, y2: 1,
              colorStops: [
                { offset: 0, color: hexToRgba(color, mode === "aggregate" ? 0.42 : 0.3) },
                { offset: 0.5, color: hexToRgba(color, mode === "aggregate" ? 0.16 : 0.1) },
                { offset: 1, color: hexToRgba(color, 0.02) },
              ],
            },
          },
          markArea: metric === "error_rate" && downMarkArea.data.length > 0 ? downMarkArea : undefined,
          markLine: i === 0 && markLine ? markLine : undefined,
          data: s.points.map((p) => [p.time, p.value]),
        };
      }),
    };
  }, [series, mode, metric, unit, locale, palette, downMarkArea, markLine, aggregateName]);

  return (
    <Card variant="bordered" className="h-full shadow-subtle">
      <CardHeader className="flex-row flex-wrap items-center justify-between gap-3 space-y-0 px-5 pt-4">
        <div className="flex flex-wrap items-center gap-2">
          <div className="inline-flex items-center gap-0.5 rounded-lg border border-border/60 bg-muted/25 p-0.5">
            {HTTP_CHART_METRICS.map((m) => (
              <button
                key={m.key}
                type="button"
                onClick={() => onChangeMetric?.(m.key)}
                className={cn(
                  "inline-flex h-7 items-center rounded-md px-2.5 text-xs font-medium transition-colors",
                  metric === m.key ? "bg-card text-foreground shadow-sm" : "text-muted-foreground",
                )}
              >
                {isFa ? m.fa : m.en}
              </button>
            ))}
          </div>
          <div className="inline-flex items-center gap-0.5 rounded-lg border border-border/60 bg-muted/25 p-0.5">
            {(["aggregate", "probes"] as const).map((m) => (
              <button
                key={m}
                type="button"
                onClick={() => onChangeMode?.(m)}
                className={cn(
                  "inline-flex h-7 items-center rounded-md px-2.5 text-xs font-medium transition-colors",
                  mode === m ? "bg-card text-foreground shadow-sm" : "text-muted-foreground",
                )}
              >
                {isFa
                  ? (m === "aggregate" ? "تجمیعی" : "به‌تفکیک پراب")
                  : (m === "aggregate" ? "Aggregate" : "Per probe")}
              </button>
            ))}
          </div>
        </div>
      </CardHeader>
      <CardContent className="px-1 pb-3 pt-1 sm:px-2">
        {isLoading ? (
          <Skeleton className="h-72 w-full rounded-lg" />
        ) : isError ? (
          <div className="flex h-72 items-center justify-center text-sm text-muted-foreground">
            {t("Unable to load data", "خطا در بارگذاری داده")}
          </div>
        ) : series.length === 0 || series.every((s) => s.points.length === 0) ? (
          <div className="flex h-72 items-center justify-center text-sm text-muted-foreground">
            {t("No data to display", "داده‌ای برای نمایش وجود ندارد")}
          </div>
        ) : (
          <EChart
            option={option}
            className="h-72 w-full"
            ariaLabel={HTTP_CHART_METRICS.find((m) => m.key === metric)?.en ?? "HTTP performance"}
            onEvents={{
              click: (params: unknown) => {
                const p = params as { data?: unknown };
                if (!p?.data || !Array.isArray(p.data)) return;
                const ts = p.data[0];
                if (typeof ts === "string") onSelectPoint?.(ts);
              },
            }}
          />
        )}
      </CardContent>
    </Card>
  );
}
