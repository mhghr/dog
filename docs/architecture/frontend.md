# Dog — Frontend Architecture

## پشته فنی

- **Next.js App Router** (با `[locale]` برای i18n و RTL/LTR)
- **React Query** برای state سمت سرور + prefetch/hydration
- **SSE** برای realtime (invalidation خفن)
- **shadcn/ui + Tailwind + Estedad**
- Feature-Sliced Design

## ساختار سطح بالا (`apps/web`)

```
apps/web/
  app/              ← صفحات Next.js (route handlers, layouts, pages)
  entities/         ← موجودیت‌های دامنه (resource, monitor, alert, agent, probe, ...)
  features/         ← قابلیت‌های محصول (monitor-management, probe-management, observability, ...)
  plugins/          ← پلاگین‌های مانیتور تایپ (monitoring/) — registry UI
  shared/           ← کد مشترک: api, data, hooks, types, ui, utils
  design-system/    ← توکن‌ها، کامپوننت‌ها، icons, themes (برند)
  widgets/          ← ویجت‌های مرکب (console-shell, dashboard, monitoring-map)
  platform/         ← auth, permissions, providers, realtime, state (zustand)
  i18n/, messages/  ← next-intl
  e2e/, tests/      ← Playwright + Vitest
```

## Convention هر Entity

هر entity در `entities/<name>/` از این زیرشاخه‌ها استفاده می‌کند:

| زیرشاخه | محتوا |
|---|---|
| `model/` | انواع (types), نتایج, metadata |
| `api/` | `xxx.api.ts` — توابع API با `apiRequest<T>` / `serverApiRequest<T>` |
| `hooks/` | React Query hooks (`use-xxx.ts`) |
| `ui/` | کامپوننت‌های نمایشی این entity |

مثال: `entities/monitor/` →
`model/{types,result,health,monitor-meta}.ts`, `api/monitor.api.ts`,
`hooks/{use-monitor,use-monitor-metrics,use-monitor-results,...}.ts`,
`ui/{monitor-detail-screen,node-monitor-tabs,...}.tsx`

## Feature Convention

هر feature در `features/<name>/`:

| زیرشاخه | محتوا |
|---|---|
| `components/` | کامپوننت‌های feature |
| `hooks/` | hooks مختص feature |
| `schemas/` | schemaهای فرم (zod یا معادل) |
| `types/` | types مختص feature |
| `ui/` | viewهای feature |
| `lib/` | منطق feature |
| `registry.ts` | ثبت feature (در صورت نیاز) |

مثال‌های موجود: `features/monitor-management/`, `features/probe-management/`,
`features/observability/{metrics,logs,traces,monitoring,resource-settings}/`,
`features/status-pages/`, `features/marketing/`.

## Plugin Registry (مانیتور تایپ‌ها)

- `plugins/monitoring/core/registry.ts`:
  - `MONITOR_TYPES`, `MONITOR_REGISTRY`, `MONITOR_TYPE_GROUPS`,
    `getMonitorDefinition(type)`, `getMonitorFormField(type, apiField)`.
- هر تایپ یک پوشه دارد: `plugins/monitoring/<type>/`
  (`definition.ts` الزامی + اختیاری `configuration.tsx`, `summary.tsx`).
- **مهم**: این registry فقط **UI metadata** نگه می‌دارد (icon, label, form,
  chart). Business logic و capabilities از Backend می‌آیند.

## API Client

- `shared/api/endpoints.ts`: **مرکز pathهای API**. UI نباید URL را hard-code
  کند؛ همه از `endpoints.<domain>...` استفاده می‌کنند.
- `shared/api/client.ts`: `apiRequest<T>` برای مرورگر (با refresh-token
  single-flight و redirect به login).
- `shared/api/server.ts`: `serverApiRequest<T>` برای Server Components /
  Route Handlers (با کوکی‌های HttpOnly و refresh rotation).
- هر entity توابع خود را در `entities/<name>/api/*.ts` با استفاده از
  `endpoints` تعریف می‌کند.

## Data Flow در UI

```
Server Component
   → serverApiRequest (prefetch)
   → React Query (hydration)
   → Client Component (use-xxx hooks)
   → SSE invalidation (platform/realtime)
   → Refetch
```

- Component نباید مستقیماً HTTP request بسازد — data fetching فقط در
  hooks/services.
- `platform/state/` شامل zustand storeها (console, user, workspace).

## Realtime / SSE

- `platform/realtime/sse-client.ts`: کلاس `SseClient` (EventSource با
  withCredentials، reconnect exponential backoff 1s→30s).
- `platform/realtime/use-live-results.ts`: تنها hook realtime؛ روی
  `/events/stream` گوش می‌دهد و invalidate را throttle می‌کند
  (`shared/data/realtime.ts`).
- معماری: `Data Event → Event Bus → API instances → SSE → Browser`.

## Charts

- `shared/ui/charts/`: `echart.tsx`, `chart-config.ts`, `sparkline.tsx`,
  `latency-chart.tsx`, `multi-location-chart.tsx`, `success-chart.tsx`.
- Downsampling در کلاینت انجام نمی‌شود؛ Backend `step_seconds` برمی‌گرداند و
  باندل نقاط را محدود می‌کند.

## Design System

- تمام UI جدید باید از `design-system/` و `shared/ui/` استفاده کند.
- از ساخت Button/Input/Card جدید بدون نیاز خودداری کنید.
- RTL/LTR با next-intl مدیریت می‌شود.
