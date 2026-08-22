// Central endpoint registry.
// All backend paths used by the web app live here so the HTTP surface is
// discoverable and consistent. Domain API objects (entities/*/api) consume
// these path builders; UI code must never hardcode `/api/...` strings.

const api = "/api" as const;

export const endpoints = {
  auth: {
    me: `${api}/auth/me`,
    refresh: `${api}/auth/refresh`,
    logout: `${api}/auth/logout`,
    otp: {
      request: `${api}/auth/otp/request`,
      verify: `${api}/auth/otp/verify`,
    },
  },

  organization: {
    create: `${api}/organizations`,
    projects: `${api}/organizations/projects`,
  },

  workspace: {
    list: `${api}/workspaces`,
  },

  resource: {
    types: `${api}/resource-types`,
    overview: `${api}/resources/overview`,
    list: `${api}/resources`,
    byId: (id: string) => `${api}/resources/${id}`,
    overviewById: (id: string) => `${api}/resources/${id}/overview`,
    monitors: (resourceId: string) => `${api}/resources/${resourceId}/monitors`,
    monitor: (resourceId: string, monitorId: string) =>
      `${api}/resources/${resourceId}/monitors/${monitorId}`,
    monitorResults: (resourceId: string, monitorId: string) =>
      `${api}/resources/${resourceId}/monitors/${monitorId}/results`,
    monitorMetrics: (resourceId: string, monitorId: string) =>
      `${api}/resources/${resourceId}/monitors/${monitorId}/metrics`,
    snmp: {
      test: (resourceId: string, monitorId: string) =>
        `${api}/resources/${resourceId}/monitors/${monitorId}/snmp/test`,
      discover: (resourceId: string, monitorId: string) =>
        `${api}/resources/${resourceId}/monitors/${monitorId}/snmp/discover`,
      discovery: (resourceId: string, monitorId: string) =>
        `${api}/resources/${resourceId}/monitors/${monitorId}/snmp/discovery`,
      interfaces: (resourceId: string, monitorId: string) =>
        `${api}/resources/${resourceId}/monitors/${monitorId}/snmp/interfaces`,
      interface: (resourceId: string, monitorId: string, ifIndex: number) =>
        `${api}/resources/${resourceId}/monitors/${monitorId}/snmp/interfaces/${ifIndex}`,
      events: (resourceId: string, monitorId: string) =>
        `${api}/resources/${resourceId}/monitors/${monitorId}/snmp/events`,
      diagnostics: (resourceId: string, monitorId: string) =>
        `${api}/resources/${resourceId}/monitors/${monitorId}/snmp/diagnostics`,
    },
    snmpTasks: {
      task: (taskId: string) => `${api}/snmp/tasks/${taskId}`,
      apply: (taskId: string) => `${api}/snmp/tasks/${taskId}/apply`,
    },
    snmpSourceIps: `${api}/snmp/source-ips`,
  },

  monitor: {
    types: `${api}/monitor-types`,
    typeParameters: (type: string) => `${api}/monitor-types/${type}/parameters`,
    list: `${api}/monitors`,
    byId: (id: string) => `${api}/monitors/${id}`,
    pause: (id: string) => `${api}/monitors/${id}/pause`,
    resume: (id: string) => `${api}/monitors/${id}/resume`,
    metrics: (id: string) => `${api}/monitors/${id}/metrics`,
    results: (id: string) => `${api}/monitors/${id}/results`,
    health: {
      rules: (id: string) => `${api}/monitors/${id}/health/rules`,
      states: (id: string) => `${api}/monitors/${id}/health/states`,
      policies: `${api}/monitors/health/policies`,
      policy: (id: string) => `${api}/monitors/health/policies/${id}`,
      monitorPolicies: (id: string) => `${api}/monitors/${id}/health/policies`,
    },
  },

  alerting: {
    policies: `${api}/alerting/policies`,
    alerts: `${api}/alerting/alerts`,
    channels: `${api}/alerting/channels`,
  },

  probe: {
    locations: `${api}/probe-locations`,
  },

  geoip: (ip: string) => `${api}/geoip?ip=${encodeURIComponent(ip)}`,

  agent: {
    list: (params: string) => `/api/admin/probe-agents${params}`,
    byId: (id: string) => `${api}/admin/probe-agents/${id}`,
    approve: (id: string) => `${api}/admin/probe-agents/${id}/approve`,
    reject: (id: string) => `${api}/admin/probe-agents/${id}/reject`,
    disable: (id: string) => `${api}/admin/probe-agents/${id}/disable`,
    enable: (id: string) => `${api}/admin/probe-agents/${id}/enable`,
    revoke: (id: string) => `${api}/admin/probe-agents/${id}/revoke`,
    drain: (id: string) => `${api}/admin/probe-agents/${id}/drain`,
    publicIp: (id: string) => `${api}/admin/probe-agents/${id}/public-ip`,
    location: (id: string) => `${api}/admin/probe-agents/${id}/location`,
    enrollmentTokens: `${api}/admin/probe-agent-enrollment-tokens`,
  },

  statusPage: {
    list: `${api}/status-pages`,
    byId: (id: string) => `${api}/status-pages/${id}`,
  },

  dashboard: {
    summary: `${api}/dashboard/summary`,
  },

  system: {
    health: `${api}/system/health`,
  },
} as const;
