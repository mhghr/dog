# Ping Monitor Failure-State Handling — Design

**Date:** 2026-08-16
**Status:** Approved by user
**Scope:** Full stack (probe / DB / VictoriaMetrics / health / frontend / charts / alerting / tests)

## Problem

When a Ping target is unreachable, the system must not represent the result as
zero values. Zero latency is a valid measurement; absence of data is not the
same as downtime. Availability state must be modeled independently from
performance metrics.

## Principles

1. **Separate health state from metrics.** UP = reachable with valid latency.
   DOWN = unreachable, no valid latency metrics. Never use `latency = 0`,
   `jitter = 0`, `RTT = 0` for an unreachable target.
2. **No fake zero metrics in VictoriaMetrics.** Downtime must appear as gaps,
   not zero lines. Availability is carried by an explicit status metric.
3. **Missing data ≠ downtime.** Down periods are derived from explicit status
   signals (`result.Status`, `monitor_ping_status`), never inferred from gaps.

## Design Decisions (user-confirmed)

- Full stack implementation of all 8 gap areas.
- Keep the existing 4 Kpi cards; adapt each when the monitor is down.
- Error taxonomy is best-effort mapped (host/network unreachable are not
  distinguishable from ICMP statistics alone).
- Implement `StatusSeriesByProbe` backend series; do NOT infer DOWN from gaps.

## 1. Backend — Ping result model & executor

**File:** `packages/shared/probe/ping.go`, `packages/shared/probe/helpers.go`

- Write `Metrics["reachability"] = 1` on success, `0` on failure. This feeds the
  health engine's `ping.reachability` rule (currently never evaluated).
- On total packet failure also write `Metrics["packet_loss_percent"] = 100`;
  keep `rtt_ms`, `min_rtt_ms`, `max_rtt_ms`, `jitter_ms` **absent** (NULL).
- Extract result-shaping into a pure function for unit testing without real ICMP.
- Error taxonomy (best-effort, in `mapErrorCode`):
  - `dns_resolution_failed` — guard/DNS failure
  - `timeout` — context deadline or all-packets-lost
  - `permission_denied` — pinger permission error
  - `network_unreachable` — other run errors
  - `unknown_error` — fallback
  - `blocked_target` — preserved

## 2. Health alignment

**Files:** `packages/shared/health/engine.go`, `packages/shared/health/catalog.go`

- `splitParamKey`: add real executor keys as fallbacks:
  - `ping.rtt.avg_ms` → `rtt_avg_ms`, `avg_rtt_ms`, `rtt_ms`
  - `ping.rtt.min_ms` → `rtt_min_ms`, `min_rtt_ms`
  - `ping.rtt.max_ms` → `rtt_max_ms`, `max_rtt_ms`
  - `ping.reachability` → `ping.reachability`, `reachability`
- Packet-loss thresholds → warning `5`, error `20` (Go catalog + frontend
  `ping-config.ts` defaults must agree).
- Reachability `0` → `HealthError` (BOOLEAN_FAILURE). Priority
  DOWN > CRITICAL > WARNING > HEALTHY emerges from the worst() ranking.

## 3. VictoriaMetrics

**File:** `packages/shared/metrics/victoria.go`

- Add `monitor_ping_status` gauge for ping: `1` up, `0` down. Keep
  `monitor_probe_success` for backward compatibility.
- Latency is only written when present → downtime never produces a
  `monitor_ping_latency_ms` point.

## 4. Frontend — cards & failed state

**Files:** `apps/web/entities/resource/ui/monitoring/ping/PingMonitoringView.tsx`,
`ping-metrics.ts`, `PingKpiCard.tsx` (presentational, unchanged logic)

Keep the 4 Kpi cards, adapt each when monitor is down:
- Availability → `0.00%`
- Latency → `∞ ms`
- Packet loss → `100%`
- Jitter → `∞ ms`
- Add a failure banner: reason (`error_code`/`error_message` from latest
  result) + "Last successful check: <time>".

## 5. Charts

**Files:** new `StatusSeriesByProbe` repo method, updated
`resource_monitor_handler.go`, `PingMetricChart.tsx`, new
`PingAvailabilityChart.tsx`.

- New backend series per probe + bucket: success ratio including failures,
  exposed on the metrics endpoint.
- Latency chart: existing gaps + DOWN-period `markArea` from status series.
- New availability chart rendering status series as UP/DOWN bands.

## 6. Alerting

`packages/shared/alerting/engine.go` already keys off `result.Status`
(down→failure, up→success) — this is the `monitor_ping_status == 0`
condition. No new engine code; verify wiring and add a test.

## 7. Tests

- Go: `ping_test.go` — shaping function for success / timeout / network failure
  / recovery (re-evaluation). Health `splitParamKey` + threshold tests.
- Frontend: extend `ping-metrics.test.ts` / `ping-health.test.ts` for down-state
  values (∞, 100%, reason, last-success).

## Data model references

- Success:
  ```json
  { "status": "up", "packets_sent": 4, "packets_received": 4,
    "packet_loss_percent": 0, "latency_ms": 42, "min_latency_ms": 38,
    "max_latency_ms": 47, "jitter_ms": 5, "error": null }
  ```
- Failure:
  ```json
  { "status": "down", "packets_sent": 4, "packets_received": 0,
    "packet_loss_percent": 100, "latency_ms": null, "min_latency_ms": null,
    "max_latency_ms": null, "jitter_ms": null, "error": "timeout" }
  ```
