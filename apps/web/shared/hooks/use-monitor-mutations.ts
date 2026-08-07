"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";

import { apiRequest } from "@/lib/api-client";
import type { CreateMonitorInput, Monitor } from "@/types/monitor";

function useInvalidateMonitors() {
  const queryClient = useQueryClient();

  return async (monitorId?: string) => {
    await queryClient.invalidateQueries({ queryKey: ["monitors"] });
    await queryClient.invalidateQueries({ queryKey: ["dashboard"] });
    if (monitorId) {
      await queryClient.invalidateQueries({ queryKey: ["monitors", monitorId] });
    }
  };
}

export function useCreateMonitor() {
  const invalidate = useInvalidateMonitors();

  return useMutation({
    mutationFn: (input: CreateMonitorInput) =>
      apiRequest<Monitor>("/api/v1/monitors", {
        method: "POST",
        body: JSON.stringify(input),
      }),
    onSuccess: () => invalidate(),
  });
}

export function useUpdateMonitor(monitorId: string) {
  const invalidate = useInvalidateMonitors();

  return useMutation({
    mutationFn: (input: CreateMonitorInput) =>
      apiRequest<Monitor>(`/api/v1/monitors/${monitorId}`, {
        method: "PUT",
        body: JSON.stringify(input),
      }),
    onSuccess: () => invalidate(monitorId),
  });
}

export function useDeleteMonitor() {
  const invalidate = useInvalidateMonitors();

  return useMutation({
    mutationFn: (monitorId: string) =>
      apiRequest<void>(`/api/v1/monitors/${monitorId}`, { method: "DELETE" }),
    onSuccess: () => invalidate(),
  });
}

export function usePauseMonitor() {
  const invalidate = useInvalidateMonitors();

  return useMutation({
    mutationFn: (monitorId: string) =>
      apiRequest<Monitor>(`/api/v1/monitors/${monitorId}/pause`, {
        method: "POST",
      }),
    onSuccess: (_, monitorId) => invalidate(monitorId),
  });
}

export function useResumeMonitor() {
  const invalidate = useInvalidateMonitors();

  return useMutation({
    mutationFn: (monitorId: string) =>
      apiRequest<Monitor>(`/api/v1/monitors/${monitorId}/resume`, {
        method: "POST",
      }),
    onSuccess: (_, monitorId) => invalidate(monitorId),
  });
}
