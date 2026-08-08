"use client";

import { useState } from "react";
import { Pencil } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";

import dynamic from "next/dynamic";

const MultiLocationChart = dynamic(() => import("@/shared/ui/charts/multi-location-chart").then((m) => m.MultiLocationChart), { ssr: false });
import { ErrorState } from "@/design-system/patterns/error-state";
import { MonitorActions } from "@/features/monitor-management/ui/monitor-actions";
import { MonitorStatusBadge } from "@/features/monitor-management/ui/monitor-status-badge";
import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import { Skeleton } from "@/shared/ui/skeleton";
import { GenericMonitorSummary } from "@/entities/monitor/ui/generic-summary";
import { NodeMonitorTabs } from "@/entities/monitor/ui/node-monitor-tabs";
import { getMonitorDefinition } from "@/plugins/monitoring/core/registry";
import { useAgents } from "@/entities/agent/hooks/use-agent";
import { useMonitor } from "@/entities/monitor/hooks/use-monitor";
import { useMonitorMetrics, type MetricsRange } from "@/entities/monitor/hooks/use-monitor-metrics";
import { useMonitorResults } from "@/entities/monitor/hooks/use-monitor-results";
import { useLocations } from "@/entities/probe/hooks/use-location";
import { Link } from "@/i18n/navigation";
import { useConsoleBase } from "@/widgets/console-shell/use-console-base";
import type { ProbeLocation } from "@/entities/probe/model/types";

export function MonitorDetailScreen({ monitorId }: { monitorId: string }) {
  const t = useTranslations("monitorDetail");
  const tMonitors = useTranslations("monitors");
  const locale = useLocale();
  const base = useConsoleBase();
  const [range, setRange] = useState<MetricsRange>("24h");

  const monitorQuery = useMonitor(monitorId);
  const metricsQuery = useMonitorMetrics(monitorId, range);
  const recentResultsQuery = useMonitorResults(monitorId, 50, 1);
  const locationsQuery = useLocations();
  const agentsQuery = useAgents();

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

  // Show active agents by name in the probe list instead of location names
  const activeAgents = agentsQuery.data?.items?.filter((a) => a.status === "active" || a.status === "approved") ?? [];
  const enrichedLocations: ProbeLocation[] = activeAgents.length > 0
    ? activeAgents.map((a) => ({
        id: a.location_id || a.id,
        name: a.name,
        code: "",
        enabled: true,
        created_at: "",
      }))
    : probeLocations;

  return (
    <div className="flex min-w-0 flex-col">
      <header className="flex flex-wrap items-center justify-between gap-4 py-3.5">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="max-w-full truncate text-xl font-semibold tracking-tight sm:text-2xl">{monitor.name}</h1>
            <MonitorStatusBadge status={monitor.last_status} />
            <Button variant="outline" size="icon-sm" asChild>
              <Link href={`${base}/nodes/${monitor.id}/edit`}>
                <Pencil aria-hidden />
                <span className="sr-only">{tMonitors("editMonitor")}</span>
              </Link>
            </Button>
          </div>
          <p dir="ltr" className="mt-1 max-w-full truncate text-start font-mono text-xs text-muted-foreground sm:text-sm">{monitor.target}</p>
        </div>
        <MonitorActions monitor={monitor} afterDeleteHref={`${base}/nodes`} />
      </header>

      <NodeMonitorTabs currentMonitor={monitor} />

      <div className="mb-4 border-b border-border/60" />

      <Summary
        monitor={monitor}
        metrics={metricsQuery.data}
        latestResult={recentResults[0] ?? null}
        recentResults={recentResults}
        probeLocations={enrichedLocations}
        locale={locale}
        rangeLabel=""
      />

      <div className="mt-4 grid min-w-0 gap-4 xl:grid-cols-[1fr_360px]">
        <Card className="overflow-hidden border-border/65 py-0 shadow-none">
          <CardContent className="px-4 py-3">
            {metricsQuery.isPending ? <Skeleton className="h-52 w-full rounded-lg" /> : metricsQuery.isError ? <ErrorState onRetry={() => void metricsQuery.refetch()} /> : (
              <MultiLocationChart
                series={enrichedLocations.map((loc) => ({
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
