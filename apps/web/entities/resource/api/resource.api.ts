import { apiRequest } from "@/shared/api";
import { endpoints } from "@/shared/api/endpoints";
import type {
  Resource,
  ResourceInput,
  ResourceType,
  MonitorTypeDef,
} from "@/entities/resource/model/types";
import type { MonitorV2, MonitorV2Input } from "@/entities/resource/model/monitor-v2";
import type { ProbeResult } from "@/entities/monitor/model/result";

export const resourcesApi = {
  listTypes() {
    return apiRequest<{ items: ResourceType[] }>(endpoints.resource.types);
  },

  listMonitorTypes() {
    return apiRequest<{ items: MonitorTypeDef[] }>(endpoints.monitor.types);
  },

  overview() {
    return apiRequest<{
      total_resources: number;
      by_type: Record<string, number>;
      by_status: Record<string, number>;
      monitors_enabled: number;
      alerts_firing: number;
    }>(endpoints.resource.overview);
  },

  list(queryString: string) {
    return apiRequest<{ items: Resource[]; pagination: { page: number; total: number; total_pages: number } }>(
      `${endpoints.resource.list}?${queryString}`
    );
  },

  getById(id: string) {
    return apiRequest<Resource>(endpoints.resource.byId(id));
  },

  create(input: ResourceInput) {
    return apiRequest<Resource>(endpoints.resource.list, { method: "POST", body: JSON.stringify(input) });
  },

  update(id: string, input: Partial<ResourceInput>) {
    return apiRequest<Resource>(endpoints.resource.byId(id), { method: "PUT", body: JSON.stringify(input) });
  },

  delete(id: string) {
    return apiRequest<void>(endpoints.resource.byId(id), { method: "DELETE" });
  },

  listMonitors(resourceId: string) {
    return apiRequest<{ items: MonitorV2[] }>(endpoints.resource.monitors(resourceId));
  },

  createMonitor(resourceId: string, input: MonitorV2Input) {
    return apiRequest<MonitorV2>(endpoints.resource.monitors(resourceId), { method: "POST", body: JSON.stringify(input) });
  },

  updateMonitor(resourceId: string, id: string, input: MonitorV2Input) {
    return apiRequest<MonitorV2>(endpoints.resource.monitor(resourceId, id), { method: "PUT", body: JSON.stringify(input) });
  },

  deleteMonitor(resourceId: string, id: string) {
    return apiRequest<void>(endpoints.resource.monitor(resourceId, id), { method: "DELETE" });
  },

  getMonitorResults(resourceId: string, monitorId: string) {
    return apiRequest<{ items: ProbeResult[] }>(
      endpoints.resource.monitorResults(resourceId, monitorId)
    );
  },

  getMonitorMetrics(resourceId: string, monitorId: string, queryString: string) {
    return apiRequest<MonitorMetricsResponse>(
      `${endpoints.resource.monitorMetrics(resourceId, monitorId)}?${queryString}`
    );
  },
};

export interface ProbeSeries {
  probe_id: string;
  probe_name: string;
  location: string;
  points: Array<{ timestamp: string; value: number }>;
  values: Array<{ timestamp: string; value: number }>;
}

export interface MonitorMetricsResponse {
  series: ProbeSeries[];
  latest: ProbeResult[];
  step_seconds: number;
  from: string;
  to: string;
}
