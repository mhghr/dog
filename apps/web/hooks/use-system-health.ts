"use client";

import { useQuery } from "@tanstack/react-query";

import { apiRequest } from "@/lib/api-client";
import type { SystemHealth } from "@/types/api";

export function useSystemHealth() {
  return useQuery({
    queryKey: ["system", "health"],
    queryFn: () => apiRequest<SystemHealth>("/api/v1/system/health"),
    refetchInterval: 10_000,
    placeholderData: (previous) => previous,
  });
}
