import { apiRequest } from "@/shared/api";
import { endpoints } from "@/shared/api/endpoints";
import type {
  AlertListResponse,
  AlertPolicy,
  AlertPolicyListResponse,
  NotificationChannel,
  NotificationChannelListResponse,
} from "@/entities/alert/model/types";

export const alertApi = {
  listPolicies() {
    return apiRequest<AlertPolicyListResponse>(endpoints.alerting.policies);
  },

  listAlerts() {
    return apiRequest<AlertListResponse>(endpoints.alerting.alerts);
  },

  createPolicy(input: Partial<AlertPolicy>) {
    return apiRequest<AlertPolicy>(endpoints.alerting.policies, {
      method: "POST",
      body: JSON.stringify(input),
    });
  },

  listChannels() {
    return apiRequest<NotificationChannelListResponse>(
      endpoints.alerting.channels,
    );
  },

  createChannel(input: Partial<NotificationChannel>) {
    return apiRequest<NotificationChannel>(endpoints.alerting.channels, {
      method: "POST",
      body: JSON.stringify(input),
    });
  },
};
