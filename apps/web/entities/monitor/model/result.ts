import type { MonitorStatus } from "./types";

export interface ProbeResult {
  id: string;
  job_id: string;
  monitor_id: string;
  probe_location_id: string;
  status: MonitorStatus;
  success: boolean;
  error_code?: string;
  error_message?: string;
  duration_millis: number;
  metrics: Record<string, number | string | boolean>;
  attributes: Record<string, unknown>;
  started_at: string;
  finished_at: string;
}

export interface MetricPoint {
  timestamp: string;
  value: number;
}

export interface ProbeSeries {
  probe_id: string;
  probe_name: string;
  location: string;
  metric_key: string;
  points: MetricPoint[];
  values: MetricPoint[];
}

export interface MonitorMetrics {
  series: {
    latency: MetricPoint[];
    success: MetricPoint[];
  };
  summary: {
    uptime_percent: number | null;
    p50_latency_ms: number | null;
    p95_latency_ms: number | null;
    p99_latency_ms: number | null;
  };
  step_seconds: number;
  from: string;
  to: string;
}

export interface LiveProbeEvent {
  monitor_id: string;
  status: MonitorStatus;
  success: boolean;
  duration_ms: number;
  error_code?: string;
  timestamp: string;
}
