import type { QueryClient } from "@tanstack/react-query";

import { endpoints } from "@/shared/api/endpoints";
import { ServerApiError, serverApiRequest } from "@/shared/api/server";
import type { Monitor } from "@/entities/resource/hooks/types";
import {
  buildMetricsQueryString,
  isPingMonitor,
  resourceMonitorMetricsQueryKey,
} from "@/entities/resource/hooks/resource-query";
import type {
  MonitorTypeDef,
  Resource,
} from "@/entities/resource/model/types";
import type { MonitorMetricsResponse } from "@/entities/resource/api/resource.api";

export class ResourceNotFoundError extends Error {
  constructor(resourceId: string) {
    super(`Resource not found: ${resourceId}`);
    this.name = "ResourceNotFoundError";
  }
}

const DEFAULT_RANGE = "1h" as const;
const PING_METRICS = [
  undefined,
  "packet_loss_percent",
  "jitter_ms",
] as const;

// Server-side preload for the Resource detail page. Fetches every query the
// page layout depends on — the resource, its monitors, the monitor types, and
// the ping-metric series the monitoring cards render — through the
// cookie-forwarding server client, so the initial server-rendered HTML already
// contains the final page state (header stats, health status and KPI cards).
// The hydrated client then owns revalidation (refetch intervals + SSE).
export async function prefetchResourceDetail(
  queryClient: QueryClient,
  resourceId: string,
): Promise<void> {
  try {
    await queryClient.fetchQuery({
      queryKey: ["resources", resourceId],
      queryFn: () =>
        serverApiRequest<Resource>(endpoints.resource.byId(resourceId), undefined, {
          refreshOn401: true,
        }),
    });
  } catch (error) {
    if (error instanceof ServerApiError && error.status === 404) {
      throw new ResourceNotFoundError(resourceId);
    }
    throw error;
  }

  await Promise.all([
    queryClient.prefetchQuery({
      queryKey: ["resources", resourceId, "monitors"],
      queryFn: () =>
        serverApiRequest<{ items: Monitor[] }>(
          endpoints.resource.monitors(resourceId),
          undefined,
          { refreshOn401: true },
        ),
      staleTime: 15_000,
    }),
    queryClient.prefetchQuery({
      queryKey: ["monitor-types"],
      queryFn: () =>
        serverApiRequest<{ items: MonitorTypeDef[] }>(
          endpoints.monitor.types,
          undefined,
          { refreshOn401: true },
        ),
      staleTime: 300_000,
    }),
  ]);

  const monitorsData =
    queryClient.getQueryData<{ items: Monitor[] }>(["resources", resourceId, "monitors"]);
  const typesData =
    queryClient.getQueryData<{ items: MonitorTypeDef[] }>(["monitor-types"]);

  const pingMonitors = (monitorsData?.items ?? []).filter((m) =>
    isPingMonitor(m, typesData?.items ?? []),
  );

  // Preload the metric series the ping cards render, so the health state and
  // KPI cards are correct in the initial HTML rather than "unknown"/skeleton.
  await Promise.all(
    pingMonitors.flatMap((monitor) =>
      PING_METRICS.map((metric) =>
        queryClient.prefetchQuery({
          queryKey: resourceMonitorMetricsQueryKey(
            resourceId,
            monitor.id,
            DEFAULT_RANGE,
            metric,
          ),
          queryFn: () =>
            serverApiRequest<MonitorMetricsResponse>(
              `${endpoints.resource.monitorMetrics(resourceId, monitor.id)}?${buildMetricsQueryString(DEFAULT_RANGE, metric)}`,
              undefined,
              { refreshOn401: true },
            ),
          staleTime: 15_000,
        }),
      ),
    ),
  );
}
