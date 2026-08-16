import type { MonitorType } from "@/entities/monitor/model/types";

// MonitorFormValues is the shape of the monitor create/edit form. It lives in
// the domain model so the monitoring plugin definitions and the form schema
// can both reference it without importing the form module (which would create
// an import cycle between the plugin registry and the form schema).
export interface MonitorFormValues {
  name: string;
  type: MonitorType;
  target: string;
  interval_seconds: number;
  timeout_millis: number;
  retries: number;
  enabled: boolean;

  warning_duration_millis?: number;
  critical_duration_millis?: number;

  http_method?: string;
  http_expected_status_codes?: string;
  http_body_contains?: string;
  http_body?: string;
  http_headers?: string;
  http_follow_redirects?: boolean;
  http_max_redirects?: number;
  http_verify_tls?: boolean;

  tcp_port?: number;

  dns_server?: string;
  dns_record_type?: string;
  dns_expected_values?: string;

  ping_packet_count?: number;
  ping_packet_interval_millis?: number;
  ping_warning_latency_millis?: number;
  ping_critical_latency_millis?: number;
  ping_warning_packet_loss_percent?: number;
  ping_critical_packet_loss_percent?: number;
  ping_warning_jitter_millis?: number;
  ping_critical_jitter_millis?: number;

  tls_port?: number;
  tls_server_name?: string;
  tls_verify_chain?: boolean;
  tls_verify_hostname?: boolean;
  tls_min_version?: string;
  tls_warning_days?: number;
  tls_critical_days?: number;
  tls_expected_issuer?: string;
  tls_expected_fingerprint?: string;

  domain_warning_days?: number;
  domain_critical_days?: number;
  domain_check_nameservers?: boolean;
  domain_expected_registrar?: string;
  domain_expected_nameservers?: string;

  smtp_port?: number;
  smtp_mode?: string;
  smtp_ehlo_domain?: string;
  smtp_require_starttls?: boolean;
  smtp_verify_tls?: boolean;
  smtp_expected_banner?: string;
  smtp_expected_capabilities?: string;

  ntp_port?: number;
  ntp_version?: number;
  ntp_max_offset_millis?: number;
  ntp_max_round_trip_millis?: number;
  ntp_stratum_min?: number;
  ntp_stratum_max?: number;
  ntp_warning_offset_millis?: number;
  ntp_warning_round_trip_millis?: number;
}
