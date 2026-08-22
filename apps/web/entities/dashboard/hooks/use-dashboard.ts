"use client";

import { useQuery } from "@tanstack/react-query";

import { dashboardApi } from "@/entities/dashboard/api/dashboard.api";

export function useDashboardSummary() {
  return useQuery({
    queryKey: ["dashboard-summary"],
    queryFn: () => dashboardApi.summary(),
    refetchInterval: 5_000,
    placeholderData: (previous) => previous,
  });
}
