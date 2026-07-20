"use client";

import { useMemo } from "react";
import { useLocale, useTranslations } from "next-intl";

import { EChart, useChartPalette } from "@/components/charts/echart";
import { makeGrid, makeTimeXAxis, makeTooltip } from "@/components/charts/chart-config";
import { EmptyState } from "@/components/common/empty-state";
import type { MetricPoint } from "@/types/result";

interface LatencyChartProps {
  data: MetricPoint[];
}

export function LatencyChart({ data }: LatencyChartProps) {
  const locale = useLocale();
  const t = useTranslations("monitorDetail");
  const palette = useChartPalette();

  const option = useMemo(() => {
    return {
      animation: false,
      grid: makeGrid(),
      tooltip: makeTooltip(palette, (value: unknown) =>
        typeof value === "number" ? `${Math.round(value)} ms` : "—",
      ),
      xAxis: makeTimeXAxis(locale, palette),
      yAxis: {
        type: "value" as const,
        axisLabel: {
          color: palette.text,
          formatter: (value: number) => `${value} ms`,
        },
        splitLine: { lineStyle: { color: palette.axis } },
      },
      series: [
        {
          type: "line",
          name: "latency",
          showSymbol: false,
          smooth: 0.2,
          lineStyle: { width: 2, color: palette.primary },
          itemStyle: { color: palette.primary },
          areaStyle: { opacity: 0.08, color: palette.primary },
          data: data.map((point) => [point.timestamp, point.value]),
        },
      ],
    };
  }, [data, locale, palette]);

  if (data.length === 0) {
    return <EmptyState title={t("noData")} />;
  }

  return <EChart option={option} ariaLabel={t("latencyChart")} />;
}
