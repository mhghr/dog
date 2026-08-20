"use client";

import { useMemo } from "react";
import type { EChartsCoreOption } from "echarts/core";
import { useTranslations } from "next-intl";

import { Card, CardContent } from "@/shared/ui/card";
import { EChart } from "@/shared/ui/charts/echart";
import { formatDuration, formatBytes } from "@/shared/utils/formatters";
import type { ProbeResult } from "@/entities/monitor/model/result";

function metricNumber(result: ProbeResult | undefined | null, key: string): number | null {
  if (!result) return null;
  const value = result.metrics?.[key];
  return typeof value === "number" ? value : null;
}

interface HttpMetricCardProps {
  icon: React.ElementType;
  label: string;
  value?: string;
  metricKey?: string;
  unit?: "ms" | "bytes";
  latestResult: ProbeResult | null;
  locale: string;
  warningThreshold?: number;
  criticalThreshold?: number;
  recentResults?: ProbeResult[];
}

function miniSparklineOption(results: ProbeResult[], metricKey: string): EChartsCoreOption {
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

export function HttpMetricCard({
  icon: Icon,
  label,
  value,
  metricKey,
  unit,
  latestResult,
  locale,
  warningThreshold,
  criticalThreshold,
  recentResults = [],
}: HttpMetricCardProps) {
  const t = useTranslations("monitorDetail");
  const rawValue = metricKey ? metricNumber(latestResult, metricKey) : null;

  const isExceeded =
    typeof rawValue === "number" && typeof criticalThreshold === "number"
      ? rawValue >= criticalThreshold
      : false;
  const isWarning =
    typeof rawValue === "number" && typeof warningThreshold === "number" && !isExceeded
      ? rawValue >= warningThreshold
      : false;

  const statusColor = latestResult
    ? isExceeded
      ? "bg-destructive"
      : isWarning
        ? "bg-warning"
        : "bg-success"
    : "bg-muted-foreground/30";

  const sparklineOption = useMemo(
    () => (metricKey ? miniSparklineOption(recentResults, metricKey) : {}),
    [recentResults, metricKey],
  );

  const formattedValue =
    value ??
    (rawValue != null
      ? unit === "ms"
        ? formatDuration(rawValue, locale)
        : unit === "bytes"
          ? formatBytes(rawValue, locale)
          : String(rawValue)
      : "—");

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
