"use client";

import { useMemo } from "react";

import type { EChartsCoreOption } from "echarts/core";
import { useTranslations } from "next-intl";

import { EChart } from "./echart";
import type { ProbeLocation } from "@/entities/probe/model/types";
import type { ProbeResult } from "@/entities/monitor/model/result";

export interface MultiLocationSeries {
  location: ProbeLocation;
  results: ProbeResult[];
}

// MultiLocationChart renders per-location latency series for a monitor,
// letting operators compare probe performance across geographic locations.
export function MultiLocationChart({
  series,
}: {
  series: MultiLocationSeries[];
}) {
  const t = useTranslations("monitorDetail");

  const option = useMemo<EChartsCoreOption>(() => {
    const rows = series.flatMap(({ location, results }) =>
      results
        .filter((r) => r.success && r.duration_millis > 0)
        .map((r) => ({
          location: location.name,
          time: new Date(r.finished_at).getTime(),
          latency: r.duration_millis,
        })),
    );

    const locations = [...new Set(rows.map((r) => r.location))];

    return {
      tooltip: {
        trigger: "axis",
      },
      legend: { type: "scroll", bottom: 0 },
      grid: { left: 8, right: 16, top: 8, bottom: 40, containLabel: true },
      xAxis: {
        type: "time",
        axisLabel: { color: "var(--muted-foreground)", fontSize: 11 },
      },
      yAxis: {
        type: "value",
        name: "ms",
        axisLabel: { color: "var(--muted-foreground)", fontSize: 11 },
        splitLine: { lineStyle: { color: "var(--border)" } },
      },
      series: locations.map((location) => ({
        name: location,
        type: "line",
        showSymbol: false,
        smooth: true,
        data: rows
          .filter((r) => r.location === location)
          .map((r) => [r.time, r.latency]),
      })),
    };
  }, [series]);

  if (series.every((s) => s.results.length === 0)) {
    return (
      <div className="flex h-52 w-full items-center justify-center rounded-lg border border-dashed border-border/60 text-sm text-muted-foreground">
        {t("noResults")}
      </div>
    );
  }

  return <EChart option={option} ariaLabel={t("latencyChart")} className="h-52 w-full" />;
}
