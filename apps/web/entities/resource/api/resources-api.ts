import { apiRequest } from "@/shared/lib/api-client";
import type { Resource, ResourceInput, ResourceType, MonitorTypeDef, Workspace } from "@/features/resources/types/resource";
import type { MonitorV2, MonitorV2Input } from "@/features/resources/hooks/types";
import type { ProbeResult } from "@/features/monitors/types/result";

export const resourcesApi = {
  listTypes() {
    return apiRequest<{ items: ResourceType[] }>("/api/v1/resource-types");
  },

  listMonitorTypes() {
    return apiRequest<{ items: MonitorTypeDef[] }>("/api/v1/monitor-types");
  },

  listWorkspaces() {
    return apiRequest<{ items: Workspace[] }>("/api/v1/workspaces");
  },

  overview() {
    return apiRequest<{
      total_resources: number;
      by_type: Record<string, number>;
      by_status: Record<string, number>;
      monitors_enabled: number;
      alerts_firing: number;
    }>("/api/v1/resources/overview");
  },

  list(queryString: string) {
    return apiRequest<{ items: Resource[]; pagination: { page: number; total: number; total_pages: number } }>(`/api/v1/resources?${queryString}`);
  },

  getById(id: string) {
    return apiRequest<Resource>(`/api/v1/resources/${id}`);
  },

  create(input: ResourceInput) {
    return apiRequest<Resource>("/api/v1/resources", { method: "POST", body: JSON.stringify(input) });
  },

  update(id: string, input: Partial<ResourceInput>) {
    return apiRequest<Resource>(`/api/v1/resources/${id}`, { method: "PUT", body: JSON.stringify(input) });
  },

  delete(id: string) {
    return apiRequest<void>(`/api/v1/resources/${id}`, { method: "DELETE" });
  },

  listMonitors(resourceId: string) {
    return apiRequest<{ items: MonitorV2[] }>(`/api/v1/resources/${resourceId}/monitors`);
  },

  createMonitor(resourceId: string, input: MonitorV2Input) {
    return apiRequest<MonitorV2>(`/api/v1/resources/${resourceId}/monitors`, { method: "POST", body: JSON.stringify(input) });
  },

  updateMonitor(resourceId: string, id: string, input: MonitorV2Input) {
    return apiRequest<MonitorV2>(`/api/v1/resources/${resourceId}/monitors/${id}`, { method: "PUT", body: JSON.stringify(input) });
  },

  deleteMonitor(resourceId: string, id: string) {
    return apiRequest<void>(`/api/v1/resources/${resourceId}/monitors/${id}`, { method: "DELETE" });
  },

  getMonitorResults(resourceId: string, monitorId: string) {
    return apiRequest<{ items: ProbeResult[] }>(
      `/api/v1/resources/${resourceId}/monitors/${monitorId}/results`
    );
  },
};
