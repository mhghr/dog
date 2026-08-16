// Centralized Ping health evaluation.
//
// The backend `last_status` is authoritative for the "down" state. For the
// richer Healthy / Warning / Critical gradation, the worst metric condition
// (compared against configured thresholds) wins — we never average states.
//
// This single evaluator is the only place that turns raw metrics into a
// normalized state; components must not re-implement this logic.

import type { PingThresholds } from "./ping-config";
import type { StatusTone } from "@/design-system/components/status-badge";

export type PingHealthState =
  | "healthy"
  | "warning"
  | "critical"
  | "down"
  | "unknown";

export interface PingHealthInput {
  /** monitor.last_status */
  lastStatus?: string;
  /** averaged across probe locations, may be null when no data */
  latency?: number | null;
  packetLoss?: number | null;
  jitter?: number | null;
  thresholds: PingThresholds;
}

const RANK: Record<PingHealthState, number> = {
  unknown: 0,
  healthy: 1,
  warning: 2,
  critical: 3,
  down: 4,
};

function worst(
  current: PingHealthState,
  candidate: PingHealthState,
): PingHealthState {
  return RANK[candidate] > RANK[current] ? candidate : current;
}

export function compareMetric(
  value: number | null | undefined,
  threshold: { warning?: number; critical?: number } | undefined,
): PingHealthState {
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
): PingHealthState {
  if (value == null) return "unknown";
  return compareMetric(value, threshold);
}

/** Availability has no configured threshold; use a simple centered rule. */
export function evaluateAvailability(percent: number | null | undefined): PingHealthState {
  if (percent == null) return "unknown";
  if (percent >= 100) return "healthy";
  if (percent >= 99) return "warning";
  return "critical";
}

export function evaluatePingHealth(input: PingHealthInput): PingHealthState {
  const { lastStatus, latency, packetLoss, jitter, thresholds } = input;

  // Backend status is authoritative for down/paused/unknown.
  if (lastStatus === "down") return "down";
  if (lastStatus === "paused") return "unknown";

  // No metric data at all → unknown, never a fake "healthy".
  if (latency == null && packetLoss == null && jitter == null) {
    return lastStatus === "up" ? "healthy" : "unknown";
  }

  let state: PingHealthState = "healthy";
  state = worst(state, compareMetric(latency, thresholds.latency));
  state = worst(state, compareMetric(packetLoss, thresholds.packetLoss));
  state = worst(state, compareMetric(jitter, thresholds.jitter));

  return state;
}

export function pingHealthTone(state: PingHealthState): StatusTone {
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
