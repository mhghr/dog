export interface DashboardSummary {
  total_monitors: number;
  status_counts: Record<string, number>;
  availability_24h: number | null;
  checks_24h: Checks24h;
  availability_series: AvailabilityPoint[];
  recent_failures: RecentFailure[];
  slowest_monitors: SlowMonitor[];
  attention_required: AttentionRequired;
}

export interface Checks24h {
  successful: number;
  failed: number;
}

export interface AvailabilityPoint {
  timestamp: string;
  successful: number;
  total: number;
  rate: number;
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

export interface AttentionRequired {
  certificates_expiring_30d: number;
  domains_expiring_45d: number;
  smtp_starttls_failures: number;
  ntp_high_offset: number;
}
