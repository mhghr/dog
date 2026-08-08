import { Badge } from "@/shared/ui/badge";
import { cn } from "@/shared/utils/cn";

export type StatusTone = "success" | "warning" | "destructive" | "info" | "muted";

const TONE_STYLES: Record<StatusTone, string> = {
  success: "border-transparent bg-success/12 text-success",
  warning: "border-transparent bg-warning/15 text-warning",
  destructive: "border-transparent bg-destructive/12 text-destructive",
  info: "border-transparent bg-info/12 text-info",
  muted: "border-transparent bg-muted text-muted-foreground",
};

const DOT_STYLES: Record<StatusTone, string> = {
  success: "bg-success",
  warning: "bg-warning",
  destructive: "bg-destructive",
  info: "bg-info",
  muted: "bg-muted-foreground/60",
};

export interface StatusBadgeProps {
  tone?: StatusTone;
  label: string;
  className?: string;
}

// Design-system StatusBadge: a semantic status pill used across entities
// (resources, monitors, agents). Domain-specific variants stay in the entity
// layers; this is the shared primitive.
export function StatusBadge({ tone = "muted", label, className }: StatusBadgeProps) {
  return (
    <Badge className={cn("gap-1.5 font-medium", TONE_STYLES[tone], className)}>
      <span
        className={cn("size-1.5 shrink-0 rounded-full", DOT_STYLES[tone])}
        aria-hidden
      />
      {label}
    </Badge>
  );
}
