import { resourcesApi } from "@/entities/resource/api/resource.api";
import { workspaceApi } from "@/entities/workspace/api/workspace.api";
import { alertApi } from "@/entities/alert/api/alert.api";
import { probeApi } from "@/entities/probe/api/probe.api";

import type { Resource, ResourceType } from "@/entities/resource/model/types";
import type { MonitorV2, MonitorV2Input } from "@/entities/resource/model/monitor-v2";
import type { Alert } from "@/entities/alert/model/types";
import type { ProbeLocation } from "@/entities/probe/model/types";

export interface ResourceOverviewData {
  resource: Resource;
  types: ResourceType[];
  monitors: MonitorV2[];
  alerts: Alert[];
  locations: ProbeLocation[];
}

// getResourceOverview composes resource + monitors + alerts + reference data
// so the resource detail page never calls four APIs from the UI layer.
export async function getResourceOverview(
  resourceId: string,
): Promise<ResourceOverviewData> {
  const [resource, typesResponse, monitorsResponse, alertsResponse, locationsResponse] =
    await Promise.all([
      resourcesApi.getById(resourceId),
      resourcesApi.listTypes(),
      resourcesApi.listMonitors(resourceId),
      alertApi.listAlerts(),
      probeApi.listLocations(),
    ]);

  return {
    resource,
    types: typesResponse.items,
    monitors: monitorsResponse.items,
    alerts: alertsResponse.items,
    locations: locationsResponse.items,
  };
}

export async function createResource(
  input: Parameters<typeof resourcesApi.create>[0],
) {
  return resourcesApi.create(input);
}

export async function deleteResource(id: string) {
  return resourcesApi.delete(id);
}

export async function listResourceTypes() {
  return resourcesApi.listTypes();
}

export async function createResourceMonitor(
  resourceId: string,
  input: MonitorV2Input,
) {
  return resourcesApi.createMonitor(resourceId, input);
}

export async function updateResourceMonitor(
  resourceId: string,
  monitorId: string,
  input: MonitorV2Input,
) {
  return resourcesApi.updateMonitor(resourceId, monitorId, input);
}

export async function deleteResourceMonitor(
  resourceId: string,
  monitorId: string,
) {
  return resourcesApi.deleteMonitor(resourceId, monitorId);
}

export async function listWorkspaces() {
  return workspaceApi.listWorkspaces();
}
