// Centralized DNS health evaluation.
//
// The backend `last_status` is authoritative for the "down" state. For the
// richer Healthy / Warning / Critical gradation, the worst metric condition
// (compared against configured thresholds) wins. Mirrors the TCP/HTTP
// evaluators.

import type { DnsThresholds } from "./dns-config";
import type { StatusTone } from "@/design-system/components/status-badge";

export type DnsHealthState =
  | "healthy"
  | "warning"
  | "critical"
  | "down"
  | "unknown";

export interface DnsHealthInput {
  /** monitor.last_status */
  lastStatus?: string;
  /** whether the DNS query succeeded */
  success?: boolean;
  /** query response time, may be null on failure */
  responseTimeMs?: number | null;
  /** expected-record assertion: 1 matched, 0 failed, null unknown */
  expectedRecordMatch?: number | null;
  thresholds: DnsThresholds;
}

const RANK: Record<DnsHealthState, number> = {
  unknown: 0,
  healthy: 1,
  warning: 2,
  critical: 3,
  down: 4,
};

function worst(
  current: DnsHealthState,
  candidate: DnsHealthState,
): DnsHealthState {
  return RANK[candidate] > RANK[current] ? candidate : current;
}

/** Compare a higher-is-worse metric against its threshold; no value → healthy. */
export function compareMetric(
  value: number | null | undefined,
  threshold: { warning?: number; critical?: number } | undefined,
): DnsHealthState {
  if (value == null || threshold == null) return "healthy";
  const critical = threshold.critical;
  const warning = threshold.warning;

  if (critical != null && value >= critical) return "critical";
  if (warning != null && value >= warning) return "warning";
  return "healthy";
}

export function evaluateMetric(
  value: number | null | undefined,
  threshold: { warning?: number; critical?: number } | undefined,
): DnsHealthState {
  if (value == null) return "unknown";
  return compareMetric(value, threshold);
}

export function evaluateAvailability(success: boolean | undefined): DnsHealthState {
  if (success === undefined) return "unknown";
  return success ? "healthy" : "critical";
}

export function evaluateDnsHealth(input: DnsHealthInput): DnsHealthState {
  const { lastStatus, success, responseTimeMs, expectedRecordMatch, thresholds } = input;

  if (lastStatus === "down") return "down";
  if (lastStatus === "paused") return "unknown";

  if (success === false) return "critical";
  if (expectedRecordMatch === 0) return "critical";

  if (responseTimeMs == null) {
    return lastStatus === "up" ? "healthy" : "unknown";
  }

  let state: DnsHealthState = "healthy";
  state = worst(state, compareMetric(responseTimeMs, thresholds.responseTime));
  return state;
}

export function dnsHealthTone(state: DnsHealthState): StatusTone {
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
