// Centralized TCP health evaluation.
//
// The backend `last_status` is authoritative for the "down" state. For the
// richer Healthy / Warning / Critical gradation, the worst metric condition
// (compared against configured thresholds) wins — we never average states.
// This mirrors the Ping/HTTP evaluators.

import type { TcpThresholds } from "./tcp-config";
import type { StatusTone } from "@/design-system/components/status-badge";

export type TcpHealthState =
  | "healthy"
  | "warning"
  | "critical"
  | "down"
  | "unknown";

export interface TcpHealthInput {
  /** monitor.last_status */
  lastStatus?: string;
  /** whether the TCP connection was established */
  success?: boolean;
  /** connect duration, may be null on failure */
  connectTimeMs?: number | null;
  thresholds: TcpThresholds;
}

const RANK: Record<TcpHealthState, number> = {
  unknown: 0,
  healthy: 1,
  warning: 2,
  critical: 3,
  down: 4,
};

function worst(
  current: TcpHealthState,
  candidate: TcpHealthState,
): TcpHealthState {
  return RANK[candidate] > RANK[current] ? candidate : current;
}

/** Compare a higher-is-worse metric against its threshold; no value → healthy. */
export function compareMetric(
  value: number | null | undefined,
  threshold: { warning?: number; critical?: number } | undefined,
): TcpHealthState {
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
): TcpHealthState {
  if (value == null) return "unknown";
  return compareMetric(value, threshold);
}

export function evaluateAvailability(success: boolean | undefined): TcpHealthState {
  if (success === undefined) return "unknown";
  return success ? "healthy" : "critical";
}

export function evaluateTcpHealth(input: TcpHealthInput): TcpHealthState {
  const { lastStatus, success, connectTimeMs, thresholds } = input;

  if (lastStatus === "down") return "down";
  if (lastStatus === "paused") return "unknown";

  if (success === false) return "critical";

  if (connectTimeMs == null) {
    return lastStatus === "up" ? "healthy" : "unknown";
  }

  let state: TcpHealthState = "healthy";
  state = worst(state, compareMetric(connectTimeMs, thresholds.connectTime));
  return state;
}

export function tcpHealthTone(state: TcpHealthState): StatusTone {
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
