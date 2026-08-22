import { apiRequest } from "@/shared/api";
import { endpoints } from "@/shared/api/endpoints";
import type {
  Resource,
  ResourceInput,
  ResourceType,
  MonitorTypeDef,
  ResourceOverview,
} from "@/entities/resource/model/types";
import type { Monitor, MonitorInput } from "@/entities/resource/hooks/types";
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

  overviewById(id: string) {
    return apiRequest<ResourceOverview>(endpoints.resource.overviewById(id));
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
    return apiRequest<{ items: Monitor[] }>(endpoints.resource.monitors(resourceId));
  },

  createMonitor(resourceId: string, input: MonitorInput) {
    return apiRequest<Monitor>(endpoints.resource.monitors(resourceId), { method: "POST", body: JSON.stringify(input) });
  },

  updateMonitor(resourceId: string, id: string, input: MonitorInput) {
    return apiRequest<Monitor>(endpoints.resource.monitor(resourceId, id), { method: "PUT", body: JSON.stringify(input) });
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

  snmpTestConnection(resourceId: string, monitorId: string) {
    return apiRequest<{ task_id: string; kind: string }>(endpoints.resource.snmp.test(resourceId, monitorId), { method: "POST" });
  },

  snmpDiscover(resourceId: string, monitorId: string) {
    return apiRequest<{ task_id: string; kind: string }>(endpoints.resource.snmp.discover(resourceId, monitorId), { method: "POST" });
  },

  snmpGetTask(taskId: string) {
    return apiRequest<SnmpTaskResponse>(endpoints.resource.snmpTasks.task(taskId));
  },

  snmpApplyTask(taskId: string) {
    return apiRequest<{
      ok: boolean;
      interfaces: number;
      sensors: number;
      vendor?: string;
      model?: string;
      sys_name?: string;
    }>(endpoints.resource.snmpTasks.apply(taskId), { method: "POST" });
  },

  snmpSourceIps() {
    return apiRequest<{ ips: string[]; port: number; note: string }>(endpoints.resource.snmpSourceIps);
  },

  snmpDiagnostics(resourceId: string, monitorId: string) {
    return apiRequest<Record<string, unknown>>(endpoints.resource.snmp.diagnostics(resourceId, monitorId));
  },

  snmpGetDiscovery(resourceId: string, monitorId: string) {
    return apiRequest<{ discovery: SnmpDiscovery | null }>(
      endpoints.resource.snmp.discovery(resourceId, monitorId)
    );
  },

  snmpListInterfaces(resourceId: string, monitorId: string) {
    return apiRequest<{ items: SnmpInterfaceRow[] }>(
      endpoints.resource.snmp.interfaces(resourceId, monitorId)
    );
  },

  snmpUpdateInterface(
    resourceId: string,
    monitorId: string,
    ifIndex: number,
    input: Partial<{
      display_name: string;
      ignore: boolean;
      monitor: boolean;
      utilization_warning: number | null;
      utilization_critical: number | null;
      oper_down_critical: boolean;
    }>,
  ) {
    return apiRequest<SnmpInterfaceRow>(endpoints.resource.snmp.interface(resourceId, monitorId, ifIndex), {
      method: "PUT",
      body: JSON.stringify(input),
    });
  },

  snmpListEvents(resourceId: string, monitorId: string, limit = 50) {
    return apiRequest<{ items: SnmpEvent[] }>(
      `${endpoints.resource.snmp.events(resourceId, monitorId)}?limit=${limit}`
    );
  },
};

export interface SnmpDeviceIdentity {
  sys_name?: string;
  sys_descr?: string;
  sys_object_id?: string;
  sys_uptime?: string;
  sys_location?: string;
  vendor?: string;
  model?: string;
  serial_number?: string;
  os?: string;
  firmware?: string;
}

export interface SnmpInterfaceRow {
  id: string;
  monitor_id: string;
  if_index: number;
  if_name: string;
  if_descr?: string;
  if_alias?: string;
  display_name?: string;
  ignore: boolean;
  monitor: boolean;
  utilization_warning?: number | null;
  utilization_critical?: number | null;
  oper_down_critical?: boolean | null;
  last_oper_status?: number | null;
  last_in_bps?: number | null;
  last_out_bps?: number | null;
  last_utilization_percent?: number | null;
  last_check_at?: string | null;
}

export interface SnmpSensor {
  name: string;
  sensor_type: string;
  value: number;
  unit: string;
  status: string;
}

export interface SnmpInterfaceSnapshot {
  if_index: number;
  if_name?: string;
  if_descr?: string;
  if_alias?: string;
  if_type?: number;
  if_speed_bps?: number;
  if_admin_status?: number;
  if_oper_status?: number;
  if_in_octets?: number;
  if_out_octets?: number;
  if_in_packets?: number;
  if_out_packets?: number;
  if_in_errors?: number;
  if_out_errors?: number;
  if_in_discards?: number;
  if_out_discards?: number;
  has_64_bit_in?: boolean;
  has_64_bit_out?: boolean;
  in_bps?: number;
  out_bps?: number;
  utilization_percent?: number;
}

export interface SnmpDiscovery {
  device: SnmpDeviceIdentity;
  interfaces: SnmpInterfaceSnapshot[];
  sensors: SnmpSensor[];
  hardware_ok?: boolean;
  discovered_at?: string;
}

export interface SnmpEvent {
  id: string;
  resource_id: string;
  monitor_id?: string;
  kind: string;
  event_type: string;
  severity: "info" | "warning" | "critical";
  source: string;
  summary: string;
  if_index?: number;
  if_name?: string;
  details?: Record<string, string>;
  created_at: string;
}

export interface SnmpTaskResult {
  ok: boolean;
  state: string;
  kind: string;
  detail?: string;
  sys_name?: string;
  sys_descr?: string;
  sys_object_id?: string;
  uptime?: string;
  discovery?: SnmpDiscovery | string;
  duration_millis?: number;
  steps?: string[];
}

export interface SnmpTaskResponse {
  task_id: string;
  kind: string;
  status: "pending" | "running" | "success" | "failed";
  created_at: string;
  finished_at?: string | null;
  result?: SnmpTaskResult;
  error?: string;
}

export interface ProbeSeries {
  probe_id: string;
  probe_name: string;
  location: string;
  metric_key?: string;
  points: Array<{ timestamp: string; value: number }>;
  values: Array<{ timestamp: string; value: number }>;
}

export interface AggregateChecks {
  total_requests: number;
  successful_requests: number;
  failed_requests: number;
}

// Range-scoped KPIs for a monitor, computed in the metric layer so the UI
// never derives P95 or error rates from raw samples.
export interface MonitorAggregateMetrics {
  checks: AggregateChecks;
  availability: number | null;
  avg_response_time_ms: number | null;
  p95_response_time_ms: number | null;
  avg_ttfb_ms: number | null;
  error_rate: number | null;
  codes_4xx: number;
  rate_4xx: number | null;
  codes_5xx: number;
  rate_5xx: number | null;
}

// Per-probe range KPIs; last-status facts are attached from the latest result.
export interface ProbeAggregateMetrics {
  probe_id: string;
  probe_name: string;
  location: string;
  checks: AggregateChecks;
  availability: number | null;
  avg_response_time_ms: number | null;
  p95_response_time_ms: number | null;
  avg_ttfb_ms: number | null;
  error_rate: number | null;
  last_checked_at: string | null;
  last_status_code: number | null;
  last_success: boolean;
}

export interface MonitorMetricsResponse {
  series: ProbeSeries[];
  latest: ProbeResult[];
  series_limit?: number;
  step_seconds: number;
  from: string;
  to: string;
  metric_key?: string;
  probe_id?: string;
  monitor_type?: string;
  last_success_at?: string | null;
  status_codes?: Array<{ code: number; count: number }>;
  aggregate?: MonitorAggregateMetrics;
  probes?: ProbeAggregateMetrics[];
  selected?: ProbeResult | null;
}
