"use client";

import { useTranslations } from "next-intl";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { RelativeTime } from "@/shared/ui/relative-time";
import type { MonitorConfigurationProps } from "@/plugins/monitoring/core/definition";
import { formatDuration } from "@/shared/utils/formatters";
import type { ProbeResult } from "@/entities/monitor/model/result";

function latestForLocation(locationId: string, results: ProbeResult[]): ProbeResult | undefined {
  return results.find((r) => (r.probe_location_id || "default") === locationId);
}

export function HttpMonitorConfiguration({ monitor, latestResult, recentResults, probeLocations, locale }: MonitorConfigurationProps) {
  const t = useTranslations("monitorDetail");
  const statusCode =
    typeof latestResult?.attributes?.status_code === "number"
      ? latestResult.attributes.status_code
      : null;
  const method =
    typeof monitor.config.method === "string"
      ? monitor.config.method.toUpperCase()
      : "GET";

  const configRows = [
    [t("method"), method],
    [t("statusCode"), statusCode != null ? String(statusCode) : "—"],
    [t("expectedStatus"), typeof monitor.config.expected_status_codes === "string" ? monitor.config.expected_status_codes : "200"],
    [t("followRedirects"), monitor.config.follow_redirects === false ? t("no") : t("yes")],
    [t("verifyTls"), monitor.config.verify_tls === false ? t("no") : t("yes")],
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
            const code =
              result && typeof result.attributes?.status_code === "number"
                ? result.attributes.status_code
                : null;
            const rtt =
              result && typeof result.metrics.response_time_ms === "number"
                ? result.metrics.response_time_ms
                : null;
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
                      <p className="text-sm font-semibold tabular-nums" dir="ltr">{code != null ? String(code) : "—"}</p>
                      <p className="text-[10px] text-muted-foreground">{t("statusCode")}</p>
                    </>
                  ) : (
                    <>
                      <p className="text-sm text-muted-foreground/50" dir="ltr">—</p>
                      <p className="text-[10px] text-muted-foreground">{t("statusCode")}</p>
                    </>
                  )}
                </div>
                <div className="w-16 text-end">
                  {result ? (
                    <p className={result.success ? "text-sm font-semibold text-success" : "text-sm font-semibold text-destructive"} dir="ltr">
                      {rtt != null ? formatDuration(rtt, locale) : "—"}
                    </p>
                  ) : (
                    <p className="text-sm text-muted-foreground/50" dir="ltr">—</p>
                  )}
                  <p className="text-[10px] text-muted-foreground">{t("currentLatency")}</p>
                </div>
              </div>
            );
          }) : <p className="py-6 text-center text-xs text-muted-foreground">{t("noLocationResults")}</p>}
        </CardContent>
      </Card>

      <Card className="border-border/70 bg-card/60 shadow-none">
        <CardHeader><CardTitle className="text-sm">{t("httpConfiguration")}</CardTitle></CardHeader>
        <CardContent className="grid gap-x-6 sm:grid-cols-2 xl:grid-cols-1">
          {configRows.map(([label, value]) => (
            <div key={label} className="flex items-center justify-between gap-4 border-b border-border/50 py-2.5">
              <span className="text-xs text-muted-foreground">{label}</span>
              <span className="truncate text-xs font-medium tabular-nums" dir="ltr">{String(value)}</span>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
