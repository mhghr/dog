"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { apiRequest } from "@/lib/api-client";
import type {
  AlertPolicy,
  AlertPolicyListResponse,
  AlertListResponse,
  NotificationChannel,
  NotificationChannelListResponse,
} from "@/types/alert";

export function useAlertPolicies() {
  return useQuery({
    queryKey: ["alert-policies"],
    queryFn: () =>
      apiRequest<AlertPolicyListResponse>("/api/v1/alerting/policies"),
    refetchInterval: 30_000,
  });
}

export function useAlerts() {
  return useQuery({
    queryKey: ["alerts"],
    queryFn: () =>
      apiRequest<AlertListResponse>("/api/v1/alerting/alerts"),
    refetchInterval: 10_000,
  });
}

export function useCreateAlertPolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: Partial<AlertPolicy>) =>
      apiRequest<AlertPolicy>("/api/v1/alerting/policies", {
        method: "POST",
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alert-policies"] });
    },
  });
}

export function useNotificationChannels() {
  return useQuery({
    queryKey: ["notification-channels"],
    queryFn: () =>
      apiRequest<NotificationChannelListResponse>(
        "/api/v1/alerting/channels",
      ),
    refetchInterval: 30_000,
  });
}

export function useCreateNotificationChannel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: Partial<NotificationChannel>) =>
      apiRequest<NotificationChannel>("/api/v1/alerting/channels", {
        method: "POST",
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notification-channels"] });
    },
  });
}
