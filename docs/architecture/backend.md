# Dog — Backend Architecture

## ساختار کلی

Backend به زبان Go و به صورت monorepo است. ماژول: `monitoring-platform`.

```
packages/shared/            ← کد اشتراکی بین همه اپ‌ها
apps/                       ← باینری‌های deployable
migrations/                 ← SQL migrations (PostgreSQL)
proto/agent/v1/             ← gRPC contract agent
infrastructure/             ← docker-compose، scripts، certs
```

## پکیج‌های `packages/shared`

| پکیج | مسئولیت |
|---|---|
| `domain/` | موجودیت‌ها، value objects، قواعد دامنه. **بدون وابستگی به زیرساخت.** |
| `application/` | پورت‌ها/سرویس‌های application (مثل `metricquery`). |
| `repository/` | پورت‌های persistence (interfaces). پیاده‌سازی در `infrastructure/postgres`. |
| `infrastructure/postgres/` | آداپترهای PostgreSQL (همه repositoryهای concrete). |
| `infrastructure/postgres/metric_query_service.go` | آداپتر query متریک (backend-driven). |
| `interfaces/http/` | chi router، handlerها، Deps (dependency wiring). |
| `probe/` | **Registry اکسیوکتورهای check** + اجرای آنها (`probe.go`, `ping.go`, `http.go`, ...). |
| `health/` | Health Engine + `catalog.go` (registry پارامترهای health هر نوع). |
| `alerting/` | Alert Engine + Notifier (webhook/email/telegram). |
| `scheduler/` | Scheduler که jobها را از مانیورهای due تولید می‌کند. |
| `worker/` | Worker: مصرف job، محدودیت‌های concurrency، ارسال نتیجه. |
| `queue/` | JobQueue abstraction: Redis Streams + NATS JetStream. |
| `ingestion/` | Ingestion Service، messagebus (NATS streams)، pipeline (validator/enricher/normalizer/publisher). |
| `telemetry/` | Unified ingestion pipeline جدید (consumer، dedup، batcher، circuit breaker). |
| `snmp/` | کلاینت SNMP، OID registry، discovery، poll، tasks، trap receiver. |
| `agents/` | Probe agents (enrollment، identity، spool، gateway، certificate). |
| `auth/` | JWT، Google OAuth، OTP/SMS، org scoping، middleware. |
| `config/` | بارگذاری config از env vars. |
| `events/` | Event bus in-process (SSE fan-out). |
| `metrics/` | Prometheus registry + VictoriaMetrics client. |
| `security/` | SSRF guard، secret helpers. |
| `heartbeat/` | Liveness heartbeats به Redis. |
| `logging/` | ساخت slog logger با format/service. |
| `httpserver/` | چرخه حیات HTTP server + healthcheck. |

## Domain Model

### هسته (Resource → Monitor)

```
Organization ─┬─ Workspace ─┬─ Resource ─┬─ Monitor (Check)
              │             │            ├─ MetricSeries
              │             │            └─ Events
              └─────────────┴─ Tags
```

- **`Resource`** (`domain/observability.go`): موجودیت مرکزی. دارای
  `resource_type_id` که capabilities را تعیین می‌کند.
- **`ResourceType`** (`domain/observability.go`): registry با `capabilities`,
  `configuration_schema`. Backend-driven؛ Frontend اطلاعات capabilities را
  hard-code نمی‌کند.
- **`Monitor`** (`domain/observability.go`): یک check فعال متصل به Resource.
  - `monitor_type_id` → کلید کانونیکال (FK به `monitor_types`).
  - `Type`/`ProbeType` فیلد مشتق‌شده از registry (`executor_key`).
  - ستون legacy `monitors.type` **Deprecated** است — هیچ query جدیدی نباید
    از آن استفاده کند. queryهای قدیمی از طریق `COALESCE(mt.executor_key,
    m.type::text)` migration شده‌اند (`infrastructure/postgres/result_repository.go`).
- **`MonitorTypeDef`** (`domain/observability.go`): ردیف registry با
  `executor_key`, `config_schema`, `metric_schema`, `health_parameters`,
  `supported_resource_types`.
- **`ProbeJob`** (`domain/job.go`): پیام scheduler→worker.
- **`ProbeResult`** (`domain/result.go`): نتیجه اجرای check با `Metrics`,
  `Attributes`, `ErrorCode`.

### سیگنال‌ها

| نوع | فایل | نکته |
|---|---|---|
| Metric | `domain/metric.go`, `metric_series.go` | MetricSample، MetricBatch، MetricSeriesRow، MetricRollup |
| Event | `domain/observability.go` (`Event`) | رویدادهای منبع‌مستقل |
| Alert | `domain/alert.go` | AlertPolicy، Alert، NotificationChannel |
| SNMP | `domain/snmp.go`, `snmp_monitoring.go` | SNMPDevice، SNMPInterfaceInfo، SNMPEvent، SNMPTask |

## Registryها (Backend Source of Truth)

| Registry | محل | محتوا |
|---|---|---|
| **Check Executor Registry** | `probe/probe.go` (`DefaultRegistry`) | نگاشت `MonitorType` → Executor |
| **Monitor Type Registry** | جدول `monitor_types` + `MonitorTypeDef` | schemaها، metric_keys، health params |
| **Health Parameter Catalog** | `health/catalog.go` (`AllParameters`) | پارامترها و thresholds هر نوع |
| **SNMP OID Registry** | `snmp/oids.go` (`DefaultRegistry`) | OIDهای vendorها |
| **Resource Type Registry** | جدول `resource_types` | capabilities، config schema |

قانون: **Backend مرجع (source of truth) است.** Frontend فقط UI metadata
(icon, label, form, chart) را نگه می‌دارد.

## Query متریک و MetricQueryService

Frontend نباید بداند متریک از PostgreSQL یا VictoriaMetrics می‌آید.

- **پورت**: `application/metricquery/query_service.go` — `QueryService` interface
  (`SeriesByProbe`, `SeriesByProbeMetric`, `StatusSeriesByProbe`,
  `LatestSuccessAt`, `StatusCodeDistribution`, `AggregateMetrics`,
  `ProbeMetrics`).
- **آداپتر فعلی**: `infrastructure/postgres/metric_query_service.go`
  (خواندن از `probe_results` / `metric_points`).
- **مصرف**: handler در `interfaces/http/resource_monitor_handler.go` فقط به
  `deps.MetricQuery` وابسته است.

برای مهاجرت به VictoriaMetrics، فقط یک آداپتر جدید بنویسید و آن را در
`apps/api/cmd/api/main.go` جایگزین کنید؛ API contract و Frontend بدون تغییر
می‌مانند.

### Downsampling

`resolveStep()` در `interfaces/http/resource_monitor_handler.go` بر اساس
Range و Resolution step را انتخاب می‌کند و حداکثر به `maxChartPoints = 1500`
نقطه در هر سری محدود می‌کند (spec section 18). تست‌ها در
`metric_query_service_test.go`.

## Scheduler

- `scheduler/scheduler.go`: تیک می‌زند، `monitors.ClaimDue` را صدا می‌زند.
- `ClaimDue` (`monitor_repository.go`): با `FOR UPDATE SKIP LOCKED` ردیف‌های
  due را claim می‌کند، `next_run_at` را جلو می‌برد و job منتشر می‌کند.
  → **چند instance scheduler می‌توانند همزمان اجرا شوند** بدون دوباره‌اجرا شدن
  مانیور.

## Queue

- **پیش‌فرض**: Redis Streams (`queue/redis_stream.go`) — `probe_jobs`.
- **NATS JetStream** (`TELEMETRY_PIPELINE_MODE=nats`): استریم‌ها در
  `ingestion/messagebus/nats.go` تعریف می‌شوند:
  - `PROBE_JOBS` ← `probe.jobs.>`
  - `TELEMETRY_EVENTS` ← `telemetry.probe.>`, `telemetry.metrics.>`
  - `METRICS` ← `metrics.>`
  - DLQها: `PROBE_JOBS_DLQ`, `TELEMETRY_DLQ`, `METRICS_DLQ`

## Worker

- `worker/worker.go`: مصرف job، اجرای اکسیوکتور از `probe.Registry`،
  retry، poison/DLQ، reclaim loop.
- Concurrency limits: global، per-check-type، per-workspace (`config.go`).
- Result به spool (دیسک) ذخیره و batched ارسال می‌شود.

## Ingestion

- API: `ingestion.Service.Ingest` → validate → ذخیره result + آپدیت مانیور →
  VictoriaMetrics → Event Bus (SSE) → Health Engine.
- Unified pipeline: `telemetry/ingest/` (consumer روی `telemetry.>`, dedup،
  batcher، circuit breaker).

## Health / Alert

- Health Engine: `health/engine.go` + `health/repository.go`
  (resource_health_state). Input: Metric، Check Result، Event، Error.
- Alert Engine: `alerting/engine.go` (state machine:
  pending/firing/recovering/resolved) + `alerting/notifier` + notification
  channels.
- اپ‌های مستقل `monitor-engine` و `alert-engine` فعلاً stub هستند و منطق داخل
  API اجرا می‌شود.

## Dependency Rules

```
Domain
  ↑
Application  (پورت‌ها)
  ↑
Infrastructure / Interfaces  (آداپترها / handlers)
```

- Domain نباید HTTP/PostgreSQL/NATS/Redis/VictoriaMetrics را بشناسد.
- Infrastructure نباید business rule تعریف کند.
- ارتباط بین bounded contexts از طریق interfaces، application services و
  events است.
