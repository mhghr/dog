// Domain event names shared across the realtime layer (SSE + WebSocket).
// Backend event names map to these constants; components and the query
// invalidation layer listen for them.
export const REAL_TIME_EVENTS = {
  probeResult: "probe-result",
  resourceUpdated: "resource.updated",
  monitorStatusChanged: "monitor.status.changed",
  alertCreated: "alert.created",
  agentHeartbeat: "agent.heartbeat",
} as const;

export type RealtimeEventName =
  (typeof REAL_TIME_EVENTS)[keyof typeof REAL_TIME_EVENTS];
