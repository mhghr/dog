# Dog — Observability (Self-Monitoring)

Dog خودش باید مانیتور شود. این سند توضیح می‌دهد که چگونه سلامت خود سرویس‌ها
سنجیده و قابل مشاهده می‌شود.

## Health Endpoints

هر سرویس باید داشته باشد:

| Endpoint | نقش | وابستگی |
|---|---|---|
| `/health/live` | Liveness — فقط بگوید process زنده است | هیچ وابستگی خارجی ندارد |
| `/health/ready` | Readiness — آیا می‌تواند ترافیک بگیرد | وابستگی‌ها را چک می‌کند (PostgreSQL، VictoriaMetrics، ...) |

پیاده‌سازی فعلی: `interfaces/http/health_handler.go` + `system_handler.go`.
- VictoriaMetrics به‌صورت غیر-fatal گزارش می‌شود (degraded، نه down).
- در `httpserver/server.go` healthcheck برای container استفاده می‌شود.

## Metrics (Prometheus)

- Registry: `packages/shared/metrics/prometheus.go` (`NewRegistry`).
- Endpoint: `GET /metrics` (Prom handler).
- Metric sets موجود: Scheduler، Worker، Ingestion، Telemetry
  (`metrics.NewIngestionMetrics(registry)` و ...).

## Metricهای مهم برای مانیتورینگ خود Dog

| دسته | مثال |
|---|---|
| API | latency، error rate، request count |
| DB | latency، pool usage |
| NATS | consumer lag، queue depth، delivery/ack |
| Worker | utilization، queue depth |
| Scheduler | lag، claim batch size |
| Ingestion | rate، error count، dedup hits |
| VictoriaMetrics | import latency، batch size |
| SSE | اتصال‌های باز، reconnect count، رویدادهای دریافتی/حذف‌شده |
| Agents/Probes | اتصال count، health |

## Logging

- Structured logging با `slog` (`packages/shared/logging/logger.go`).
- فیلدهای استاندارد: `service`, `level`, `timestamp`, `request_id`,
  `trace_id`, `workspace_id`, `resource_id`, `component`.
- Secrets و PII هرگز log نمی‌شوند.

## Heartbeat

- `packages/shared/heartbeat/heartbeat.go`: لiveness توزیع‌شده از طریق Redis
  keyها (برای componentهای distributed).
- Probe agents هر `last_seen_at` را در `probe_agents` به‌روز می‌کنند.

## Trace Correlation

- Correlation ID در pipeline حفظ می‌شود (`job_id`, `event_id`).
- برای distributed tracing داخلی (API → NATS → Worker → Ingestion →
  Storage) فیلدهای `trace_id`/`span_id` در جریان آینده (OTEL) اضافه می‌شوند.

## Checklist برای هر سرویس جدید

1. `/health/live` + `/health/ready` داشته باشد.
2. Metricهای Prometheus ثبت کند.
3. Structured logging با slog.
4. در docker-compose / deployment تنظیم شود.
