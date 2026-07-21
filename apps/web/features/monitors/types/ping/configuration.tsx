"use client";

import { useTranslations } from "next-intl";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { RelativeTime } from "@/components/common/relative-time";
import type { MonitorConfigurationProps } from "@/features/monitors/core/definition";
import { formatDuration } from "@/lib/formatters";

function attribute(result: MonitorConfigurationProps["latestResult"], key: string) {
  return result?.attributes?.[key];
}

export function PingMonitorConfiguration({ monitor, latestResult, recentResults, probeLocations, locale }: MonitorConfigurationProps) {
  const t = useTranslations("monitorDetail");
  const sent = attribute(latestResult, "packets_sent");
  const received = attribute(latestResult, "packets_received");
  const resolvedIp = attribute(latestResult, "resolved_ip");
  const packetCount = monitor.config.packet_count;
  const packetInterval = monitor.config.packet_interval_millis;

  const rows = [
    [t("resolvedIp"), typeof resolvedIp === "string" ? resolvedIp : "—"],
    [t("packetsSent"), typeof sent === "number" ? String(sent) : "—"],
    [t("packetsReceived"), typeof received === "number" ? String(received) : "—"],
    [t("packetCount"), typeof packetCount === "number" ? String(packetCount) : "4"],
    [t("packetInterval"), typeof packetInterval === "number" ? formatDuration(packetInterval, locale) : "200 ms"],
    [t("timeout"), formatDuration(monitor.timeout_millis, locale)],
  ] as const;

  const locationById = new Map(probeLocations.map((location) => [location.id, location]));
  const latestByLocation = new Map<string, (typeof recentResults)[number]>();
  for (const result of recentResults) {
    const key = result.probe_location_id || "default";
    if (!latestByLocation.has(key)) latestByLocation.set(key, result);
  }

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(300px,.65fr)]">
      <Card className="border-border/70 bg-card/60 shadow-none">
        <CardHeader><CardTitle className="text-sm">{t("locationResults")}</CardTitle></CardHeader>
        <CardContent className="space-y-1">
          {[...latestByLocation.entries()].map(([locationId, result]) => {
            const location = locationById.get(locationId);
            const loss = typeof result.metrics.packet_loss_percent === "number" ? result.metrics.packet_loss_percent : null;
            return (
              <div key={locationId} className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-4 border-b border-border/50 py-3 last:border-0">
                <div className="min-w-0"><div className="flex items-center gap-2"><span className={result.success ? "size-2 rounded-full bg-success" : "size-2 rounded-full bg-destructive"} /><span className="truncate text-sm font-medium">{location?.name ?? t("unknownLocation")}</span>{location?.code ? <span className="rounded bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground">{location.code}</span> : null}</div><p className="mt-1 ps-4 text-[11px] text-muted-foreground"><RelativeTime value={result.started_at} /></p></div>
                <div className="text-end"><p className="text-sm font-semibold tabular-nums" dir="ltr">{formatDuration(result.duration_millis, locale)}</p><p className="text-[10px] text-muted-foreground">RTT</p></div>
                <div className="w-16 text-end"><p className={result.success ? "text-sm font-semibold text-success" : "text-sm font-semibold text-destructive"} dir="ltr">{loss != null ? `${loss.toFixed(1)}%` : "—"}</p><p className="text-[10px] text-muted-foreground">{t("packetLoss")}</p></div>
              </div>
            );
          })}
          {latestByLocation.size === 0 ? <p className="py-6 text-center text-xs text-muted-foreground">{t("noLocationResults")}</p> : null}
        </CardContent>
      </Card>

      <Card className="border-border/70 bg-card/60 shadow-none">
        <CardHeader><CardTitle className="text-sm">{t("pingConfiguration")}</CardTitle></CardHeader>
        <CardContent className="grid gap-x-6 sm:grid-cols-2 xl:grid-cols-1">
          {rows.map(([label, value]) => <div key={label} className="flex items-center justify-between gap-4 border-b border-border/50 py-2.5"><span className="text-xs text-muted-foreground">{label}</span><span className="truncate text-xs font-medium tabular-nums" dir="ltr">{value}</span></div>)}
        </CardContent>
      </Card>
    </div>
  );
}
