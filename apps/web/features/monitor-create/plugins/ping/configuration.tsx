"use client";

import { useTranslations } from "next-intl";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { RelativeTime } from "@/components/common/relative-time";
import type { MonitorConfigurationProps } from "@/features/monitors/core/definition";
import { formatDuration } from "@/lib/formatters";
import type { ProbeResult } from "@/types/result";

function latestForLocation(locationId: string, results: ProbeResult[]): ProbeResult | undefined {
  return results.find((r) => (r.probe_location_id || "default") === locationId);
}

export function PingMonitorConfiguration({ monitor, latestResult, recentResults, probeLocations, locale }: MonitorConfigurationProps) {
  const t = useTranslations("monitorDetail");
  const sent = latestResult?.attributes?.packets_sent;
  const received = latestResult?.attributes?.packets_received;
  const resolvedIp = latestResult?.attributes?.resolved_ip;
  const packetCount = monitor.config.packet_count;
  const packetInterval = monitor.config.packet_interval_millis;

  const configRows = [
    [t("resolvedIp"), typeof resolvedIp === "string" ? resolvedIp : "—"],
    [t("packetsSent"), typeof sent === "number" ? String(sent) : "—"],
    [t("packetsReceived"), typeof received === "number" ? String(received) : "—"],
    [t("packetCount"), typeof packetCount === "number" ? String(packetCount) : "4"],
    [t("packetInterval"), typeof packetInterval === "number" ? formatDuration(packetInterval, locale) : "200 ms"],
    [t("timeout"), formatDuration(monitor.timeout_millis, locale)],
  ] as const;

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(300px,.65fr)]">
      <Card className="border-border/70 bg-card/60 shadow-none">
        <CardHeader><CardTitle className="text-sm">{t("locationResults")}</CardTitle></CardHeader>
        <CardContent className="space-y-1">
          {probeLocations.length > 0 ? probeLocations.map((location) => {
            const locationId = location.id;
            const result = latestForLocation(locationId, recentResults);
            const loss = result && typeof result.metrics.packet_loss_percent === "number" ? result.metrics.packet_loss_percent : null;
            return (
              <div key={locationId} className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-4 border-b border-border/50 py-3 last:border-0">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <span className={result ? (result.success ? "size-2 rounded-full bg-success" : "size-2 rounded-full bg-destructive") : "size-2 rounded-full bg-muted-foreground/30"} />
                    <span className="truncate text-sm font-medium">{location.name}</span>
                    {location.code ? <span className="rounded bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground">{location.code}</span> : null}
                  </div>
                  {result ? (
                    <p className="mt-1 ps-4 text-[11px] text-muted-foreground"><RelativeTime value={result.started_at} /></p>
                  ) : (
                    <p className="mt-1 ps-4 text-[11px] text-muted-foreground">{"—"}</p>
                  )}
                </div>
                <div className="text-end">
                  {result ? (
                    <>
                      <p className="text-sm font-semibold tabular-nums" dir="ltr">{formatDuration(result.duration_millis, locale)}</p>
                      <p className="text-[10px] text-muted-foreground">RTT</p>
                    </>
                  ) : (
                    <>
                      <p className="text-sm text-muted-foreground/50" dir="ltr">—</p>
                      <p className="text-[10px] text-muted-foreground">RTT</p>
                    </>
                  )}
                </div>
                <div className="w-16 text-end">
                  {result ? (
                    <p className={result.success ? "text-sm font-semibold text-success" : "text-sm font-semibold text-destructive"} dir="ltr">{loss != null ? `${loss.toFixed(1)}%` : "—"}</p>
                  ) : (
                    <p className="text-sm text-muted-foreground/50" dir="ltr">—</p>
                  )}
                  <p className="text-[10px] text-muted-foreground">{t("packetLoss")}</p>
                </div>
              </div>
            );
          }) : <p className="py-6 text-center text-xs text-muted-foreground">{t("noLocationResults")}</p>}
        </CardContent>
      </Card>

      <Card className="border-border/70 bg-card/60 shadow-none">
        <CardHeader><CardTitle className="text-sm">{t("pingConfiguration")}</CardTitle></CardHeader>
        <CardContent className="grid gap-x-6 sm:grid-cols-2 xl:grid-cols-1">
          {configRows.map(([label, value]) => (
            <div key={label} className="flex items-center justify-between gap-4 border-b border-border/50 py-2.5">
              <span className="text-xs text-muted-foreground">{label}</span>
              <span className="truncate text-xs font-medium tabular-nums" dir="ltr">{value}</span>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
