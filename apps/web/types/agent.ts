export type AgentStatus =
  | "pending"
  | "approved"
  | "active"
  | "offline"
  | "disabled"
  | "rejected"
  | "revoked"
  | "draining"
  | "updating";

export interface ProbeAgent {
  id: string;
  location_id: string;
  name: string;
  hostname: string;
  version: string;
  operating_system: string;
  architecture: string;
  public_ip: string;
  capabilities: string[];
  max_concurrency: number;
  running_jobs: number;
  status: AgentStatus;
  last_seen_at?: string;
  created_at: string;
  latitude?: number | null;
  longitude?: number | null;
  city?: string;
  country?: string;
}

export interface AgentListResponse {
  items: ProbeAgent[];
}

export interface EnrollmentToken {
  token: string;
  location_id: string;
  expires_at: string;
}

export interface CreateTokenInput {
  location_code: string;
  ttl_minutes: number;
}

export interface UnusedToken {
  id: string;
  token_label: string;
  location_id: string;
  expires_at: string;
  created_at: string;
}

export interface TokenListResponse {
  items: UnusedToken[];
}
