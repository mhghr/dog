import type { Monitor, MonitorType } from "@/types/monitor";

export function monitorHost(target: string, type: MonitorType): string {
  if (type === "http") {
    try {
      return new URL(target).hostname.toLowerCase().replace(/^www\./, "");
    } catch {
      return target.toLowerCase();
    }
  }

  const targetWithoutPort = target.replace(/^\[([^\]]+)\](?::\d+)?$/, "$1");
  const lastColon = targetWithoutPort.lastIndexOf(":");
  if (lastColon > 0 && /^\d+$/.test(targetWithoutPort.slice(lastColon + 1))) {
    return targetWithoutPort.slice(0, lastColon).toLowerCase().replace(/^www\./, "");
  }
  return targetWithoutPort.toLowerCase().replace(/^www\./, "");
}

export function belongsToSameNode(left: Monitor, right: Monitor): boolean {
  return monitorHost(left.target, left.type) === monitorHost(right.target, right.type);
}
