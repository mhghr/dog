"use client";

import { useState } from "react";
import { Pencil } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";

import dynamic from "next/dynamic";

const MultiLocationChart = dynamic(() => import("@/components/charts/multi-location-chart").then((m) => m.MultiLocationChart), { ssr: false });
import { ErrorState } from "@/components/common/error-state";
import { MonitorActions } from "@/components/monitors/monitor-actions";
import { MonitorStatusBadge } from "@/components/monitors/monitor-status-badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { GenericMonitorSummary } from "@/features/monitors/detail/generic-summary";
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
        <Skeleton className="h-52 rounded-xl" />
      </div>
    );
  }

  if (monitorQuery.isError) return <ErrorState onRetry={() => void monitorQuery.refetch()} />;

  const monitor = monitorQuery.data;
  const Summary = (getMonitorDefinition(monitor.type).Summary ?? GenericMonitorSummary);
  const recentResults = recentResultsQuery.data?.items ?? [];
  const probeLocations = locationsQuery.data?.items ?? [];

  return (
    <div className="flex min-w-0 flex-col">
      <header className="flex flex-wrap items-center justify-between gap-4 py-3.5">
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

      <NodeMonitorTabs currentMonitor={monitor} />

      <div className="mb-4 border-b border-border/60" />

      <Summary
        monitor={monitor}
        metrics={metricsQuery.data}
        latestResult={recentResults[0] ?? null}
        recentResults={recentResults}
        probeLocations={probeLocations}
        locale={locale}
        rangeLabel=""
      />

      <div className="mt-4 grid min-w-0 gap-4 xl:grid-cols-[1fr_360px]">
        <Card className="overflow-hidden border-border/65 py-0 shadow-none">
          <CardContent className="px-4 py-3">
            {metricsQuery.isPending ? <Skeleton className="h-52 w-full rounded-lg" /> : metricsQuery.isError ? <ErrorState onRetry={() => void metricsQuery.refetch()} /> : (
              <MultiLocationChart
                series={probeLocations.map((loc) => ({
                  location: loc,
                  results: recentResults.filter((r) => (r.probe_location_id || "default") === loc.id),
                }))}
              />
            )}
          </CardContent>
        </Card>

        <div className="flex flex-col gap-2 rounded-xl border border-border/65 bg-card/50 px-4 py-3">
          <p className="text-xs font-medium text-muted-foreground">بازه زمانی</p>
          <div className="flex gap-1">
            {(["24h", "7d", "30d"] as const).map((r) => (
              <button
                key={r}
                type="button"
                onClick={() => setRange(r)}
                className={`rounded-md px-3 py-1 text-xs font-medium transition-colors ${range === r ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground hover:bg-muted/80"}`}
              >
                {r}
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
