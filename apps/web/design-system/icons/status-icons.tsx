"use client";

import { CircleCheck, AlertTriangle, CircleAlert, MinusCircle, Activity, Pause } from "lucide-react";
import type { LucideIcon } from "lucide-react";

type StatusDef = {
  icon: LucideIcon;
  label: string;
  color: string;
  dot: string;
  badge: string;
};

export const STATUS_ICONS: Record<string, StatusDef> = {
  up: {
    icon: CircleCheck,
    label: "Up",
    color: "text-emerald-500",
    dot: "bg-emerald-500",
    badge: "border-emerald-500/25 bg-emerald-500/10 text-emerald-500",
  },
  healthy: {
    icon: CircleCheck,
    label: "Healthy",
    color: "text-emerald-500",
    dot: "bg-emerald-500",
    badge: "border-emerald-500/25 bg-emerald-500/10 text-emerald-500",
  },
  down: {
    icon: CircleAlert,
    label: "Down",
    color: "text-red-500",
    dot: "bg-red-500",
    badge: "border-red-500/25 bg-red-500/10 text-red-500",
  },
  critical: {
    icon: CircleAlert,
    label: "Critical",
    color: "text-red-500",
    dot: "bg-red-500",
    badge: "border-red-500/25 bg-red-500/10 text-red-500",
  },
  warning: {
    icon: AlertTriangle,
    label: "Warning",
    color: "text-amber-500",
    dot: "bg-amber-500",
    badge: "border-amber-500/25 bg-amber-500/10 text-amber-500",
  },
  degraded: {
    icon: AlertTriangle,
    label: "Degraded",
    color: "text-amber-500",
    dot: "bg-amber-500",
    badge: "border-amber-500/25 bg-amber-500/10 text-amber-500",
  },
  unknown: {
    icon: MinusCircle,
    label: "Unknown",
    color: "text-muted-foreground",
    dot: "bg-muted-foreground",
    badge: "border-muted-foreground/25 bg-muted-foreground/10 text-muted-foreground",
  },
  paused: {
    icon: Pause,
    label: "Paused",
    color: "text-muted-foreground",
    dot: "bg-muted-foreground",
    badge: "border-muted-foreground/25 bg-muted-foreground/10 text-muted-foreground",
  },
  active: {
    icon: Activity,
    label: "Active",
    color: "text-emerald-500",
    dot: "bg-emerald-500",
    badge: "border-emerald-500/25 bg-emerald-500/10 text-emerald-500",
  },
};

export const DEFAULT_STATUS_ICON: StatusDef = STATUS_ICONS.unknown;

export function getStatusIcon(status?: string): StatusDef {
  if (!status) return DEFAULT_STATUS_ICON;
  return STATUS_ICONS[status] ?? DEFAULT_STATUS_ICON;
}
