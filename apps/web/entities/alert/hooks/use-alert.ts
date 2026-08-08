"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { alertApi } from "@/entities/alert/api/alert.api";
import type { AlertPolicy, NotificationChannel } from "@/entities/alert/model/types";

export function useAlertPolicies() {
  return useQuery({
    queryKey: ["alert-policies"],
    queryFn: () => alertApi.listPolicies(),
    refetchInterval: 30_000,
    placeholderData: (previous) => previous,
  });
}

export function useAlerts() {
  return useQuery({
    queryKey: ["alerts"],
    queryFn: () => alertApi.listAlerts(),
    refetchInterval: 10_000,
    placeholderData: (previous) => previous,
  });
}

export function useCreateAlertPolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: Partial<AlertPolicy>) => alertApi.createPolicy(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alert-policies"] });
    },
  });
}

export function useNotificationChannels() {
  return useQuery({
    queryKey: ["notification-channels"],
    queryFn: () => alertApi.listChannels(),
    refetchInterval: 30_000,
    placeholderData: (previous) => previous,
  });
}

export function useCreateNotificationChannel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: Partial<NotificationChannel>) =>
      alertApi.createChannel(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notification-channels"] });
    },
  });
}
