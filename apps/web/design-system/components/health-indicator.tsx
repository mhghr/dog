import { cn } from "@/shared/utils/cn";

export type HealthState = "healthy" | "degraded" | "down" | "unknown";

const RING_STYLES: Record<HealthState, string> = {
  healthy: "text-success",
  degraded: "text-warning",
  down: "text-destructive",
  unknown: "text-muted-foreground",
};

const LABEL_STYLES: Record<HealthState, string> = {
  healthy: "bg-success/12 text-success",
  degraded: "bg-warning/15 text-warning",
  down: "bg-destructive/12 text-destructive",
  unknown: "bg-muted text-muted-foreground",
};

export interface HealthIndicatorProps {
  state: HealthState;
  label?: string;
  className?: string;
}

// Design-system HealthIndicator: a donut-style health ring for resources and
// monitors. The ring uses the semantic state color; an optional label shows
// the human-readable state.
export function HealthIndicator({ state, label, className }: HealthIndicatorProps) {
  const pct = state === "healthy" ? 100 : state === "degraded" ? 66 : state === "down" ? 33 : 0;

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <span className={cn("relative grid size-8 place-items-center", RING_STYLES[state])}>
        <svg viewBox="0 0 36 36" className="size-8 -rotate-90" aria-hidden>
          <circle
            cx="18"
            cy="18"
            r="15.5"
            fill="none"
            strokeWidth="3"
            className="stroke-current opacity-15"
          />
          <circle
            cx="18"
            cy="18"
            r="15.5"
            fill="none"
            strokeWidth="3"
            strokeLinecap="round"
            strokeDasharray="97.4"
            strokeDashoffset={97.4 - (97.4 * pct) / 100}
            className="stroke-current"
          />
        </svg>
      </span>
      {label ? (
        <span className={cn("rounded-full px-2 py-0.5 text-xs font-medium", LABEL_STYLES[state])}>
          {label}
        </span>
      ) : null}
    </div>
  );
}
