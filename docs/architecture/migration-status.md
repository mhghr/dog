# Dog — Migration Status (وضعیت ریفکتور معماری)

> این سند «حافظه جلسه» است. در شروع هر جلسه، agent باید این فایل و
> [feature-development.md](feature-development.md) را بخواند تا بداند
> کجا هستیم و قدم بعدی چیست.

## وضعیت کلی

- هدف: تبدیل Dog به معماری Enterprise / Horizontally Scalable طبق
  [overview.md](overview.md) — بدون شکستن قابلیت‌های Monitoring فعلی.
- قانون: هیچ تغییر بزرگ بدون build/test. هر فاز باید Buildable و Testable باشد.
- مسیر ۱۵ فازی در spec اصلی: `docs/architecture/overview.md` → بخش ۷۳.

## فازهای تکمیلشده ✅

| فاز | شرح | فایلهای کلیدی |
|---|---|---|
| **1+2** | ایزولهسازی legacy `monitors.type` — queryها به `monitor_types.executor_key` + `COALESCE` fallback | `packages/shared/infrastructure/postgres/result_repository.go` |
| **6** | `MetricQueryService` — پورت application + آداپتر PostgreSQL | `packages/shared/application/metricquery/query_service.go`, `packages/shared/infrastructure/postgres/metric_query_service.go`, wiring در `apps/api/cmd/api/main.go` |
| **8** | Distributed SSE — `Publisher` + `NATSRelay` (فعالسازی opt-in با `LIVE_EVENTS_NATS=1`) | `packages/shared/events/bus.go`, `packages/shared/events/nats.go`, `packages/shared/ingestion/service.go` |
| **10** | **Health Engine مستقل** — `apps/monitor-engine` پر شد؛ consumer روی `engine.health.eval` (durable queue-group) + health endpoint | `apps/monitor-engine/cmd/monitor-engine/main.go`, `packages/shared/engines/consumer.go`, `packages/shared/health/engine.go` |
| **11** | **Alert Engine مستقل** — `apps/alert-engine` پر شد؛ consumer روی `engine.alert.eval` (durable queue-group) + health endpoint | `apps/alert-engine/cmd/alert-engine/main.go`, `packages/shared/engines/consumer.go` |

> فازهای ۱۰+۱۱ با «ResultRouter» (پکیج `packages/shared/engines`) از مسیر synchronous خارج شدند.
> `ResultRouter` تک نقطهای است که بعد از persist نتیجه، engineها را صدا میزند. حالت هر engine
> با `MONITOR_ENGINE_MODE` / `ALERT_ENGINE_MODE` (`inline` پیشفرض | `nats`) قابل بازگشت است.
> در حالت `nats`، API نتیجه را روی `engine.health.eval` / `engine.alert.eval` (stream جدید
> `ENGINE_EVENTS`) publish میکند و باینری مستقل مصرف میکند. دو تغییر رفتاری پذیرفتهشده:
> (الف) نتایج مسیر gateway هم alert میگیرند، (ب) health notifications فعال شدند
> (`EvaluateResult` حالا `[]EvaluateOutcome` برمیگرداند و `NotificationEngine` دیگر dead code نیست).

### تستهای اضافهشده
- `packages/shared/infrastructure/postgres/metric_query_service_test.go` (delegation)
- `packages/shared/interfaces/http/metric_query_service_test.go` (downsampling/`resolveStep`)
- `packages/shared/events/nats_test.go` (echo-drop, wire format, URL clean)
- `packages/shared/engines/router_test.go` (inline vs nats publish، mixed mode، nil-safe)
- `packages/shared/engines/consumer_test.go` (decode، redelivery on error، malformed ack)
- `packages/shared/health/outcomes_test.go` (state transition → outcome)

## فازهای باقیمانده ⏳

| فاز | شرح | قدم بعدی مشخص |
|---|---|---|
| **3** | Unified Registry — کاهش registryهای پراکنده، تعریف canonical | سندسازی + consolidation؛ کم‌ریسک |
| **4** | Collection Method abstraction (`probe\|agent\|snmp\|otel\|sdk\|api`) | در Domain/Schema اضافه شود |
| **5** | SNMP → Collection Method (نه نوع مانیتور) | بازطراحی UX + `monitor_types` |
| **7** | VictoriaMetrics به‌عنوان primary query (آداپتر دوم برای MetricQueryService) | آداپتر جدید + جایگزینی در `main.go` |
| **9** | Distributed Scheduler/Worker hardening (misfire, jitter, retry) | افزایش به `scheduler/scheduler.go` + `worker/` |
| **12** | Signal architecture (Event/Error/Log/Trace مدل‌های کانونیکال) | مدل‌ها + registry |
| **13** | Extension points برای Logs/Traces/Errors | بدون backend کامل — فقط ساختار |
| **14** | Load/Failure testing | k6/scripts + سناریوها |
| **15** | Legacy cleanup — حذف کامل `monitors.type` | backfill `monitor_type_id` + migration drop ستون |

## قدم بعدی پیشنهادی (فاز ۹)

با انجام فازهای ۱۰+۱۱، sink اصلی ingestion دیگر synchronous نیست. قدم بعدی
سختسازی Scheduler/Worker است: `misfire` (اجرای jobهای از دسترفته)، `jitter`
(جلوگیری از thundering herd)، و `retry` با backoff.

### شروع فاز ۹ (Scheduler/Worker hardening) — گام‌ها

1. **بررسی قبلی**: `packages/shared/scheduler/scheduler.go`, `packages/shared/worker/`,
   `packages/shared/queue/nats_queue.go`.
2. **درک مرز**: scheduler فعلاً `ClaimDue` + `FOR UPDATE SKIP LOCKED` روی
   PostgreSQL انجام میدهد؛ در حالت NATS job روی `probe.jobs.*` publish میشود.
3. **طراحی**: jitter تصادفی به اسکن scheduler، تشخیص misfire از روی
   `next_run_at` جاافتاده، و retry برای اجراهای ناموفق.
4. **فایلها**: `scheduler/scheduler.go` + گزینههای config جدید.
5. **Validation**: `go build ./...`, `go test ./...` + تست integration queue.

> ⚠️ فاز ۷ (VictoriaMetrics به‌عنوان primary query) جایگزین سادهتری است اگر بخواهی
> مسیر query را قبل از مقیاسپذیری اجرا بهبود بدهی.

## دستورات Validation

```bash
go build ./...          # build
go vet ./...            # vet
go test ./...           # همه تست‌های Go
# frontend (فقط اگر تغییر frontend دادید):
cd apps/web && npx tsc --noEmit
cd apps/web && pnpm test
```

## قواعد هنگام ادامه

1. قبل از هر تغییر، فایل‌های مربوطه را با Repowise/گشت بزن (AGENTS.md).
2. بدون `go build` + `go test` سبز، فاز بعدی را شروع نکن.
3. جابه‌جایی logic بین سرویس‌ها فقط با فهم مرزها (NATS subjects, callers).
4. هر فاز جدید = commit جداگانه با پیام شفاف.
5. پس از هر فاز، این فایل (migration-status.md) را به‌روز کن.
