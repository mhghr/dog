"use client";

import { Card, CardContent } from "@/shared/ui/card";
import { cn } from "@/shared/utils/cn";
import { Sparkline, type SparklineSeries } from "@/shared/ui/charts/sparkline";
import type { ProbeHealth } from "./http-metrics";

const STATE_TEXT: Record<ProbeHealth, string> = {
  healthy: "text-success",
  warning: "text-warning",
  critical: "text-destructive",
  down: "text-destructive",
  unknown: "text-muted-foreground",
};

const STATE_GLOW: Record<ProbeHealth, string> = {
  healthy: "dark:neon-glow-current dark:text-success",
  warning: "dark:neon-glow-current dark:text-warning",
  critical: "dark:neon-glow-current dark:text-destructive",
  down: "dark:neon-glow-current dark:text-destructive",
  unknown: "dark:text-muted-foreground",
};

export interface HttpKpiLegendRow {
  name: string;
  value: string;
}

interface HttpKpiCardProps {
  label: string;
  value: string;
  unit?: string;
  state: ProbeHealth;
  /** One sparkline series per probe — drawn exactly like the Ping KPI cards. */
  spark?: SparklineSeries[];
  colors?: string[];
  sparkLabel?: string;
  /** Per-probe rows: location name + current value, listed beside the sparkline. */
  rows?: HttpKpiLegendRow[];
  /** Optional sub-rows below the value (e.g. min/max/p95). */
  meta?: Array<{ label: string; value: string }>;
}

// HTTP parameter card mirroring the Ping KPI card: label with a status dot and
// the value placed in front of the title, plus a per-probe sparkline and a
// per-probe value legend.
export function HttpKpiCard({
  label,
  value,
  unit,
  state,
  spark,
  colors,
  sparkLabel,
  rows,
  meta,
}: HttpKpiCardProps) {
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
      <span
        aria-hidden
        className={cn(
          "pointer-events-none absolute -top-10 right-0 size-36 rounded-full bg-current opacity-[0.07] blur-3xl dark:opacity-[0.12]",
          stateText,
        )}
      />

      <CardContent className="relative flex h-full flex-col gap-2.5 p-3.5">
        <div className="flex items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-1.5">
            <span
              className={cn("size-1.5 shrink-0 rounded-full bg-current", stateText, stateGlow)}
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
            {unit ? <span className="ms-1 text-xs font-medium text-muted-foreground">{unit}</span> : null}
          </p>
        </div>

        {spark && spark.length > 0 && (
          <div className="flex items-end justify-end gap-4">
            <Sparkline
              series={spark}
              colors={colors ?? []}
              height={36}
              lineWidth={2}
              fillOpacity={0.4}
              className="-mx-0.5 h-9 w-2/5 min-w-0"
              ariaLabel={sparkLabel}
            />
            {rows && rows.length > 0 && (
              <ul className="flex shrink-0 flex-col items-end gap-1.5">
                {rows.map((row) => (
                  <li key={row.name} className="truncate text-[10px] leading-none text-muted-foreground">
                    {row.name}
                    <span className="ms-1.5 font-semibold tabular-nums text-foreground" dir="ltr">
                      {row.value}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}

        {meta && meta.length > 0 && (
          <dl className="mt-auto flex flex-wrap gap-x-4 gap-y-1 border-t border-border/40 pt-2">
            {meta.map((row) => (
              <div key={row.label} className="flex items-center gap-1.5 text-[10px]">
                <dt className="text-muted-foreground">{row.label}</dt>
                <dd className="font-semibold tabular-nums text-foreground" dir="ltr">
                  {row.value}
                </dd>
              </div>
            ))}
          </dl>
        )}
      </CardContent>
    </Card>
  );
}
