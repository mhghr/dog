// Per-domain query key factories.
//
// Every domain exposes a flat `all` key plus `list`/`detail` derived keys so
// invalidations can target whole families. Keys are scoped by workspace so a
// multi-tenant session never leaks data between workspaces.
//
// Usage:
//   useQuery({ queryKey: resourceKeys.list(workspaceId, params), ... })
//   queryClient.invalidateQueries({ queryKey: resourceKeys.all(workspaceId) })

export type WorkspaceScope = string;

const scope = (workspaceId?: WorkspaceScope) => workspaceId ?? "*";

export const workspaceKeys = {
  all: () => ["workspaces"] as const,
  detail: (id: string) => ["workspaces", id] as const,
};

export const resourceKeys = {
  all: (workspaceId?: WorkspaceScope) =>
    ["workspaces", scope(workspaceId), "resources"] as const,
  lists: (workspaceId?: WorkspaceScope) =>
    ["workspaces", scope(workspaceId), "resources", "list"] as const,
  list: (workspaceId: WorkspaceScope, params: Record<string, unknown>) =>
    [
      "workspaces",
      workspaceId,
      "resources",
      "list",
      JSON.stringify(params),
    ] as const,
  detail: (workspaceId: WorkspaceScope, id: string) =>
    ["workspaces", workspaceId, "resources", "detail", id] as const,
  overview: (workspaceId: WorkspaceScope) =>
    ["workspaces", workspaceId, "resources", "overview"] as const,
  types: () => ["resource-types"] as const,
  monitors: (workspaceId: WorkspaceScope, resourceId: string) =>
    ["workspaces", workspaceId, "resources", "detail", resourceId, "monitors"] as const,
  monitorResults: (
    workspaceId: WorkspaceScope,
    resourceId: string,
    monitorId: string,
  ) =>
    [
      "workspaces",
      workspaceId,
      "resources",
      "detail",
      resourceId,
      "monitors",
      monitorId,
      "results",
    ] as const,
  monitorMetrics: (
    workspaceId: WorkspaceScope,
    resourceId: string,
    monitorId: string,
    range: string,
  ) =>
    [
      "workspaces",
      workspaceId,
      "resources",
      "detail",
      resourceId,
      "monitors",
      monitorId,
      "metrics",
      range,
    ] as const,
};

export const monitorKeys = {
  all: () => ["monitors"] as const,
  lists: () => ["monitors", "list"] as const,
  list: (params: Record<string, unknown>) =>
    ["monitors", "list", JSON.stringify(params)] as const,
  detail: (id: string) => ["monitors", id] as const,
  results: (id: string, params: Record<string, unknown>) =>
    ["monitors", id, "results", JSON.stringify(params)] as const,
  metrics: (id: string, range: string) =>
    ["monitors", id, "metrics", range] as const,
  types: () => ["monitor-types"] as const,
  health: {
    rules: (id: string) => ["monitors", id, "health", "rules"] as const,
    states: (id: string) => ["monitors", id, "health", "states"] as const,
    policies: (id: string) => ["monitors", id, "health", "policies"] as const,
  },
};

export const alertKeys = {
  all: () => ["alerts"] as const,
  list: (params: Record<string, unknown>) =>
    ["alerts", JSON.stringify(params)] as const,
  policies: () => ["alert-policies"] as const,
  channels: () => ["notification-channels"] as const,
};

export const agentKeys = {
  all: (status?: string) =>
    status ? (["agents", status] as const) : (["agents"] as const),
  tokens: () => ["probe-tokens"] as const,
  locations: () => ["probe-locations"] as const,
};

export const statusPageKeys = {
  all: () => ["status-pages"] as const,
  detail: (id: string) => ["status-pages", id] as const,
};

export const authKeys = {
  me: () => ["auth", "me"] as const,
};

export const dashboardKeys = {
  summary: () => ["dashboard", "summary"] as const,
};

export const systemKeys = {
  health: () => ["system", "health"] as const,
};

export const organizationKeys = {
  projects: () => ["organization", "projects"] as const,
};

export const parameterKeys = {
  catalog: () => ["parameter-catalog"] as const,
  rules: (monitorId: string) => ["parameter-rules", monitorId] as const,
  states: (monitorId: string) => ["parameter-health-states", monitorId] as const,
  policies: (monitorId: string) => ["notification-policies", monitorId] as const,
};
