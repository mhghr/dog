// Central endpoint registry.
// All backend paths used by the web app live here so the HTTP surface is
// discoverable and consistent. Domain API objects (entities/*/api) consume
// these path builders; UI code must never hardcode `/api/...` strings.

const v1 = "/api/v1" as const;

export const endpoints = {
  auth: {
    me: `${v1}/auth/me`,
    refresh: `${v1}/auth/refresh`,
    logout: `${v1}/auth/logout`,
    otp: {
      request: `${v1}/auth/otp/request`,
      verify: `${v1}/auth/otp/verify`,
    },
  },

  organization: {
    create: `${v1}/organizations`,
    projects: `${v1}/organizations/projects`,
  },

  workspace: {
    list: `${v1}/workspaces`,
  },

  resource: {
    types: `${v1}/resource-types`,
    overview: `${v1}/resources/overview`,
    list: `${v1}/resources`,
    byId: (id: string) => `${v1}/resources/${id}`,
    monitors: (resourceId: string) => `${v1}/resources/${resourceId}/monitors`,
    monitor: (resourceId: string, monitorId: string) =>
      `${v1}/resources/${resourceId}/monitors/${monitorId}`,
    monitorResults: (resourceId: string, monitorId: string) =>
      `${v1}/resources/${resourceId}/monitors/${monitorId}/results`,
    monitorMetrics: (resourceId: string, monitorId: string) =>
      `${v1}/resources/${resourceId}/monitors/${monitorId}/metrics`,
  },

  monitor: {
    types: `${v1}/monitor-types`,
    typeParameters: (type: string) => `${v1}/monitor-types/${type}/parameters`,
    list: `${v1}/monitors`,
    byId: (id: string) => `${v1}/monitors/${id}`,
    pause: (id: string) => `${v1}/monitors/${id}/pause`,
    resume: (id: string) => `${v1}/monitors/${id}/resume`,
    metrics: (id: string) => `${v1}/monitors/${id}/metrics`,
    results: (id: string) => `${v1}/monitors/${id}/results`,
    health: {
      rules: (id: string) => `${v1}/monitors/${id}/health/rules`,
      states: (id: string) => `${v1}/monitors/${id}/health/states`,
      policies: `${v1}/monitors/health/policies`,
      policy: (id: string) => `${v1}/monitors/health/policies/${id}`,
      monitorPolicies: (id: string) => `${v1}/monitors/${id}/health/policies`,
    },
  },

  alerting: {
    policies: `${v1}/alerting/policies`,
    alerts: `${v1}/alerting/alerts`,
    channels: `${v1}/alerting/channels`,
  },

  probe: {
    locations: `${v1}/probe-locations`,
  },

  agent: {
    list: (params: string) => `/api/v1/admin/probe-agents${params}`,
    byId: (id: string) => `${v1}/admin/probe-agents/${id}`,
    approve: (id: string) => `${v1}/admin/probe-agents/${id}/approve`,
    reject: (id: string) => `${v1}/admin/probe-agents/${id}/reject`,
    disable: (id: string) => `${v1}/admin/probe-agents/${id}/disable`,
    enable: (id: string) => `${v1}/admin/probe-agents/${id}/enable`,
    revoke: (id: string) => `${v1}/admin/probe-agents/${id}/revoke`,
    drain: (id: string) => `${v1}/admin/probe-agents/${id}/drain`,
    publicIp: (id: string) => `${v1}/admin/probe-agents/${id}/public-ip`,
    location: (id: string) => `${v1}/admin/probe-agents/${id}/location`,
    enrollmentTokens: `${v1}/admin/probe-agent-enrollment-tokens`,
  },

  statusPage: {
    list: `${v1}/status-pages`,
    byId: (id: string) => `${v1}/status-pages/${id}`,
  },

  dashboard: {
    summary: `${v1}/dashboard/summary`,
  },

  system: {
    health: `${v1}/system/health`,
  },
} as const;
