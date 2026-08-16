"use client";

import { useLocale } from "next-intl";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
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

const CELL = "shrink-0 tabular-nums text-[13px]";

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
      className="border-border/70 bg-white/70 shadow-[var(--shadow-panel)] backdrop-blur-xl dark:bg-white/[0.03]"
    >
      <CardHeader className="px-5 pt-4">
        <CardTitle className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
          {isFa ? "موقعیت‌های پراب" : "Probe locations"}
        </CardTitle>
      </CardHeader>
      <CardContent className="px-5 pb-4">
        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-11 w-full rounded-lg" />
            ))}
          </div>
        ) : stats.length === 0 ? (
          <p className="py-4 text-sm text-muted-foreground">
            {isFa ? "هیچ پرابی داده ندارد" : "No probe has recent data"}
          </p>
        ) : (
          <div className="flex flex-col">
            {stats.map((stat) => {
              const state = locationHealth(stat, thresholds);
              const label = STATE_LABEL[state];
              return (
                <div
                  key={stat.probeId}
                  className="flex flex-wrap items-center justify-between gap-3 border-b border-border/50 py-2.5 transition-colors last:border-0 hover:bg-muted/30"
                >
                  <div className="flex min-w-0 items-center gap-2.5">
                    <span className="min-w-0 truncate text-sm font-medium" dir="auto">
                      {stat.location}
                    </span>
                    <StatusBadge tone={pingHealthTone(state)} label={isFa ? label.fa : label.en} />
                  </div>
                  <div className="flex shrink-0 items-center gap-4">
                    <span className={cn(CELL, "text-muted-foreground")} dir="ltr">
                      {stat.latency == null ? "—" : `${Math.round(stat.latency)} ms`}
                    </span>
                    <span
                      className={cn(CELL, "text-muted-foreground", stat.packetLoss != null && stat.packetLoss > 0 && "text-amber-600 dark:text-amber-400")}
                      dir="ltr"
                    >
                      {stat.packetLoss == null ? "—" : `${stat.packetLoss.toFixed(1)}% loss`}
                    </span>
                    <span className={cn(CELL, "hidden text-muted-foreground sm:inline")} dir="ltr">
                      {stat.jitter == null ? "—" : `${Math.round(stat.jitter)} ms`}
                    </span>
                    <span className="hidden text-xs text-muted-foreground md:inline">
                      {formatRelativeTime(stat.lastCheckedAt, locale)}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
