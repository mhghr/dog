"use client";

import { useMemo } from "react";
import type { EChartsCoreOption } from "echarts/core";
import { Activity, Radio, Waves, TrendingUp } from "lucide-react";
import { useTranslations } from "next-intl";

import { Card, CardContent } from "@/shared/ui/card";
import { EChart } from "@/shared/ui/charts/echart";
import { Skeleton } from "@/shared/ui/skeleton";
import { formatDuration, formatPercent } from "@/shared/utils/formatters";
import type { ProbeResult } from "@/entities/monitor/model/result";

function metricNumber(result: ProbeResult | undefined | null, key: string): number | null {
  if (!result) return null;
  const value = result.metrics?.[key];
  return typeof value === "number" ? value : null;
}

interface PingMetricCardProps {
  icon: React.ElementType;
  label: string;
  metricKey: string;
  latestResult: ProbeResult | null;
  locale: string;
  format: "ms" | "percent";
  warningThreshold?: number;
  criticalThreshold?: number;
  recentResults?: ProbeResult[];
}

function miniSparklineOption(results: ProbeResult[], metricKey: string, format: "ms" | "percent"): EChartsCoreOption {
  const points = results
    .filter((r) => r.success && typeof r.metrics?.[metricKey] === "number")
    .map((r) => ({
      time: new Date(r.finished_at).getTime(),
      value: r.metrics[metricKey] as number,
    }))
    .reverse()
    .slice(0, 48);

  if (points.length < 2) return {};

  const values = points.map((p) => p.value);
  const min = Math.min(...values);
  const max = Math.max(...values);

  return {
    grid: { left: 0, right: 0, top: 2, bottom: 2 },
    xAxis: { show: false, type: "time", data: points.map((p) => p.time) },
    yAxis: { show: false, type: "value", min: min * 0.9, max: max * 1.1 },
    series: [
      {
        type: "line",
        data: points.map((p) => [p.time, p.value]),
        showSymbol: false,
        smooth: true,
        lineStyle: { color: "var(--primary)", width: 1.5 },
        areaStyle: {
          color: {
            type: "linear",
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: "rgba(59,130,246,0.15)" },
              { offset: 1, color: "rgba(59,130,246,0.01)" },
            ],
          },
        },
      },
    ],
  };
}

export function PingMetricCard({
  icon: Icon,
  label,
  metricKey,
  latestResult,
  locale,
  format: fmt,
  warningThreshold,
  criticalThreshold,
  recentResults = [],
}: PingMetricCardProps) {
  const t = useTranslations("monitorDetail");
  const value = metricNumber(latestResult, metricKey);
  const isExceeded =
    typeof value === "number" && typeof criticalThreshold === "number"
      ? value >= criticalThreshold
      : false;
  const isWarning =
    typeof value === "number" && typeof warningThreshold === "number" && !isExceeded
      ? value >= warningThreshold
      : false;

  const statusColor = latestResult
    ? isExceeded
      ? "bg-destructive"
      : isWarning
        ? "bg-warning"
        : "bg-success"
    : "bg-muted-foreground/30";

  const sparklineOption = useMemo(
    () => miniSparklineOption(recentResults, metricKey, fmt),
    [recentResults, metricKey, fmt],
  );

  const formattedValue =
    value != null
      ? fmt === "ms"
        ? formatDuration(value, locale)
        : formatPercent(value, locale)
      : "—";

  return (
    <Card className="border-border/70 bg-card/60 shadow-none">
      <CardContent className="p-3.5">
        <div className="flex items-center gap-2 mb-2.5">
          <Icon className="size-3.5 text-primary" aria-hidden />
          <span className="text-xs font-medium">{label}</span>
          <span className={`size-2 rounded-full ml-auto ${statusColor}`} />
        </div>

        <div className="flex items-end justify-between gap-3">
          <div>
            <p className="text-2xl font-semibold tabular-nums tracking-tight" dir="ltr">
              {formattedValue}
            </p>
            {latestResult?.finished_at && (
              <p className="mt-0.5 text-[10px] text-muted-foreground">
                {t("lastCheck")}: {new Date(latestResult.finished_at).toLocaleTimeString()}
              </p>
            )}
          </div>

          {sparklineOption.series ? (
            <div className="h-10 w-24 shrink-0">
              <EChart option={sparklineOption} ariaLabel={label} className="h-full w-full" />
            </div>
          ) : null}
        </div>
      </CardContent>
    </Card>
  );
}
