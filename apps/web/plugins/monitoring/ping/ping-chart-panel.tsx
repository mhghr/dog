"use client";

import { useMemo, useState } from "react";
import type { EChartsCoreOption } from "echarts/core";
import { useTranslations } from "next-intl";

import { EChart } from "@/shared/ui/charts/echart";
import { Skeleton } from "@/shared/ui/skeleton";
import { usePingSeriesByMetric, type MetricsRange } from "@/entities/monitor/hooks/use-ping-metrics";
import type { ProbeLocation } from "@/entities/probe/model/types";

const PING_METRICS = [
  { key: "rtt_ms", label: "RTT", unit: "ms" },
  { key: "packet_loss_percent", label: "packetLoss", unit: "%" },
  { key: "jitter_ms", label: "jitter", unit: "ms" },
  { key: "availability", label: "availability", unit: "%" },
] as const;

type PingMetricTab = (typeof PING_METRICS)[number]["key"];

interface PingChartPanelProps {
  monitorId: string;
  probeLocations: ProbeLocation[];
}

export function PingChartPanel({ monitorId, probeLocations }: PingChartPanelProps) {
  const t = useTranslations("monitorDetail");
  const [activeMetric, setActiveMetric] = useState<PingMetricTab>("rtt_ms");
  const [range, setRange] = useState<MetricsRange>("24h");

  const { data, isPending } = usePingSeriesByMetric(monitorId, activeMetric, range);

  const option = useMemo<EChartsCoreOption>(() => {
    if (!data?.items?.length) {
      return {};
    }

    const colors = [
      "#3b82f6", "#ef4444", "#22c55e", "#f59e0b", "#8b5cf6",
      "#06b6d4", "#ec4899", "#84cc16",
    ];

    const active = PING_METRICS.find((m) => m.key === activeMetric);
    const unit = active?.unit ?? "";

    return {
      tooltip: {
        trigger: "axis",
        valueFormatter: (value: unknown) => (typeof value === "number" ? `${value.toFixed(2)} ${unit}` : String(value)),
      },
      legend: {
        type: "scroll",
        bottom: 0,
        textStyle: { fontSize: 11 },
      },
      grid: { left: 8, right: 16, top: 8, bottom: 40, containLabel: true },
      xAxis: {
        type: "time",
        axisLabel: { color: "var(--muted-foreground)", fontSize: 11 },
      },
      yAxis: {
        type: "value",
        name: unit,
        axisLabel: { color: "var(--muted-foreground)", fontSize: 11 },
        splitLine: { lineStyle: { color: "var(--border)" } },
      },
      color: colors,
      series: data.items.map((s) => ({
        name: s.probe_name || s.location || "—",
        type: "line",
        showSymbol: false,
        smooth: true,
        data: s.points.map((p) => [new Date(p.timestamp).getTime(), p.value]),
      })),
    };
  }, [data, activeMetric]);

  const hasData = data?.items?.some((s) => s.points.length > 0) ?? false;

  return (
    <div className="overflow-hidden rounded-xl border border-border/65 bg-card/50 shadow-none">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border/50 px-4 py-2.5">
        <div className="flex gap-1">
          {PING_METRICS.map((metric) => (
            <button
              key={metric.key}
              type="button"
              onClick={() => setActiveMetric(metric.key)}
              className={`rounded-md px-3 py-1 text-xs font-medium transition-colors ${
                activeMetric === metric.key
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground hover:bg-muted/80"
              }`}
            >
              {t(metric.label)}
            </button>
          ))}
        </div>
        <div className="flex gap-1">
          {(["24h", "7d", "30d"] as const).map((r) => (
            <button
              key={r}
              type="button"
              onClick={() => setRange(r)}
              className={`rounded-md px-3 py-1 text-xs font-medium transition-colors ${
                range === r
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground hover:bg-muted/80"
              }`}
            >
              {r}
            </button>
          ))}
        </div>
      </div>

      <div className="px-4 py-3">
        {isPending ? (
          <Skeleton className="h-52 w-full rounded-lg" />
        ) : !hasData ? (
          <div className="flex h-52 w-full items-center justify-center rounded-lg border border-dashed border-border/60 text-sm text-muted-foreground">
            {t("noResults")}
          </div>
        ) : (
          <EChart option={option} ariaLabel={t("pingCharts")} className="h-52 w-full" />
        )}
      </div>
    </div>
  );
}
