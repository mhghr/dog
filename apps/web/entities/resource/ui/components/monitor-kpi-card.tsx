"use client";

import { Card, CardContent } from "@/shared/ui/card";
import { Sparkline, type SparklineSeries } from "@/shared/ui/charts/sparkline";
import { cn } from "@/shared/utils/cn";

export type KpiHealth = "healthy" | "warning" | "critical" | "unknown";

const HEALTH_DOT: Record<KpiHealth, string> = {
  healthy: "bg-emerald-500 shadow-[0_0_8px_1px_rgba(16,185,129,0.5)]",
  warning: "bg-amber-500 shadow-[0_0_8px_1px_rgba(245,158,11,0.5)]",
  critical: "bg-rose-500 shadow-[0_0_8px_1px_rgba(244,63,94,0.5)]",
  unknown: "bg-muted-foreground/50",
};

// The metric value is colored by health; the card border stays neutral so the
// health state reads through the value, not the card chrome.
const VALUE_TEXT: Record<KpiHealth, string> = {
  healthy: "text-success dark:neon-text-current",
  warning: "text-warning dark:neon-text-current",
  critical: "text-destructive dark:neon-text-current",
  unknown: "text-foreground",
};

const REASON_TEXT: Record<KpiHealth, string> = {
  healthy: "text-muted-foreground",
  warning: "text-warning",
  critical: "text-destructive",
  unknown: "text-muted-foreground",
};

export interface MonitorKpiCardProps {
  title: string;
  /** Always the latest real value from the most recent check. */
  value: string;
  unit?: string;
  /** Aggregation computed over the selected time range: value left, label right. */
  secondary?: { label?: string; value: string };
  /** Trend for the selected time range (one series per probe). */
  spark?: SparklineSeries[];
  health: KpiHealth;
  /** Shown when the card is warning/critical so the problem is visible at once. */
  reason?: string;
}

// Unified KPI card used at the top of every monitor section. The metric value
// sits in front of the title and is the only element colored by health; the
// card border stays neutral and the mini sparkline shows the selected range.
export function MonitorKpiCard({
  title,
  value,
  unit,
  secondary,
  spark,
  health,
  reason,
}: MonitorKpiCardProps) {
  const notHealthy = health === "warning" || health === "critical";

  return (
    <Card
      variant="bordered"
      className={cn(
        "group relative h-full overflow-hidden bg-card shadow-subtle transition-[border-color,box-shadow] duration-300",
        "hover:border-border/70 hover:shadow-card-hover dark:hover:border-primary/40 dark:hover:shadow-glow",
      )}
    >
      <CardContent className="flex h-full min-h-40 flex-col p-3.5">
        {/* Top row: metric name + value in front of the title */}
        <div className="flex items-start justify-between gap-2">
          <p className="min-w-0 flex-1 truncate pt-1 text-[10px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
            {title}
          </p>
          <p className={cn("shrink-0 text-xl font-bold leading-none tracking-tight tabular-nums", VALUE_TEXT[health])} dir="ltr">
            {value}
            {unit ? <span className="ms-1 text-[11px] font-medium text-muted-foreground">{unit}</span> : null}
          </p>
        </div>

        {/* Secondary aggregation: number on the left, label on the right. Forced
            LTR so the value stays on the left even in RTL layouts. */}
        {secondary ? (
          <div dir="ltr" className="mt-2 flex items-center justify-between gap-2">
            <span className="truncate text-[11px] tabular-nums text-muted-foreground">
              {secondary.value}
            </span>
            {secondary.label ? (
              <span dir="auto" className="truncate text-[11px] text-muted-foreground/80">
                {secondary.label}
              </span>
            ) : null}
          </div>
        ) : null}

        {/* Reason when warning/critical */}
        {notHealthy && reason ? (
          <p className={cn("mt-1 flex items-center gap-1 text-[11px] font-medium", REASON_TEXT[health])} dir="auto">
            <span className="size-1 rounded-full bg-current" aria-hidden />
            {reason}
          </p>
        ) : null}

        {/* Mini trend for the selected range */}
        <div className="mt-auto pt-2">
          {spark && spark.length > 0 ? (
            <Sparkline
              series={spark}
              colors={["#8B5CF6", "#14B8A6", "#22C55E", "#F59E0B", "#F43F5E"]}
              height={28}
              lineWidth={1.75}
              fillOpacity={0.3}
              className="h-7 w-full"
            />
          ) : (
            <div className="h-7 w-full" />
          )}
        </div>
      </CardContent>
    </Card>
  );
}
