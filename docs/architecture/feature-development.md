# Dog — Feature Development Guide (راهنمای رسمی تیم)

> اگر می‌خواهید یک Feature جدید به Dog اضافه کنید، این سند را دنبال کنید.
> هدف: هر Developer بداند **دقیقاً** کجا و **چطور** کد بنویسد — بدون اختراع
> ساختار جدید.

## جریان تصمیم قبل از شروع

هر Feature باید به این چهار پرسش پاسخ دهد:

1. **Resource**: روی چه نوع Resource ای کار می‌کند؟ (یا Resource جدید؟)
2. **Capability**: چه capability جدیدی است؟ (check؟ metric؟ log؟ trace؟ error؟ event؟)
3. **Signal**: خروجی اصلی چیست؟ (Check Result؟ Metric؟ Event؟ ...)
4. **Collection Method**: داده چگونه جمع می‌شود؟ (`probe | agent | snmp | otel | sdk | api`)

> اگر Feature شما «Metric از طریق SNMP» است: Signal=Metric، Collection=SNMP.
> اگر «Check از طریق Probe» است: Signal=Check، Collection=probe.

---

## مثال مرجع: افزودن «Redis Monitoring»

ما در طول این سند «Redis Monitoring» را به‌عنوان مثال پیاده می‌کنیم تا هر
قسمت مشخص باشد.

### ۱. Backend — Domain

**کجا**: `packages/shared/domain/`

- اگر مدل جدیدی لازم است: `packages/shared/domain/metrics/redis.go` یا در فایل
  موجود همان دامنه.
- اگر فقط یک نوع جدید از یک موجودیت موجود است، فقط registry را توسعه دهید.

```go
// packages/shared/domain/metrics/redis.go
package domain

type RedisMetrics struct {
    ConnectionsActive float64 `json:"connections_active"`
    MemoryUsedBytes   float64 `json:"memory_used_bytes"`
    HitRate           float64 `json:"hit_rate"`
}
```

> قانون: Domain نباید به HTTP/PostgreSQL/NATS/Redis/VictoriaMetrics وابسته باشد.

### ۲. Backend — Registry (Check/Metric)

**کجا**:

- **Check**: `packages/shared/probe/probe.go` (`DefaultRegistry`) + یک اکسیوکتور
  در `packages/shared/probe/redis.go`.
- **Monitor Type**: ردیف در جدول `monitor_types` (seed migration) +
  `MonitorTypeCode()` در `domain/observability.go` در صورت نیاز به mapping.
- **Health Parameter**: `packages/shared/health/catalog.go` (`AllParameters`).
- **SNMP OID**: فقط برای SNMP — `packages/shared/snmp/oids.go`.

```go
// packages/shared/probe/redis.go
func NewRedisExecutor(deps ExecutorDeps) Executor {
    return &redisExecutor{deps: deps}
}
// در DefaultRegistry:
//     registry.Register(domain.MonitorRedis, NewRedisExecutor(deps))
```

### ۳. Backend — Application Service

**کجا**: `packages/shared/application/`

اگر نیاز به use case یا پورت دارید:

- پورت: `packages/shared/application/redis/query_service.go` (interface)
- سرویس: پیاده‌سازی در `infrastructure/` (آداپتر).

> مثال موجود: `application/metricquery/query_service.go` + آداپتر آن در
> `infrastructure/postgres/metric_query_service.go`.

اگر Feature فقط از سرویس‌های موجود استفاده می‌کند، پورت جدید نسازید.

### ۴. Backend — Infrastructure

**کجا**: `packages/shared/infrastructure/postgres/`

- Repository جدید: `redis_repository.go` (آداپتر پورت از مرحله ۳).
- اگر storage جدید (VictoriaMetrics/Redis/...) لازم شد:
  `packages/shared/infrastructure/<backend>/...` — فقط در صورت نیاز.

### ۵. Database Migration

**کجا**: `migrations/NNNNNN_<name>.up.sql` + `.down.sql`

- فقط اگر schema جدید لازم است.
- طبق [migrations.md](migrations.md): idempotent، backward-aware، کوچک.
- برای seed کردن `monitor_types`، از `INSERT ... ON CONFLICT (slug) DO NOTHING`.

### ۶. Backend — API Contract

**کجا**: `packages/shared/interfaces/http/`

- Handler: `interfaces/http/redis_handler.go`
- مسیر: در `interfaces/http/router.go` ثبت کنید.
- Tenant isolation: از `resourceBelongsToOrg` / `monitorBelongsToOrg` استفاده
  کنید (یا معادل برای موجودیت جدید).
- خروجی با `writeJSON` / خطا با `writeDomainError`.

```go
router.Route("/resources/{resourceID}/redis", func(r chi.Router) {
    r.Use(orgScoped)
    r.Get("/metrics", handler.redisMetrics)
})
```

### ۷. Frontend — Entity

**کجا**: `apps/web/entities/redis/`

```
entities/redis/
  model/types.ts     ← interface RedisMetrics
  api/redis.api.ts   ← redisApi.listMetrics() → apiRequest<T>(endpoints.redis...)
  hooks/use-redis.ts ← React Query hook
  ui/                ← کامپوننت‌های نمایشی (در صورت نیاز)
```

### ۸. Frontend — Feature

**کجا**: `apps/web/features/`

```
features/redis-monitoring/
  components/
  hooks/
  schemas/
  types/
  ui/
  registry.ts
```

اگر Feature به مانیتور تایپ جدید مرتبط است (چک)، پلاگین UI:

### ۹. Frontend — Plugin Registry

**کجا**: `apps/web/plugins/monitoring/<type>/`

- `plugins/monitoring/redis/definition.ts` (الزامی) + اختیاری
  `configuration.tsx`, `summary.tsx`.
- در `plugins/monitoring/core/registry.ts` ثبت کنید (فقط UI metadata —
  business logic از Backend).

### ۱۰. Frontend — API Endpoints

**کجا**: `apps/web/shared/api/endpoints.ts`

- path جدید را اینجا اضافه کنید، **هرگز** در Component ها hard-code نکنید.

```ts
// shared/api/endpoints.ts
redis: {
  metrics: (resourceId: string) => `/resources/${resourceId}/redis/metrics`,
},
```

### ۱۱. Realtime Event (در صورت نیاز)

- اگر باید realtime به‌روزرسانی شود: یک event name در
  `apps/web/platform/realtime/events.ts` و invalidation در
  `use-live-results.ts` اضافه کنید.
- در Backend: بعد از ingestion، `events.Bus.Publish(...)` صدا بزنید.

### ۱۲. Health Rules (در صورت نیاز)

- برای Metricهای جدید، پارامتر health در `health/catalog.go` + seed در
  `monitor_types.health_parameters` اضافه کنید.

### ۱۳. Alert Policy (در صورت نیاز)

- Alert Policy روی پارامترهای health تعریف می‌شود (`alert_policies.conditions`).

### ۱۴. Tests

برای هر Feature حداقل:

| سطح | محل | مثال |
|---|---|---|
| Domain test | `packages/shared/domain/..._test.go` | validation قوانین |
| Application/port test | کنار آداپتر | `metric_query_service_test.go` |
| API test | `packages/shared/interfaces/http/..._test.go` | handler با fake repo |
| Frontend test | `apps/web/tests/` | hooks / api client |
| E2E | `apps/web/e2e/` (Playwright) | سناریوی کامل |
| Queue integration (اگر async) | کنار pipeline | مصرف/انتشار job |

### ۱۵. Documentation

- اگر ساختار یا جریان جدید معرفی کردید، این سند (feature-development.md) را
  به‌روز کنید.

---

## چک‌لیست نهایی Feature

- [ ] Domain تعریف شد (`packages/shared/domain/`)
- [ ] Capability تعریف شد
- [ ] Signal مشخص شد (Check/Metric/Log/Trace/Error/Event)
- [ ] Collection Method مشخص شد (probe/agent/snmp/otel/sdk/api)
- [ ] Registry Backend به‌روز شد (probe / health catalog / monitor_types)
- [ ] Migration DB (در صورت نیاز) — idempotent
- [ ] Application Service / Port (در صورت نیاز)
- [ ] API Contract (handler + route + tenant isolation)
- [ ] Frontend Entity (`entities/<name>/`)
- [ ] Frontend Feature (`features/<name>/`)
- [ ] Plugin Registry (`plugins/monitoring/<type>/`) — فقط در صورت چک
- [ ] Endpoint در `shared/api/endpoints.ts` ثبت شد
- [ ] Config Schema (در صورت نیاز)
- [ ] Health Rules (در صورت نیاز)
- [ ] Storage Adapter (در صورت نیاز)
- [ ] Realtime Event (در صورت نیاز)
- [ ] Tests (Domain/Application/API/Frontend/E2E)
- [ ] Documentation به‌روز شد

---

## قواعد طلایی

1. **پیش از ساخت، به دنبال مشابه موجود بگردید** — اگر دو implementation موازی
   دیدید، یکی باید Legacy/Deprecated باشد.
2. **ابسترکشن فقط وقتی که مصرف می‌شود** — پورت بدون آداپتر استفاده‌شده، پوچ است.
3. **Backend Source of Truth** — Frontend فقط UI metadata نگه می‌دارد.
4. **Tenant isolation** در همه queryها.
5. **Idempotency** برای job/result/event/alert/notification.
6. **هیچ کد جدیدی فقط برای اینکه «Enterprise» به نظر برسد** — معماری یعنی
   پیش‌بینی‌پذیری و مرزهای واضح، نه کد اضافی.
