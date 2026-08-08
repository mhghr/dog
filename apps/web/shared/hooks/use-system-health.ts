"use client";

import { useQuery } from "@tanstack/react-query";

import { apiRequest } from "@/shared/api";
import type { SystemHealth } from "@/shared/types/api";

export function useSystemHealth() {
  return useQuery({
    queryKey: ["system", "health"],
    queryFn: () => apiRequest<SystemHealth>("/api/v1/system/health"),
    refetchInterval: 10_000,
    placeholderData: (previous) => previous,
  });
}
