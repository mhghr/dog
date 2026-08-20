// Centralized TLS health evaluation.
//
// The backend `last_status` is authoritative for the "down" state. Handshake
// time is HIGHER_IS_WORSE; certificate expiry days is LOWER_IS_WORSE (fewer
// remaining days → worse). The worst condition wins.

import type { TlsThresholds } from "./tls-config";
import type { StatusTone } from "@/design-system/components/status-badge";

export type TlsHealthState =
  | "healthy"
  | "warning"
  | "critical"
  | "down"
  | "unknown";

export interface TlsHealthInput {
  /** monitor.last_status */
  lastStatus?: string;
  /** whether the TLS handshake + certificate checks succeeded */
  success?: boolean;
  /** handshake duration, may be null on failure */
  handshakeTimeMs?: number | null;
  /** days until certificate expiry, may be null on failure */
  certificateExpiryDays?: number | null;
  /** whether the endpoint certificate was verified (false when verification disabled) */
  verified?: boolean;
  thresholds: TlsThresholds;
}

const RANK: Record<TlsHealthState, number> = {
  unknown: 0,
  healthy: 1,
  warning: 2,
  critical: 3,
  down: 4,
};

function worst(
  current: TlsHealthState,
  candidate: TlsHealthState,
): TlsHealthState {
  return RANK[candidate] > RANK[current] ? candidate : current;
}

/** Compare a higher-is-worse metric against its threshold; no value → healthy. */
export function compareMetric(
  value: number | null | undefined,
  threshold: { warning?: number; critical?: number } | undefined,
): TlsHealthState {
  if (value == null || threshold == null) return "healthy";
  const critical = threshold.critical;
  const warning = threshold.warning;

  if (critical != null && value >= critical) return "critical";
  if (warning != null && value >= warning) return "warning";
  return "healthy";
}

/**
 * Compare a lower-is-worse metric (e.g. days until expiry) against its
 * threshold: a value at or below the critical threshold is critical, at or
 * below the warning threshold is a warning.
 */
export function compareLowerIsWorse(
  value: number | null | undefined,
  threshold: { warning?: number; critical?: number } | undefined,
): TlsHealthState {
  if (value == null || threshold == null) return "healthy";
  const critical = threshold.critical;
  const warning = threshold.warning;

  if (critical != null && value <= critical) return "critical";
  if (warning != null && value <= warning) return "warning";
  return "healthy";
}

export function evaluateMetric(
  value: number | null | undefined,
  threshold: { warning?: number; critical?: number } | undefined,
): TlsHealthState {
  if (value == null) return "unknown";
  return compareMetric(value, threshold);
}

export function evaluateExpiry(
  value: number | null | undefined,
  threshold: { warning?: number; critical?: number } | undefined,
): TlsHealthState {
  if (value == null) return "unknown";
  return compareLowerIsWorse(value, threshold);
}

export function evaluateAvailability(success: boolean | undefined): TlsHealthState {
  if (success === undefined) return "unknown";
  return success ? "healthy" : "critical";
}

export function evaluateTlsHealth(input: TlsHealthInput): TlsHealthState {
  const {
    lastStatus,
    success,
    handshakeTimeMs,
    certificateExpiryDays,
    verified,
    thresholds,
  } = input;

  if (lastStatus === "down") return "down";
  if (lastStatus === "paused") return "unknown";

  // Verification being disabled is not a healthy state by itself: the
  // certificate is untrusted, so surface it as a warning so the operator
  // notices the gap.
  if (success && verified === false) {
    return worst("warning", evaluateMetric(handshakeTimeMs, thresholds.handshakeTime));
  }

  if (success === false) return "critical";

  if (handshakeTimeMs == null && certificateExpiryDays == null) {
    return lastStatus === "up" ? "healthy" : "unknown";
  }

  let state: TlsHealthState = "healthy";
  state = worst(state, compareMetric(handshakeTimeMs, thresholds.handshakeTime));
  state = worst(state, compareLowerIsWorse(certificateExpiryDays, thresholds.certificateExpiryDays));
  return state;
}

export function tlsHealthTone(state: TlsHealthState): StatusTone {
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
