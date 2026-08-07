"use client";

import { useQuery } from "@tanstack/react-query";

import { apiRequest } from "@/lib/api-client";
import type { DashboardSummary } from "@/types/api";

export function useDashboardSummary() {
  return useQuery({
    queryKey: ["dashboard", "summary"],
    queryFn: () => apiRequest<DashboardSummary>("/api/v1/dashboard/summary"),
    refetchInterval: 15_000,
    placeholderData: (previous) => previous,
  });
}
