"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { resourcesApi } from "@/entities/resource/api/resource.api";
import type { ResourceInput } from "@/entities/resource/model/types";
import type { ProbeResult } from "@/entities/monitor/model/result";
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

export function useResourceOverview() {
  return useQuery({
    queryKey: ["resources", "overview"],
    queryFn: () => resourcesApi.overview(),
    refetchInterval: 30_000,
  });
}

// Single data source for a resource's overview snapshot — status, current
// metric values and sparkline trends in one request. All Metric Cards on the
// page consume this one query instead of firing a request per metric.
export function useResourceOverviewById(resourceId: string | undefined) {
  return useQuery({
    queryKey: ["resources", resourceId, "overview"],
    queryFn: () => resourcesApi.overviewById(resourceId!),
    enabled: Boolean(resourceId),
    staleTime: 10_000,
    gcTime: 300_000,
    placeholderData: (previous) => previous,
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

export function useDeleteResource() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => resourcesApi.delete(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["resources"] }); },
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

export function useDeleteResourceMonitor(resourceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => resourcesApi.deleteMonitor(resourceId, id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["resources", resourceId, "monitors"] }); },
  });
}

export function useResourceMonitorResults(resourceId: string | undefined, monitorId: string | undefined) {
  return useQuery({
    queryKey: ["resources", resourceId, "monitors", monitorId, "results"],
    queryFn: () => resourcesApi.getMonitorResults(resourceId!, monitorId!),
    enabled: Boolean(resourceId) && Boolean(monitorId),
    refetchInterval: 15_000,
    select: (data): ProbeResult | null => data.items[0] ?? null,
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
