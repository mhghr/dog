"use client";

import { useQuery } from "@tanstack/react-query";

import { apiRequest } from "@/lib/api-client";
import type { ResultListResponse } from "@/types/api";

export function useMonitorResults(monitorId: string, limit = 50, page = 1) {
  return useQuery({
    queryKey: ["monitors", monitorId, "results", { limit, page }],
    queryFn: () =>
      apiRequest<ResultListResponse>(
        `/api/v1/monitors/${monitorId}/results?limit=${limit}&page=${page}`,
      ),
    enabled: Boolean(monitorId),
    refetchInterval: 10_000,
    placeholderData: (previous) => previous,
  });
}
