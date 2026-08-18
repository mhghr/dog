"use client";

import { Card, CardContent } from "@/shared/ui/card";
import { cn } from "@/shared/utils/cn";
import { Sparkline, type SparklineSeries } from "@/shared/ui/charts/sparkline";
import type { PingHealthState } from "./ping-health";

// Neon status mapping — every state carries its own glow. The stripe uses
// `bg-current` + `neon-glow-current` so the glow follows the state color.
const STATE_TEXT: Record<PingHealthState, string> = {
  healthy: "text-success",
  warning: "text-warning",
  critical: "text-destructive",
  down: "text-destructive",
  unknown: "text-muted-foreground",
};

const STATE_GLOW: Record<PingHealthState, string> = {
  healthy: "dark:neon-glow-current dark:text-success",
  warning: "dark:neon-glow-current dark:text-warning",
  critical: "dark:neon-glow-current dark:text-destructive",
  down: "dark:neon-glow-current dark:text-destructive",
  unknown: "dark:text-muted-foreground",
};

export function PingKpiCard({
  label,
  value,
  unit,
  state,
  spark,
  colors,
  sparkLabel,
}: {
  label: string;
  value: string;
  unit?: string;
  state: PingHealthState;
  spark?: SparklineSeries[];
  colors?: string[];
  sparkLabel?: string;
}) {
  const stateText = STATE_TEXT[state];
  const stateGlow = STATE_GLOW[state];

  return (
    <Card
      variant="bordered"
      className={cn(
        "group relative h-full overflow-hidden bg-card shadow-subtle transition-[border-color,box-shadow] duration-300",
        "hover:border-border/70 hover:shadow-card-hover dark:hover:border-primary/50 dark:hover:shadow-glow",
      )}
    >
      {/* faint state-tinted radial bloom behind the value */}
      <span
        aria-hidden
        className={cn(
          "pointer-events-none absolute -top-10 right-0 size-36 rounded-full bg-current opacity-[0.07] blur-3xl dark:opacity-[0.12]",
          stateText,
        )}
      />

      <CardContent className="relative flex h-full flex-col gap-3 p-3.5">
        <div className="flex items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-1.5">
            <span
              className={cn(
                "size-1.5 shrink-0 rounded-full bg-current",
                stateText,
                stateGlow,
              )}
              aria-hidden
            />
            <p className="truncate text-[10px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
              {label}
            </p>
          </div>

          <p
            className={cn(
              "shrink-0 text-3xl font-bold leading-none tracking-tight tabular-nums",
              stateText,
              state === "unknown" ? "" : "dark:neon-text-current",
            )}
            dir="ltr"
          >
            {value}
            {unit ? (
              <span className="ms-1 text-xs font-medium text-muted-foreground">{unit}</span>
            ) : null}
          </p>
        </div>

        {spark && spark.length > 0 && (
          <div className="mt-auto pt-1">
            <Sparkline
              series={spark}
              colors={colors ?? []}
              height={40}
              className="-mx-0.5 h-10"
              ariaLabel={sparkLabel}
            />
          </div>
        )}
      </CardContent>
    </Card>
  );
}
