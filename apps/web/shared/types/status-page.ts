export type PublicOverallStatus =
  | "operational"
  | "partial_outage"
  | "major_outage";

export interface StatusPageComponentConfig {
  id: string;
  monitor_id: string;
  monitor_name: string;
  display_name: string;
  sort_order: number;
}

export interface StatusPage {
  id: string;
  slug: string;
  name: string;
  description: string;
  enabled: boolean;
  component_count: number;
  components?: StatusPageComponentConfig[];
  created_at: string;
  updated_at: string;
}

export interface StatusPageInput {
  slug: string;
  name: string;
  description: string;
  enabled: boolean;
  components: { monitor_id: string; display_name: string }[];
}

export interface PublicStatusComponent {
  name: string;
  status: "up" | "down" | "unknown" | "paused";
  uptime_24h: number | null;
  uptime_7d: number | null;
  uptime_30d: number | null;
}

export interface PublicStatusPage {
  name: string;
  description: string;
  status: PublicOverallStatus;
  components: PublicStatusComponent[];
  checked_at: string;
}
