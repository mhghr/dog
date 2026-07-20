# Local development (Windows + WSL)

در این حالت فقط Redis و VictoriaMetrics داخل Docker روی WSL اجرا می‌شوند. کد
Next.js و سرویس‌های Go روی ویندوز اجرا می‌شوند و PostgreSQL نصب‌شده روی ویندوز
استفاده می‌شود.

## پورت‌ها

| سرویس | آدرس |
|---|---|
| Web | `http://localhost:2000` |
| API | `http://localhost:5000` |
| Scheduler health | `http://localhost:5001` |
| Worker health | `http://localhost:5002` |
| Redis | `localhost:6380` |
| VictoriaMetrics | `http://localhost:8428/vmui` |
| PostgreSQL | `localhost:5432/datadog` |

## اجرا

از PowerShell و در ریشه پروژه:

```powershell
pnpm install
pnpm infra:up
pnpm db:migrate
pnpm dev
```

`pnpm dev` وب، API، scheduler و worker را هم‌زمان اجرا می‌کند و متغیرهای فایل
`.env` را برای همه آن‌ها بارگذاری می‌کند. برای متوقف‌کردن پردازه‌ها `Ctrl+C`
را بزنید.

برای مشاهده لاگ یا خاموش‌کردن زیرساخت:

```powershell
pnpm infra:logs
pnpm infra:down
```

اگر PostgreSQL اتصال Docker به ویندوز را قبول نکرد، در `postgresql.conf` مقدار
`listen_addresses` و در `pg_hba.conf` دسترسی شبکه Docker را بررسی کنید. اجرای
native بک‌اند همچنان با `DATABASE_URL` و `localhost` انجام می‌شود؛ فقط migration
داخل کانتینر از `host.docker.internal` استفاده می‌کند.
