"use client";

import { useQuery } from "@tanstack/react-query";

import { apiRequest } from "@/lib/api-client";
import type { Monitor } from "@/types/monitor";

export function useMonitor(monitorId: string) {
  return useQuery({
    queryKey: ["monitors", monitorId],
    queryFn: () => apiRequest<Monitor>(`/api/v1/monitors/${monitorId}`),
    enabled: Boolean(monitorId),
    refetchInterval: 10_000,
  });
}
