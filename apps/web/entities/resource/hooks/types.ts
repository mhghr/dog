import type { MonitorStatus } from "@/entities/monitor/model/types";

export interface Monitor {
  id: string;
  resource_id: string;
  monitor_type_id: string;
  health_profile_id?: string | null;
  created_by?: string | null;
  name: string;
  enabled: boolean;
  configuration: Record<string, unknown>;
  severity: string;
  interval_seconds: number;
  timeout_millis: number;
  retries: number;
  last_status: MonitorStatus;
  last_checked_at?: string | null;
  next_run_at: string;
  created_at: string;
  updated_at: string;
  resource_target?: string;
}

export interface MonitorInput {
  monitor_type_id: string;
  health_profile_id?: string | null;
  name: string;
  enabled?: boolean;
  configuration?: Record<string, unknown>;
  severity?: string;
  interval_seconds: number;
  timeout_millis: number;
  retries: number;
}
