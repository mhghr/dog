import type { ReactNode } from "react";

import { Card, CardContent } from "@/shared/ui/card";
import { cn } from "@/shared/utils/cn";

export interface MetricCardProps {
  label: string;
  value: ReactNode;
  hint?: ReactNode;
  icon?: ReactNode;
  tone?: "default" | "success" | "warning" | "destructive";
  className?: string;
}

const TONE_TEXT: Record<NonNullable<MetricCardProps["tone"]>, string> = {
  default: "text-foreground",
  success: "text-success",
  warning: "text-warning",
  destructive: "text-destructive",
};

// Design-system MetricCard: the standard KPI tile used across dashboards and
// entity detail pages (monitors, resources, probes).
export function MetricCard({
  label,
  value,
  hint,
  icon,
  tone = "default",
  className,
}: MetricCardProps) {
  return (
    <Card className={cn("h-full", className)} variant="bordered">
      <CardContent className="flex h-full flex-col gap-1 p-4">
        <div className="flex items-center justify-between gap-2">
          <p className="truncate text-xs font-medium text-muted-foreground">{label}</p>
          {icon ? <span className="shrink-0 text-muted-foreground">{icon}</span> : null}
        </div>
        <p className={cn("text-xl font-semibold tabular-nums", TONE_TEXT[tone])}>{value}</p>
        {hint ? <p className="truncate text-xs text-muted-foreground">{hint}</p> : null}
      </CardContent>
    </Card>
  );
}
