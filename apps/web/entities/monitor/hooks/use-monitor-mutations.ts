"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";

import { monitorApi } from "@/entities/monitor/api/monitor.api";
import type { CreateMonitorInput } from "@/entities/monitor/model/types";

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
    mutationFn: (input: CreateMonitorInput) => monitorApi.create(input),
    onSuccess: () => invalidate(),
  });
}

export function useDeleteMonitor() {
  const invalidate = useInvalidateMonitors();

  return useMutation({
    mutationFn: (monitorId: string) => monitorApi.delete(monitorId),
    onSuccess: () => invalidate(),
  });
}

export function usePauseMonitor() {
  const invalidate = useInvalidateMonitors();

  return useMutation({
    mutationFn: (monitorId: string) => monitorApi.pause(monitorId),
    onSuccess: (_, monitorId) => invalidate(monitorId),
  });
}

export function useResumeMonitor() {
  const invalidate = useInvalidateMonitors();

  return useMutation({
    mutationFn: (monitorId: string) => monitorApi.resume(monitorId),
    onSuccess: (_, monitorId) => invalidate(monitorId),
  });
}
