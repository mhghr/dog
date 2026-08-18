"use client";

import { useQuery } from "@tanstack/react-query";

import { monitorApi } from "@/entities/monitor/api/monitor.api";
import type { ProbeSeries } from "@/entities/monitor/model/result";
import type { ProbeResult } from "@/entities/monitor/model/result";

export type MetricsRange = "24h" | "7d" | "30d";

const RANGE_MILLIS: Record<MetricsRange, number> = {
  "24h": 24 * 60 * 60 * 1000,
  "7d": 7 * 24 * 60 * 60 * 1000,
  "30d": 30 * 24 * 60 * 60 * 1000,
};

interface PingSeriesResponse {
  items: ProbeSeries[];
  latest: ProbeResult[];
  step_seconds: number;
  from: string;
  to: string;
}

export function usePingSeriesByMetric(
  monitorId: string,
  metricKey: string,
  range: MetricsRange,
) {
  return useQuery({
    queryKey: ["monitors", monitorId, "ping-series", metricKey, range],
    queryFn: async () => {
      const to = new Date();
      const from = new Date(to.getTime() - RANGE_MILLIS[range]);

      const query = new URLSearchParams({
        from: from.toISOString(),
        to: to.toISOString(),
        step: "auto",
        metric: metricKey,
      });

      const res = await monitorApi.metrics(monitorId, query.toString());
      return res as unknown as PingSeriesResponse;
    },
    enabled: Boolean(monitorId) && metricKey !== "availability",
    refetchInterval: 60_000,
    placeholderData: (previous) => previous,
  });
}
