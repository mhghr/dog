"use client";

import type { PingHealthState } from "./ping-health";
import { cn } from "@/shared/utils/cn";

const STATE_LABEL: Record<PingHealthState, { en: string; fa: string }> = {
  healthy: { en: "Healthy", fa: "سالم" },
  warning: { en: "Warning", fa: "هشدار" },
  critical: { en: "Critical", fa: "بحرانی" },
  down: { en: "Down", fa: "قطع" },
  unknown: { en: "Unknown", fa: "نامشخص" },
};

const STATE_META: Record<PingHealthState, { dot: string; text: string; ring: string; icon: string }> = {
  healthy: {
    dot: "bg-emerald-500",
    text: "text-emerald-700 dark:text-emerald-300",
    ring: "border-emerald-500/20 bg-emerald-500/[0.06] dark:shadow-[0_0_16px_-4px_var(--success)]",
    icon: "text-emerald-500",
  },
  warning: {
    dot: "bg-amber-500",
    text: "text-amber-700 dark:text-amber-300",
    ring: "border-amber-500/20 bg-amber-500/[0.06] dark:shadow-[0_0_16px_-4px_var(--warning)]",
    icon: "text-amber-500",
  },
  critical: {
    dot: "bg-rose-500",
    text: "text-rose-700 dark:text-rose-300",
    ring: "border-rose-500/20 bg-rose-500/[0.06] dark:shadow-[0_0_16px_-4px_var(--destructive)]",
    icon: "text-rose-500",
  },
  down: {
    dot: "bg-rose-500",
    text: "text-rose-700 dark:text-rose-300",
    ring: "border-rose-500/25 bg-rose-500/[0.07] dark:shadow-[0_0_20px_-4px_var(--destructive)]",
    icon: "text-rose-500",
  },
  unknown: {
    dot: "bg-muted-foreground/50",
    text: "text-muted-foreground",
    ring: "border-border/60 bg-muted/40",
    icon: "text-muted-foreground",
  },
};

export function PingStatusSummary({
  state,
  detail,
  isStale,
  isFa,
}: {
  state: PingHealthState;
  detail?: string;
  isStale: boolean;
  isFa: boolean;
}) {
  const meta = STATE_META[state];
  const label = STATE_LABEL[state];

  return (
    <div
      className={cn(
        "flex items-center gap-3 rounded-2xl border px-4 py-3",
        meta.ring,
      )}
    >
      <span
        className={cn(
          "relative flex size-2.5 shrink-0 rounded-full",
          meta.dot,
        )}
        aria-hidden
      >
        <span
          className={cn(
            "absolute inline-flex h-full w-full animate-ping rounded-full opacity-60",
            meta.dot,
          )}
        />
      </span>      <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
        <span className={cn("text-sm font-semibold", meta.text)}>
          {isFa ? label.fa : label.en}
        </span>
        {isStale && (
          <span className="rounded-md border border-warning/20 bg-warning/10 px-1.5 py-0.5 text-[11px] font-medium text-warning">
            {isFa ? "داده قدیمی" : "Stale data"}
          </span>
        )}
        {detail && (
          <span className="truncate text-xs text-muted-foreground">{detail}</span>
        )}
      </div>
    </div>
  );
}
