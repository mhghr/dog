// Pure query-key and query-string builders for the resource entity.
// This module intentionally has NO "use client" directive so it can be
// imported by both server components (prefetch) and client hooks.

import type { Monitor } from "@/entities/resource/hooks/types";
import type { MonitorTypeDef } from "@/entities/resource/model/types";

export type MetricsRange = "15m" | "1h" | "6h" | "24h" | "7d" | "30d";

export const RANGE_MILLIS: Record<MetricsRange, number> = {
  "15m": 15 * 60 * 1000,
  "1h": 60 * 60 * 1000,
  "6h": 6 * 60 * 60 * 1000,
  "24h": 24 * 60 * 60 * 1000,
  "7d": 7 * 24 * 60 * 60 * 1000,
  "30d": 30 * 24 * 60 * 60 * 1000,
};

export interface ResourceListParams {
  page?: number;
  pageSize?: number;
  search?: string;
  status?: string;
  resourceTypeId?: string;
}

// Shared query-string builder for the resource list, so the server-side
// prefetch and the client hook resolve the same cache entry.
export function resourceListQueryString(params: ResourceListParams): string {
  const query = new URLSearchParams();
  query.set("page", String(params.page ?? 1));
  query.set("page_size", String(params.pageSize ?? 20));
  if (params.search) query.set("search", params.search);
  if (params.status) query.set("status", params.status);
  if (params.resourceTypeId) query.set("resource_type_id", params.resourceTypeId);
  return query.toString();
}

// Shared query key builder so the server-side prefetch and the client hook
// resolve the same cache entry for a monitor's metric series.
export function resourceMonitorMetricsQueryKey(
  resourceId: string | undefined,
  monitorId: string | undefined,
  range: MetricsRange,
  metric?: string,
) {
  return [
    "resources",
    resourceId,
    "monitors",
    monitorId,
    "metrics",
    range,
    metric ?? "",
  ];
}

// Shared query-string builder for the metrics endpoint, so server prefetch and
// client fetch produce identical request URLs.
export function buildMetricsQueryString(
  range: MetricsRange,
  metric?: string,
): string {
  const to = new Date();
  const from = new Date(to.getTime() - RANGE_MILLIS[range]);
  const query = new URLSearchParams({
    from: from.toISOString(),
    to: to.toISOString(),
    step: "auto",
  });
  if (metric) query.set("metric", metric);
  return query.toString();
}

// Classifies a resource monitor as a ping monitor by its monitor type.
export function isPingMonitor(
  monitor: Monitor,
  types: MonitorTypeDef[],
): boolean {
  const type = types.find((t) => t.id === monitor.monitor_type_id);
  return (
    type?.executor_key === "ping" ||
    type?.slug === "ping" ||
    type?.name?.toLowerCase() === "ping"
  );
}

// Classifies a resource monitor as an HTTP monitor by its monitor type.
export function isHttpMonitor(
  monitor: Monitor,
  types: MonitorTypeDef[],
): boolean {
  const type = types.find((t) => t.id === monitor.monitor_type_id);
  return (
    type?.executor_key === "http" ||
    type?.slug === "http" ||
    (type?.name?.toLowerCase().includes("http") ?? false)
  );
}

// Classifies a resource monitor as a TCP port monitor by its monitor type.
export function isTcpMonitor(
  monitor: Monitor,
  types: MonitorTypeDef[],
): boolean {
  const type = types.find((t) => t.id === monitor.monitor_type_id);
  return (
    type?.executor_key === "tcp" ||
    type?.slug === "tcp" ||
    (type?.name?.toLowerCase().includes("tcp") ?? false)
  );
}

// Classifies a resource monitor as a DNS monitor by its monitor type.
export function isDnsMonitor(
  monitor: Monitor,
  types: MonitorTypeDef[],
): boolean {
  const type = types.find((t) => t.id === monitor.monitor_type_id);
  return (
    type?.executor_key === "dns" ||
    type?.slug === "dns" ||
    type?.name?.toLowerCase() === "dns resolution" ||
    (type?.name?.toLowerCase().includes("dns") ?? false)
  );
}

// Classifies a resource monitor as a TLS/SSL certificate monitor. The seeded
// monitor type uses slug "ssl" with executor_key "tls".
export function isTlsMonitor(
  monitor: Monitor,
  types: MonitorTypeDef[],
): boolean {
  const type = types.find((t) => t.id === monitor.monitor_type_id);
  return (
    type?.executor_key === "tls" ||
    type?.slug === "ssl" ||
    type?.slug === "tls" ||
    (type?.name?.toLowerCase().includes("ssl") ?? false) ||
    (type?.name?.toLowerCase().includes("tls") ?? false) ||
    (type?.name?.toLowerCase().includes("certificate") ?? false)
  );
}
