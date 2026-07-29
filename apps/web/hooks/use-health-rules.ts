"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { apiRequest } from "@/lib/api-client";
import type { MonitorType } from "@/types/monitor";
import type {
  NotificationChannel,
  NotificationPolicy,
  ParameterHealthState,
  ParameterRule,
} from "@/types/health";
import { getParametersForType } from "@/features/monitors/health/parameter-catalog";

export function useParameterCatalog(monitorType: MonitorType) {
  return useQuery({
    queryKey: ["parameter-catalog", monitorType],
    queryFn: () => Promise.resolve(getParametersForType(monitorType)),
    enabled: Boolean(monitorType),
    staleTime: Infinity,
  });
}

export function useParameterRules(monitorId: string) {
  return useQuery({
    queryKey: ["parameter-rules", monitorId],
    queryFn: () =>
      apiRequest<ParameterRule[]>(
        `/api/v1/monitors/${monitorId}/health/rules`,
      ),
    enabled: Boolean(monitorId),
    refetchInterval: 15_000,
    placeholderData: (previous) => previous,
  });
}

export function useParameterHealthStates(monitorId: string) {
  return useQuery({
    queryKey: ["parameter-health-states", monitorId],
    queryFn: () =>
      apiRequest<ParameterHealthState[]>(
        `/api/v1/monitors/${monitorId}/health/states`,
      ),
    enabled: Boolean(monitorId),
    refetchInterval: 15_000,
    placeholderData: (previous) => previous,
  });
}

export function useUpdateParameterRule(monitorId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: Partial<ParameterRule>) =>
      apiRequest<ParameterRule>(
        `/api/v1/monitors/${monitorId}/health/rules`,
        { method: "PUT", body: JSON.stringify(input) },
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["parameter-rules", monitorId] });
      queryClient.invalidateQueries({
        queryKey: ["parameter-health-states", monitorId],
      });
    },
  });
}

export function useNotificationChannels() {
  return useQuery({
    queryKey: ["notification-channels"],
    queryFn: () =>
      apiRequest<NotificationChannel[]>(
        "/api/v1/alerting/channels",
      ),
    refetchInterval: 30_000,
  });
}

export function useNotificationPolicies(monitorId: string) {
  return useQuery({
    queryKey: ["notification-policies", monitorId],
    queryFn: () =>
      apiRequest<NotificationPolicy[]>(
        `/api/v1/monitors/${monitorId}/health/policies`,
      ),
    enabled: Boolean(monitorId),
    refetchInterval: 15_000,
  });
}

export function useCreateNotificationPolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: Partial<NotificationPolicy>) =>
      apiRequest<NotificationPolicy>("/api/v1/monitors/health/policies", {
        method: "POST",
        body: JSON.stringify(input),
      }),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["notification-policies", variables.monitor_id],
      });
    },
  });
}

export function useUpdateNotificationPolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: Partial<NotificationPolicy> & { id: string }) =>
      apiRequest<NotificationPolicy>(
        `/api/v1/monitors/health/policies/${input.id}`,
        { method: "PUT", body: JSON.stringify(input) },
      ),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["notification-policies", variables.monitor_id],
      });
    },
  });
}
