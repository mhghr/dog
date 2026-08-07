export const MONITOR_TYPE_VALUES = [
  "http", "tcp", "dns", "ping", "tls", "domain_expiration", "smtp", "ntp",
] as const;

export type MonitorType = (typeof MONITOR_TYPE_VALUES)[number];

export type MonitorStatus = "up" | "down" | "unknown" | "paused";

export interface LastResultSummary {
  success: boolean;
  duration_millis: number;
  error_code: string | null;
  metrics?: Record<string, number>;
}

export interface Monitor {
  id: string;
  name: string;
  type: MonitorType;
  target: string;
  interval_seconds: number;
  timeout_millis: number;
  retries: number;
  enabled: boolean;
  config: Record<string, unknown>;
  last_status: MonitorStatus;
  last_checked_at: string | null;
  next_run_at: string;
  created_at: string;
  updated_at: string;
  last_result?: LastResultSummary | null;
}

export interface CreateMonitorInput {
  name: string;
  type: MonitorType;
  target: string;
  interval_seconds: number;
  timeout_millis: number;
  retries: number;
  enabled: boolean;
  config: Record<string, unknown>;
}

export interface ProbeLocation {
  id: string;
  name: string;
  code: string;
  enabled: boolean;
  created_at: string;
}
