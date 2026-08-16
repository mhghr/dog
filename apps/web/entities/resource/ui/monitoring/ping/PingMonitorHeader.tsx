"use client";

import { useLocale } from "next-intl";

import { StatusBadge } from "@/design-system/components";
import { pingHealthTone, type PingHealthState } from "./ping-health";
import { formatRelativeTime } from "@/shared/utils/formatters";
import { PingTimeRangeSelector } from "./PingTimeRangeSelector";
import type { MetricsRange } from "@/entities/resource/hooks/use-resource";

const STATE_LABEL: Record<PingHealthState, { en: string; fa: string }> = {
  healthy: { en: "Healthy", fa: "سالم" },
  warning: { en: "Warning", fa: "هشدار" },
  critical: { en: "Critical", fa: "بحرانی" },
  down: { en: "Down", fa: "قطع" },
  unknown: { en: "Unknown", fa: "نامشخص" },
};

export function PingMonitorHeader({
  lastCheckedAt,
  overallState,
  isLive,
  range,
  onRangeChange,
}: {
  lastCheckedAt: string | null | undefined;
  overallState: PingHealthState;
  isLive: boolean;
  range: MetricsRange;
  onRangeChange: (range: MetricsRange) => void;
}) {
  const locale = useLocale();
  const isFa = locale === "fa";
  const label = STATE_LABEL[overallState];

  // "Live" only makes sense when the monitor is actually healthy/operational.
  // Showing it alongside a Down/Critical status would read as a contradiction.
  const showLive = isLive && overallState !== "down" && overallState !== "critical";

  return (
    <div className="flex flex-wrap items-center gap-3">
      {showLive && (
        <span className="flex shrink-0 items-center gap-1.5 rounded-full border border-emerald-500/20 bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-emerald-600 dark:text-emerald-400">
          <span className="relative flex size-1.5">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-500 opacity-60" />
            <span className="relative inline-flex size-1.5 rounded-full bg-emerald-500" />
          </span>
          {isFa ? "زنده" : "Live"}
        </span>
      )}

      <div className="ms-auto flex shrink-0 items-center gap-3">
        <StatusBadge tone={pingHealthTone(overallState)} label={isFa ? label.fa : label.en} />
        {lastCheckedAt && (
          <span
            className="hidden text-xs tabular-nums text-muted-foreground sm:inline"
            suppressHydrationWarning
          >
            {isFa ? "آخرین بررسی: " : "Last check: "}
            {formatRelativeTime(lastCheckedAt, locale)}
          </span>
        )}
        <PingTimeRangeSelector range={range} onChange={onRangeChange} />
      </div>
    </div>
  );
}
