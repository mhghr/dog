export interface AlertPolicyScope {
  monitor_ids?: string[];
  tags?: Record<string, string>;
}

export interface AlertConditions {
  consecutive_failures?: number;
  high_latency_ms?: number;
  packet_loss_percent?: number;
  ssl_expiring_days?: number;
  domain_expiring_days?: number;
  smtp_starttls_fail?: boolean;
  ntp_offset_ms?: number;
  dns_mismatch?: boolean;
}

export interface AlertPolicy {
  id: string;
  organization_id: string;
  project_id: string;
  name: string;
  scope: AlertPolicyScope;
  conditions: AlertConditions;
  severity: "info" | "warning" | "critical";
  opening_failures: number;
  resolving_successes: number;
  cooldown_seconds: number;
  renotify_seconds: number;
  enabled: boolean;
  channel_ids: string[];
  created_at: string;
  updated_at: string;
}

export type AlertState = "pending" | "firing" | "recovering" | "resolved" | "suppressed";

export interface Alert {
  id: string;
  organization_id: string;
  policy_id: string;
  monitor_id: string;
  state: AlertState;
  severity: "info" | "warning" | "critical";
  title: string;
  description: string;
  dedup_key: string;
  consecutive_failures: number;
  consecutive_successes: number;
  opened_at: string | null;
  resolved_at: string | null;
  created_at: string;
}

export interface NotificationChannel {
  id: string;
  organization_id: string;
  name: string;
  type: "email" | "webhook" | "telegram";
  config: Record<string, string>;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface AlertListResponse {
  items: Alert[];
}

export interface AlertPolicyListResponse {
  items: AlertPolicy[];
}

export interface NotificationChannelListResponse {
  items: NotificationChannel[];
}
