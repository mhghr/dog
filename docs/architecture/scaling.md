# Dog — Scaling & High Availability

## اصول

- هیچ فرض «single-instance» نباید وجود داشته باشد.
- Workerها **stateless** و Horizontally Scalable هستند.
- Scheduler با `FOR UPDATE SKIP LOCKED` چند-instance است.
- Queue مسیر اصلی NATS JetStream است (Redis برای cache/lock/session).

## Deployable Units

| سرویس | مقیاس‌پذیری | نکته |
|---|---|---|
| `api` | N instances | state مشترک در PostgreSQL/NATS/Redis |
| `scheduler` | N instances | ClaimDue + SKIP LOCKED → بدون double-schedule |
| `worker` | N instances | stateless؛ concurrency limits |
| `agent-gateway` | N instances | gateway stateless؛ identity در DB |
| `monitor-engine` | N instances (آینده) | health evaluation از queue مصرف می‌کند |
| `alert-engine` | N instances (آینده) | dedup + state در DB |
| `web` | N instances | Next.js |

## Scheduler — Distributed

- `monitors.ClaimDue`: `FOR UPDATE SKIP LOCKED` + جلو بردن `next_run_at`
  در همان تراکنش.
- ویژگی‌ها: idempotency (بازارclaim دوباره نمی‌شود)، lease (lock تا commit)،
  misfire handling (بسته شدن در بچ بعدی)، jitter/retry.
- Multi-instance crash مقاوم است: اگر یک instance بمیرد، مانیورها توسط
  instance دیگر در تیک بعدی claim می‌شوند (زمانی که next_run_at برسد).

## Worker — Stateless & Backpressure

- Concurrency limits (`config.go`):
  - `WorkerConcurrency` (global — همیشه bound کل fleet)
  - `WorkerPerTypeConcurrency` (per-check-type، مثل `http=500,ping=200`)
  - `WorkerPerWorkspaceConcurrency` (یک tenant نمی‌تواند کل fleet را اشباع کند)
- Spool روی دیسک + batcher → result delivery.
- Poison / DLQ + reclaim loop.
- SNMP tasks: NATS reply (`snmp.tasks.result.<id>`) — اجرای async.

## Queue — NATS JetStream

- استریم‌ها: `PROBE_JOBS`, `TELEMETRY_EVENTS`, `METRICS` + DLQها
  (`ingestion/messagebus/nats.go`).
- Durable consumers + Ack + Retry + DLQ.
- Consumer lag قابل مانیتورینگ (راهنمای [observability.md](observability.md)).
- Redis Streams مسیر legacy/پیش‌فرض local است؛ NATS برای production.

## Database HA

- PostgreSQL: Primary + Replica + Backup + Point-in-time recovery.
- Application باید connection pool درست داشته باشد (pgxpool)؛ pool برای
  تعداد instanceها محاسبه شود.
- Time-series در VictoriaMetrics (نه PostgreSQL).

## VictoriaMetrics

- Cluster-ready؛ application به topology خاص VM وابسته نیست.
- Metric labels کنترل‌شده — جلوگیری از high cardinality. labelهای
  user-generated بدون limit ممنوع.

## Distributed SSE

Live events از ingestion از طریق یک باس اشتراکی بین همه replicaهای API پخش
می‌شوند تا هر instance همان stream را به browserهای متصل خودش بدهد:

```
Data Event → DistributedPublisher → (local Bus + NATS `events.live`)
          → NATSRelay (هر API replica) → local Bus → SSE → Browser
```

- `events.Publisher` interface در `packages/shared/events/bus.go`؛
  `DistributedPublisher` + `NATSRelay` در `packages/shared/events/nats.go`.
- فعال‌سازی: `LIVE_EVENTS_NATS=1` + `NATS_URL` در `apps/api`. بدون flag، رفتار
  قبلی (فقط local Bus) حفظ می‌شود.
- **Echo-drop**: هر رویداد دارای هدر `dog-event-origin` است؛ relay رویدادهای
  instance خودش را دور می‌اندازد تا subscriber محلی کپی تکراری نبیند.
- **Graceful degradation**: اگر NATS در دسترس نباشد، publish فقط log می‌شود —
  ingestion هرگز به live stream وابسته نیست (PostgreSQL منبع حقیقت است).
- Reconnect: `nats.MaxReconnects(-1)` + reconnect backoff در relay/پابلیشر.

## فازهای آینده (Health/Alert Engines مستقل)

- `monitor-engine` / `alert-engine` فعلاً stub هستند و منطق داخل API اجرا
  می‌شود. برای مقیاس بالا باید به اپ‌های مستقل تبدیل شوند که از همان
  queue/باس consume می‌کنند.

## نکات Load

- پروفایل‌های بار: 1K/10K/100K user، 10K/100K monitor، 1M checks/hour،
  10M metrics/min.
- Metricهای بار: p50/p95/p99، error rate، queue lag، DB load، VM load.
- Downsampling سمت backend (`resolveStep` → ≤1500 points) جلوی overload مرورگر
  را می‌گیرد.
