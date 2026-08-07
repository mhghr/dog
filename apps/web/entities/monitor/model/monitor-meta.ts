import { MONITOR_REGISTRY, MONITOR_TYPE_GROUPS, MONITOR_TYPES } from "@/features/monitors/core/registry";
import type { MonitorStatus, MonitorType } from "@/types/monitor";

// Compatibility facade. New feature code should import the registry directly.
export { MONITOR_TYPE_GROUPS, MONITOR_TYPES };

export const MONITOR_TYPE_ICONS = Object.fromEntries(
  MONITOR_TYPES.map((type) => [type, MONITOR_REGISTRY[type].icon]),
) as Record<MonitorType, (typeof MONITOR_REGISTRY)[MonitorType]["icon"]>;

export const DEFAULT_INTERVALS = Object.fromEntries(
  MONITOR_TYPES.map((type) => [type, MONITOR_REGISTRY[type].defaultIntervalSeconds]),
) as Record<MonitorType, number>;

export const MIN_INTERVALS = Object.fromEntries(
  MONITOR_TYPES.map((type) => [type, MONITOR_REGISTRY[type].minimumIntervalSeconds]),
) as Record<MonitorType, number>;

export const STATUS_STYLES: Record<
  MonitorStatus,
  { badge: string; dot: string }
> = {
  up: {
    badge: "border-transparent bg-success/12 text-success",
    dot: "bg-success",
  },
  down: {
    badge: "border-transparent bg-destructive/12 text-destructive",
    dot: "bg-destructive",
  },
  unknown: {
    badge: "border-transparent bg-muted text-muted-foreground",
    dot: "bg-muted-foreground/60",
  },
  paused: {
    badge: "border-transparent bg-warning/15 text-warning",
    dot: "bg-warning",
  },
};
