import type { ReactNode } from "react";

import { Card, CardContent } from "@/shared/ui/card";
import { Sparkline } from "@/shared/ui/sparkline";
import { cn } from "@/shared/utils/cn";

export type MetricStatus = "healthy" | "warning" | "critical" | "unknown";

export interface MetricCardProps {
  title: string;
  value: ReactNode;
  unit?: string;
  status?: MetricStatus;
  trend?: number[];
  lastUpdated?: string;
  hint?: ReactNode;
  icon?: ReactNode;
  className?: string;
}

const STATUS_TEXT: Record<MetricStatus, string> = {
  healthy: "text-success",
  warning: "text-warning",
  critical: "text-destructive",
  unknown: "text-muted-foreground",
};

const STATUS_DOT: Record<MetricStatus, string> = {
  healthy: "bg-success",
  warning: "bg-warning",
  critical: "bg-destructive",
  unknown: "bg-muted-foreground/50",
};

// Design-system MetricCard: the standard, reusable KPI tile. Every metric
// (CPU, Memory, Ping, HTTP, ...) renders through this one component — only the
// configuration (title/value/status/trend) differs, never the component.
export function MetricCard({
  title,
  value,
  unit,
  status = "unknown",
  trend,
  lastUpdated,
  hint,
  icon,
  className,
}: MetricCardProps) {
  return (
    <Card
      variant="bordered"
      className={cn("h-full shadow-subtle", className)}
    >
      <CardContent className="flex h-full flex-col gap-1.5 p-4">
        <div className="flex items-center justify-between gap-2">
          <p className="truncate text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
            {title}
          </p>
          {icon ? <span className="shrink-0 text-muted-foreground">{icon}</span> : null}
        </div>

        <div className="flex items-baseline gap-1">
          <span className={cn("text-2xl font-semibold tracking-tight tabular-nums", STATUS_TEXT[status])} dir="ltr">
            {value}
          </span>
          {unit ? (
            <span className="text-sm font-medium text-muted-foreground">{unit}</span>
          ) : null}
        </div>

        <div className="flex items-center gap-1.5">
          <span className={cn("size-1.5 rounded-full", STATUS_DOT[status])} aria-hidden />
          <span className="text-xs font-medium capitalize text-muted-foreground">{status}</span>
        </div>

        {trend && trend.length > 1 ? (
          <Sparkline values={trend} className="mt-1 w-full" stroke={status === "critical" ? "var(--destructive)" : status === "warning" ? "var(--warning)" : "var(--success)"} />
        ) : null}

        {hint ? <p className="truncate text-xs text-muted-foreground">{hint}</p> : null}
        {lastUpdated ? (
          <p className="mt-auto truncate text-[11px] text-muted-foreground/70">{lastUpdated}</p>
        ) : null}
      </CardContent>
    </Card>
  );
}
