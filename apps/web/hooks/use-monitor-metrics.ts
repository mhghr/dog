"use client";

import { useQuery } from "@tanstack/react-query";

import { apiRequest } from "@/lib/api-client";
import type { MonitorMetrics } from "@/types/result";

export type MetricsRange = "24h" | "7d" | "30d";

const RANGE_MILLIS: Record<MetricsRange, number> = {
  "24h": 24 * 60 * 60 * 1000,
  "7d": 7 * 24 * 60 * 60 * 1000,
  "30d": 30 * 24 * 60 * 60 * 1000,
};

export function useMonitorMetrics(monitorId: string, range: MetricsRange) {
  return useQuery({
    queryKey: ["monitors", monitorId, "metrics", range],
    queryFn: () => {
      const to = new Date();
      const from = new Date(to.getTime() - RANGE_MILLIS[range]);

      const query = new URLSearchParams({
        from: from.toISOString(),
        to: to.toISOString(),
        step: "auto",
      });

      return apiRequest<MonitorMetrics>(
        `/api/v1/monitors/${monitorId}/metrics?${query.toString()}`,
      );
    },
    enabled: Boolean(monitorId),
    refetchInterval: 60_000,
    placeholderData: (previous) => previous,
  });
}
