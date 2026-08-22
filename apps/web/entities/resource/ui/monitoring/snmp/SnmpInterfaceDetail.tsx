"use client";

import { useMemo, useState } from "react";

import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/shared/ui/sheet";
import { Skeleton } from "@/shared/ui/skeleton";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { cn } from "@/shared/utils/cn";
import { EChart, useChartPalette } from "@/shared/ui/charts/echart";
import { makeGrid, makeTimeXAxis, makeTooltip, hexToRgba } from "@/shared/ui/charts/chart-config";
import { useResourceMonitorMetrics } from "@/entities/resource/hooks/use-resource";
import type { MetricsRange } from "@/entities/resource/hooks/use-resource";
import type { ProbeSeries } from "@/entities/resource/api/resource.api";
import { operStatusLabel } from "./snmp-metrics";
import { interfaceDisplayName } from "./snmp-config";
import type { SnmpInterfaceTableRow } from "./SnmpInterfacesCard";

const CHART_FONT = "'bakh', 'estedad', ui-sans-serif, system-ui, sans-serif";

function poolSeries(series: ProbeSeries[] | undefined, transform?: (v: number) => number): Array<[number, number]> {
  const points = (series ?? []).flatMap((s) =>
    s.points.map((p) => [Date.parse(p.timestamp), transform ? transform(p.value) : p.value] as [number, number]),
  );
  return points.filter(([t, v]) => Number.isFinite(t) && Number.isFinite(v));
}

function formatBits(bps: number | undefined): string {
  if (bps == null) return "—";
  if (bps >= 1_000_000_000) return `${(bps / 1_000_000_000).toFixed(2)} Gbps`;
  if (bps >= 1_000_000) return `${(bps / 1_000_000).toFixed(2)} Mbps`;
  if (bps >= 1_000) return `${(bps / 1_000).toFixed(1)} Kbps`;
  return `${Math.round(bps)} bps`;
}

type DetailTab = "traffic" | "utilization" | "errors";

function DetailChart({
  tab,
  range,
  resourceId,
  monitorId,
  ifIndex,
  isFa,
}: {
  tab: DetailTab;
  range: MetricsRange;
  resourceId: string;
  monitorId: string;
  ifIndex: number;
  isFa: boolean;
}) {
  const locale = isFa ? "fa" : "en";
  const palette = useChartPalette();

  const metricKeys = useMemo(() => {
    switch (tab) {
      case "traffic":
        return { in: `if_${ifIndex}_in_bps`, out: `if_${ifIndex}_out_bps` } as const;
      case "utilization":
        return { util: `if_${ifIndex}_utilization_percent` } as const;
      case "errors":
        return {
          errors: `if_${ifIndex}_in_errors`,
          discards: `if_${ifIndex}_in_discards`,
        } as const;
    }
  }, [tab, ifIndex]);

  const qIn = useResourceMonitorMetrics(resourceId, monitorId, range, metricKeys.in);
  const qOut = useResourceMonitorMetrics(resourceId, monitorId, range, metricKeys.out);
  const qUtil = useResourceMonitorMetrics(resourceId, monitorId, range, metricKeys.util);
  const qErrors = useResourceMonitorMetrics(resourceId, monitorId, range, metricKeys.errors);
  const qDiscards = useResourceMonitorMetrics(resourceId, monitorId, range, metricKeys.discards);

  const isLoading = [qIn, qOut, qUtil, qErrors, qDiscards].some((q) => q.isPending);

  const option = useMemo(() => {
    const timeFormatter = (value: unknown) => {
      if (typeof value !== "number") return "";
      const date = new Date(value);
      if (date.getMinutes() % 10 !== 0) return "";
      return new Intl.DateTimeFormat(locale, { hour: "2-digit", minute: "2-digit" }).format(date);
    };

    const series: Array<Record<string, unknown>> = [];
    if (tab === "traffic") {
      series.push(
        {
          type: "line",
          name: isFa ? "ورودی" : "Inbound",
          showSymbol: false,
          sampling: "lttb",
          data: poolSeries(qIn.data?.series, (v) => v / 1_000_000),
          lineStyle: { width: 2.5, color: palette.series[0], shadowBlur: 14, shadowColor: hexToRgba(palette.series[0], 0.5) },
          itemStyle: { color: palette.series[0] },
          areaStyle: { color: hexToRgba(palette.series[0], 0.15) },
        },
        {
          type: "line",
          name: isFa ? "خروجی" : "Outbound",
          showSymbol: false,
          sampling: "lttb",
          data: poolSeries(qOut.data?.series, (v) => v / 1_000_000),
          lineStyle: { width: 2.5, color: palette.series[1], shadowBlur: 14, shadowColor: hexToRgba(palette.series[1], 0.5) },
          itemStyle: { color: palette.series[1] },
          areaStyle: { color: hexToRgba(palette.series[1], 0.15) },
        },
      );
    } else if (tab === "utilization") {
      series.push({
        type: "line",
        name: isFa ? "مصرف پهنای باند" : "Utilization",
        showSymbol: false,
        sampling: "lttb",
        data: poolSeries(qUtil.data?.series),
        lineStyle: { width: 2.5, color: palette.series[0], shadowBlur: 14, shadowColor: hexToRgba(palette.series[0], 0.5) },
        itemStyle: { color: palette.series[0] },
        areaStyle: { color: hexToRgba(palette.series[0], 0.15) },
      });
    } else {
      series.push(
        {
          type: "line",
          name: isFa ? "خطاها" : "Errors",
          showSymbol: false,
          sampling: "lttb",
          data: poolSeries(qErrors.data?.series),
          lineStyle: { width: 2, color: palette.series[2], shadowBlur: 12, shadowColor: hexToRgba(palette.series[2], 0.5) },
          itemStyle: { color: palette.series[2] },
        },
        {
          type: "line",
          name: isFa ? "افت بسته" : "Discards",
          showSymbol: false,
          sampling: "lttb",
          data: poolSeries(qDiscards.data?.series),
          lineStyle: { width: 2, color: palette.series[3], shadowBlur: 12, shadowColor: hexToRgba(palette.series[3], 0.5) },
          itemStyle: { color: palette.series[3] },
        },
      );
    }

    return {
      animation: false,
      grid: makeGrid({ top: 24, right: 16, bottom: 40, left: 56 }),
      tooltip: { ...makeTooltip(palette, (value: unknown) => String(value)), textStyle: { color: palette.tooltipText, fontSize: 12, fontFamily: CHART_FONT } },
      xAxis: {
        ...makeTimeXAxis(locale, palette, CHART_FONT),
        axisLabel: { color: palette.text, fontFamily: CHART_FONT, hideOverlap: true, interval: 0, formatter: timeFormatter },
      },
      yAxis: {
        type: "value",
        axisLabel: { color: palette.text, fontFamily: CHART_FONT, formatter: (v: number) => Math.round(v).toLocaleString() },
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { lineStyle: { color: palette.axis, opacity: 0.35 } },
      },
      legend: { bottom: 0, left: "center", icon: "roundRect", itemWidth: 14, itemHeight: 7, textStyle: { color: palette.text, fontFamily: CHART_FONT, fontSize: 11 } },
      series,
    };
  }, [tab, isFa, locale, palette, qIn.data, qOut.data, qUtil.data, qErrors.data, qDiscards.data]);

  if (isLoading) return <Skeleton className="h-64 w-full rounded-lg" />;
  return <EChart option={option} className="h-64 w-full" ariaLabel={tab} />;
}

export function SnmpInterfaceDetail({
  resourceId,
  monitorId,
  row,
  range,
  isFa,
  onOpenChange,
}: {
  resourceId: string;
  monitorId: string;
  row: SnmpInterfaceTableRow;
  range: MetricsRange;
  isFa: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const [tab, setTab] = useState<DetailTab>("traffic");
  const s = row.snapshot;
  const name = interfaceDisplayName(s.if_index, s.if_name, row.setting ? [row.setting] : []);
  const up = s.if_oper_status === 1;

  const stat = (label: string, value: string) => (
    <div className="flex flex-col gap-0.5 rounded-lg border border-border/40 px-3 py-2">
      <span className="text-[10px] text-muted-foreground">{label}</span>
      <span className="text-sm font-semibold tabular-nums text-foreground" dir="ltr">
        {value}
      </span>
    </div>
  );

  return (
    <Sheet open onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-xl">
        <SheetHeader>
          <SheetTitle className="flex flex-wrap items-center gap-2">
            <span dir="auto">{name}</span>
            <Badge
              variant="outline"
              className={cn(
                "px-2 py-0.5 text-[10px]",
                up ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-500" : "border-destructive/30 bg-destructive/10 text-destructive",
              )}
            >
              {operStatusLabel(s.if_oper_status)}
            </Badge>
          </SheetTitle>
        </SheetHeader>

        <div className="mt-4 flex flex-col gap-4 px-4">
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            {stat(t("Inbound", "ورودی"), formatBits(s.in_bps))}
            {stat(t("Outbound", "خروجی"), formatBits(s.out_bps))}
            {stat(t("Utilization", "مصرف"), s.utilization_percent == null ? "—" : `${s.utilization_percent.toFixed(1)}%`)}
            {stat(t("Speed", "سرعت"), s.if_speed_bps ? `${(s.if_speed_bps / 1_000_000_000).toFixed(2)} Gbps` : "—")}
            {stat(t("In errors", "خطا ورودی"), String(s.if_in_errors ?? 0))}
            {stat(t("Out errors", "خطا خروجی"), String(s.if_out_errors ?? 0))}
            {stat(t("In discards", "افت ورودی"), String(s.if_in_discards ?? 0))}
            {stat(t("Out discards", "افت خروجی"), String(s.if_out_discards ?? 0))}
          </div>

          {s.if_descr && (
            <p className="text-xs text-muted-foreground" dir="auto">
              <span className="font-medium text-foreground">{t("Description", "توضیحات")}:</span> {s.if_descr}
            </p>
          )}

          <div className="inline-flex w-fit items-center gap-0.5 rounded-lg border border-border/60 bg-muted/25 p-0.5">
            {(
              [
                { key: "traffic", label: t("Traffic", "ترافیک") },
                { key: "utilization", label: t("Utilization", "مصرف") },
                { key: "errors", label: t("Errors & Discards", "خطا و افت") },
              ] as Array<{ key: DetailTab; label: string }>
            ).map((item) => (
              <Button
                key={item.key}
                type="button"
                size="sm"
                variant="ghost"
                className={cn("h-7 px-2.5 text-xs", tab === item.key ? "bg-card text-foreground shadow-sm" : "text-muted-foreground")}
                onClick={() => setTab(item.key)}
              >
                {item.label}
              </Button>
            ))}
          </div>

          <DetailChart
            tab={tab}
            range={range}
            resourceId={resourceId}
            monitorId={monitorId}
            ifIndex={s.if_index}
            isFa={isFa}
          />
        </div>
      </SheetContent>
    </Sheet>
  );
}
