"use client";

import { useState } from "react";
import { Pencil } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";

import dynamic from "next/dynamic";

import { ErrorState } from "@/components/common/error-state";

const MultiLocationChart = dynamic(() => import("@/components/charts/multi-location-chart"), { ssr: false });
import { MonitorActions } from "@/components/monitors/monitor-actions";
import { MonitorStatusBadge } from "@/components/monitors/monitor-status-badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { GenericMonitorSummary } from "@/features/monitors/detail/generic-summary";
import { GenericMonitorConfiguration } from "@/features/monitors/detail/generic-configuration";
import { NodeMonitorTabs } from "@/features/monitors/detail/node-monitor-tabs";
import { getMonitorDefinition } from "@/features/monitors/core/registry";
import { ParameterRulesTable } from "@/features/monitors/health/parameter-rules-table";
import { useMonitor } from "@/hooks/use-monitor";
import { useMonitorMetrics, type MetricsRange } from "@/hooks/use-monitor-metrics";
import { useMonitorResults } from "@/hooks/use-monitor-results";
import { useLocations } from "@/hooks/use-locations";
import { useParameterRules, useParameterHealthStates } from "@/hooks/use-health-rules";
import { Link } from "@/i18n/navigation";

export function MonitorDetailScreen({ monitorId }: { monitorId: string }) {
  const t = useTranslations("monitorDetail");
  const tMonitors = useTranslations("monitors");
  const tHealth = useTranslations("health");
  const locale = useLocale();
  const [range, setRange] = useState<MetricsRange>("24h");
  const [detailTab, setDetailTab] = useState("overview");

  const monitorQuery = useMonitor(monitorId);
  const metricsQuery = useMonitorMetrics(monitorId, range);
  const recentResultsQuery = useMonitorResults(monitorId, 50, 1);
  const locationsQuery = useLocations();
  const rulesQuery = useParameterRules(monitorId);
  const healthStatesQuery = useParameterHealthStates(monitorId);

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
  const probeLocations = locationsQuery.data?.items ?? [];
  const rangeLabel = t(`range${range}` as "range24h");

  return (
    <div className="flex min-w-0 flex-col gap-5">
      <header className="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-border/65 bg-card/45 px-4 py-3.5 shadow-sm shadow-foreground/[0.02]">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="max-w-full truncate text-xl font-semibold tracking-tight sm:text-2xl">{monitor.name}</h1>
            <MonitorStatusBadge status={monitor.last_status} />
            <Button variant="outline" size="icon-sm" asChild>
              <Link href={`/app/nodes/${monitor.id}/edit`}>
                <Pencil aria-hidden />
                <span className="sr-only">{tMonitors("editMonitor")}</span>
              </Link>
            </Button>
          </div>
          <p dir="ltr" className="mt-1 max-w-full truncate text-start font-mono text-xs text-muted-foreground sm:text-sm">{monitor.target}</p>
        </div>
        <MonitorActions monitor={monitor} afterDeleteHref="/app/nodes" />
      </header>

      <div className="rounded-t-xl border-x border-t border-border/60 bg-card/30 px-2 pt-1">
        <NodeMonitorTabs currentMonitor={monitor} />
      </div>

      <Tabs value={detailTab} onValueChange={setDetailTab}>
        <TabsList>
          <TabsTrigger value="overview">{t("latencyChart")}</TabsTrigger>
          <TabsTrigger value="parameters">{tHealth("parameters")}</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-4 flex flex-col gap-5">
          <Summary
            monitor={monitor}
            metrics={metricsQuery.data}
            latestResult={latestResult}
            recentResults={recentResults}
            probeLocations={probeLocations}
            locale={locale}
            rangeLabel={rangeLabel}
          />

          <Card className="overflow-hidden border-border/65 py-0 shadow-none">
            <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-3 border-b border-border/55 px-4 py-3">
              <CardTitle className="text-base">{t("latencyChart")}</CardTitle>
              <Tabs value={range} onValueChange={(value) => setRange(value as MetricsRange)}>
                <TabsList>
                  <TabsTrigger value="24h">{t("range24h")}</TabsTrigger>
                  <TabsTrigger value="7d">{t("range7d")}</TabsTrigger>
                  <TabsTrigger value="30d">{t("range30d")}</TabsTrigger>
                </TabsList>
              </Tabs>
            </CardHeader>
            <CardContent className="px-4 py-4">
              {metricsQuery.isPending ? <Skeleton className="h-72 w-full rounded-lg" /> : metricsQuery.isError ? <ErrorState onRetry={() => void metricsQuery.refetch()} /> : (
                <MultiLocationChart
                  series={probeLocations.map((loc) => ({
                    location: loc,
                    results: recentResults.filter((r) => (r.probe_location_id || "default") === loc.id),
                  }))}
                />
              )}
            </CardContent>
          </Card>

          <Configuration
            monitor={monitor}
            latestResult={latestResult}
            recentResults={recentResults}
            probeLocations={probeLocations}
            locale={locale}
          />
        </TabsContent>

        <TabsContent value="parameters" className="mt-4">
          <ParameterRulesTable
            monitorId={monitor.id}
            monitorType={monitor.type}
            rules={rulesQuery.data}
            healthStates={healthStatesQuery.data}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}
