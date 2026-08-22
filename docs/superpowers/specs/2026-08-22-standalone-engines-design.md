# Standalone Monitor & Alert Engines — Design

**Date:** 2026-08-22
**Status:** Implemented (phases 10-11)
**Scope:** Backend (API seam / monitor-engine app / alert-engine app / NATS subjects / config / tests)

## Problem

Today the Health Engine and Alert Engine run synchronously inside the API:

- Health evaluation lives inside `ingestion.Service.Ingest` (`apps/api` HTTP and
  Redis-gateway paths only).
- Alert evaluation lives inside `result_handler.go` (HTTP path only — gateway
  results never trigger alerts).
- In NATS mode (`TELEMETRY_PIPELINE_MODE=nats`) results are persisted by the
  telemetry consumer with **no** health or alert evaluation at all.
- Health notifications (`health.NotificationEngine`) are never invoked (dead code).

Phases 10+11 of the architecture migration require emptying the
`apps/monitor-engine` and `apps/alert-engine` stubs into real, independently
scalable binaries that consume results from the message bus.

## Design Decisions (user-confirmed)

- Approach A: NATS event-driven engines with an `inline` fallback, switchable
  per-engine via feature flag (`MONITOR_ENGINE_MODE`, `ALERT_ENGINE_MODE`).
- Behavior changes accepted in `inline` mode:
  (a) Gateway-path results now also trigger alert evaluation (bug fix).
  (b) Health notifications start working (previously dead code).
- Reuse the existing `ingest.Envelope` wire format for the result payload.
- Single `ResultRouter` seam is the only place engine logic is invoked from the
  API; both persistence paths route through it.

## 1. Config & Subjects

**File:** `packages/shared/config/config.go`

New config fields:
- `MonitorEngineMode string` ← `MONITOR_ENGINE_MODE` (`inline` | `nats`, default `inline`)
- `AlertEngineMode string` ← `ALERT_ENGINE_MODE` (`inline` | `nats`, default `inline`)

New JetStream stream `ENGINE_EVENTS` (added in `messagebus.NATSBus.ensureStreams`,
`packages/shared/ingestion/messagebus/nats.go`):
- Subjects: `engine.health.eval`, `engine.alert.eval`
- Retention: `WorkQueuePolicy`, FileStorage, 24h MaxAge, replicas from `NATS_REPLICAS`

Wire format reuses `ingest.Envelope` (already carries `EventID` + dedup-friendly
value envelope); the value is a `domain.ProbeResult`.

## 2. API — ResultRouter seam

**New package:** `packages/shared/engines`

`Router` is constructed with the two mode flags, an optional NATS bus, the
health/alert engines and notifiers. `RouteResult(ctx, result)`:
- **`inline`**: run `health.Engine.EvaluateResult` + `health.NotificationEngine`
  (health), then `alerting.Engine.Evaluate` + `alerting.Notifier.Dispatch`
  (alert) — the same engines the API uses today, now wired once.
- **`nats`**: publish the result envelope to `engine.health.eval` and
  `engine.alert.eval` (one per enabled subject); nothing runs in-process.
- A nil/invalid mode is treated as `inline`.

Call sites (both mode-aware, only one path is active per result in each
deployment mode, so no double delivery):
1. `ingestion.Service.Ingest` after a successful insert (`inserted == true`).
2. `telemetry/ingest` consumer `processProbeResult` after `IngestProbeResult`
   returns `inserted == true`. (The telemetry consumer is not wired into any
   binary yet — this seam is prepared for the enterprise path; wiring that
   consumer is out of scope for this phase.)

`ingestion.Service` drops its `healthEngine` / `healthNotif` fields and takes the
`Router` instead. `result_handler.go` drops its inline `AlertEngine.Evaluate` +
`Notifier.Dispatch` calls — the router is now the single funnel (fixes the
gateway alert gap). The API creates a `messagebus.NATSBus` when either mode is
`nats` and passes it to the router.

## 3. monitor-engine

**File:** `apps/monitor-engine/cmd/monitor-engine/main.go`

- `config.Load()`, `logging.New(..., "monitor-engine")`, `Require(DATABASE_URL, NATS_URL)`.
- Postgres pool + `postgres.NewHealthRepository`.
- `messagebus.NewNATSBus` → durable queue-group subscription on
  `engine.health.eval` (`Queue: "monitor-engines"`, `Durable: "monitor-engine"`,
  `Stream: "ENGINE_EVENTS"`, `DeliverNew: true`).
- Handler: unmarshal envelope → `health.Engine.EvaluateResult`.
- Health engine contract change: `EvaluateResult(ctx, result) ([]EvaluateOutcome, error)`
  — the internal `persistAndReturn` already computes `PreviousState`/`NewState`;
  thread an `*EvaluateOutcome` out through the ~6 evaluator call sites. The
  engine worker then calls `health.NotificationEngine.Evaluate` per outcome,
  wiring the previously dead notification path (decision b).
- Health/readiness endpoint via `httpserver` (pattern: `apps/worker`).

## 4. alert-engine

**File:** `apps/alert-engine/cmd/alert-engine/main.go`

- Same bootstrapping; `postgres.NewAlertRepository` + `NewChannelRepository`.
- Durable queue-group subscription on `engine.alert.eval`
  (`Queue: "alert-engines"`, `Durable: "alert-engine"`, `Stream: "ENGINE_EVENTS"`).
- Handler: unmarshal envelope → `alerting.Engine.Evaluate` →
  `alerting.Notifier.Dispatch` per firing event.
- Health/readiness endpoint via `httpserver`.

## 5. Error handling & idempotency

- At-least-once delivery. A handler error returns the error → NATS redelivers.
- Malformed payloads return `nil` (no redelivery).
- Health eval is idempotent (`parameter_health_state` upsert keyed on
  monitor+parameter). Alert eval is idempotent via its `policy:monitor` dedup
  key. Duplicate results are already filtered at ingestion (`inserted == false`).

## 6. Validation

- `go build ./...`, `go vet ./...`, `go test ./...`
- New unit tests: `engines` router (inline vs nats publish, using a fake bus),
  `health` engine outcome emission, consumer handler (fake bus) for both engines.
- Manual smoke: run API in `nats` mode + both engines, POST a probe result,
  observe health state rows update and alert firing.

## Non-goals (this phase)

- No change to the scheduler/worker queue (`PROBE_JOBS`) or probe execution.
- No k8s/HA manifests; engines run as plain binaries like the worker today.
- Alert-engine does not receive health outcomes; it stays driven by probe results
  (alerting stays independent, matching today's contract).
