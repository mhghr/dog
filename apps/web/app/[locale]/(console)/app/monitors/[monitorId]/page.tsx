"use client";

import { use, useState } from "react";
import { useLocale, useTranslations } from "next-intl";

import { ErrorState } from "@/components/common/error-state";
import { RelativeTime } from "@/components/common/relative-time";
import { LatencyChart } from "@/components/charts/latency-chart";
import { SuccessChart } from "@/components/charts/success-chart";
import { MonitorActions } from "@/components/monitors/monitor-actions";
import { MonitorStatusBadge } from "@/components/monitors/monitor-status-badge";
import { MonitorTypeLabel } from "@/components/monitors/monitor-type-label";
import {
  ResultTable,
  ResultTableSkeleton,
} from "@/components/results/result-table";
import { ResultDetailSheet } from "@/components/results/result-detail-sheet";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useMonitor } from "@/hooks/use-monitor";
import { useMonitorMetrics, type MetricsRange } from "@/hooks/use-monitor-metrics";
import { useMonitorResults } from "@/hooks/use-monitor-results";
import { formatDuration, formatInterval, formatPercent } from "@/lib/formatters";
import type { ProbeResult } from "@/types/result";

function SummaryCard({ label, value }: { label: string; value: string }) {
  return (
    <Card>
      <CardContent className="flex flex-col gap-1">
        <span className="text-sm text-muted-foreground">{label}</span>
        <span className="text-xl font-semibold tabular-nums" dir="ltr">
          {value}
        </span>
      </CardContent>
    </Card>
  );
}

export default function MonitorDetailPage({
  params,
}: {
  params: Promise<{ monitorId: string }>;
}) {
  const { monitorId } = use(params);

  const t = useTranslations("monitorDetail");
  const tMonitors = useTranslations("monitors");
  const tResults = useTranslations("results");
  const tFields = useTranslations("monitors.fields");
  const locale = useLocale();

  const [range, setRange] = useState<MetricsRange>("24h");
  const [selectedResult, setSelectedResult] = useState<ProbeResult | null>(null);

  const monitorQuery = useMonitor(monitorId);
  const metricsQuery = useMonitorMetrics(monitorId, range);
  const resultsQuery = useMonitorResults(monitorId, 50);

  if (monitorQuery.isPending) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-16 rounded-xl" />
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-20 rounded-xl" />
          ))}
        </div>
        <Skeleton className="h-72 rounded-xl" />
      </div>
    );
  }

  if (monitorQuery.isError) {
    return <ErrorState onRetry={() => void monitorQuery.refetch()} />;
  }

  const monitor = monitorQuery.data;
  const summary = metricsQuery.data?.summary;

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="max-w-full truncate text-2xl font-semibold tracking-tight">
              {monitor.name}
            </h1>
            <MonitorStatusBadge status={monitor.last_status} />
          </div>
          <p
            dir="ltr"
            className="mt-1 max-w-full truncate text-start font-mono text-sm text-muted-foreground"
          >
            {monitor.target}
          </p>
          <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground">
            <MonitorTypeLabel type={monitor.type} />
            <span>
              {tMonitors("lastCheck")}:{" "}
              <RelativeTime value={monitor.last_checked_at} />
            </span>
            <span>
              {t("nextRun")}: <RelativeTime value={monitor.next_run_at} />
            </span>
          </div>
        </div>

        <MonitorActions monitor={monitor} afterDeleteHref="/app/nodes" />
      </div>

      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <SummaryCard
          label={`${t("uptime")} (${t(`range${range}` as "range24h")})`}
          value={formatPercent(summary?.uptime_percent ?? null, locale)}
        />
        <SummaryCard
          label={t("currentLatency")}
          value={
            monitor.last_result
              ? formatDuration(monitor.last_result.duration_millis, locale)
              : "—"
          }
        />
        <SummaryCard
          label={t("p95Latency")}
          value={formatDuration(summary?.p95_latency_ms ?? null, locale)}
        />
        <SummaryCard
          label={tMonitors("interval")}
          value={formatInterval(monitor.interval_seconds, locale)}
        />
      </div>

      <Card>
        <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-3">
          <CardTitle className="text-base">{t("latencyChart")}</CardTitle>
          <Tabs value={range} onValueChange={(value) => setRange(value as MetricsRange)}>
            <TabsList>
              <TabsTrigger value="24h">{t("range24h")}</TabsTrigger>
              <TabsTrigger value="7d">{t("range7d")}</TabsTrigger>
              <TabsTrigger value="30d">{t("range30d")}</TabsTrigger>
            </TabsList>
          </Tabs>
        </CardHeader>
        <CardContent>
          {metricsQuery.isPending ? (
            <Skeleton className="h-72 w-full rounded-lg" />
          ) : metricsQuery.isError ? (
            <ErrorState onRetry={() => void metricsQuery.refetch()} />
          ) : (
            <LatencyChart data={metricsQuery.data.series.latency} />
          )}
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <Card className="xl:col-span-2">
          <CardHeader>
            <CardTitle className="text-base">{t("recentResults")}</CardTitle>
          </CardHeader>
          <CardContent>
            {resultsQuery.isPending ? (
              <ResultTableSkeleton />
            ) : resultsQuery.isError ? (
              <ErrorState onRetry={() => void resultsQuery.refetch()} />
            ) : resultsQuery.data.items.length === 0 ? (
              <p className="py-8 text-center text-sm text-muted-foreground">
                {tResults("empty")}
              </p>
            ) : (
              <ResultTable
                results={resultsQuery.data.items}
                onSelect={setSelectedResult}
              />
            )}
          </CardContent>
        </Card>

        <div className="flex flex-col gap-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">{t("successChart")}</CardTitle>
            </CardHeader>
            <CardContent>
              {metricsQuery.isPending ? (
                <Skeleton className="h-40 w-full rounded-lg" />
              ) : metricsQuery.isError ? null : (
                <SuccessChart data={metricsQuery.data.series.success} />
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">{t("configuration")}</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-2 text-sm">
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">
                  {tFields("intervalSeconds")}
                </span>
                <span className="tabular-nums" dir="ltr">
                  {monitor.interval_seconds}s
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">
                  {tFields("timeoutMillis")}
                </span>
                <span className="tabular-nums" dir="ltr">
                  {monitor.timeout_millis}ms
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">{tFields("retries")}</span>
                <span className="tabular-nums" dir="ltr">
                  {monitor.retries}
                </span>
              </div>
              <pre
                dir="ltr"
                className="mt-2 max-h-48 overflow-auto rounded-lg border border-border bg-secondary/50 p-3 font-mono text-xs leading-relaxed"
              >
                {JSON.stringify(monitor.config, null, 2)}
              </pre>
            </CardContent>
          </Card>
        </div>
      </div>

      <ResultDetailSheet
        result={selectedResult}
        onClose={() => setSelectedResult(null)}
      />
    </div>
  );
}
