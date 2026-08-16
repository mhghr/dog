# پلتفرم مانیتورینگ Agentless — فاز اول

پیاده‌سازی کامل فاز اول سند `monitoring.md`: یک پلتفرم مانیتورینگ Agentless با هشت نوع Probe، معماری Control Plane / Worker، نتایج زنده (SSE) و کنسول وب دوزبانه (فارسی/انگلیسی).

## معماری

```text
Web (Next.js)  ──REST/SSE──▶  API (Go/chi)
                                │        ▲
                                 ▼        │ POST /internal/results
                            PostgreSQL   │
                                         │
Scheduler (Go) ──XADD──▶ Redis Streams ──XREADGROUP──▶ Worker (Go)
     │                                                    │
     └── FOR UPDATE SKIP LOCKED                           └── HTTP/TCP/DNS/Ping/TLS/Domain/SMTP/NTP

Ingestion → PostgreSQL + VictoriaMetrics + SSE Event Bus
```

سه باینری مستقل از یک Codebase مشترک:

| باینری | مسئولیت |
|---|---|
| `cmd/api` | REST API، اعتبارسنجی، Ingestion نتایج، SSE Gateway، Health/Metrics |
| `cmd/scheduler` | یافتن Monitorهای Due با `FOR UPDATE SKIP LOCKED` و انتشار Job در Redis Stream |
| `cmd/worker` | مصرف Jobها با Consumer Group، اجرای Probe با Timeout/Retry، ارسال نتیجه |

## انواع مانیتور (۸ نوع)

`http` · `tcp` · `dns` · `ping` · `tls` · `domain_expiration` · `smtp` · `ntp`

هر نوع دارای: پیکربندی اختصاصی، اعتبارسنجی سمت سرور و کلاینت، حداقل Interval مخصوص (بخش ۵۱ سند)، کدهای خطای استاندارد (بخش ۲۹ و ۴۵–۴۸ سند) و متریک‌های اختصاصی.

## اجرای محلی

```bash
cp .env.example .env
docker compose up --build
```

| سرویس | آدرس |
|---|---|
| کنسول وب | http://localhost:2000 |
| API | http://localhost:8080 (کامپوز) / :5000 (لوکال طبق .env) |
| VictoriaMetrics UI | http://localhost:8428/vmui |

نمونه ساخت Monitor:

```bash
curl -X POST http://localhost:5000/api/monitors \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Example Website",
    "type": "http",
    "target": "https://example.com",
    "interval_seconds": 60,
    "timeout_millis": 5000,
    "retries": 1,
    "config": {
      "method": "GET",
      "expected_status_codes": [200],
      "follow_redirects": true,
      "verify_tls": true
    }
  }'
```

### اجرای بدون Docker (توسعه)

```bash
# وابستگی‌ها: PostgreSQL 16، Redis 7، VictoriaMetrics
migrate -path migrations -database "$DATABASE_URL" up

make run-api        # یا: go run ./cmd/api
make run-scheduler
make run-worker

pnpm install
pnpm dev                # Web :2000 + API :5000 + scheduler + worker
```

## API

```text
GET    /health/live | /health/ready | /metrics
POST   /api/auth/google/exchange   (وب: تبادل code — بدون auth)
POST   /api/auth/google/mobile     (موبایل: ارسال id_token گوگل)
POST   /api/auth/otp/request       (ارسال کد یکبارمصرف به موبایل)
POST   /api/auth/otp/verify        (تایید کد و ورود)
POST   /api/auth/refresh           (چرخش رفرش‌توکن)
POST   /api/auth/logout
GET    /api/auth/me                (نیازمند auth)
GET    /api/dashboard/summary                       (auth)
GET    /api/monitors?page=&page_size=&type=&status=&search=&sort=&order=   (auth)
POST   /api/monitors                                (auth)
GET    /api/monitors/{id}                           (auth)
PUT    /api/monitors/{id}                           (auth)
DELETE /api/monitors/{id}                           (auth)
POST   /api/monitors/{id}/pause | /resume           (auth)
GET    /api/monitors/{id}/results?limit=&page=      (auth)
GET    /api/monitors/{id}/metrics?from=&to=&step=auto  (auth)
GET    /api/probe-locations                         (auth)
GET    /api/system/health                           (auth)
GET    /events/stream            (SSE — auth با کوکی)
POST   /internal/results         (Bearer WORKER_TOKEN)
```

فرمت خطا (بخش ۳۹.۳۴ سند):

```json
{ "error": { "code": "validation_failed", "message": "...", "fields": { "target": ["..."] }, "request_id": "..." } }
```

## احراز هویت

بدون صفحه ثبت‌نام: با اولین لاگین موفق (گوگل یا موبایل)، کاربر به‌صورت خودکار در دیتابیس ساخته می‌شود.

### روش‌های ورود

1. **گوگل (وب)** — Authorization Code Flow:
   - `GET /api/auth/google/start` (روی وب، پورت 2000) → ریدایرکت به گوگل با `state` امضاشده در کوکی
   - گوگل → `GET /api/auth/callback/google` (مطابق کنسول گوگل) → تبادل code توسط API گو (secret فقط سمت API) → ست‌شدن کوکی‌ها → ریدایرکت به داشبورد
2. **گوگل (موبایل/Native)** — اپ id_token گوگل را به `POST /api/auth/google/mobile` می‌فرستد و توکن‌ها را در بدنه JSON می‌گیرد.
3. **موبایل (OTP)** — `otp/request` کد ۶ رقمی می‌سازد (TTL پنج دقیقه، حداکثر ۵ تلاش، Rate Limit: هر ۶۰ ثانیه یک‌بار و ۵ بار در ساعت). ارسال SMS پشت interface `SMSSender` است (پیش‌فرض `log`؛ در حالت توسعه کد در پاسخ برمی‌گردد و در UI نمایش داده می‌شود). شماره‌ها به E.164 نرمال می‌شوند (پشتیبانی ارقام فارسی و `09... → +989...`).

### توکن‌ها

- **Access Token**: JWT (HS256) با عمر ۱۵ دقیقه — در کوکی HttpOnly `mp_at` (وب) یا هدر `Authorization: Bearer` (موبایل).
- **Refresh Token**: رشته تصادفی ۳۸۴ بیتی با عمر ۳۰ روز — فقط hash (SHA-256) در جدول `auth_refresh_tokens` ذخیره می‌شود؛ کوکی HttpOnly `mp_rt`.
- **Rotation**: هر فراخوانی `refresh` توکن قبلی را باطل و توکن جدید صادر می‌کند. استفاده مجدد از توکن باطل‌شده (نشانه سرقت) باعث ابطال تمام نشست‌های کاربر می‌شود.
- کلاینت وب روی 401 به‌صورت شفاف یک‌بار refresh می‌کند و درخواست را تکرار می‌کند؛ تا وقتی رفرش‌توکن معتبر است هرگز لاگین مجدد خواسته نمی‌شود.
- Logout رفرش‌توکن را باطل و کوکی‌ها را پاک می‌کند.

### تنظیمات گوگل (کنسول Google Cloud)

```text
Authorized JavaScript origin:  http://localhost:2000
Authorized redirect URI:       http://localhost:2000/api/auth/callback/google
```

مقادیر در `.env` (ریشه، برای API) و `apps/web/.env.local` (برای وب) قرار دارند. `AUTH_JWT_SECRET` در Production باید حتماً تغییر کند (API خارج از development با secret پیش‌فرض بالا نمی‌آید).

## امنیت (SSRF)

- تمام اتصالات خروجی Probeها از `security.Guard` عبور می‌کنند: Resolve → اعتبارسنجی همه IPها → Dial فقط به IP تاییدشده (بدون Re-resolve؛ بستن حفره TOCTOU).
- رنج‌های Private/Reserved/Metadata (بخش ۲۶ سند) مسدود هستند؛ ریدایرکت‌های HTTP هم در لایه Dial دوباره بررسی می‌شوند.
- URLهای دارای Credential ممنوع؛ اندازه Body پاسخ محدود (1MB).
- `SSRF_ALLOW_PRIVATE=true` فقط برای توسعه محلی است (در compose برای Worker فعال است تا بتوانید سرویس‌های داخلی را تست کنید). **در Production حتماً false.**
- Ingestion داخلی با Bearer Token و مقایسه Constant-time محافظت می‌شود.
- هدرهای حساس هرگز لاگ نمی‌شوند؛ Drawer نتایج، کلیدهای حساس را نمایش نمی‌دهد.

## قابلیت اطمینان

- Idempotency: ایندکس یکتای `job_id` + `ON CONFLICT DO NOTHING` (ثبت تکراری = No-op).
- صف: Redis Streams با Consumer Group، `XAUTOCLAIM` برای بازیابی Jobهای رهاشده، Dead-letter برای پیام‌های سمی و عبور از سقف تحویل (۵ بار).
- Retry محلی Probe: `retries+1` تلاش با Backoff نیم‌ثانیه‌ای و Timeout مستقل هر تلاش.
- Graceful shutdown در هر سه باینری؛ Health Check باینری برای کانتینرهای distroless (`/api healthcheck`).
- Heartbeat کامپوننت‌ها در Redis با TTL → صفحه System Health (وضعیت Scheduler/Workerها + Lag صف).

## Observability

- لاگ ساخت‌یافته `slog` (JSON در Production) با `request_id`.
- `/metrics` پرومتئوسی در هر سه سرویس با متریک‌های بخش ۳۱ سند (jobs published/received/completed/failed، duplicate results، queue_pending_jobs و …).
- متریک‌های Probe به VictoriaMetrics (فرمت Prometheus Import، ارسال Async و Batch).

## فرانت‌اند

Next.js 16 (App Router) + TypeScript + Tailwind v4 + shadcn/ui (نصب‌شده فقط با CLI رسمی، `components.json` کامیت شده) + TanStack Query + RHF/Zod + ECharts + next-intl + next-themes.

- دوزبانه کامل fa/en با مسیر `/[locale]/…` و RTL/LTR (فونت فارسی Estedad به‌صورت Local).
- تم System/Light/Dark بدون Flash.
- صفحات: Landing، Login/Signup (استاب — Auth خارج از فاز ۱)، `/app/dashboard`، `/app/monitors` (فیلتر/جستجو/صفحه‌بندی)، ساخت/ویرایش با فرم Dynamic هشت نوع + خلاصه Config، جزئیات Monitor (Uptime/نمودارها/نتایج/Drawer)، `/app/locations`، `/app/system`، `/app/settings`، `/status/[slug]` (Placeholder).
- Live: SSE با Invalidation ترتل‌شده + Polling طبق جدول بخش ۳۹.۲۴ سند.

## تست‌ها

```bash
go test ./... -cover          # Go: domain, security(SSRF), probes (HTTP/TCP/DNS/NTP/SMTP/TLS با سرورهای Fake محلی), retry
pnpm --filter web test        # Vitest: formatters, schemas/config-builder
pnpm --filter web test:e2e    # Playwright (نیازمند استک بالا: compose + web dev)
```

وضعیت اجرا در این مخزن: `go build/vet/test` ✅ — `next build` ✅ — `vitest` ✅ (۱۵ تست) — `eslint` ✅

نکته: `-race` روی این ماشین (windows/386) در دسترس نیست؛ در CI لینوکسی `make test` را با `-race` اجرا کنید.

## تصمیم‌های پیاده‌سازی (انحراف‌های آگاهانه از سند)

1. **Enum کامل از ابتدا**: چون پروژه Greenfield است، هر ۸ نوع در `000001_init` تعریف شد (به‌جای `ALTER TYPE` در مایگریشن دوم).
2. **نویسنده واحد متریک**: سند در بخش ۴.۳ Worker و در بخش ۲۰ Ingestion را نویسنده VM معرفی می‌کند؛ برای جلوگیری از دوباره‌نویسی و ساده‌ماندن Workerها، فقط Ingestion به VM می‌نویسد.
3. **سری‌های نمودار از PostgreSQL**: Endpoint متریک‌ها با `date_bin` و Percentileهای PG محاسبه می‌شود (در مقیاس MVP دقیق و بدون وابستگی به VM). VM همچنان همه متریک‌ها را دریافت می‌کند و تعویض Backend این Endpoint یک نقطه توسعه مشخص است.
4. **NTP بدون وابستگی**: کلاینت NTP (RFC 5905، محاسبه Offset/RTT، بررسی Origin-timestamp و Kiss-of-Death) داخلی نوشته شد.
5. **Domain Expiration**: RDAP (rdap.org bootstrap) با Fallback WHOIS + کش In-memory یک‌ساعته برای رعایت Rate Limit رجیستری‌ها.
6. **Next 16 / shadcn v5**: فایل `proxy.ts` جایگزین `middleware.ts` شده (تغییر رسمی Next 16) و کامپوننت `form` قدیمی shadcn با `field` + RHF Controller جایگزین شده است. init با فلگ رسمی `--rtl` انجام شد.
7. **Worker در Docker با کاربر root + NET_RAW** برای ICMP Privileged (مستندسازی‌شده در `deployments/Dockerfile.worker`)؛ جایگزین: `sysctl net.ipv4.ping_group_range` و `PING_PRIVILEGED=false`.

## ساختار مخزن

```text
├── cmd/{api,scheduler,worker}/
├── internal/
│   ├── api/  config/  domain/  events/  heartbeat/  httpserver/
│   ├── ingestion/  logging/  metrics/  postgres/  probe/  queue/
│   ├── repository/  scheduler/  security/  worker/
├── migrations/
├── deployments/Dockerfile.{api,scheduler,worker}
├── apps/web/            ← Next.js console + landing
├── docker-compose.yml   ← postgres, redis, victoriametrics, migrate, api, scheduler, worker, web
├── Makefile  .env.example  go.mod  pnpm-workspace.yaml
└── monitoring.md        ← سند مرجع
```

## گام‌های بعد از MVP

Transactional Outbox، چند Probe Location، Alerting/Notification، Result Confirmation، RBAC و API Key، Status Page عمومی، uPlot برای نمودارهای پرتراکم، انتقال Query نمودارها به VictoriaMetrics.
