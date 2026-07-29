"use client";

import { Activity, Radio, Waves } from "lucide-react";
import { useTranslations } from "next-intl";

import { Card, CardContent } from "@/components/ui/card";
import type { MonitorSummaryProps } from "@/features/monitors/core/definition";
import { formatDuration } from "@/lib/formatters";
import type { ProbeResult } from "@/types/result";
import type { ProbeLocation } from "@/types/monitor";

function metric(result: ProbeResult | undefined, key: string): number | null {
  if (!result) return null;
  const value = result.metrics?.[key];
  return typeof value === "number" ? value : null;
}

interface LocationRowProps {
  location: ProbeLocation;
  result: ProbeResult | undefined;
  value: (result: ProbeResult) => string;
}
function LocationRow({ location, result, value }: LocationRowProps) {
  return (
    <div className="flex min-w-0 items-center justify-between gap-4 border-b border-border/40 py-2 last:border-0">
      <div className="flex min-w-0 items-center gap-2">
        <span className={result ? (result.success ? "size-1.5 shrink-0 rounded-full bg-success" : "size-1.5 shrink-0 rounded-full bg-destructive") : "size-1.5 shrink-0 rounded-full bg-muted-foreground/30"} />
        <span className="truncate text-xs text-muted-foreground">{location.name}</span>
        {location.code ? <span className="shrink-0 font-mono text-[9px] text-muted-foreground/70">{location.code}</span> : null}
      </div>
      <span className="shrink-0 text-sm font-semibold tabular-nums" dir="ltr">
        {result ? value(result) : "—"}
      </span>
    </div>
  );
}

export function PingMonitorSummary({ recentResults, probeLocations, locale }: MonitorSummaryProps) {
  const t = useTranslations("monitorDetail");

  const latestByLocation = new Map<string, ProbeResult>();
  for (const result of recentResults) {
    const id = result.probe_location_id || "default";
    if (!latestByLocation.has(id)) latestByLocation.set(id, result);
  }

  return (
    <section className="grid min-w-0 gap-3 md:grid-cols-3">
      <Card className="min-w-0 border-border/70 bg-card/60 shadow-none">
        <CardContent className="p-3">
          <div className="flex items-center gap-2 border-b border-border/60 pb-2.5 text-xs font-medium">
            <Activity className="size-3.5 text-primary" aria-hidden />
            <span>{t("rttRange")}</span>
          </div>
          <div className="pt-1">
            {probeLocations.map((location) => (
              <LocationRow
                key={location.id}
                location={location}
                result={latestByLocation.get(location.id)}
                value={(result) => formatDuration(metric(result, "avg_rtt_ms") ?? result.duration_millis, locale)}
              />
            ))}
            {probeLocations.length === 0 ? <p className="py-4 text-center text-xs text-muted-foreground">{t("noLocationResults")}</p> : null}
          </div>
        </CardContent>
      </Card>

      <Card className="min-w-0 border-border/70 bg-card/60 shadow-none">
        <CardContent className="p-3">
          <div className="flex items-center gap-2 border-b border-border/60 pb-2.5 text-xs font-medium">
            <Radio className="size-3.5 text-primary" aria-hidden />
            <span>{t("packetLoss")}</span>
          </div>
          <div className="pt-1">
            {probeLocations.map((location) => (
              <LocationRow
                key={location.id}
                location={location}
                result={latestByLocation.get(location.id)}
                value={(result) => {
                  const loss = metric(result, "packet_loss_percent");
                  return loss != null ? `${loss.toFixed(1)}%` : "—";
                }}
              />
            ))}
            {probeLocations.length === 0 ? <p className="py-4 text-center text-xs text-muted-foreground">{t("noLocationResults")}</p> : null}
          </div>
        </CardContent>
      </Card>

      <Card className="min-w-0 border-border/70 bg-card/60 shadow-none">
        <CardContent className="p-3">
          <div className="flex items-center gap-2 border-b border-border/60 pb-2.5 text-xs font-medium">
            <Waves className="size-3.5 text-primary" aria-hidden />
            <span>{t("jitter")}</span>
          </div>
          <div className="pt-1">
            {probeLocations.map((location) => (
              <LocationRow
                key={location.id}
                location={location}
                result={latestByLocation.get(location.id)}
                value={(result) => formatDuration(metric(result, "stddev_rtt_ms"), locale)}
              />
            ))}
            {probeLocations.length === 0 ? <p className="py-4 text-center text-xs text-muted-foreground">{t("noLocationResults")}</p> : null}
          </div>
        </CardContent>
      </Card>
    </section>
  );
}
