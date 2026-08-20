"use client";

import { useLocale } from "next-intl";

import { Card, CardContent } from "@/shared/ui/card";
import { Skeleton } from "@/shared/ui/skeleton";
import { StatusBadge } from "@/design-system/components";
import { cn } from "@/shared/utils/cn";
import { formatRelativeTime } from "@/shared/utils/formatters";
import { evaluatePingHealth, pingHealthTone, type PingHealthState } from "./ping-health";
import type { PingProbeStat } from "./ping-metrics";
import type { PingThresholds } from "./ping-config";

const STATE_LABEL: Record<PingHealthState, { en: string; fa: string }> = {
  healthy: { en: "Healthy", fa: "سالم" },
  warning: { en: "Warning", fa: "هشدار" },
  critical: { en: "Critical", fa: "بحرانی" },
  down: { en: "Down", fa: "قطع" },
  unknown: { en: "Unknown", fa: "نامشخص" },
};

function locationHealth(stat: PingProbeStat, thresholds: PingThresholds): PingHealthState {
  return evaluatePingHealth({
    lastStatus: stat.success ? "up" : "down",
    latency: stat.latency,
    packetLoss: stat.packetLoss,
    jitter: stat.jitter,
    thresholds,
  });
}

const NUM_CELL = "shrink-0 tabular-nums text-[13px] text-muted-foreground dark:text-foreground/80";

export function PingProbeLocations({
  stats,
  thresholds,
  isLoading,
}: {
  stats: PingProbeStat[];
  thresholds: PingThresholds;
  isLoading: boolean;
}) {
  const locale = useLocale();
  const isFa = locale === "fa";

  return (
    <Card
      variant="bordered"
      className="shadow-subtle transition-[border-color,box-shadow] duration-300 dark:hover:border-primary/40 dark:hover:shadow-glow"
    >
      <CardContent className="overflow-x-auto px-4 pb-4 pt-4">
        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-11 w-full rounded-lg" />
            ))}
          </div>
        ) : stats.length === 0 ? (
          <p className="py-4 text-sm text-muted-foreground">
            {isFa ? "هیچ پرابی داده اخیر ندارد" : "No probe has recent data"}
          </p>
        ) : (
          <table className="w-full min-w-[560px] border-collapse text-left">
            <thead>
              <tr className="border-b border-border/60 text-[11px] font-semibold uppercase tracking-[0.05em] text-muted-foreground">
                <th className="px-1 pb-2.5 font-semibold">{isFa ? "موقعیت" : "Location"}</th>
                <th className="px-1 pb-2.5 font-semibold">{isFa ? "وضعیت" : "Status"}</th>
                <th className="px-1 pb-2.5 text-right font-semibold" dir="ltr">
                  {isFa ? "تأخیر" : "Latency"}
                </th>
                <th className="px-1 pb-2.5 text-right font-semibold" dir="ltr">
                  {isFa ? "افت بسته" : "Packet loss"}
                </th>
                <th className="px-1 pb-2.5 text-right font-semibold" dir="ltr">
                  {isFa ? "نوسان" : "Jitter"}
                </th>
                <th className="px-1 pb-2.5 text-right font-semibold">
                  {isFa ? "آخرین بررسی" : "Last check"}
                </th>
              </tr>
            </thead>
            <tbody>
              {stats.map((stat) => {
                const state = locationHealth(stat, thresholds);
                const label = STATE_LABEL[state];
                return (
                  <tr
                    key={stat.probeId}
                    className="border-b border-border/40 transition-colors last:border-0 hover:bg-muted/30"
                  >
                    <td className="px-1 py-2.5 text-sm font-medium" dir="auto">
                      {stat.location}
                    </td>
                    <td className="px-1 py-2.5">
                      <StatusBadge tone={pingHealthTone(state)} label={isFa ? label.fa : label.en} />
                    </td>
                    <td className={cn(NUM_CELL, "text-right", state === "healthy" && "dark:neon-text-current dark:text-emerald-300")} dir="ltr">
                      {stat.latency == null ? "—" : `${Math.round(stat.latency)} ms`}
                    </td>
                    <td
                      className={cn(
                        NUM_CELL,
                        "text-right",
                        stat.packetLoss != null && stat.packetLoss > 0 && "text-warning dark:text-warning",
                      )}
                      dir="ltr"
                    >
                      {stat.packetLoss == null ? "—" : `${stat.packetLoss.toFixed(1)}%`}
                    </td>
                    <td className={cn(NUM_CELL, "text-right")} dir="ltr">
                      {stat.jitter == null ? "—" : `${Math.round(stat.jitter)} ms`}
                    </td>
                    <td className="px-1 py-2.5 text-right text-xs text-muted-foreground">
                      {formatRelativeTime(stat.lastCheckedAt, locale)}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </CardContent>
    </Card>
  );
}
