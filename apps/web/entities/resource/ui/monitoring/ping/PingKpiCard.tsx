"use client";

import { Card, CardContent } from "@/shared/ui/card";
import { cn } from "@/shared/utils/cn";
import type { PingHealthState } from "./ping-health";

export interface PingKpiRow {
  label: string;
  value: string;
  tone?: "success" | "warning" | "destructive" | "muted";
}

const STATE_ACCENT: Record<PingHealthState, string> = {
  healthy: "from-emerald-400/70 to-teal-400/0",
  warning: "from-amber-400/70 to-amber-400/0",
  critical: "from-rose-400/70 to-rose-400/0",
  down: "from-rose-500/80 to-rose-500/0",
  unknown: "from-zinc-300/70 to-zinc-300/0",
};

const STATE_TEXT: Record<PingHealthState, string> = {
  healthy: "text-emerald-600 dark:text-emerald-400",
  warning: "text-amber-600 dark:text-amber-400",
  critical: "text-rose-600 dark:text-rose-400",
  down: "text-rose-600 dark:text-rose-400",
  unknown: "text-muted-foreground",
};

const ROW_TONE_TEXT: Record<string, string> = {
  success: "text-emerald-600 dark:text-emerald-400",
  warning: "text-amber-600 dark:text-amber-400",
  destructive: "text-rose-600 dark:text-rose-400",
  muted: "text-foreground/70",
};

export function PingKpiCard({
  label,
  value,
  unit,
  state,
  rows,
}: {
  label: string;
  value: string;
  unit?: string;
  state: PingHealthState;
  rows: PingKpiRow[];
}) {
  return (
    <Card
      variant="bordered"
      className="group relative h-full overflow-hidden border border-border/40 bg-white/70 shadow-[var(--shadow-panel)] backdrop-blur-xl transition-all duration-200 hover:border-border/60 hover:shadow-[0_4px_20px_rgba(0,0,0,0.06)] dark:bg-white/[0.03] dark:hover:bg-white/[0.05]"
    >
      {/* health accent line */}
      <span
        aria-hidden
        className={cn(
          "pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r",
          STATE_ACCENT[state],
        )}
      />

      <CardContent className="flex h-full flex-col gap-1.5 p-3">
        <div className="flex items-center justify-between gap-2">
          <p className="truncate text-[10px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
            {label}
          </p>
          <span
            className={cn("size-1.5 shrink-0 rounded-full", STATE_TEXT[state], "bg-current")}
            aria-hidden
          />
        </div>

        <p className={cn("text-lg font-semibold tracking-tight tabular-nums", STATE_TEXT[state])} dir="ltr">
          {value}
          {unit ? (
            <span className="ms-1 text-xs font-medium text-muted-foreground">{unit}</span>
          ) : null}
        </p>

        {rows.length > 0 && (
          <dl className="mt-0.5 space-y-0.5 border-t border-border/50 pt-2">
            {rows.map((row) => (
              <div
                key={row.label}
                className="flex items-center justify-between gap-2 text-[11px] leading-tight"
              >
                <dt className="truncate text-muted-foreground" dir="auto">
                  {row.label}
                </dt>
                <dd
                  dir="ltr"
                  className={cn(
                    "shrink-0 font-medium tabular-nums",
                    ROW_TONE_TEXT[row.tone ?? "muted"],
                  )}
                >
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
