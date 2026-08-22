# Dog — Architecture Overview

> راهنمای صفر تا صد معماری و ساختار پروژه Dog برای اعضای تیم.
> برای افزودن یک Feature جدید ابتدا
> [feature-development.md](feature-development.md) را بخوانید.
> **برای ادامه ریفکتور، اول [migration-status.md](migration-status.md) را
> بخوانید** — وضعیت فازها و قدم بعدی آنجا ثبت شده است.

## Dog چیست؟

Dog یک پلتفرم **Monitoring / Observability** سازمانی است که هر موجودیت قابل
مشاهده (سرور، وب‌سایت، API، تجهیزات شبکه، container و ...) را به عنوان
**Resource** مدل می‌کند و قابلیت‌های مختلف **Checks**, **Metrics**, **Logs**,
**Traces**, **Errors**, **Events** را روی آن ارائه می‌دهد.

## نقشه معماری (North Star)

```
                      USERS
                        │
                        ▼
                 ┌─────────────┐
                 │   Web UI    │   (apps/web — Next.js)
                 └──────┬──────┘
                        │ HTTPS / SSE
                 ┌──────▼──────┐
                 │   API       │   (apps/api — control plane)
                 └──────┬──────┘
                        │
     ┌──────────────────┼───────────────────┐
     │                  │                   │
     ▼                  ▼                   ▼
 Resources          Checks              Signals
                          │        ┌───────┼───────┐
                          │        │       │       │
                          │     Metric  Log   Trace
                          │        │       │       │
                          │       Error   Event
                          │
                   Collection Layer
                          │
     ┌──────────┬─────────┼────────┬────────────┐
     ▼          ▼         ▼        ▼            ▼
   Probe      Agent      SNMP     OTEL         SDK
                          │
                          ▼
                   NATS JetStream (Queue / Event Bus)
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
       Workers         Ingestion       Engines
                          │          (Health / Alert)
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
     PostgreSQL      VictoriaMetrics   Future Stores
     Control Plane    Time Series      (Logs/Traces/Errors)
                          │
                          ▼
                    Health / Alert / SSE → Web UI
```

## لایه‌های اصلی

| لایه | نقش | مسیر |
|---|---|---|
| **Domain** | موجودیت‌ها و قواعد کسب‌وکار، بدون وابستگی به زیرساخت | `packages/shared/domain/` |
| **Application** | سرویس‌های استفاده (use cases) و پورت‌ها (ports/interfaces) | `packages/shared/application/` |
| **Infrastructure** | آداپترهای PostgreSQL / VictoriaMetrics / NATS / Redis | `packages/shared/infrastructure/` |
| **Interfaces** | HTTP API، SSE، gRPC | `packages/shared/interfaces/http/` |
| **Registry** | ثبت check/metric/collection/signal انواع | `probe.Registry`, `health/catalog.go`, جداول `*_types` |
| **Pipeline** | ingestion/normalization/enrichment/routing | `packages/shared/ingestion/`, `packages/shared/telemetry/` |
| **Queue / Bus** | NATS JetStream، Redis Streams | `packages/shared/queue/`, `ingestion/messagebus/` |
| **Apps** | باینری‌های مستقل deployable | `apps/*/cmd/*/main.go` |
| **Frontend** | Next.js Feature-Sliced | `apps/web/` |

## اپلیکیشن‌ها (Backend Binaries)

| اپ | نقش | وضعیت |
|---|---|---|
| `apps/api` | Control plane: REST API، SSE، ingestion، health/alert engine، auth، SNMP trap | Active |
| `apps/scheduler` | Claim مانیورهای due و انتشار jobها در Queue | Active |
| `apps/worker` | مصرف jobها، اجرای اکسیوکتورها، ارسال نتیجه | Active |
| `apps/agent-gateway` | Gateway برای probe agents (mTLS) | Active |
| `apps/probe-gateway` | باینری خود probe agent | Active |
| `apps/metric-processor` | پردازش legacy متریک — **Deprecated** (جایگزین: telemetry pipeline) | Deprecated |
| `apps/monitor-engine` | Health Engine مستقل | Stub (فعلاً داخل API) |
| `apps/alert-engine` | Alert Engine مستقل | Stub (فعلاً داخل API) |

## سیگنال‌ها (Signals)

سیگنال‌ها مستقل از روش جمع‌آوری هستند:

| سیگنال | مثال | روش جمع‌آوری |
|---|---|---|
| **Check** | Ping، HTTP، TCP، DNS، SSL، SMTP، NTP، Domain Expiry | probe |
| **Metric** | CPU، Memory، Interface Traffic | agent / snmp / otel |
| **Log** | Application Log | agent / otel |
| **Trace** | HTTP Span | otel |
| **Error** | NullPointerException | sdk / otel |
| **Event** | SNMP Trap «Interface Down» | trap receiver |

این تفکیک باید در Domain، Database، API، Frontend و Registry یکسان رعایت شود.

## مفاهیم مرکزی

- **Resource**: موجودیت مرکزی (Organization → Workspace → Resource).
  هر چیزی که مانیتور می‌شود Resource است.
- **Monitor**: یک Check فعال متصل به Resource. مانیور باید `monitor_type_id`
  داشته باشد (registry). ستون legacy `monitors.type` **Deprecated** است.
- **Resource Type / Monitor Type**: موجودیت‌های registry با capabilities و
  schemaهای JSON (backend-driven).
- **Probe / Agent**: روش‌های اجرای job (Probe برای remote checks، Agent برای
  host telemetry).
- **Collection Method**: `probe | agent | snmp | otel | sdk | api`.

## جریان اصلی داده (خلاصه)

```
Scheduler → (ClaimDue + FOR UPDATE SKIP LOCKED) → Queue (NATS JetStream / Redis)
   → Worker (stateless, concurrency limits) → Executor (probe.Registry)
   → Result → Ingestion → PostgreSQL + VictoriaMetrics + SSE
   → Health Engine → Alert Engine → Notification
```

## مستندات مرتبط

| سند | محتوا |
|---|---|
| [backend.md](backend.md) | ساختار پکیج‌های Go، domain model، registryها |
| [frontend.md](frontend.md) | ساختار Next.js، entities/features/plugins، data flow |
| [data-flow.md](data-flow.md) | مسیرهای end-to-end داده |
| [scaling.md](scaling.md) | مقیاس‌پذیری افقی، queue، lease، HA |
| [observability.md](observability.md) | Self-monitoring، health endpoints، metrics |
| [snmp.md](snmp.md) | معماری SNMP (discovery، polling، trap) |
| [migrations.md](migrations.md) | راهنمای migrationهای دیتابیس |
| [feature-development.md](feature-development.md) | **چگونه یک Feature جدید اضافه کنیم** |
