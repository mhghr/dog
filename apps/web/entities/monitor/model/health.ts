export type HealthState = "OK" | "WARNING" | "ERROR" | "UNKNOWN";

export type RuleMode = "INHERIT_DEFAULT" | "USE_PROFILE" | "CUSTOM" | "DISABLED";

export type RuleProfile = "Sensitive" | "Recommended" | "Relaxed";

export type ParameterDirection =
  | "HIGHER_IS_WORSE"
  | "LOWER_IS_WORSE"
  | "BOOLEAN_FAILURE"
  | "ENUM_STATE"
  | "RANGE_DEVIATION"
  | "CHANGE_EVENT"
  | "RATE"
  | "COUNT";

export interface ParameterDefinition {
  key: string;
  name: string;
  description: string;
  data_type: string;
  unit: string;
  direction: ParameterDirection;
  default_profile: RuleProfile;
}

export interface ParameterRule {
  id?: string;
  monitor_id: string;
  parameter_key: string;
  mode: RuleMode;
  profile?: RuleProfile;
  aggregation: string;
  window_type: string;
  window_value: number;
  warning_operator: string;
  warning_value?: number;
  error_operator: string;
  error_value?: number;
  recovery_operator?: string;
  recovery_value?: number;
  minimum_samples: number;
  consecutive_failures?: number;
  consecutive_successes?: number;
  missing_data_policy: string;
  missed_checks: number;
  cooldown_seconds: number;
  enabled: boolean;
}

export interface ParameterHealthState {
  monitor_id: string;
  parameter_key: string;
  current_state: HealthState;
  current_value?: number;
  evaluated_at?: string;
  previous_state?: HealthState;
  state_changed_at?: string;
}

export type NotificationChannelType =
  | "email"
  | "telegram"
  | "slack"
  | "discord"
  | "teams"
  | "webhook";

export interface NotificationChannel {
  id: string;
  name: string;
  type: NotificationChannelType;
  config: Record<string, unknown>;
  enabled: boolean;
}

export type NotificationTrigger =
  | "STATUS_ENTERED_WARNING"
  | "STATUS_ENTERED_ERROR"
  | "STATUS_ENTERED_UNKNOWN"
  | "RECOVERED_TO_OK"
  | "DEGRADED_FROM_ERROR_TO_WARNING"
  | "REPEATED_WARNING"
  | "REPEATED_ERROR"
  | "NO_DATA"
  | "FLAPPING_DETECTED";

export interface NotificationPolicy {
  id?: string;
  monitor_id: string;
  parameter_key?: string;
  channel_id: string;
  triggers: NotificationTrigger[];
  delay_seconds: number;
  repeat_interval_seconds: number;
  cooldown_seconds: number;
  enabled: boolean;
}
