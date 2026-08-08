"use client";

import { useMemo } from "react";
import { useLocale, useTranslations } from "next-intl";

import { EChart, useChartPalette } from "@/shared/ui/charts/echart";
import { makeGrid, makeTimeXAxis, makeTooltip } from "@/shared/ui/charts/chart-config";
import { EmptyState } from "@/design-system/patterns/empty-state";
import type { MetricPoint } from "@/entities/monitor/model/result";

interface SuccessChartProps {
  data: MetricPoint[];
}

export function SuccessChart({ data }: SuccessChartProps) {
  const locale = useLocale();
  const t = useTranslations("monitorDetail");
  const palette = useChartPalette();

  const option = useMemo(() => {
    return {
      animation: false,
      grid: makeGrid(),
      tooltip: makeTooltip(palette, (value: unknown) =>
        typeof value === "number" ? `${Math.round(value * 100)}%` : "—",
      ),
      xAxis: makeTimeXAxis(locale, palette),
      yAxis: {
        type: "value" as const,
        min: 0,
        max: 1,
        axisLabel: {
          color: palette.text,
          formatter: (value: number) => `${Math.round(value * 100)}%`,
        },
        splitLine: { lineStyle: { color: palette.axis } },
      },
      series: [
        {
          type: "line",
          name: "success",
          showSymbol: false,
          step: "end",
          lineStyle: { width: 2, color: palette.success },
          itemStyle: { color: palette.success },
          areaStyle: { opacity: 0.08, color: palette.success },
          data: data.map((point) => [point.timestamp, point.value]),
        },
      ],
    };
  }, [data, locale, palette]);

  if (data.length === 0) {
    return <EmptyState title={t("noData")} />;
  }

  return <EChart option={option} ariaLabel={t("successChart")} />;
}
