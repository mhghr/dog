"use client";

import { useQuery } from "@tanstack/react-query";

import { monitorApi } from "@/entities/monitor/api/monitor.api";
import type { MonitorListResponse } from "@/shared/types/api";
import type { MonitorStatus, MonitorType } from "@/entities/monitor/model/types";

export interface MonitorListParams {
  page?: number;
  pageSize?: number;
  type?: MonitorType | "all";
  status?: MonitorStatus | "all";
  search?: string;
  sort?: string;
  order?: "asc" | "desc";
}

export function monitorListQueryString(params: MonitorListParams): string {
  const query = new URLSearchParams();

  query.set("page", String(params.page ?? 1));
  query.set("page_size", String(params.pageSize ?? 20));

  if (params.type && params.type !== "all") query.set("type", params.type);
  if (params.status && params.status !== "all") query.set("status", params.status);
  if (params.search) query.set("search", params.search);
  if (params.sort) query.set("sort", params.sort);
  if (params.order) query.set("order", params.order);

  return query.toString();
}

export function useMonitors(params: MonitorListParams) {
  const queryString = monitorListQueryString(params);

  return useQuery({
    queryKey: ["monitors", "list", queryString],
    queryFn: () => monitorApi.list(queryString),
    refetchInterval: 15_000,
    placeholderData: (previous) => previous,
  });
}

export function useMonitor(monitorId: string) {
  return useQuery({
    queryKey: ["monitors", monitorId],
    queryFn: () => monitorApi.getById(monitorId),
    enabled: Boolean(monitorId),
    refetchInterval: 10_000,
  });
}
