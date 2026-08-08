import { resourcesApi } from "@/entities/resource/api/resource.api";

export interface ResourceHealthSummary {
  resourceId: string;
  totalMonitors: number;
  enabledMonitors: number;
  activeMonitors: number;
  status: "healthy" | "degraded" | "down" | "unknown";
}

// resourceHealth combines the monitor configuration with live status from the
// resource's monitor collection into a single health summary for the UI.
export async function getResourceHealth(
  resourceId: string,
): Promise<ResourceHealthSummary> {
  const { items } = await resourcesApi.listMonitors(resourceId);

  const totalMonitors = items.length;
  const enabledMonitors = items.filter((m) => m.enabled).length;
  const activeMonitors = items.filter((m) => m.enabled && m.last_status === "up").length;

  let status: ResourceHealthSummary["status"] = "unknown";
  if (enabledMonitors === 0) {
    status = "unknown";
  } else if (activeMonitors === enabledMonitors) {
    status = "healthy";
  } else if (activeMonitors === 0) {
    status = "down";
  } else {
    status = "degraded";
  }

  return { resourceId, totalMonitors, enabledMonitors, activeMonitors, status };
}
