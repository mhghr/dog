"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { resourcesApi } from "@/entities/resource/api/resource.api";
import type { ResourceInput } from "@/entities/resource/model/types";
import type { ProbeResult } from "@/entities/monitor/model/result";
import type { MonitorV2Input, MonitorV2 } from "@/entities/resource/model/monitor-v2";

export { type MonitorV2, type MonitorV2Input };

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

export interface ResourceListParams {
  page?: number;
  pageSize?: number;
  search?: string;
  status?: string;
  resourceTypeId?: string;
}

export function useResources(params: ResourceListParams) {
  const query = new URLSearchParams();
  query.set("page", String(params.page ?? 1));
  query.set("page_size", String(params.pageSize ?? 20));
  if (params.search) query.set("search", params.search);
  if (params.status) query.set("status", params.status);
  if (params.resourceTypeId) query.set("resource_type_id", params.resourceTypeId);
  const queryString = query.toString();

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
    mutationFn: (input: MonitorV2Input) => resourcesApi.createMonitor(resourceId, input),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["resources", resourceId, "monitors"] }); },
  });
}

export function useUpdateResourceMonitor(resourceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...input }: MonitorV2Input & { id: string }) => resourcesApi.updateMonitor(resourceId, id, input),
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

export type MetricsRange = "15m" | "1h" | "6h" | "24h" | "7d" | "30d";

const RANGE_MILLIS: Record<MetricsRange, number> = {
  "15m": 15 * 60 * 1000,
  "1h": 60 * 60 * 1000,
  "6h": 6 * 60 * 60 * 1000,
  "24h": 24 * 60 * 60 * 1000,
  "7d": 7 * 24 * 60 * 60 * 1000,
  "30d": 30 * 24 * 60 * 60 * 1000,
};

export function useResourceMonitorMetrics(
  resourceId: string | undefined,
  monitorId: string | undefined,
  range: MetricsRange,
) {
  return useQuery({
    queryKey: ["resources", resourceId, "monitors", monitorId, "metrics", range],
    queryFn: () => {
      const to = new Date();
      const from = new Date(to.getTime() - RANGE_MILLIS[range]);
      const query = new URLSearchParams({
        from: from.toISOString(),
        to: to.toISOString(),
        step: "auto",
      });
      return resourcesApi.getMonitorMetrics(resourceId!, monitorId!, query.toString());
    },
    enabled: Boolean(resourceId) && Boolean(monitorId),
    refetchInterval: range === "15m" || range === "1h" ? 15_000 : 60_000,
    placeholderData: (prev) => prev,
  });
}
