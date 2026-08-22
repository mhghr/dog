# Dog — Database Migrations

## ابزار و محل

- Migrations در `migrations/` با فرمت `NNNNNN_name.up.sql` / `.down.sql`.
- ابزار: `golang-migrate` (Makefile: `migrate-up`, `migrate-down`).
- ترتیب اجرا: عددی.

## قواعد اجباری

1. **Idempotent**: اجرای مجدد نباید خطا دهد (یا `IF NOT EXISTS` / `DO`
   block).
2. **Backward-aware**: هنگام تغییر ستون، data موجود را حفظ کن
   (مثل `000026_monitors_unify` که `monitors.type` را nullable کرد و
   `monitor_type_id` را کانونیکال اعلام کرد).
3. **Reversible where possible**: هر `.up.sql` یک `.down.sql` داشته باشد.
4. هر migration **کوچک** و با یک هدف.
5. در محیط‌های با drift، از migrationهای repair استفاده شده است
   (مثل `000027_resource_types_repair`, `000036_alert_policies_workspace`) —
   قبل از تغییر schema به خروجی واقعی هر محیط دقت کنید.

## نقش‌های جدولی

| حوزه | جدول‌ها | نکته |
|---|---|---|
| **Tenancy** | `organizations`, `workspaces`, `workspace_members`, `users` | org→workspace→resource |
| **Resources** | `resources`, `resource_types`, `tags`, `resource_tags` | capabilities در `resource_types` |
| **Monitoring** | `monitors`, `monitor_types`, `probe_assignments`, `monitor_jobs`, `probe_results`, `probe_locations` | `monitor_type_id` FK کانونیکال؛ `monitors.type` legacy |
| **Health** | `parameter_catalog`, `parameter_rules`, `parameter_health_state`, `resource_health_state`, `health_notification_channels`, `notification_policies` | `resource_health_state` وضعیت کل resource |
| **Alerting** | `notification_channels`, `alert_policies`, `alert_events` | `alert_events` عملاً جدول incidents |
| **Metrics** | `metric_series`, `metric_points`, `metric_rollups` | time-series فعلی PostgreSQL؛ VM مسیر اصلی scale |
| **Agents** | `probe_agents`, `probe_agent_enrollment_tokens`, `monitoring_agents`, `monitoring_bootstrap_tokens`, `monitoring_agent_heartbeats` (range partition), `monitoring_agent_configs` | دو سیستم agent مجزا |
| **SNMP** | `snmp_credentials`, `snmp_devices`, `snmp_discovery`, `snmp_interfaces`, `snmp_events`, `snmp_tasks` | credentials encrypted |
| **Audit** | `audit_logs` | action enum ~30 مورد |
| **Telemetry** | `telemetry_event_dedup`, `telemetry_dlq_events` | dedup + DLQ |
| **Status Pages** | `status_pages`, `status_page_components` | public projection |

## نکات مهاجرت مهم

### Legacy `monitors.type`

- تاریخچه: ستون enum `type` از `000001`؛ `monitor_type_id` در `000022` اضافه
  شد؛ `000026` ستون `type` را nullable کرد (Deprecated).
- وضعیت کد: queryهای قدیمی به `COALESCE(mt.executor_key, m.type::text)`
  migration شده‌اند (`result_repository.go`).
- قانون: **هیچ query جدید نباید `monitors.type` را بخواند.**
- حذف کامل: فاز legacy cleanup — ابتدا مطمئن شوید همه ردیف‌ها `monitor_type_id`
  دارند (backfill)، سپس `type` و enum `monitor_type` حذف شود.

### زمان‌سری

- `metric_points` index: BRIN روی `time` + PK `(time, series_id)`.
- `metric_rollups` برای aggregation در بازه‌های بزرگ.
- در Scale بالا، مسیر اصلی query به VictoriaMetrics منتقل می‌شود
  (از طریق `MetricQueryService` — [backend.md](backend.md)).

## چگونه یک migration جدید بنویسیم

1. `NNNNNN_<name>.up.sql` و `NNNNNN_<name>.down.sql` بسازید (شماره بعد از
   آخرین موجود؛ `000037` ...).
2. تغییرات را Backward-aware نگه دارید.
3. اگر enum/type هست، `DO $$ ... $$` یا `IF NOT EXISTS` برای idempotency.
4. Indexها را بر اساس query pattern طراحی کنید.
5. Test: روی دیتابیس محلی `make migrate-up` و `make migrate-down`.
