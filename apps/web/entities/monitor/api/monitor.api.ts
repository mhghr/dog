import { apiRequest } from "@/shared/api";
import { endpoints } from "@/shared/api/endpoints";
import type { MonitorListResponse, ResultListResponse } from "@/shared/types/api";
import type { CreateMonitorInput, Monitor } from "@/entities/monitor/model/types";
import type { MonitorMetrics } from "@/entities/monitor/model/result";
import type {
  NotificationChannel,
  NotificationPolicy,
  ParameterHealthState,
  ParameterRule,
} from "@/entities/monitor/model/health";

export const monitorApi = {
  list(queryString: string) {
    return apiRequest<MonitorListResponse>(`${endpoints.monitor.list}?${queryString}`);
  },

  getById(id: string) {
    return apiRequest<Monitor>(endpoints.monitor.byId(id));
  },

  create(input: CreateMonitorInput) {
    return apiRequest<Monitor>(endpoints.monitor.list, {
      method: "POST",
      body: JSON.stringify(input),
    });
  },

  update(id: string, input: CreateMonitorInput) {
    return apiRequest<Monitor>(endpoints.monitor.byId(id), {
      method: "PUT",
      body: JSON.stringify(input),
    });
  },

  delete(id: string) {
    return apiRequest<void>(endpoints.monitor.byId(id), { method: "DELETE" });
  },

  pause(id: string) {
    return apiRequest<Monitor>(endpoints.monitor.pause(id), { method: "POST" });
  },

  resume(id: string) {
    return apiRequest<Monitor>(endpoints.monitor.resume(id), { method: "POST" });
  },

  metrics(id: string, queryString: string) {
    return apiRequest<MonitorMetrics>(`${endpoints.monitor.metrics(id)}?${queryString}`);
  },

  results(id: string, queryString: string) {
    return apiRequest<ResultListResponse>(`${endpoints.monitor.results(id)}?${queryString}`);
  },

  healthRules(id: string) {
    return apiRequest<ParameterRule[]>(endpoints.monitor.health.rules(id));
  },

  updateHealthRule(id: string, input: Partial<ParameterRule>) {
    return apiRequest<ParameterRule>(endpoints.monitor.health.rules(id), {
      method: "PUT",
      body: JSON.stringify(input),
    });
  },

  healthStates(id: string) {
    return apiRequest<ParameterHealthState[]>(endpoints.monitor.health.states(id));
  },

  notificationChannels() {
    return apiRequest<NotificationChannel[]>(endpoints.alerting.channels);
  },

  notificationPolicies(id: string) {
    return apiRequest<NotificationPolicy[]>(endpoints.monitor.health.monitorPolicies(id));
  },

  createNotificationPolicy(input: Partial<NotificationPolicy>) {
    return apiRequest<NotificationPolicy>(endpoints.monitor.health.policies, {
      method: "POST",
      body: JSON.stringify(input),
    });
  },

  updateNotificationPolicy(input: Partial<NotificationPolicy> & { id: string }) {
    return apiRequest<NotificationPolicy>(endpoints.monitor.health.policy(input.id), {
      method: "PUT",
      body: JSON.stringify(input),
    });
  },
};
