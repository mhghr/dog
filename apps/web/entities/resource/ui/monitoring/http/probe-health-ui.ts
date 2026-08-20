import type { ProbeHealth } from "./http-metrics";

export function healthTone(health: ProbeHealth): "success" | "warning" | "destructive" | "muted" {
  switch (health) {
    case "healthy":
      return "success";
    case "warning":
      return "warning";
    case "critical":
    case "down":
      return "destructive";
    default:
      return "muted";
  }
}

export function healthLabel(health: ProbeHealth, isFa: boolean): string {
  switch (health) {
    case "healthy":
      return isFa ? "سالم" : "Healthy";
    case "warning":
      return isFa ? "هشدار" : "Warning";
    case "critical":
      return isFa ? "بحرانی" : "Critical";
    case "down":
      return isFa ? "قطع" : "Down";
    default:
      return isFa ? "نامشخص" : "Unknown";
  }
}
