"use client";

import { useMemo, useState } from "react";
import { useLocale } from "next-intl";

import { EChart, useChartPalette } from "@/shared/ui/charts/echart";
import { makeGrid, makeTimeXAxis, makeTooltip } from "@/shared/ui/charts/chart-config";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { Button } from "@/shared/ui/button";
import { Skeleton } from "@/shared/ui/skeleton";
import { cn } from "@/shared/utils/cn";
import {
  useResourceMonitorMetrics,
  type MetricsRange,
} from "@/entities/resource/hooks/use-resource";
import type { ProbeResult } from "@/entities/monitor/model/result";

const RANGES: MetricsRange[] = ["15m", "1h", "6h", "24h", "7d", "30d"];

function formatValue(value: number): string {
  if (value >= 1000) return `${(value / 1000).toFixed(2)}s`;
  return `${Math.round(value)}ms`;
}

function getMetricValue(result: ProbeResult, keys: string[]): number | null {
  for (const key of keys) {
    const raw = result.metrics?.[key];
    if (typeof raw === "number") return raw;
    if (typeof raw === "string" && !Number.isNaN(Number(raw))) return Number(raw);
  }
  return null;
}

const PALETTE = ["#4F66F0", "#0D9464", "#DC3035", "#F59E0B", "#8B5CF6", "#06B6D4", "#EC4899", "#84CC16"];

export function ResourceMonitorDashboard({
  resourceId,
  monitorId,
  metricKeys,
  isFa,
}: {
  resourceId: string;
  monitorId: string;
  metricKeys: string[];
  isFa: boolean;
}) {
  const locale = useLocale();
  const [range, setRange] = useState<MetricsRange>("1h");
  const metricsQuery = useResourceMonitorMetrics(resourceId, monitorId, range);

  const data = metricsQuery.data;
  const series = data?.series ?? [];
  const latest = data?.latest ?? [];

  // Aggregate per-probe current values for the metric cards
  const perProbeLatest = useMemo(() => {
    return latest.map((result) => ({
      probeName: (result.attributes?.probe_name as string) || result.probe_location_id || "—",
      result,
    }));
  }, [latest]);

  // Cards: average latency, availability, and one card per metric key
  const cards = useMemo(() => {
    const avgLatency = latest.length
      ? latest.reduce((sum, r) => sum + r.duration_millis, 0) / latest.length
      : null;
    const availability = latest.length
      ? (latest.filter((r) => r.success).length / latest.length) * 100
      : null;

    const metricCards = metricKeys.slice(0, 4).map((key) => {
      const values = latest
        .map((r) => getMetricValue(r, [key]))
        .filter((v): v is number => v !== null);
      const avg = values.length
        ? values.reduce((a, b) => a + b, 0) / values.length
        : null;
      return { key, avg };
    });

    return {
      avgLatency,
      availability,
      metricCards,
    };
  }, [latest, metricKeys]);

  const palette = useChartPalette();

  const chartOption = useMemo(() => {
    return {
      animation: false,
      grid: makeGrid(),
      tooltip: makeTooltip(palette, (value: unknown) =>
        typeof value === "number" ? formatValue(value) : String(value),
      ),
      xAxis: makeTimeXAxis(locale, palette),
      yAxis: {
        type: "value" as const,
        axisLabel: { color: palette.text, formatter: (value: number) => formatValue(value) },
        splitLine: { lineStyle: { color: palette.axis } },
      },
      legend: {
        type: "scroll" as const,
        bottom: 0,
        textStyle: { color: palette.text },
      },
      series: series.map((s, index) => ({
        type: "line" as const,
        name: s.probe_name || `probe-${index + 1}`,
        showSymbol: false,
        smooth: 0.2,
        lineStyle: { width: 2, color: PALETTE[index % PALETTE.length] },
        itemStyle: { color: PALETTE[index % PALETTE.length] },
        data: s.points.map((p) => [p.timestamp, p.value]),
      })),
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [series, locale]);

  if (metricsQuery.isPending) {
    return (
      <div className="space-y-4">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-xl" />
          ))}
        </div>
        <Skeleton className="h-72 rounded-xl" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Time range selector */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="inline-flex w-fit items-center rounded-lg border border-border/70 bg-muted/25 p-1">
          {RANGES.map((r) => (
            <Button
              key={r}
              type="button"
              variant={range === r ? "secondary" : "ghost"}
              size="sm"
              className="h-7 px-2.5 text-xs"
              onClick={() => setRange(r)}
            >
              {r}
            </Button>
          ))}
        </div>
      </div>

      {/* Metric cards */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <MetricCard
          label={isFa ? "میانگین تأخیر" : "Avg latency"}
          value={cards.avgLatency == null ? "—" : formatValue(cards.avgLatency)}
        />
        <MetricCard
          label={isFa ? "دسترس‌پذیری" : "Availability"}
          value={cards.availability == null ? "—" : `${cards.availability.toFixed(1)}%`}
        />
        {cards.metricCards.map((card) => (
          <MetricCard
            key={card.key}
            label={card.key}
            value={card.avg == null ? "—" : formatValue(card.avg)}
          />
        ))}
      </div>

      {/* Per-probe current values */}
      {perProbeLatest.length > 0 ? (
        <Card variant="bordered">
          <CardHeader>
            <CardTitle>{isFa ? "مقادیر هر پراب" : "Values per probe"}</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col">
            {perProbeLatest.map(({ probeName, result }) => (
              <div
                key={probeName}
                className="flex items-center justify-between gap-3 border-b border-border/50 py-2.5 last:border-0"
              >
                <div className="flex items-center gap-2.5">
                  <span className="size-2 rounded-full bg-current" style={{ color: PALETTE[perProbeLatest.findIndex((p) => p.probeName === probeName) % PALETTE.length] }} />
                  <span className="text-sm font-medium">{probeName}</span>
                  {result.success ? (
                    <span className="text-xs text-success">{isFa ? "موفق" : "ok"}</span>
                  ) : (
                    <span className="text-xs text-destructive">{isFa ? "ناموفق" : "fail"}</span>
                  )}
                </div>
                <span className="text-sm tabular-nums text-muted-foreground">
                  {result.duration_millis}ms
                </span>
              </div>
            ))}
          </CardContent>
        </Card>
      ) : null}

      {/* Chart */}
      {series.length > 0 ? (
        <Card variant="bordered">
          <CardContent className="pt-4">
            <EChart option={chartOption} className="h-72 w-full" />
          </CardContent>
        </Card>
      ) : (
        <Card variant="bordered">
          <CardContent className="flex items-center justify-center py-12 text-sm text-muted-foreground">
            {isFa ? "داده‌ای برای نمایش نیست" : "No data to display"}
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <Card variant="bordered">
      <CardContent className="p-4">
        <p className="truncate text-xs text-muted-foreground">{label}</p>
        <p className="mt-1 text-xl font-semibold tabular-nums" dir="ltr">
          {value}
        </p>
      </CardContent>
    </Card>
  );
}
