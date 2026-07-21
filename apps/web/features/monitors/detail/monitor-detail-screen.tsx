"use client";

import { useState } from "react";
import { Pencil } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";

import { LatencyChart } from "@/components/charts/latency-chart";
import { ErrorState } from "@/components/common/error-state";
import { MonitorActions } from "@/components/monitors/monitor-actions";
import { MonitorStatusBadge } from "@/components/monitors/monitor-status-badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { GenericMonitorSummary } from "@/features/monitors/detail/generic-summary";
import { GenericMonitorConfiguration } from "@/features/monitors/detail/generic-configuration";
import { NodeMonitorTabs } from "@/features/monitors/detail/node-monitor-tabs";
import { getMonitorDefinition } from "@/features/monitors/core/registry";
import { useMonitor } from "@/hooks/use-monitor";
import { useMonitorMetrics, type MetricsRange } from "@/hooks/use-monitor-metrics";
import { useMonitorResults } from "@/hooks/use-monitor-results";
import { useLocations } from "@/hooks/use-locations";
import { Link } from "@/i18n/navigation";

export function MonitorDetailScreen({ monitorId }: { monitorId: string }) {
  const t = useTranslations("monitorDetail");
  const tMonitors = useTranslations("monitors");
  const locale = useLocale();
  const [range, setRange] = useState<MetricsRange>("24h");

  const monitorQuery = useMonitor(monitorId);
  const metricsQuery = useMonitorMetrics(monitorId, range);
  const recentResultsQuery = useMonitorResults(monitorId, 50, 1);
  const locationsQuery = useLocations();

  if (monitorQuery.isPending) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-16 rounded-xl" />
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, index) => <Skeleton key={index} className="h-24 rounded-xl" />)}
        </div>
        <Skeleton className="h-72 rounded-xl" />
      </div>
    );
  }

  if (monitorQuery.isError) return <ErrorState onRetry={() => void monitorQuery.refetch()} />;

  const monitor = monitorQuery.data;
  const definition = getMonitorDefinition(monitor.type);
  const Summary = definition.Summary ?? GenericMonitorSummary;
  const Configuration = definition.Configuration ?? GenericMonitorConfiguration;
  const recentResults = recentResultsQuery.data?.items ?? [];
  const latestResult = recentResults[0] ?? null;
  const rangeLabel = t(`range${range}` as "range24h");

  return (
    <div className="flex flex-col gap-6">
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="max-w-full truncate text-2xl font-semibold tracking-tight">{monitor.name}</h1>
            <MonitorStatusBadge status={monitor.last_status} />
            <Button variant="outline" size="icon-sm" asChild>
              <Link href={`/app/monitors/${monitor.id}/edit`}>
                <Pencil aria-hidden />
                <span className="sr-only">{tMonitors("editMonitor")}</span>
              </Link>
            </Button>
          </div>
          <p dir="ltr" className="mt-0.5 max-w-full truncate text-start font-mono text-sm text-muted-foreground">{monitor.target}</p>
        </div>
        <MonitorActions monitor={monitor} afterDeleteHref="/app/nodes" />
      </header>

      <NodeMonitorTabs currentMonitor={monitor} />

      <Summary
        monitor={monitor}
        metrics={metricsQuery.data}
        latestResult={latestResult}
        recentResults={recentResults}
        probeLocations={locationsQuery.data?.items ?? []}
        locale={locale}
        rangeLabel={rangeLabel}
      />

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
          {metricsQuery.isPending ? <Skeleton className="h-72 w-full rounded-lg" /> : metricsQuery.isError ? <ErrorState onRetry={() => void metricsQuery.refetch()} /> : <LatencyChart data={metricsQuery.data.series.latency} />}
        </CardContent>
      </Card>

      <Configuration
        monitor={monitor}
        latestResult={latestResult}
        recentResults={recentResults}
        probeLocations={locationsQuery.data?.items ?? []}
        locale={locale}
      />
    </div>
  );
}
