"use client";

import { useQuery } from "@tanstack/react-query";

import { monitorApi } from "@/entities/monitor/api/monitor.api";
import type { ResultListResponse } from "@/shared/types/api";

export function useMonitorResults(monitorId: string, limit = 50, page = 1) {
  return useQuery({
    queryKey: ["monitors", monitorId, "results", { limit, page }],
    queryFn: () =>
      monitorApi.results(monitorId, `limit=${limit}&page=${page}`),
    enabled: Boolean(monitorId),
    refetchInterval: 10_000,
    placeholderData: (previous) => previous,
  });
}
