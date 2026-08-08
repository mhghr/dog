"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { monitorApi } from "@/entities/monitor/api/monitor.api";
import type { MonitorType } from "@/entities/monitor/model/types";
import type {
  NotificationChannel,
  NotificationPolicy,
  ParameterHealthState,
  ParameterRule,
} from "@/entities/monitor/model/health";
import { getParametersForType } from "@/features/monitor-management/schemas/parameter-catalog";

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
    queryFn: () => monitorApi.healthRules(monitorId),
    enabled: Boolean(monitorId),
    refetchInterval: 15_000,
    placeholderData: (previous) => previous,
  });
}

export function useParameterHealthStates(monitorId: string) {
  return useQuery({
    queryKey: ["parameter-health-states", monitorId],
    queryFn: () => monitorApi.healthStates(monitorId),
    enabled: Boolean(monitorId),
    refetchInterval: 15_000,
    placeholderData: (previous) => previous,
  });
}

export function useUpdateParameterRule(monitorId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: Partial<ParameterRule>) =>
      monitorApi.updateHealthRule(monitorId, input),
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
    queryFn: () => monitorApi.notificationChannels(),
    refetchInterval: 30_000,
  });
}

export function useNotificationPolicies(monitorId: string) {
  return useQuery({
    queryKey: ["notification-policies", monitorId],
    queryFn: () => monitorApi.notificationPolicies(monitorId),
    enabled: Boolean(monitorId),
    refetchInterval: 15_000,
  });
}

export function useCreateNotificationPolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: Partial<NotificationPolicy>) =>
      monitorApi.createNotificationPolicy(input),
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
      monitorApi.updateNotificationPolicy(input),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["notification-policies", variables.monitor_id],
      });
    },
  });
}
