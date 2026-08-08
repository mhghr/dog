export interface Tag {
  id: string;
  organization_id: string;
  key: string;
  value: string;
}

export interface TagInput {
  key: string;
  value: string;
}

export interface ResourceType {
  id: string;
  name: string;
  category: string;
  slug: string;
  icon: string;
  capabilities: string[];
  configuration_schema: Record<string, unknown>;
  created_at: string;
}

export interface Resource {
  id: string;
  organization_id: string;
  workspace_id?: string | null;
  resource_type_id: string;
  created_by?: string | null;
  name: string;
  description: string;
  target: string;
  status: string;
  metadata: Record<string, unknown>;
  type_name?: string;
  type_category?: string;
  type_icon?: string;
  monitors_count?: number;
  health_status?: string;
  health_score?: number;
  tags?: Tag[];
  created_at: string;
  updated_at: string;
}

export interface ResourceInput {
  workspace_id?: string | null;
  resource_type_id: string;
  name: string;
  description?: string;
  target?: string;
  metadata?: Record<string, unknown>;
  tags?: TagInput[];
}

export interface MonitorTypeDef {
  id: string;
  name: string;
  slug: string;
  category: string;
  execution_type: string;
  executor_key: string;
  description: string;
  icon: string;
  enabled: boolean;
  metric_keys: string[];
  config_schema: Record<string, unknown>;
  default_configuration: Record<string, unknown>;
  metric_schema: Record<string, unknown>;
  health_parameters: Record<string, unknown>;
  supported_resource_types: string[];
  created_at: string;
  updated_at: string;
}
