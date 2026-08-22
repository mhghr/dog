# Dog — Data Flow

این سند مسیرهای end-to-end داده را در Dog نشان می‌دهد.

## 1. جریان Monitoring (Check) — جریان اصلی

```
[API] create monitor (POST /resources/{id}/monitors)
   │  monitor_type_id از registry، target از resource
   ▼
[PostgreSQL] monitors (باید enabled + next_run_at<=NOW باشد)
   │
   ▼  Scheduler tick
[Scheduler] monitors.ClaimDue (FOR UPDATE SKIP LOCKED)
   │  می‌سازد: domain.ProbeJob {id, monitor_id, resource_id, workspace_id,
   │           type(executor_key), target, timeout, retries, config}
   │  و `next_run_at` را جلو می‌برد (idempotent — بدون double-scheduling)
   ▼
[Queue] Redis Stream `probe_jobs` OR NATS JetStream `probe.jobs`
   │
   ▼
[Worker] consume (limits: global/per-type/per-workspace)
   │  Executor از probe.Registry (ping/http/tcp/dns/tls/smtp/ntp/snmp/...)
   │  spool به دیسک + batcher
   ▼
[Result] domain.ProbeResult {success, error_code, metrics, attributes, duration}
   │  HTTP POST /internal/results[/batch] (Bearer WORKER_TOKEN)
   │  OR NATS `telemetry.probe.result` → TELEMETRY_EVENTS
   ▼
[API] ingestion.Service.Ingest
   │  1. validate
   │  2. InsertAndUpdateMonitor (idempotent روی job_id؛ آپدیت monitors.last_status)
   │  3. VictoriaMetrics.Enqueue (باتچ → /api/v1/import/prometheus)
   │  4. events.Bus.Publish("probe-result") → SSE /events/stream
   │  5. health.Engine.EvaluateResult → resource_health_state
   │  6. alerting.Engine → AlertPolicy state machine → Notification
   ▼
[Frontend] SSE → useLiveResults → React Query invalidation
```

## 2. جریان Metrics (Agent Telemetry)

```
[Monitoring Agent] collect (cpu/memory/disk/network)
   → batch → NATS `metrics.<tenant>.<agent>` → METRICS stream
   → processor → VictoriaMetrics (api/v1/import/prometheus)
   → heartbeat → PostgreSQL (monitoring_agent_heartbeats — range partition)
```

## 3. جریان SNMP

### Polling (Metric)

```
[Scheduler/API] SNMP task → NATS `snmp.tasks.<id>`
   → Worker/SNMP collector: GET/GETBULK با OID registry
   → normalize → ProbeResult/metrics → ingestion pipeline (مانند جریان 1)
   → snmp_interfaces, snmp_events (PostgreSQL)
```

### Discovery

```
[API] POST /snmp/discovery → snmp.Walk
   → SNMPDiscoveryResult {vendor, model, interfaces, cpu, memory, sensors}
   → ذخیره در snmp_discovery + پیشنهاد OIDها
```

### Trap (Event)

```
[Device] UDP/162 → TrapReceiver (API)
   → normalize → SNMPEvent (event، نه metric!)
   → ذخیره + (اختیاری) alerting
```

قانون: Trap یک **Event** است (مثل Interface Down)؛ Polling منبع **Metric**ها
(Interface Traffic, CPU).

## 4. جریان Probe Agents

```
[probe agent] enroll (بوت‌استرپ token) → identity → mTLS
   → agent-gateway (HTTP/MTLS)
   → heartbeat + job delivery + result delivery
   → gateway → Redis pub/sub `probe_results` → API consumeGatewayResults → ingestion
```

## 5. جریان SSE / Realtime

```
[Ingestion] publisher.Publish("probe-result", payload)
   │  local Bus (همان instance) + (اختیاری) NATS `events.live`
   ▼
[هر API instance] NATSRelay → local Bus → SSE /events/stream
   ▼
[Browser] SseClient + useLiveResults
   → throttleInvalidate (3s) → React Query refetch
```

- Single-instance: `LIVE_EVENTS_NATS` تنظیم نشده → فقط local Bus (رفتار قبلی).
- Multi-instance: `LIVE_EVENTS_NATS=1` → `DistributedPublisher` رویدادها را به
  NATS فَناوت می‌کند و `NATSRelay` در هر replica آن‌ها را به bus محلی خودش
  می‌دهد (جزئیات در [scaling.md](scaling.md)).
- Echo-drop: رویدادهای همان instance دوباره سرو نمی‌شوند.

## 6. جریان Status Pages

```
[API] status_pages + status_page_components
   → PublicBySlug (بدون auth) → مسیر عمومی /status/<slug>
```

## Correlation / Tenant در تمام مسیر

- **Tenant scope**: همه queryها با organization/workspace filter اجرا می‌شوند
  (handlerها: `resourceBelongsToOrg`, `monitorBelongsToOrg` → 404 برای جلوگیری
  از enumeration).
- **Idempotency**:
  - `probe_results` unique روی `job_id` (InsertAndUpdateMonitor)
  - `metric_points` unique روی `(time, series_id)`
  - `telemetry_event_dedup` برای پیام‌های NATS
  - `snmp_interfaces` unique روی `(monitor_id, if_index)`
- **Correlation**: `job_id` / `event_id` در resultها و رویدادها حفظ می‌شود.
