"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { resourcesApi } from "@/entities/resource/api/resource.api";
import type { ResourceInput } from "@/entities/resource/model/types";
import type { MonitorInput, Monitor } from "@/entities/resource/hooks/types";
import {
  buildMetricsQueryString,
  resourceListQueryString,
  resourceMonitorMetricsQueryKey,
  type MetricsRange,
  type ResourceListParams,
} from "@/entities/resource/hooks/resource-query";

export { type Monitor, type MonitorInput };
export {
  RANGE_MILLIS,
  buildMetricsQueryString,
  isPingMonitor,
  resourceListQueryString,
  resourceMonitorMetricsQueryKey,
  type MetricsRange,
  type ResourceListParams,
} from "@/entities/resource/hooks/resource-query";

export function useResourceTypes() {
  return useQuery({
    queryKey: ["resource-types"],
    queryFn: () => resourcesApi.listTypes(),
    staleTime: 300_000,
  });
}

export function useMonitorTypes() {
  return useQuery({
    queryKey: ["monitor-types"],
    queryFn: () => resourcesApi.listMonitorTypes(),
    staleTime: 300_000,
  });
}

export function useCreateResource() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ResourceInput) => resourcesApi.create(input),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["resources"] }); },
  });
}

export function useResources(params: ResourceListParams) {
  const queryString = resourceListQueryString(params);

  return useQuery({
    queryKey: ["resources", "list", queryString],
    queryFn: () => resourcesApi.list(queryString),
    refetchInterval: 15_000,
    placeholderData: (prev) => prev,
  });
}

export function useResource(id: string | undefined) {
  return useQuery({
    queryKey: ["resources", id],
    queryFn: () => resourcesApi.getById(id!),
    enabled: Boolean(id),
    placeholderData: (prev) => prev,
  });
}

export function useUpdateResource(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: Partial<ResourceInput>) => resourcesApi.update(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["resources", id] });
      qc.invalidateQueries({ queryKey: ["resources", "list"] });
    },
  });
}

export function useResourceMonitors(resourceId: string | undefined) {
  return useQuery({
    queryKey: ["resources", resourceId, "monitors"],
    queryFn: () => resourcesApi.listMonitors(resourceId!),
    enabled: Boolean(resourceId),
    refetchInterval: 15_000,
    placeholderData: (prev) => prev,
  });
}

export function useCreateResourceMonitor(resourceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: MonitorInput) => resourcesApi.createMonitor(resourceId, input),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["resources", resourceId, "monitors"] }); },
  });
}

export function useUpdateResourceMonitor(resourceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...input }: MonitorInput & { id: string }) => resourcesApi.updateMonitor(resourceId, id, input),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["resources", resourceId, "monitors"] }); },
  });
}

export function useResourceMonitorMetrics(
  resourceId: string | undefined,
  monitorId: string | undefined,
  range: MetricsRange,
  metric?: string,
) {
  return useQuery({
    queryKey: resourceMonitorMetricsQueryKey(resourceId, monitorId, range, metric),
    queryFn: () =>
      resourcesApi.getMonitorMetrics(
        resourceId!,
        monitorId!,
        buildMetricsQueryString(range, metric),
      ),
    enabled: Boolean(resourceId) && Boolean(monitorId),
	staleTime: 15_000,
    refetchInterval: range === "15m" || range === "1h" ? 15_000 : 60_000,
    placeholderData: (prev) => prev,
  });
}

// Fetches the explicit per-probe availability series (success ratio 0..1) for
// a ping monitor. Down periods come from this signal, never from latency gaps.
export function useResourceMonitorStatus(
  resourceId: string | undefined,
  monitorId: string | undefined,
  range: MetricsRange,
) {
  return useResourceMonitorMetrics(resourceId, monitorId, range, "status");
}
