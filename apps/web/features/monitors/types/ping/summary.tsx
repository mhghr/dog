"use client";

import { Activity, Radio, Waves } from "lucide-react";
import { useTranslations } from "next-intl";

import { Card, CardContent } from "@/components/ui/card";
import type { MonitorSummaryProps } from "@/features/monitors/core/definition";
import { formatDuration } from "@/lib/formatters";
import type { ProbeResult } from "@/types/result";

function metric(result: ProbeResult, key: string): number | null {
  const value = result.metrics?.[key];
  return typeof value === "number" ? value : null;
}

interface LocationResult {
  id: string;
  name: string;
  code?: string;
  result: ProbeResult;
}

function LocationMetricCard({
  icon: Icon,
  title,
  locations,
  value,
}: {
  icon: typeof Activity;
  title: string;
  locations: LocationResult[];
  value: (result: ProbeResult) => string;
}) {
  const t = useTranslations("monitorDetail");

  return (
    <Card className="min-w-0 border-border/70 bg-card/60 shadow-none">
      <CardContent className="p-3">
        <div className="flex items-center gap-2 border-b border-border/60 pb-2.5 text-xs font-medium">
          <Icon className="size-3.5 text-primary" aria-hidden />
          <span>{title}</span>
        </div>
        <div className="pt-1">
          {locations.map((location) => (
            <div key={location.id} className="flex min-w-0 items-center justify-between gap-4 border-b border-border/40 py-2 last:border-0">
              <div className="flex min-w-0 items-center gap-2">
                <span className={location.result.success ? "size-1.5 shrink-0 rounded-full bg-success" : "size-1.5 shrink-0 rounded-full bg-destructive"} />
                <span className="truncate text-xs text-muted-foreground">{location.name}</span>
                {location.code ? <span className="shrink-0 font-mono text-[9px] text-muted-foreground/70">{location.code}</span> : null}
              </div>
              <span className="shrink-0 text-sm font-semibold tabular-nums" dir="ltr">{value(location.result)}</span>
            </div>
          ))}
          {locations.length === 0 ? <p className="py-4 text-center text-xs text-muted-foreground">{t("noLocationResults")}</p> : null}
        </div>
      </CardContent>
    </Card>
  );
}

export function PingMonitorSummary({ recentResults, probeLocations, locale }: MonitorSummaryProps) {
  const t = useTranslations("monitorDetail");
  const locationById = new Map(probeLocations.map((location) => [location.id, location]));
  const latestByLocation = new Map<string, ProbeResult>();

  for (const result of recentResults) {
    const id = result.probe_location_id || "default";
    if (!latestByLocation.has(id)) latestByLocation.set(id, result);
  }

  const locations: LocationResult[] = [...latestByLocation.entries()].map(([id, result]) => {
    const location = locationById.get(id);
    return { id, result, name: location?.name ?? t("unknownLocation"), code: location?.code };
  });

  return (
    <section className="grid min-w-0 gap-3 md:grid-cols-3">
      <LocationMetricCard
        icon={Activity}
        title={t("rttRange")}
        locations={locations}
        value={(result) => formatDuration(metric(result, "avg_rtt_ms") ?? result.duration_millis, locale)}
      />
      <LocationMetricCard
        icon={Radio}
        title={t("packetLoss")}
        locations={locations}
        value={(result) => {
          const loss = metric(result, "packet_loss_percent");
          return loss != null ? `${loss.toFixed(1)}%` : "—";
        }}
      />
      <LocationMetricCard
        icon={Waves}
        title={t("jitter")}
        locations={locations}
        value={(result) => formatDuration(metric(result, "stddev_rtt_ms"), locale)}
      />
    </section>
  );
}
