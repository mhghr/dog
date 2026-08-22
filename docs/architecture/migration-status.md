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

### تستهای اضافهشده
- `packages/shared/infrastructure/postgres/metric_query_service_test.go` (delegation)
- `packages/shared/interfaces/http/metric_query_service_test.go` (downsampling/`resolveStep`)
- `packages/shared/events/nats_test.go` (echo-drop, wire format, URL clean)

## فازهای باقیمانده ⏳

| فاز | شرح | قدم بعدی مشخص |
|---|---|---|
| **3** | Unified Registry — کاهش registryهای پراکنده، تعریف canonical | سندسازی + consolidation؛ کم‌ریسک |
| **4** | Collection Method abstraction (`probe\|agent\|snmp\|otel\|sdk\|api`) | در Domain/Schema اضافه شود |
| **5** | SNMP → Collection Method (نه نوع مانیتور) | بازطراحی UX + `monitor_types` |
| **7** | VictoriaMetrics به‌عنوان primary query (آداپتر دوم برای MetricQueryService) | آداپتر جدید + جایگزینی در `main.go` |
| **9** | Distributed Scheduler/Worker hardening (misfire, jitter, retry) | افزایش به `scheduler/scheduler.go` + `worker/` |
| **10** | **Health Engine مستقل** — `apps/monitor-engine` (فعلاً stub) | جابه‌جایی منطق health از API به اپ مستقل + مصرف از queue |
| **11** | **Alert Engine مستقل** — `apps/alert-engine` (فعلاً stub) | همانند فاز ۱۰ برای alerting |
| **12** | Signal architecture (Event/Error/Log/Trace مدل‌های کانونیکال) | مدل‌ها + registry |
| **13** | Extension points برای Logs/Traces/Errors | بدون backend کامل — فقط ساختار |
| **14** | Load/Failure testing | k6/scripts + سناریوها |
| **15** | Legacy cleanup — حذف کامل `monitors.type` | backfill `monitor_type_id` + migration drop ستون |

## قدم بعدی پیشنهادی (فاز ۱۰ و ۱۱)

بزرگ‌ترین و مهم‌ترین قدم باقی‌مانده، خالی‌سازی stub های `monitor-engine` و
`alert-engine` است. منطق فعلاً داخل `apps/api` اجرا می‌شود
(`health.NewEngine` و `alerting.NewEngine` در `apps/api/cmd/api/main.go`).

### شروع فاز ۱۰ (Health Engine مستقل) — گام‌ها

1. **بررسی قبلی**: `packages/shared/health/engine.go`, `health/repository.go`,
   `health/catalog.go`, `apps/monitor-engine/cmd/monitor-engine/main.go` (استاب).
2. **درک مرز**: health evaluation الان در `ingestion.Service.Ingest`
   (`s.healthEngine.EvaluateResult`) صدا زده می‌شود — این را باید از مسیر
   synchronous خارج کرد.
3. **طراحی**: نتیجه در `probe_results` ذخیره می‌شود → publish رویداد
   (`telemetry.probe.result` یا subject جدید `health.eval`) → monitor-engine
   consume کند → `resource_health_state` آپدیت شود.
4. **فایلها**:
   - `apps/monitor-engine/cmd/monitor-engine/main.go` — wiring کامل (مثل API)
   - consumer جدید در `packages/shared/health/` برای مصرف queue
   - `ingestion.Service` → به‌جای اجرای synchronous، رویداد را publish کند
5. **Validation**: `go build ./...`, `go test ./...`, سپس تست یکپارچه‌سازی queue.

> ⚠️ مرز سرویس‌ها را رعایت کن (AGENTS.md): منطق health نباید دوباره در دو جا
> کپی شود. جابه‌جایی از API به اپ مستقل باید با feature flag قابل بازگشت باشد.

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
