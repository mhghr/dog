// Centralized HTTP health evaluation.
//
// The backend `last_status` is authoritative for the "down" state. For the
// richer Healthy / Warning / Critical gradation, the worst metric condition
// (compared against configured thresholds) wins — we never average states.
// This mirrors the Ping evaluator (see monitoring/ping/ping-health.ts).

import type { HttpThresholds } from "./http-config";
import type { StatusTone } from "@/design-system/components/status-badge";

export type HttpHealthState =
  | "healthy"
  | "warning"
  | "critical"
  | "down"
  | "unknown";

export interface HttpHealthInput {
  /** monitor.last_status */
  lastStatus?: string;
  /** whether the last response reached the server and passed validation */
  success?: boolean;
  /** HTTP status code, may be null on transport failure */
  statusCode?: number | null;
  /** total response time, may be null on transport failure */
  responseTimeMs?: number | null;
  /** TTFB, may be null on transport failure */
  ttfbMs?: number | null;
  /** content assertion result: 1 matched, 0 failed, null unknown */
  contentAssertion?: number | null;
  thresholds: HttpThresholds;
}

const RANK: Record<HttpHealthState, number> = {
  unknown: 0,
  healthy: 1,
  warning: 2,
  critical: 3,
  down: 4,
};

function worst(
  current: HttpHealthState,
  candidate: HttpHealthState,
): HttpHealthState {
  return RANK[candidate] > RANK[current] ? candidate : current;
}

/** Compare a metric against its threshold; no value → healthy. */
export function compareMetric(
  value: number | null | undefined,
  threshold: { warning?: number; critical?: number } | undefined,
): HttpHealthState {
  if (value == null || threshold == null) return "healthy";
  const critical = threshold.critical;
  const warning = threshold.warning;

  if (critical != null && value >= critical) return "critical";
  if (warning != null && value >= warning) return "warning";
  return "healthy";
}

/** Evaluate a single metric against its threshold (no data → unknown). */
export function evaluateMetric(
  value: number | null | undefined,
  threshold: { warning?: number; critical?: number } | undefined,
): HttpHealthState {
  if (value == null) return "unknown";
  return compareMetric(value, threshold);
}

/**
 * Availability has no configured threshold — the backend reports
 * reachability as a boolean. A failed request means unavailable (critical);
 * a successful response means available.
 */
export function evaluateAvailability(success: boolean | undefined): HttpHealthState {
  if (success === undefined) return "unknown";
  return success ? "healthy" : "critical";
}

export function evaluateHttpHealth(input: HttpHealthInput): HttpHealthState {  const { lastStatus, success, statusCode, responseTimeMs, ttfbMs, contentAssertion, thresholds } = input;

  // Backend status is authoritative for down/paused/unknown.
  if (lastStatus === "down") return "down";
  if (lastStatus === "paused") return "unknown";

  // An explicit transport failure or failed assertion is always critical.
  if (success === false) return "critical";
  if (contentAssertion === 0) return "critical";

  // No metric data at all → unknown, never a fake "healthy".
  if (statusCode == null && responseTimeMs == null && ttfbMs == null) {
    return lastStatus === "up" ? "healthy" : "unknown";
  }

  let state: HttpHealthState = "healthy";
  state = worst(state, compareMetric(responseTimeMs, thresholds.responseTime));
  state = worst(state, compareMetric(ttfbMs, thresholds.ttfb));

  return state;
}

export function httpHealthTone(state: HttpHealthState): StatusTone {
  switch (state) {
    case "healthy":
      return "success";
    case "warning":
      return "warning";
    case "critical":
      return "destructive";
    case "down":
      return "destructive";
    case "unknown":
      return "muted";
  }
}
