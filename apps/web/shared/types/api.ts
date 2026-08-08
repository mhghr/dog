import type { Monitor } from "@/entities/monitor/model/types";
import type { ProbeLocation } from "@/entities/probe/model/types";
import type { ProbeResult } from "@/entities/monitor/model/result";

export interface Pagination {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

export interface MonitorListResponse {
  items: Monitor[];
  pagination: Pagination;
}

export interface ResultListResponse {
  items: ProbeResult[];
  pagination: Pagination;
}

export interface LocationListResponse {
  items: ProbeLocation[];
}

export interface RecentFailure {
  monitor_id: string;
  monitor_name: string;
  monitor_type: string;
  error_code: string | null;
  error_message: string | null;
  started_at: string;
}

export interface SlowMonitor {
  monitor_id: string;
  monitor_name: string;
  monitor_type: string;
  duration_millis: number;
}

export interface DashboardSummary {
  total_monitors: number;
  status_counts: Record<"up" | "down" | "unknown" | "paused", number>;
  availability_24h: number | null;
  checks_24h: {
    successful: number;
    failed: number;
  };
  recent_failures: RecentFailure[];
  slowest_monitors: SlowMonitor[];
  attention_required: {
    certificates_expiring_30d: number;
    domains_expiring_45d: number;
    smtp_starttls_failures: number;
    ntp_high_offset: number;
  };
}

export interface ComponentHealth {
  name: string;
  status: "healthy" | "unhealthy" | "degraded" | "unknown";
  last_seen?: string;
}

export interface SystemHealth {
  status: "healthy" | "degraded" | "unhealthy";
  components: ComponentHealth[];
  workers: ComponentHealth[];
  queue: {
    lag: number;
    pending: number;
  };
  checked_at: string;
}

export interface ApiErrorBody {
  error: {
    code: string;
    message: string;
    fields?: Record<string, string[]>;
    request_id?: string;
  };
}
