# پیاده‌سازی فاز اول پلتفرم مانیتورینگ Agentless

## 1. هدف فاز اول

هدف این فاز، ساخت هسته یک پلتفرم مانیتورینگ Agentless است که بتواند بدون نصب Agent روی سرور مقصد، سرویس‌ها را از طریق Probeهای مختلف بررسی کند.

Probeهای این فاز:

- HTTP/HTTPS
- ICMP Ping
- TCP Connect
- DNS Query

قابلیت‌های اصلی:

- تعریف، ویرایش و حذف Monitor
- زمان‌بندی اجرای Monitorها
- اجرای Probeها توسط Worker
- ذخیره نتایج
- تعیین وضعیت `UP`، `DOWN` و `UNKNOWN`
- Retry پس از شکست
- مشاهده آخرین وضعیت و تاریخچه نتایج
- اجرای محلی با Docker Compose
- معماری قابل توسعه برای Multi-region و Alerting

---

## 2. محدوده MVP

### داخل محدوده

- یک Control Plane
- یک یا چند Probe Worker
- PostgreSQL برای داده‌های تنظیماتی
- Redis Streams برای صف اجرای Probeها
- VictoriaMetrics برای متریک‌های Time Series
- REST API
- اجرای دوره‌ای Monitorها
- ثبت نتیجه هر Probe
- Docker Compose
- لاگ ساخت‌یافته
- Health Check سرویس‌ها

### خارج از محدوده این فاز

- احراز هویت کامل و RBAC
- Multi-tenancy کامل
- Billing
- Alerting و Notification
- Incident Management
- Status Page
- Private Probe
- Browser Synthetic Monitoring
- SNMP
- APM، Log و Trace
- Kubernetes Deployment

---

## 3. معماری کلان

```text
                         ┌──────────────────────┐
                         │      Web Client      │
                         └──────────┬───────────┘
                                    │ REST
                                    ▼
┌────────────────────────────────────────────────────────┐
│                     Control Plane                      │
│                                                        │
│  ┌──────────────┐   ┌───────────────┐                 │
│  │ API Service  │   │   Scheduler   │                 │
│  └──────┬───────┘   └───────┬───────┘                 │
│         │                   │                          │
│         ▼                   ▼                          │
│  ┌──────────────┐   ┌───────────────┐                 │
│  │ PostgreSQL   │   │ Redis Streams │                 │
│  └──────────────┘   └───────┬───────┘                 │
└─────────────────────────────┼──────────────────────────┘
                              │ Probe Job
                              ▼
                    ┌─────────────────────┐
                    │    Probe Worker     │
                    │                     │
                    │ HTTP / ICMP / TCP   │
                    │ DNS Probe Executor  │
                    └──────────┬──────────┘
                               │
              ┌────────────────┴─────────────────┐
              │                                  │
              ▼                                  ▼
      ┌──────────────────┐              ┌──────────────────┐
      │ Result Ingestion │              │ VictoriaMetrics  │
      └────────┬─────────┘              └──────────────────┘
               │
               ▼
       ┌──────────────┐
       │ PostgreSQL   │
       └──────────────┘
```

---

## 4. اجزای سیستم

### 4.1 API Service

مسئولیت‌ها:

- مدیریت Monitorها
- اعتبارسنجی ورودی
- ارائه آخرین وضعیت
- ارائه تاریخچه اجرای Probeها
- دریافت نتیجه Workerها
- Health Check

### 4.2 Scheduler

مسئولیت‌ها:

- یافتن Monitorهایی که زمان اجرای آن‌ها رسیده است
- تولید Probe Job
- انتشار Job در Redis Stream
- جلوگیری از اجرای تکراری
- به‌روزرسانی زمان اجرای بعدی

### 4.3 Probe Worker

مسئولیت‌ها:

- دریافت Job از Redis Stream
- اجرای Probe متناسب با نوع Monitor
- اعمال Timeout
- Retry محلی
- تولید نتیجه استاندارد
- ارسال نتیجه به Ingestion API
- ثبت Metric در VictoriaMetrics

### 4.4 PostgreSQL

داده‌های اصلی:

- Monitor
- Probe Location
- Probe Result Summary
- Scheduler State

### 4.5 Redis Streams

کاربرد:

- صف Probe Job
- Consumer Group برای Workerها
- Ack کردن Jobهای موفق
- بازیابی Jobهای رهاشده

### 4.6 VictoriaMetrics

ذخیره متریک‌های عددی:

- Latency
- DNS Duration
- TCP Connect Duration
- TLS Duration
- HTTP Total Duration
- Packet Loss
- DNS Response Time

---

## 5. تکنولوژی پیشنهادی

| بخش | تکنولوژی |
|---|---|
| زبان Backend | Go 1.24+ |
| HTTP Framework | Chi |
| Database | PostgreSQL 16 |
| Queue | Redis Streams |
| Time Series | VictoriaMetrics |
| Migration | golang-migrate |
| SQL Layer | sqlc یا pgx |
| Logging | slog |
| Config | Environment Variables |
| Container | Docker |
| Local Orchestration | Docker Compose |
| Metrics | Prometheus Client |
| Tracing آینده | OpenTelemetry |

---

## 6. ساختار پروژه

```text
monitoring-platform/
├── cmd/
│   ├── api/
│   │   └── main.go
│   ├── scheduler/
│   │   └── main.go
│   └── worker/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── domain/
│   │   ├── monitor.go
│   │   ├── job.go
│   │   └── result.go
│   ├── repository/
│   │   ├── monitor_repository.go
│   │   └── result_repository.go
│   ├── postgres/
│   │   ├── monitor_repository.go
│   │   └── result_repository.go
│   ├── queue/
│   │   └── redis_stream.go
│   ├── scheduler/
│   │   └── scheduler.go
│   ├── probe/
│   │   ├── probe.go
│   │   ├── http.go
│   │   ├── tcp.go
│   │   ├── dns.go
│   │   └── ping.go
│   ├── ingestion/
│   │   └── service.go
│   ├── api/
│   │   ├── router.go
│   │   ├── monitor_handler.go
│   │   └── result_handler.go
│   └── metrics/
│       └── victoria.go
├── migrations/
│   ├── 000001_init.up.sql
│   └── 000001_init.down.sql
├── deployments/
│   ├── Dockerfile.api
│   ├── Dockerfile.scheduler
│   └── Dockerfile.worker
├── docker-compose.yml
├── Makefile
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

---

## 7. مدل دامنه

### 7.1 Monitor

```go
package domain

import "time"

type MonitorType string

const (
	MonitorHTTP MonitorType = "http"
	MonitorTCP  MonitorType = "tcp"
	MonitorDNS  MonitorType = "dns"
	MonitorPing MonitorType = "ping"
)

type MonitorStatus string

const (
	StatusUp      MonitorStatus = "up"
	StatusDown    MonitorStatus = "down"
	StatusUnknown MonitorStatus = "unknown"
	StatusPaused  MonitorStatus = "paused"
)

type Monitor struct {
	ID              string
	Name            string
	Type            MonitorType
	Target          string
	IntervalSeconds int
	TimeoutMillis   int
	Retries         int
	Enabled         bool
	Config          map[string]any
	LastStatus      MonitorStatus
	LastCheckedAt   *time.Time
	NextRunAt       time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
```

### 7.2 Probe Job

```go
package domain

import "time"

type ProbeJob struct {
	ID              string         `json:"id"`
	MonitorID       string         `json:"monitor_id"`
	Type            MonitorType    `json:"type"`
	Target          string         `json:"target"`
	TimeoutMillis   int            `json:"timeout_millis"`
	Retries         int            `json:"retries"`
	Config          map[string]any `json:"config"`
	ProbeLocationID string         `json:"probe_location_id"`
	ScheduledAt     time.Time      `json:"scheduled_at"`
}
```

### 7.3 Probe Result

```go
package domain

import "time"

type ProbeResult struct {
	ID              string         `json:"id"`
	JobID           string         `json:"job_id"`
	MonitorID       string         `json:"monitor_id"`
	ProbeLocationID string         `json:"probe_location_id"`
	Status          MonitorStatus  `json:"status"`
	Success         bool           `json:"success"`
	ErrorCode       string         `json:"error_code,omitempty"`
	ErrorMessage    string         `json:"error_message,omitempty"`
	DurationMillis  int64          `json:"duration_millis"`
	Metrics         map[string]any `json:"metrics"`
	Attributes      map[string]any `json:"attributes"`
	StartedAt       time.Time      `json:"started_at"`
	FinishedAt      time.Time      `json:"finished_at"`
}
```

---

## 8. طرح دیتابیس

### فایل `migrations/000001_init.up.sql`

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE monitor_type AS ENUM (
    'http',
    'tcp',
    'dns',
    'ping'
);

CREATE TYPE monitor_status AS ENUM (
    'up',
    'down',
    'unknown',
    'paused'
);

CREATE TABLE probe_locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL UNIQUE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE monitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    type monitor_type NOT NULL,
    target TEXT NOT NULL,
    interval_seconds INTEGER NOT NULL CHECK (interval_seconds >= 10),
    timeout_millis INTEGER NOT NULL DEFAULT 5000
        CHECK (timeout_millis BETWEEN 100 AND 60000),
    retries INTEGER NOT NULL DEFAULT 1
        CHECK (retries BETWEEN 0 AND 5),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_status monitor_status NOT NULL DEFAULT 'unknown',
    last_checked_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX monitors_next_run_idx
    ON monitors(next_run_at)
    WHERE enabled = TRUE;

CREATE TABLE probe_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    probe_location_id UUID REFERENCES probe_locations(id),
    status monitor_status NOT NULL,
    success BOOLEAN NOT NULL,
    error_code VARCHAR(100),
    error_message TEXT,
    duration_millis BIGINT NOT NULL,
    metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX probe_results_job_id_idx
    ON probe_results(job_id);

CREATE INDEX probe_results_monitor_time_idx
    ON probe_results(monitor_id, started_at DESC);

INSERT INTO probe_locations(name, code)
VALUES ('Local Development', 'local-dev');
```

### فایل `migrations/000001_init.down.sql`

```sql
DROP TABLE IF EXISTS probe_results;
DROP TABLE IF EXISTS monitors;
DROP TABLE IF EXISTS probe_locations;
DROP TYPE IF EXISTS monitor_status;
DROP TYPE IF EXISTS monitor_type;
```

---

## 9. قرارداد API

### 9.1 ایجاد Monitor

```http
POST /api/v1/monitors
Content-Type: application/json
```

نمونه درخواست HTTP:

```json
{
  "name": "Main Website",
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
}
```

نمونه پاسخ:

```json
{
  "id": "c3555c21-9540-4985-a7db-0dd58fd74c35",
  "name": "Main Website",
  "type": "http",
  "target": "https://example.com",
  "last_status": "unknown",
  "next_run_at": "2026-07-16T12:00:00Z"
}
```

### 9.2 فهرست Monitorها

```http
GET /api/v1/monitors
```

### 9.3 جزئیات Monitor

```http
GET /api/v1/monitors/{monitor_id}
```

### 9.4 ویرایش Monitor

```http
PUT /api/v1/monitors/{monitor_id}
```

### 9.5 حذف Monitor

```http
DELETE /api/v1/monitors/{monitor_id}
```

### 9.6 Pause و Resume

```http
POST /api/v1/monitors/{monitor_id}/pause
POST /api/v1/monitors/{monitor_id}/resume
```

### 9.7 نتایج Monitor

```http
GET /api/v1/monitors/{monitor_id}/results?limit=100
```

### 9.8 دریافت نتیجه Worker

```http
POST /internal/v1/results
Authorization: Bearer <worker-token>
```

---

## 10. پیکربندی Probeها

### 10.1 HTTP

```json
{
  "method": "GET",
  "headers": {
    "User-Agent": "MonitoringPlatform/1.0"
  },
  "body": "",
  "expected_status_codes": [200, 204],
  "body_contains": "healthy",
  "follow_redirects": true,
  "max_redirects": 5,
  "verify_tls": true
}
```

### 10.2 TCP

```json
{
  "port": 5432
}
```

Target می‌تواند یکی از این دو حالت باشد:

```text
db.example.com
db.example.com:5432
```

### 10.3 DNS

```json
{
  "server": "1.1.1.1:53",
  "record_type": "A",
  "expected_values": [
    "203.0.113.10"
  ]
}
```

### 10.4 Ping

```json
{
  "packet_count": 4,
  "packet_interval_millis": 200
}
```

---

## 11. Interface مشترک Probe

```go
package probe

import (
	"context"

	"monitoring-platform/internal/domain"
)

type Executor interface {
	Type() domain.MonitorType
	Execute(
		ctx context.Context,
		job domain.ProbeJob,
	) domain.ProbeResult
}

type Registry struct {
	executors map[domain.MonitorType]Executor
}

func NewRegistry(executors ...Executor) *Registry {
	registry := &Registry{
		executors: make(map[domain.MonitorType]Executor),
	}

	for _, executor := range executors {
		registry.executors[executor.Type()] = executor
	}

	return registry
}

func (r *Registry) Get(
	monitorType domain.MonitorType,
) (Executor, bool) {
	executor, ok := r.executors[monitorType]
	return executor, ok
}
```

---

## 12. پیاده‌سازی HTTP Probe

```go
package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"monitoring-platform/internal/domain"
)

type HTTPExecutor struct{}

func NewHTTPExecutor() *HTTPExecutor {
	return &HTTPExecutor{}
}

func (e *HTTPExecutor) Type() domain.MonitorType {
	return domain.MonitorHTTP
}

func (e *HTTPExecutor) Execute(
	ctx context.Context,
	job domain.ProbeJob,
) domain.ProbeResult {
	startedAt := time.Now()

	result := domain.ProbeResult{
		ID:              uuid.NewString(),
		JobID:           job.ID,
		MonitorID:       job.MonitorID,
		ProbeLocationID: job.ProbeLocationID,
		Status:          domain.StatusDown,
		Success:         false,
		Metrics:         map[string]any{},
		Attributes:      map[string]any{},
		StartedAt:       startedAt,
	}

	method := stringConfig(job.Config, "method", http.MethodGet)
	body := stringConfig(job.Config, "body", "")
	verifyTLS := boolConfig(job.Config, "verify_tls", true)
	followRedirects := boolConfig(
		job.Config,
		"follow_redirects",
		true,
	)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: !verifyTLS,
		},
		DisableKeepAlives: true,
	}

	client := &http.Client{
		Transport: transport,
	}

	if !followRedirects {
		client.CheckRedirect = func(
			req *http.Request,
			via []*http.Request,
		) error {
			return http.ErrUseLastResponse
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		job.Target,
		strings.NewReader(body),
	)
	if err != nil {
		return finishFailure(result, "invalid_request", err)
	}

	if rawHeaders, ok := job.Config["headers"].(map[string]any); ok {
		for key, value := range rawHeaders {
			req.Header.Set(key, fmt.Sprint(value))
		}
	}

	response, err := client.Do(req)
	if err != nil {
		return finishFailure(result, "http_request_failed", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(
		io.LimitReader(response.Body, 1024*1024),
	)
	if err != nil {
		return finishFailure(result, "body_read_failed", err)
	}

	result.Attributes["status_code"] = response.StatusCode
	result.Attributes["content_length"] = len(responseBody)
	result.Attributes["final_url"] = response.Request.URL.String()

	expectedCodes := intSliceConfig(
		job.Config,
		"expected_status_codes",
		[]int{http.StatusOK},
	)

	if !containsInt(expectedCodes, response.StatusCode) {
		return finishFailure(
			result,
			"unexpected_status_code",
			fmt.Errorf(
				"expected one of %v, received %d",
				expectedCodes,
				response.StatusCode,
			),
		)
	}

	expectedBody := stringConfig(job.Config, "body_contains", "")
	if expectedBody != "" &&
		!strings.Contains(string(responseBody), expectedBody) {
		return finishFailure(
			result,
			"body_assertion_failed",
			fmt.Errorf("expected text was not found"),
		)
	}

	finishedAt := time.Now()
	result.Success = true
	result.Status = domain.StatusUp
	result.FinishedAt = finishedAt
	result.DurationMillis = finishedAt.Sub(startedAt).Milliseconds()
	result.Metrics["total_duration_ms"] = result.DurationMillis

	if response.TLS != nil && len(response.TLS.PeerCertificates) > 0 {
		certificate := response.TLS.PeerCertificates[0]
		result.Attributes["tls_issuer"] = certificate.Issuer.String()
		result.Attributes["tls_expires_at"] = certificate.NotAfter
		result.Attributes["tls_days_remaining"] =
			int(time.Until(certificate.NotAfter).Hours() / 24)
	}

	return result
}
```

توابع کمکی:

```go
package probe

import (
	"fmt"
	"time"

	"monitoring-platform/internal/domain"
)

func finishFailure(
	result domain.ProbeResult,
	code string,
	err error,
) domain.ProbeResult {
	finishedAt := time.Now()

	result.Success = false
	result.Status = domain.StatusDown
	result.ErrorCode = code
	result.ErrorMessage = err.Error()
	result.FinishedAt = finishedAt
	result.DurationMillis =
		finishedAt.Sub(result.StartedAt).Milliseconds()
	result.Metrics["total_duration_ms"] = result.DurationMillis

	return result
}

func stringConfig(
	config map[string]any,
	key string,
	defaultValue string,
) string {
	value, ok := config[key]
	if !ok {
		return defaultValue
	}

	result, ok := value.(string)
	if !ok {
		return defaultValue
	}

	return result
}

func boolConfig(
	config map[string]any,
	key string,
	defaultValue bool,
) bool {
	value, ok := config[key]
	if !ok {
		return defaultValue
	}

	result, ok := value.(bool)
	if !ok {
		return defaultValue
	}

	return result
}

func intSliceConfig(
	config map[string]any,
	key string,
	defaultValue []int,
) []int {
	value, ok := config[key]
	if !ok {
		return defaultValue
	}

	values, ok := value.([]any)
	if !ok {
		return defaultValue
	}

	result := make([]int, 0, len(values))
	for _, item := range values {
		switch number := item.(type) {
		case float64:
			result = append(result, int(number))
		case int:
			result = append(result, number)
		}
	}

	if len(result) == 0 {
		return defaultValue
	}

	return result
}

func containsInt(values []int, expected int) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}

	return false
}

func errorf(format string, values ...any) error {
	return fmt.Errorf(format, values...)
}
```

---

## 13. پیاده‌سازی TCP Probe

```go
package probe

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"

	"monitoring-platform/internal/domain"
)

type TCPExecutor struct{}

func NewTCPExecutor() *TCPExecutor {
	return &TCPExecutor{}
}

func (e *TCPExecutor) Type() domain.MonitorType {
	return domain.MonitorTCP
}

func (e *TCPExecutor) Execute(
	ctx context.Context,
	job domain.ProbeJob,
) domain.ProbeResult {
	startedAt := time.Now()

	result := domain.ProbeResult{
		ID:              uuid.NewString(),
		JobID:           job.ID,
		MonitorID:       job.MonitorID,
		ProbeLocationID: job.ProbeLocationID,
		Status:          domain.StatusDown,
		Metrics:         map[string]any{},
		Attributes:      map[string]any{},
		StartedAt:       startedAt,
	}

	target := job.Target
	if _, _, err := net.SplitHostPort(target); err != nil {
		port := stringConfig(job.Config, "port", "")
		if port == "" {
			return finishFailure(
				result,
				"invalid_target",
				fmt.Errorf("TCP port is required"),
			)
		}

		target = net.JoinHostPort(target, port)
	}

	dialer := net.Dialer{}
	connection, err := dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		return finishFailure(result, "tcp_connect_failed", err)
	}
	defer connection.Close()

	finishedAt := time.Now()

	result.Success = true
	result.Status = domain.StatusUp
	result.FinishedAt = finishedAt
	result.DurationMillis = finishedAt.Sub(startedAt).Milliseconds()
	result.Metrics["connect_duration_ms"] = result.DurationMillis
	result.Attributes["remote_address"] =
		connection.RemoteAddr().String()

	return result
}
```

---

## 14. پیاده‌سازی DNS Probe

کتابخانه پیشنهادی:

```bash
go get github.com/miekg/dns
```

```go
package probe

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/miekg/dns"

	"monitoring-platform/internal/domain"
)

type DNSExecutor struct{}

func NewDNSExecutor() *DNSExecutor {
	return &DNSExecutor{}
}

func (e *DNSExecutor) Type() domain.MonitorType {
	return domain.MonitorDNS
}

func (e *DNSExecutor) Execute(
	ctx context.Context,
	job domain.ProbeJob,
) domain.ProbeResult {
	startedAt := time.Now()

	result := domain.ProbeResult{
		ID:              uuid.NewString(),
		JobID:           job.ID,
		MonitorID:       job.MonitorID,
		ProbeLocationID: job.ProbeLocationID,
		Status:          domain.StatusDown,
		Metrics:         map[string]any{},
		Attributes:      map[string]any{},
		StartedAt:       startedAt,
	}

	server := stringConfig(
		job.Config,
		"server",
		"1.1.1.1:53",
	)
	recordType := strings.ToUpper(
		stringConfig(job.Config, "record_type", "A"),
	)

	queryType, ok := dns.StringToType[recordType]
	if !ok {
		return finishFailure(
			result,
			"invalid_record_type",
			fmt.Errorf("unsupported DNS record type: %s", recordType),
		)
	}

	message := new(dns.Msg)
	message.SetQuestion(
		dns.Fqdn(job.Target),
		queryType,
	)
	message.RecursionDesired = true

	client := &dns.Client{
		Net: "udp",
	}

	response, duration, err := client.ExchangeContext(
		ctx,
		message,
		server,
	)
	if err != nil {
		return finishFailure(result, "dns_query_failed", err)
	}

	if response.Rcode != dns.RcodeSuccess {
		return finishFailure(
			result,
			"dns_rcode_failed",
			fmt.Errorf(
				"DNS returned RCODE %s",
				dns.RcodeToString[response.Rcode],
			),
		)
	}

	values := make([]string, 0, len(response.Answer))
	for _, answer := range response.Answer {
		switch record := answer.(type) {
		case *dns.A:
			values = append(values, record.A.String())
		case *dns.AAAA:
			values = append(values, record.AAAA.String())
		case *dns.CNAME:
			values = append(values, record.Target)
		case *dns.MX:
			values = append(values, record.Mx)
		case *dns.TXT:
			values = append(values, strings.Join(record.Txt, ""))
		case *dns.NS:
			values = append(values, record.Ns)
		}
	}

	if len(values) == 0 {
		return finishFailure(
			result,
			"empty_dns_answer",
			fmt.Errorf("DNS response contains no matching answers"),
		)
	}

	expectedValues := stringSliceConfig(
		job.Config,
		"expected_values",
		nil,
	)

	if len(expectedValues) > 0 &&
		!hasCommonValue(values, expectedValues) {
		return finishFailure(
			result,
			"dns_assertion_failed",
			fmt.Errorf(
				"received values %v do not match expected values %v",
				values,
				expectedValues,
			),
		)
	}

	finishedAt := time.Now()

	result.Success = true
	result.Status = domain.StatusUp
	result.FinishedAt = finishedAt
	result.DurationMillis = finishedAt.Sub(startedAt).Milliseconds()
	result.Metrics["dns_duration_ms"] = duration.Milliseconds()
	result.Attributes["answers"] = values
	result.Attributes["rcode"] = dns.RcodeToString[response.Rcode]
	result.Attributes["server"] = server

	return result
}

func stringSliceConfig(
	config map[string]any,
	key string,
	defaultValue []string,
) []string {
	value, ok := config[key]
	if !ok {
		return defaultValue
	}

	values, ok := value.([]any)
	if !ok {
		return defaultValue
	}

	result := make([]string, 0, len(values))
	for _, item := range values {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}

	return result
}

func hasCommonValue(
	actual []string,
	expected []string,
) bool {
	for _, actualValue := range actual {
		for _, expectedValue := range expected {
			if actualValue == expectedValue {
				return true
			}

			actualIP := net.ParseIP(actualValue)
			expectedIP := net.ParseIP(expectedValue)

			if actualIP != nil &&
				expectedIP != nil &&
				actualIP.Equal(expectedIP) {
				return true
			}
		}
	}

	return false
}
```

---

## 15. پیاده‌سازی ICMP Ping Probe

Ping خام در Linux نیازمند Capability است.

کتابخانه پیشنهادی:

```bash
go get github.com/prometheus-community/pro-bing
```

```go
package probe

import (
	"context"
	"fmt"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/google/uuid"

	"monitoring-platform/internal/domain"
)

type PingExecutor struct{}

func NewPingExecutor() *PingExecutor {
	return &PingExecutor{}
}

func (e *PingExecutor) Type() domain.MonitorType {
	return domain.MonitorPing
}

func (e *PingExecutor) Execute(
	ctx context.Context,
	job domain.ProbeJob,
) domain.ProbeResult {
	startedAt := time.Now()

	result := domain.ProbeResult{
		ID:              uuid.NewString(),
		JobID:           job.ID,
		MonitorID:       job.MonitorID,
		ProbeLocationID: job.ProbeLocationID,
		Status:          domain.StatusDown,
		Metrics:         map[string]any{},
		Attributes:      map[string]any{},
		StartedAt:       startedAt,
	}

	pinger, err := probing.NewPinger(job.Target)
	if err != nil {
		return finishFailure(result, "invalid_ping_target", err)
	}

	packetCount := intConfig(job.Config, "packet_count", 4)
	pinger.Count = packetCount
	pinger.Timeout = time.Duration(job.TimeoutMillis) * time.Millisecond
	pinger.SetPrivileged(true)

	runDone := make(chan error, 1)
	go func() {
		runDone <- pinger.Run()
	}()

	select {
	case <-ctx.Done():
		pinger.Stop()
		return finishFailure(
			result,
			"ping_timeout",
			ctx.Err(),
		)
	case err := <-runDone:
		if err != nil {
			return finishFailure(result, "ping_failed", err)
		}
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv == 0 {
		return finishFailure(
			result,
			"packet_loss_100",
			fmt.Errorf("no ICMP reply received"),
		)
	}

	finishedAt := time.Now()

	result.Success = true
	result.Status = domain.StatusUp
	result.FinishedAt = finishedAt
	result.DurationMillis = finishedAt.Sub(startedAt).Milliseconds()
	result.Metrics["packet_loss_percent"] = stats.PacketLoss
	result.Metrics["min_rtt_ms"] = stats.MinRtt.Milliseconds()
	result.Metrics["avg_rtt_ms"] = stats.AvgRtt.Milliseconds()
	result.Metrics["max_rtt_ms"] = stats.MaxRtt.Milliseconds()
	result.Metrics["stddev_rtt_ms"] = stats.StdDevRtt.Milliseconds()
	result.Attributes["packets_sent"] = stats.PacketsSent
	result.Attributes["packets_received"] = stats.PacketsRecv

	return result
}

func intConfig(
	config map[string]any,
	key string,
	defaultValue int,
) int {
	value, ok := config[key]
	if !ok {
		return defaultValue
	}

	switch number := value.(type) {
	case int:
		return number
	case float64:
		return int(number)
	default:
		return defaultValue
	}
}
```

در Docker برای Worker باید Capability زیر اضافه شود:

```yaml
cap_add:
  - NET_RAW
```

---

## 16. اجرای Probe با Retry

```go
package probe

import (
	"context"
	"time"

	"monitoring-platform/internal/domain"
)

func ExecuteWithRetry(
	parentCtx context.Context,
	executor Executor,
	job domain.ProbeJob,
) domain.ProbeResult {
	attempts := job.Retries + 1
	var lastResult domain.ProbeResult

	for attempt := 1; attempt <= attempts; attempt++ {
		ctx, cancel := context.WithTimeout(
			parentCtx,
			time.Duration(job.TimeoutMillis)*time.Millisecond,
		)

		lastResult = executor.Execute(ctx, job)
		cancel()

		lastResult.Attributes["attempt"] = attempt
		lastResult.Attributes["max_attempts"] = attempts

		if lastResult.Success {
			return lastResult
		}

		if attempt < attempts {
			select {
			case <-parentCtx.Done():
				return lastResult
			case <-time.After(500 * time.Millisecond):
			}
		}
	}

	return lastResult
}
```

---

## 17. Redis Streams

### Streamها

```text
probe_jobs
probe_results_dead_letter
```

### Consumer Group

```text
probe_workers
```

### Job Payload

```json
{
  "id": "639001b0-bf60-4cfa-844c-c94d08a095ba",
  "monitor_id": "1d055209-b7f4-41cb-8505-1f55bf7ed608",
  "type": "http",
  "target": "https://example.com",
  "timeout_millis": 5000,
  "retries": 1,
  "probe_location_id": "18ad722d-47f8-4e6e-934c-c57223a63cd5",
  "config": {
    "method": "GET",
    "expected_status_codes": [200]
  },
  "scheduled_at": "2026-07-16T12:00:00Z"
}
```

### انتشار Job

```go
func (q *RedisQueue) Publish(
	ctx context.Context,
	job domain.ProbeJob,
) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal probe job: %w", err)
	}

	return q.client.XAdd(
		ctx,
		&redis.XAddArgs{
			Stream: "probe_jobs",
			Values: map[string]any{
				"payload": payload,
			},
		},
	).Err()
}
```

### دریافت Job

```go
func (q *RedisQueue) Consume(
	ctx context.Context,
	consumerName string,
) ([]redis.XMessage, error) {
	streams, err := q.client.XReadGroup(
		ctx,
		&redis.XReadGroupArgs{
			Group:    "probe_workers",
			Consumer: consumerName,
			Streams:  []string{"probe_jobs", ">"},
			Count:    10,
			Block:    5 * time.Second,
		},
	).Result()

	if err != nil && err != redis.Nil {
		return nil, err
	}

	if len(streams) == 0 {
		return nil, nil
	}

	return streams[0].Messages, nil
}
```

---

## 18. Scheduler

الگوریتم ساده Scheduler:

1. هر یک ثانیه اجرا شود.
2. Monitorهای Due را با `FOR UPDATE SKIP LOCKED` بخواند.
3. برای هر Monitor یک Job تولید کند.
4. Job را در Redis منتشر کند.
5. `next_run_at` را به‌روز کند.
6. Transaction را Commit کند.

Query:

```sql
SELECT
    id,
    name,
    type,
    target,
    interval_seconds,
    timeout_millis,
    retries,
    config,
    next_run_at
FROM monitors
WHERE enabled = TRUE
  AND next_run_at <= NOW()
ORDER BY next_run_at
LIMIT $1
FOR UPDATE SKIP LOCKED;
```

به‌روزرسانی:

```sql
UPDATE monitors
SET
    next_run_at = NOW() +
        (interval_seconds * INTERVAL '1 second'),
    updated_at = NOW()
WHERE id = $1;
```

نمونه Service:

```go
func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			if err := s.scheduleBatch(ctx); err != nil {
				s.logger.Error(
					"scheduler batch failed",
					"error",
					err,
				)
			}
		}
	}
}
```

نکته مهم:

انتشار Job و به‌روزرسانی دیتابیس به صورت کاملاً Atomic نیست. برای MVP قابل قبول است، اما نسخه پایدارتر باید از الگوی Transactional Outbox استفاده کند.

---

## 19. Worker Loop

```go
func (w *Worker) Run(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		messages, err := w.queue.Consume(
			ctx,
			w.consumerName,
		)
		if err != nil {
			w.logger.Error(
				"consume jobs failed",
				"error",
				err,
			)
			time.Sleep(time.Second)
			continue
		}

		for _, message := range messages {
			if err := w.handleMessage(ctx, message); err != nil {
				w.logger.Error(
					"handle probe job failed",
					"message_id",
					message.ID,
					"error",
					err,
				)
				continue
			}

			if err := w.queue.Ack(ctx, message.ID); err != nil {
				w.logger.Error(
					"ack job failed",
					"message_id",
					message.ID,
					"error",
					err,
				)
			}
		}
	}
}
```

اجرای Job:

```go
func (w *Worker) handleMessage(
	ctx context.Context,
	message redis.XMessage,
) error {
	rawPayload, ok := message.Values["payload"]
	if !ok {
		return fmt.Errorf("payload is missing")
	}

	var job domain.ProbeJob
	if err := json.Unmarshal(
		[]byte(fmt.Sprint(rawPayload)),
		&job,
	); err != nil {
		return fmt.Errorf("decode probe job: %w", err)
	}

	executor, ok := w.registry.Get(job.Type)
	if !ok {
		return fmt.Errorf(
			"unsupported monitor type: %s",
			job.Type,
		)
	}

	result := probe.ExecuteWithRetry(
		ctx,
		executor,
		job,
	)

	if err := w.resultClient.Send(ctx, result); err != nil {
		return fmt.Errorf("send probe result: %w", err)
	}

	return nil
}
```

---

## 20. Result Ingestion

وظایف:

1. اعتبارسنجی Worker Token
2. جلوگیری از ثبت تکراری `job_id`
3. ثبت نتیجه در `probe_results`
4. به‌روزرسانی وضعیت Monitor
5. ارسال Metric به VictoriaMetrics

Transaction:

```sql
INSERT INTO probe_results (
    id,
    job_id,
    monitor_id,
    probe_location_id,
    status,
    success,
    error_code,
    error_message,
    duration_millis,
    metrics,
    attributes,
    started_at,
    finished_at
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12, $13
)
ON CONFLICT (job_id) DO NOTHING;

UPDATE monitors
SET
    last_status = $2,
    last_checked_at = $3,
    updated_at = NOW()
WHERE id = $1;
```

---

## 21. VictoriaMetrics

VictoriaMetrics از Prometheus Remote Write یا فرمت Import پشتیبانی می‌کند.

نمونه Metricها:

```text
monitor_probe_success{
  monitor_id="...",
  monitor_type="http",
  probe_location="local-dev"
} 1

monitor_probe_duration_seconds{
  monitor_id="...",
  monitor_type="http",
  probe_location="local-dev"
} 0.182

monitor_http_status_code{
  monitor_id="...",
  probe_location="local-dev"
} 200

monitor_ping_packet_loss_percent{
  monitor_id="...",
  probe_location="local-dev"
} 0
```

برای MVP می‌توان Metricها را با فرمت Prometheus Import ارسال کرد:

```go
func (c *VictoriaClient) Write(
	ctx context.Context,
	result domain.ProbeResult,
	monitorType string,
) error {
	successValue := 0
	if result.Success {
		successValue = 1
	}

	lines := []string{
		fmt.Sprintf(
			`monitor_probe_success{monitor_id="%s",monitor_type="%s",probe_location="%s"} %d %d`,
			result.MonitorID,
			monitorType,
			result.ProbeLocationID,
			successValue,
			result.FinishedAt.UnixMilli(),
		),
		fmt.Sprintf(
			`monitor_probe_duration_seconds{monitor_id="%s",monitor_type="%s",probe_location="%s"} %f %d`,
			result.MonitorID,
			monitorType,
			result.ProbeLocationID,
			float64(result.DurationMillis)/1000,
			result.FinishedAt.UnixMilli(),
		),
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v1/import/prometheus",
		strings.NewReader(strings.Join(lines, "\n")),
	)
	if err != nil {
		return err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return fmt.Errorf(
			"VictoriaMetrics returned %s",
			response.Status,
		)
	}

	return nil
}
```

---

## 22. Docker Compose

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: monitoring
      POSTGRES_USER: monitoring
      POSTGRES_PASSWORD: monitoring
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test:
        - CMD-SHELL
        - pg_isready -U monitoring -d monitoring
      interval: 5s
      timeout: 3s
      retries: 10

  redis:
    image: redis:7-alpine
    command:
      - redis-server
      - --appendonly
      - "yes"
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test:
        - CMD
        - redis-cli
        - ping
      interval: 5s
      timeout: 3s
      retries: 10

  victoriametrics:
    image: victoriametrics/victoria-metrics:latest
    command:
      - -storageDataPath=/victoria-metrics-data
      - -retentionPeriod=30d
    ports:
      - "8428:8428"
    volumes:
      - victoria_data:/victoria-metrics-data

  migrate:
    image: migrate/migrate:latest
    depends_on:
      postgres:
        condition: service_healthy
    volumes:
      - ./migrations:/migrations
    command:
      - -path=/migrations
      - -database=postgres://monitoring:monitoring@postgres:5432/monitoring?sslmode=disable
      - up
    restart: "no"

  api:
    build:
      context: .
      dockerfile: deployments/Dockerfile.api
    environment:
      APP_ENV: development
      HTTP_ADDRESS: :8080
      DATABASE_URL: postgres://monitoring:monitoring@postgres:5432/monitoring?sslmode=disable
      REDIS_ADDRESS: redis:6379
      VICTORIA_URL: http://victoriametrics:8428
      WORKER_TOKEN: local-worker-token
    ports:
      - "8080:8080"
    depends_on:
      migrate:
        condition: service_completed_successfully
      redis:
        condition: service_healthy

  scheduler:
    build:
      context: .
      dockerfile: deployments/Dockerfile.scheduler
    environment:
      DATABASE_URL: postgres://monitoring:monitoring@postgres:5432/monitoring?sslmode=disable
      REDIS_ADDRESS: redis:6379
      PROBE_LOCATION_ID: replace-with-local-location-id
    depends_on:
      migrate:
        condition: service_completed_successfully
      redis:
        condition: service_healthy

  worker:
    build:
      context: .
      dockerfile: deployments/Dockerfile.worker
    environment:
      REDIS_ADDRESS: redis:6379
      API_BASE_URL: http://api:8080
      WORKER_TOKEN: local-worker-token
      WORKER_NAME: worker-local-01
    cap_add:
      - NET_RAW
    depends_on:
      redis:
        condition: service_healthy
      api:
        condition: service_started

volumes:
  postgres_data:
  redis_data:
  victoria_data:
```

---

## 23. Dockerfile نمونه

### API

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/api \
    ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/api /api

EXPOSE 8080

ENTRYPOINT ["/api"]
```

برای Scheduler و Worker فقط مسیر Binary تغییر می‌کند.

---

## 24. متغیرهای محیطی

### `.env.example`

```dotenv
APP_ENV=development
HTTP_ADDRESS=:8080

DATABASE_URL=postgres://monitoring:monitoring@localhost:5432/monitoring?sslmode=disable

REDIS_ADDRESS=localhost:6379
REDIS_PASSWORD=
REDIS_DATABASE=0

VICTORIA_URL=http://localhost:8428

WORKER_TOKEN=change-me
WORKER_NAME=worker-local-01

SCHEDULER_BATCH_SIZE=100
SCHEDULER_INTERVAL=1s

PROBE_LOCATION_ID=
```

---

## 25. Health Check

Endpointها:

```http
GET /health/live
GET /health/ready
```

### Liveness

فقط نشان می‌دهد Process در حال اجرا است.

```json
{
  "status": "ok"
}
```

### Readiness

اتصالات زیر را بررسی می‌کند:

- PostgreSQL
- Redis
- VictoriaMetrics

```json
{
  "status": "ok",
  "dependencies": {
    "postgres": "ok",
    "redis": "ok",
    "victoriametrics": "ok"
  }
}
```

---

## 26. امنیت

Agentless Monitoring مستعد SSRF است.

### محدودیت‌های ضروری

برای Probeهای عمومی این IPها باید مسدود شوند:

```text
0.0.0.0/8
10.0.0.0/8
100.64.0.0/10
127.0.0.0/8
169.254.0.0/16
172.16.0.0/12
192.0.0.0/24
192.168.0.0/16
198.18.0.0/15
224.0.0.0/4
240.0.0.0/4
::1/128
fc00::/7
fe80::/10
```

### مراحل اعتبارسنجی Target

1. Parse کردن URL یا Host
2. Resolve کردن DNS
3. بررسی تمام IPهای Resolve شده
4. مسدود کردن IPهای Private و Reserved
5. بررسی مجدد بعد از Redirect
6. محدود کردن تعداد Redirect
7. محدود کردن اندازه Response Body
8. محدود کردن Portهای مجاز
9. غیرفعال کردن URL دارای Username و Password
10. جلوگیری از دسترسی به Cloud Metadata

نمونه Validator:

```go
func ValidatePublicIP(ip net.IP) error {
	blockedCIDRs := []string{
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}

	for _, rawCIDR := range blockedCIDRs {
		_, cidr, err := net.ParseCIDR(rawCIDR)
		if err != nil {
			return err
		}

		if cidr.Contains(ip) {
			return fmt.Errorf(
				"target IP %s is not publicly routable",
				ip.String(),
			)
		}
	}

	return nil
}
```

---

## 27. وضعیت Monitor

در فاز اول:

```text
Probe Success  -> UP
Probe Failure  -> DOWN
No Result Yet  -> UNKNOWN
Disabled       -> PAUSED
```

در فاز دوم بهتر است حالت‌های زیر اضافه شوند:

```text
SUSPECTED
DEGRADED
MAINTENANCE
```

---

## 28. Idempotency

هر Probe Job باید `job_id` منحصربه‌فرد داشته باشد.

در دیتابیس:

```sql
CREATE UNIQUE INDEX probe_results_job_id_idx
ON probe_results(job_id);
```

در Ingestion:

```sql
INSERT INTO probe_results (...)
VALUES (...)
ON CONFLICT (job_id) DO NOTHING;
```

این کار مانع ثبت دو نتیجه در صورت Retry شدن ارسال Worker می‌شود.

---

## 29. خطاهای استاندارد Probe

```text
invalid_target
blocked_target
timeout
dns_resolution_failed
dns_query_failed
dns_assertion_failed
tcp_connect_failed
tls_handshake_failed
tls_certificate_expired
http_request_failed
unexpected_status_code
body_assertion_failed
ping_failed
packet_loss_100
worker_internal_error
```

فرمت خطا:

```json
{
  "success": false,
  "status": "down",
  "error_code": "tcp_connect_failed",
  "error_message": "dial tcp 203.0.113.10:5432: connection refused"
}
```

---

## 30. Logging

از `log/slog` استفاده شود.

نمونه:

```go
logger.Info(
	"probe completed",
	"job_id", result.JobID,
	"monitor_id", result.MonitorID,
	"type", job.Type,
	"success", result.Success,
	"duration_ms", result.DurationMillis,
	"error_code", result.ErrorCode,
)
```

موارد حساس مانند Headerهای Authorization، Cookie، Token و Body نباید در Log ذخیره شوند.

---

## 31. Observability داخلی

هر سرویس Endpoint زیر را ارائه کند:

```http
GET /metrics
```

Metricهای داخلی:

```text
scheduler_jobs_published_total
scheduler_publish_errors_total
worker_jobs_received_total
worker_jobs_completed_total
worker_jobs_failed_total
worker_probe_duration_seconds
ingestion_results_total
ingestion_duplicate_results_total
queue_pending_jobs
```

---

## 32. تست‌ها

### Unit Test

- HTTP status assertion
- HTTP body assertion
- TCP connection
- DNS record parsing
- DNS expected values
- Retry behavior
- Timeout behavior
- SSRF validator
- Config parser

### Integration Test

- PostgreSQL Repository
- Redis Stream Publish/Consume/Ack
- Result ingestion
- Scheduler locking
- Duplicate result handling

### End-to-End Test

1. ساخت Monitor
2. اجرای Scheduler
3. دریافت Job توسط Worker
4. اجرای Probe
5. ارسال Result
6. به‌روزرسانی Monitor
7. نمایش Result در API

---

## 33. Makefile

```makefile
.PHONY: run test lint migrate-up migrate-down docker-up docker-down

run-api:
	go run ./cmd/api

run-scheduler:
	go run ./cmd/scheduler

run-worker:
	go run ./cmd/worker

test:
	go test ./... -race -cover

lint:
	golangci-lint run

migrate-up:
	migrate \
		-path migrations \
		-database "$(DATABASE_URL)" \
		up

migrate-down:
	migrate \
		-path migrations \
		-database "$(DATABASE_URL)" \
		down 1

docker-up:
	docker compose up --build

docker-down:
	docker compose down
```

---

## 34. مراحل پیاده‌سازی

### Sprint 1: هسته پروژه

- ساخت Repository
- Config Loader
- Logging
- PostgreSQL
- Migration
- Domain Model
- Health Check
- Docker Compose

### Sprint 2: Monitor API

- Create Monitor
- List Monitor
- Get Monitor
- Update Monitor
- Delete Monitor
- Pause و Resume
- Validation

### Sprint 3: Queue و Scheduler

- Redis Streams
- Job Schema
- Scheduler Loop
- Due Monitor Query
- Consumer Group
- Idempotency

### Sprint 4: HTTP و TCP

- Probe Interface
- HTTP Executor
- TCP Executor
- Retry
- Timeout
- Result Ingestion

### Sprint 5: DNS و Ping

- DNS Executor
- Ping Executor
- NET_RAW Capability
- Error Codes
- Probe Metrics

### Sprint 6: History و Stability

- Results API
- VictoriaMetrics
- Internal Metrics
- SSRF Protection
- Integration Tests
- Load Test
- Documentation

---

## 35. معیار پذیرش فاز اول

فاز اول زمانی تکمیل است که:

- Monitor از طریق API ساخته شود.
- Scheduler آن را در زمان مناسب وارد Queue کند.
- Worker Job را دریافت کند.
- Worker Probe مناسب را اجرا کند.
- نتیجه در PostgreSQL ذخیره شود.
- وضعیت Monitor به‌روزرسانی شود.
- Metric در VictoriaMetrics ثبت شود.
- نتیجه از API قابل مشاهده باشد.
- اجرای تکراری Result باعث Duplicate نشود.
- Timeout و Retry درست کار کنند.
- Targetهای Private و خطرناک مسدود شوند.
- تمام سرویس‌ها با Docker Compose اجرا شوند.
- HTTP، TCP، DNS و Ping تست End-to-End داشته باشند.

---

## 36. نمونه اجرای محلی

```bash
cp .env.example .env
docker compose up --build
```

ایجاد Monitor:

```bash
curl -X POST http://localhost:8080/api/v1/monitors \
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

مشاهده Monitorها:

```bash
curl http://localhost:8080/api/v1/monitors
```

مشاهده نتایج:

```bash
curl \
  'http://localhost:8080/api/v1/monitors/<monitor-id>/results?limit=20'
```

VictoriaMetrics UI:

```text
http://localhost:8428/vmui
```

---

## 37. تصمیم‌های معماری مهم

### چرا Microservice کامل نه؟

در فاز اول سه Binary کافی است:

- API
- Scheduler
- Worker

همه آن‌ها می‌توانند از یک Codebase و یک Domain مشترک استفاده کنند. شکستن زودهنگام سیستم به تعداد زیادی سرویس، هزینه Deploy، Debug و توسعه را افزایش می‌دهد.

### چرا Redis Streams؟

- ساده‌تر از Kafka
- پشتیبانی از Consumer Group
- Ack و Pending List
- مناسب حجم MVP
- راه‌اندازی آسان

### چرا PostgreSQL و VictoriaMetrics باهم؟

PostgreSQL برای Config و داده‌های رابطه‌ای مناسب است، اما نگهداری بلندمدت تمام Sampleهای Time Series در آن باعث بزرگ‌شدن سریع جدول‌ها می‌شود. VictoriaMetrics برای متریک‌های پرتعداد و Queryهای زمانی مناسب‌تر است.

### چرا Worker مستقل؟

Worker مستقل امکان اضافه‌شدن Locationهای مختلف را فراهم می‌کند. در آینده Worker می‌تواند در Frankfurt، Dubai یا داخل شبکه مشتری اجرا شود، بدون اینکه Control Plane تغییر اساسی کند.

---

## 38. بهبودهای بعد از MVP

پس از تثبیت فاز اول:

- Transactional Outbox
- چند Probe Location
- Result Confirmation
- False-positive Reduction
- Alert Policy
- Incident Management
- Maintenance Window
- Webhook، Email و Telegram
- SSL Probe مستقل
- Domain Expiration
- SMTP و UDP
- Private Probe
- API Key و RBAC
- Organization و Project
- SLO و SLA Reporting
- Public Status Page
- Kubernetes Deployment


---

# 39. فرانت‌اند و طراحی UI فاز اول

## 39.1 هدف فرانت‌اند

هدف فرانت‌اند در فاز اول، فراهم کردن یک رابط مدیریتی ساده، سریع و قابل توسعه برای مدیریت Monitorها و مشاهده وضعیت آن‌ها است.

کاربر باید بتواند:

- وضعیت کلی سیستم را در Dashboard مشاهده کند.
- Monitor جدید بسازد.
- Monitorها را ویرایش، Pause، Resume و حذف کند.
- جزئیات هر Monitor را مشاهده کند.
- تاریخچه Probe Resultها را ببیند.
- نمودار Latency و Uptime را مشاهده کند.
- وضعیت سرویس‌ها را بر اساس نوع و Status فیلتر کند.
- خطاهای Probe را با جزئیات بررسی کند.
- سلامت Backend و Probe Workerها را مشاهده کند.

در این فاز، UI باید روی Desktop و Tablet به‌خوبی کار کند و نسخه Mobile نیز حداقل قابلیت مشاهده و مدیریت پایه را داشته باشد.

---

## 39.2 تکنولوژی پیشنهادی فرانت‌اند

| بخش | تکنولوژی |
|---|---|
| Monitoring Console | React + Vite |
| Language | TypeScript |
| Styling | Tailwind CSS |
| UI Components | shadcn/ui |
| Server State | TanStack Query |
| Form Management | React Hook Form |
| Validation | Zod |
| General Charts | Apache ECharts |
| Dense Time-series Charts | uPlot |
| Live Transport | Server-Sent Events |
| Icons | Lucide React |
| Date Handling | date-fns |
| Testing | Vitest + Testing Library |
| E2E Testing | Playwright |
| Package Manager | pnpm |

دلایل انتخاب:

- Vite برای کنسول داخلی، HMR سریع و معماری SPA ساده‌تری فراهم می‌کند.
- Next.js برای Landing Page، SEO، مستندات و Status Page عمومی استفاده می‌شود.
- TypeScript خطاهای مدل داده را کاهش می‌دهد.
- TanStack Query مدیریت Cache، Refetch و Loading State را ساده می‌کند.
- React Hook Form و Zod برای فرم‌های Dynamic مربوط به Probeها مناسب هستند.
- shadcn/ui امکان ساخت UI حرفه‌ای بدون وابستگی شدید به یک Design System بسته را فراهم می‌کند.

---

## 39.3 معماری فرانت‌اند

```text
Browser
   │
   ▼
Next.js Application
├── App Router
├── Layout & Navigation
├── Server Components
├── Client Components
├── TanStack Query
├── Forms & Validation
└── Charts
   │
   ▼
REST API
   │
   ▼
Control Plane API
```

فرانت‌اند مستقیماً به PostgreSQL، Redis یا VictoriaMetrics متصل نمی‌شود. تمام ارتباط‌ها باید از طریق REST API کنترل‌پلین انجام شوند.

---

## 39.4 ساختار پروژه فرانت‌اند

```text
web/
├── app/
│   ├── layout.tsx
│   ├── page.tsx
│   ├── providers.tsx
│   ├── dashboard/
│   │   └── page.tsx
│   ├── monitors/
│   │   ├── page.tsx
│   │   ├── new/
│   │   │   └── page.tsx
│   │   └── [monitorId]/
│   │       ├── page.tsx
│   │       └── edit/
│   │           └── page.tsx
│   ├── locations/
│   │   └── page.tsx
│   ├── system/
│   │   └── page.tsx
│   └── settings/
│       └── page.tsx
├── components/
│   ├── layout/
│   │   ├── app-sidebar.tsx
│   │   ├── app-header.tsx
│   │   └── page-container.tsx
│   ├── dashboard/
│   │   ├── status-summary.tsx
│   │   ├── monitor-overview-chart.tsx
│   │   └── recent-failures.tsx
│   ├── monitors/
│   │   ├── monitor-table.tsx
│   │   ├── monitor-card.tsx
│   │   ├── monitor-form.tsx
│   │   ├── monitor-status-badge.tsx
│   │   ├── monitor-actions.tsx
│   │   └── monitor-filters.tsx
│   ├── probe/
│   │   ├── http-config-fields.tsx
│   │   ├── tcp-config-fields.tsx
│   │   ├── dns-config-fields.tsx
│   │   └── ping-config-fields.tsx
│   ├── charts/
│   │   ├── latency-chart.tsx
│   │   ├── uptime-chart.tsx
│   │   └── packet-loss-chart.tsx
│   ├── results/
│   │   ├── result-table.tsx
│   │   └── result-detail-drawer.tsx
│   └── ui/
├── hooks/
│   ├── use-monitors.ts
│   ├── use-monitor.ts
│   ├── use-monitor-results.ts
│   └── use-system-health.ts
├── lib/
│   ├── api-client.ts
│   ├── query-client.ts
│   ├── schemas.ts
│   ├── formatters.ts
│   └── constants.ts
├── types/
│   ├── monitor.ts
│   ├── result.ts
│   └── api.ts
├── public/
├── tests/
├── Dockerfile
├── package.json
├── pnpm-lock.yaml
├── next.config.ts
├── tailwind.config.ts
└── tsconfig.json
```

---

## 39.5 صفحات اصلی

### 39.5.1 Dashboard

مسیر:

```text
/dashboard
```

اجزای اصلی:

- تعداد کل Monitorها
- تعداد Monitorهای Up
- تعداد Monitorهای Down
- تعداد Monitorهای Unknown
- تعداد Monitorهای Paused
- درصد Availability کلی
- نمودار تعداد Checkهای موفق و ناموفق
- آخرین Failureها
- Monitorهای دارای بیشترین Latency
- وضعیت Probe Locationها

Wireframe پیشنهادی:

```text
┌─────────────────────────────────────────────────────────────┐
│ Header                                      User / Settings │
├──────────────┬──────────────────────────────────────────────┤
│ Sidebar      │ Dashboard                                    │
│              │                                              │
│ Dashboard    │ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ │
│ Monitors     │ │ Total  │ │   Up   │ │  Down  │ │Unknown │ │
│ Locations    │ └────────┘ └────────┘ └────────┘ └────────┘ │
│ System       │                                              │
│ Settings     │ ┌──────────────────────────────────────────┐ │
│              │ │ Availability / Probe Success Chart       │ │
│              │ └──────────────────────────────────────────┘ │
│              │                                              │
│              │ ┌────────────────────┐ ┌───────────────────┐ │
│              │ │ Recent Failures    │ │ Slowest Monitors  │ │
│              │ └────────────────────┘ └───────────────────┘ │
└──────────────┴──────────────────────────────────────────────┘
```

---

### 39.5.2 فهرست Monitorها

مسیر:

```text
/monitors
```

قابلیت‌ها:

- جستجو بر اساس نام یا Target
- فیلتر بر اساس Type
- فیلتر بر اساس Status
- مرتب‌سازی بر اساس Name، Status، Last Check و Latency
- Pagination
- ایجاد Monitor جدید
- Pause، Resume، Edit و Delete
- مشاهده سریع آخرین Error

ستون‌های جدول:

| ستون | توضیح |
|---|---|
| Name | نام Monitor |
| Type | HTTP، TCP، DNS یا Ping |
| Target | دامنه، URL یا Host |
| Status | وضعیت فعلی |
| Last Check | آخرین زمان بررسی |
| Duration | مدت آخرین Probe |
| Interval | فاصله اجرای Probe |
| Actions | عملیات |

---

### 39.5.3 ساخت Monitor

مسیر:

```text
/monitors/new
```

فرم به صورت مرحله‌ای یا Dynamic طراحی شود.

فیلدهای مشترک:

- Name
- Monitor Type
- Target
- Interval
- Timeout
- Retries
- Enabled

فیلدهای اختصاصی بر اساس Type نمایش داده شوند.

#### HTTP

- Method
- Headers
- Request Body
- Expected Status Codes
- Body Contains
- Follow Redirects
- Verify TLS

#### TCP

- Host
- Port

#### DNS

- Domain
- DNS Server
- Record Type
- Expected Values

#### Ping

- Host
- Packet Count
- Packet Interval

در پایین فرم، یک خلاصه از Config تولیدشده نمایش داده شود.

---

### 39.5.4 جزئیات Monitor

مسیر:

```text
/monitors/{monitorId}
```

بخش‌های صفحه:

- عنوان و Status
- Target
- نوع Monitor
- Last Checked
- Current Latency
- Uptime در 24 ساعت، 7 روز و 30 روز
- نمودار Latency
- نمودار Success/Failure
- آخرین Resultها
- آخرین Error
- تنظیمات Monitor
- Pause، Resume، Edit و Delete

Wireframe:

```text
┌──────────────────────────────────────────────────────────────┐
│ Main Website                      [UP] [Pause] [Edit] [...]  │
│ https://example.com                                         │
├──────────────────────────────────────────────────────────────┤
│ Uptime 24h │ Uptime 7d │ Uptime 30d │ Current Latency       │
│ 99.99%     │ 99.95%    │ 99.90%     │ 184 ms                │
├──────────────────────────────────────────────────────────────┤
│ Latency Chart                                                │
│                                                              │
├───────────────────────────────┬──────────────────────────────┤
│ Recent Results                │ Monitor Configuration        │
│ Success / Failure / Duration  │ Interval / Timeout / Retry   │
└───────────────────────────────┴──────────────────────────────┘
```

---

### 39.5.5 ویرایش Monitor

مسیر:

```text
/monitors/{monitorId}/edit
```

از همان کامپوننت `MonitorForm` استفاده شود و اطلاعات قبلی به‌عنوان Default Value قرار بگیرد.

---

### 39.5.6 Probe Locations

مسیر:

```text
/locations
```

در فاز اول فقط مشاهده وضعیت Location کافی است.

اطلاعات:

- Name
- Code
- Enabled
- Last Seen
- Worker Count
- Queue Lag
- Health

---

### 39.5.7 System Health

مسیر:

```text
/system
```

نمایش وضعیت:

- API
- PostgreSQL
- Redis
- VictoriaMetrics
- Scheduler
- Workerها

نمونه:

```text
API               Healthy
PostgreSQL        Healthy
Redis             Healthy
VictoriaMetrics   Healthy
Scheduler         Healthy
Worker local-01   Healthy
```

---

## 39.6 مدل TypeScript

### فایل `types/monitor.ts`

```ts
export type MonitorType = "http" | "tcp" | "dns" | "ping";

export type MonitorStatus =
  | "up"
  | "down"
  | "unknown"
  | "paused";

export interface Monitor {
  id: string;
  name: string;
  type: MonitorType;
  target: string;
  interval_seconds: number;
  timeout_millis: number;
  retries: number;
  enabled: boolean;
  config: Record<string, unknown>;
  last_status: MonitorStatus;
  last_checked_at: string | null;
  next_run_at: string;
  created_at: string;
  updated_at: string;
}

export interface CreateMonitorInput {
  name: string;
  type: MonitorType;
  target: string;
  interval_seconds: number;
  timeout_millis: number;
  retries: number;
  enabled: boolean;
  config: Record<string, unknown>;
}
```

### فایل `types/result.ts`

```ts
import type { MonitorStatus } from "./monitor";

export interface ProbeResult {
  id: string;
  job_id: string;
  monitor_id: string;
  probe_location_id: string;
  status: MonitorStatus;
  success: boolean;
  error_code?: string;
  error_message?: string;
  duration_millis: number;
  metrics: Record<string, number | string>;
  attributes: Record<string, unknown>;
  started_at: string;
  finished_at: string;
}
```

---

## 39.7 API Client

### فایل `lib/api-client.ts`

```ts
const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  "http://localhost:8080";

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public details?: unknown,
  ) {
    super(message);
  }
}

export async function apiRequest<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });

  if (!response.ok) {
    let details: unknown;

    try {
      details = await response.json();
    } catch {
      details = await response.text();
    }

    throw new ApiError(
      `API request failed with status ${response.status}`,
      response.status,
      details,
    );
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}
```

---

## 39.8 TanStack Query

### Provider

```tsx
"use client";

import {
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { useState } from "react";

export function AppProviders({
  children,
}: {
  children: React.ReactNode;
}) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 10_000,
            refetchOnWindowFocus: true,
            retry: 1,
          },
        },
      }),
  );

  return (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
}
```

### Hook فهرست Monitorها

```ts
import { useQuery } from "@tanstack/react-query";
import { apiRequest } from "@/lib/api-client";
import type { Monitor } from "@/types/monitor";

interface MonitorListResponse {
  items: Monitor[];
  total: number;
}

export function useMonitors() {
  return useQuery({
    queryKey: ["monitors"],
    queryFn: () =>
      apiRequest<MonitorListResponse>("/api/v1/monitors"),
    refetchInterval: 15_000,
  });
}
```

### Hook جزئیات Monitor

```ts
export function useMonitor(monitorId: string) {
  return useQuery({
    queryKey: ["monitors", monitorId],
    queryFn: () =>
      apiRequest<Monitor>(
        `/api/v1/monitors/${monitorId}`,
      ),
    enabled: Boolean(monitorId),
    refetchInterval: 10_000,
  });
}
```

---

## 39.9 Zod Schema فرم Monitor

### فایل `lib/schemas.ts`

```ts
import { z } from "zod";

const baseMonitorSchema = z.object({
  name: z
    .string()
    .min(2, "Name must contain at least 2 characters")
    .max(200),
  type: z.enum(["http", "tcp", "dns", "ping"]),
  target: z.string().min(1, "Target is required"),
  interval_seconds: z
    .number()
    .int()
    .min(10)
    .max(86400),
  timeout_millis: z
    .number()
    .int()
    .min(100)
    .max(60000),
  retries: z
    .number()
    .int()
    .min(0)
    .max(5),
  enabled: z.boolean(),
});

export const monitorFormSchema =
  baseMonitorSchema.superRefine((value, context) => {
    if (
      value.type === "http" &&
      !value.target.startsWith("http://") &&
      !value.target.startsWith("https://")
    ) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["target"],
        message: "HTTP target must start with http:// or https://",
      });
    }

    if (value.type === "tcp" && value.target.includes("/")) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["target"],
        message: "TCP target must be a hostname or IP address",
      });
    }
  });

export type MonitorFormValues =
  z.infer<typeof monitorFormSchema>;
```

---

## 39.10 فرم Dynamic برای Probeها

```tsx
"use client";

import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";

import {
  monitorFormSchema,
  type MonitorFormValues,
} from "@/lib/schemas";

export function MonitorForm() {
  const form = useForm<MonitorFormValues>({
    resolver: zodResolver(monitorFormSchema),
    defaultValues: {
      name: "",
      type: "http",
      target: "",
      interval_seconds: 60,
      timeout_millis: 5000,
      retries: 1,
      enabled: true,
    },
  });

  const monitorType = form.watch("type");

  async function onSubmit(values: MonitorFormValues) {
    const config = buildProbeConfig(values);

    await apiRequest("/api/v1/monitors", {
      method: "POST",
      body: JSON.stringify({
        ...values,
        config,
      }),
    });
  }

  return (
    <form onSubmit={form.handleSubmit(onSubmit)}>
      {/* Common fields */}

      {monitorType === "http" && (
        <HTTPConfigFields form={form} />
      )}

      {monitorType === "tcp" && (
        <TCPConfigFields form={form} />
      )}

      {monitorType === "dns" && (
        <DNSConfigFields form={form} />
      )}

      {monitorType === "ping" && (
        <PingConfigFields form={form} />
      )}

      <button type="submit">
        Create monitor
      </button>
    </form>
  );
}
```

پیشنهاد بهتر این است که Schema هر Probe به‌صورت Discriminated Union نوشته شود تا فیلدهای اختصاصی نیز توسط Zod اعتبارسنجی شوند.

---

## 39.11 Status Badge

```tsx
import type { MonitorStatus } from "@/types/monitor";

const statusStyles: Record<MonitorStatus, string> = {
  up: "bg-emerald-100 text-emerald-700",
  down: "bg-red-100 text-red-700",
  unknown: "bg-slate-100 text-slate-700",
  paused: "bg-amber-100 text-amber-700",
};

export function MonitorStatusBadge({
  status,
}: {
  status: MonitorStatus;
}) {
  return (
    <span
      className={[
        "inline-flex rounded-full px-2.5 py-1",
        "text-xs font-medium uppercase",
        statusStyles[status],
      ].join(" ")}
    >
      {status}
    </span>
  );
}
```

در نسخه فارسی UI می‌توان Labelها را چنین نمایش داد:

```text
up       -> فعال
down     -> قطع
unknown  -> نامشخص
paused   -> متوقف
```

---

## 39.12 جدول Monitorها

```tsx
"use client";

import { useMonitors } from "@/hooks/use-monitors";
import { MonitorStatusBadge } from "./monitor-status-badge";

export function MonitorTable() {
  const monitorsQuery = useMonitors();

  if (monitorsQuery.isLoading) {
    return <MonitorTableSkeleton />;
  }

  if (monitorsQuery.isError) {
    return (
      <ErrorState
        title="Failed to load monitors"
        onRetry={() => monitorsQuery.refetch()}
      />
    );
  }

  return (
    <div className="overflow-hidden rounded-xl border">
      <table className="w-full">
        <thead>
          <tr>
            <th>Name</th>
            <th>Target</th>
            <th>Type</th>
            <th>Status</th>
            <th>Last check</th>
            <th>Duration</th>
            <th />
          </tr>
        </thead>

        <tbody>
          {monitorsQuery.data?.items.map((monitor) => (
            <tr key={monitor.id}>
              <td>{monitor.name}</td>
              <td className="max-w-xs truncate">
                {monitor.target}
              </td>
              <td>{monitor.type.toUpperCase()}</td>
              <td>
                <MonitorStatusBadge
                  status={monitor.last_status}
                />
              </td>
              <td>
                {monitor.last_checked_at
                  ? formatRelativeDate(
                      monitor.last_checked_at,
                    )
                  : "Never"}
              </td>
              <td>
                {monitor.last_result?.duration_millis
                  ? `${monitor.last_result.duration_millis} ms`
                  : "—"}
              </td>
              <td>
                <MonitorActions monitor={monitor} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
```

برای پشتیبانی از این جدول، بهتر است API فهرست Monitorها، خلاصه آخرین Result را نیز بازگرداند.

---

## 39.13 قرارداد بهبود‌یافته API برای فهرست Monitorها

```http
GET /api/v1/monitors?page=1&page_size=20&type=http&status=down&search=api
```

پاسخ:

```json
{
  "items": [
    {
      "id": "c3555c21-9540-4985-a7db-0dd58fd74c35",
      "name": "Main Website",
      "type": "http",
      "target": "https://example.com",
      "interval_seconds": 60,
      "timeout_millis": 5000,
      "retries": 1,
      "enabled": true,
      "last_status": "up",
      "last_checked_at": "2026-07-16T12:00:00Z",
      "next_run_at": "2026-07-16T12:01:00Z",
      "last_result": {
        "success": true,
        "duration_millis": 184,
        "error_code": null
      }
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 125,
    "total_pages": 7
  }
}
```

---

## 39.14 API خلاصه Dashboard

Endpoint پیشنهادی:

```http
GET /api/v1/dashboard/summary
```

پاسخ:

```json
{
  "total_monitors": 120,
  "status_counts": {
    "up": 108,
    "down": 5,
    "unknown": 4,
    "paused": 3
  },
  "availability_24h": 99.95,
  "checks_24h": {
    "successful": 16850,
    "failed": 42
  },
  "recent_failures": [
    {
      "monitor_id": "uuid",
      "monitor_name": "Payment API",
      "error_code": "timeout",
      "started_at": "2026-07-16T11:58:00Z"
    }
  ]
}
```

---

## 39.15 API نمودار Monitor

Endpoint پیشنهادی:

```http
GET /api/v1/monitors/{monitor_id}/metrics
  ?from=2026-07-15T00:00:00Z
  &to=2026-07-16T00:00:00Z
  &step=5m
```

پاسخ:

```json
{
  "series": {
    "latency": [
      {
        "timestamp": "2026-07-16T10:00:00Z",
        "value": 184
      }
    ],
    "success": [
      {
        "timestamp": "2026-07-16T10:00:00Z",
        "value": 1
      }
    ]
  },
  "summary": {
    "uptime_percent": 99.95,
    "p50_latency_ms": 120,
    "p95_latency_ms": 310,
    "p99_latency_ms": 580
  }
}
```

Backend این Endpoint می‌تواند Query لازم را از VictoriaMetrics انجام دهد و نتیجه ساده‌شده را به فرانت‌اند تحویل دهد.

---

## 39.16 نمودار Latency

```tsx
"use client";

import {
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

interface LatencyPoint {
  timestamp: string;
  value: number;
}

export function LatencyChart({
  data,
}: {
  data: LatencyPoint[];
}) {
  return (
    <div className="h-72 w-full">
      <ResponsiveContainer>
        <LineChart data={data}>
          <XAxis
            dataKey="timestamp"
            tickFormatter={(value) =>
              new Date(value).toLocaleTimeString()
            }
          />
          <YAxis
            unit=" ms"
            width={70}
          />
          <Tooltip
            labelFormatter={(value) =>
              new Date(value).toLocaleString()
            }
            formatter={(value) => [`${value} ms`, "Latency"]}
          />
          <Line
            dataKey="value"
            type="monotone"
            dot={false}
            strokeWidth={2}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
```

---

## 39.17 Result Detail Drawer

با کلیک روی هر Result، یک Drawer یا Dialog باز شود و موارد زیر را نمایش دهد:

- Status
- Started At
- Finished At
- Duration
- Probe Location
- Error Code
- Error Message
- Metrics
- Attributes

نمونه نمایش:

```text
Probe Result

Status             Down
Duration           5,002 ms
Probe Location     local-dev
Error Code         timeout
Error Message      context deadline exceeded
Started At         2026-07-16 12:00:00
Finished At        2026-07-16 12:00:05

Metrics
total_duration_ms  5002

Attributes
attempt            2
max_attempts       2
```

اطلاعات حساس مانند Header یا Body کامل نباید نمایش داده شوند.

---

## 39.18 عملیات Monitor

عملیات:

- Pause
- Resume
- Edit
- Delete

### Mutation نمونه

```ts
import {
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";

export function usePauseMonitor() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (monitorId: string) =>
      apiRequest(
        `/api/v1/monitors/${monitorId}/pause`,
        { method: "POST" },
      ),
    onSuccess: async (_, monitorId) => {
      await queryClient.invalidateQueries({
        queryKey: ["monitors"],
      });

      await queryClient.invalidateQueries({
        queryKey: ["monitors", monitorId],
      });
    },
  });
}
```

قبل از Delete باید Confirmation Dialog نمایش داده شود.

---

## 39.19 Navigation و Layout

Sidebar پیشنهادی:

```text
Overview
├── Dashboard

Monitoring
├── Monitors
├── Probe Locations

Platform
├── System Health
├── Settings
```

Header:

- عنوان صفحه
- Search
- Theme Toggle
- User Menu
- دکمه Create Monitor

---

## 39.20 Design System

### رنگ وضعیت‌ها

| وضعیت | مفهوم |
|---|---|
| Green | Up |
| Red | Down |
| Gray | Unknown |
| Amber | Paused |
| Blue | Informational |

### اصول UI

- Status فقط با رنگ نمایش داده نشود؛ Icon و Text نیز داشته باشد.
- خطاهای مهم در بالای صفحه Monitor نمایش داده شوند.
- جدول‌ها Skeleton داشته باشند.
- Empty State مناسب طراحی شود.
- عملیات مخرب Confirmation داشته باشند.
- Timestampها هم Relative و هم Absolute قابل مشاهده باشند.
- Targetهای طولانی Ellipsis داشته باشند.
- Error Messageهای طولانی در Drawer نمایش داده شوند.
- UI باید RTL را پشتیبانی کند.

---

## 39.21 پشتیبانی RTL و فارسی

در `app/layout.tsx`:

```tsx
export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="fa" dir="rtl">
      <body>
        <AppProviders>
          {children}
        </AppProviders>
      </body>
    </html>
  );
}
```

برای نسخه دوزبانه، بهتر است از Internationalization استفاده شود.

ساختار پیشنهادی:

```text
app/
├── [locale]/
│   ├── layout.tsx
│   ├── dashboard/
│   └── monitors/
messages/
├── fa.json
└── en.json
```

---

## 39.22 Responsive Design

### Desktop

- Sidebar ثابت
- جدول کامل
- نمودارها در دو ستون
- Summary Cardهای چهار یا پنج‌ستونه

### Tablet

- Sidebar جمع‌شونده
- جدول با Scroll افقی
- نمودارها تک‌ستونه یا دو ستونه

### Mobile

- Bottom Navigation یا Drawer
- Monitorها به شکل Card
- Summary Cardها دو ستونه
- عملیات در Dropdown
- نمودارها Full Width

---

## 39.23 Loading، Empty و Error State

برای هر صفحه سه حالت باید طراحی شود.

### Loading

- Skeleton برای Card
- Skeleton برای Table
- Placeholder برای Chart

### Empty

مثلاً در صفحه Monitorها:

```text
هنوز هیچ مانیتوری ایجاد نشده است.
اولین مانیتور HTTP، Ping، TCP یا DNS خود را بسازید.
[ایجاد مانیتور]
```

### Error

```text
دریافت اطلاعات با خطا مواجه شد.
[تلاش مجدد]
```

---

## 39.24 Real-time و Polling

برای فاز اول WebSocket ضروری نیست.

پیشنهاد Polling:

| داده | Interval |
|---|---|
| Dashboard Summary | 15 ثانیه |
| Monitor List | 15 ثانیه |
| Monitor Detail | 10 ثانیه |
| Recent Results | 10 ثانیه |
| System Health | 10 ثانیه |
| Historical Charts | بدون Poll یا 60 ثانیه |

بعداً می‌توان Server-Sent Events یا WebSocket اضافه کرد.

---

## 39.25 Next.js Configuration

### فایل `next.config.ts`

```ts
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  reactStrictMode: true,
};

export default nextConfig;
```

### متغیر محیطی

```dotenv
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

در محیط Docker:

```dotenv
NEXT_PUBLIC_API_BASE_URL=http://api:8080
```

نکته:

متغیرهای `NEXT_PUBLIC_*` در زمان Build در Bundle قرار می‌گیرند. برای Deploymentهای مختلف بهتر است از Reverse Proxy یا Runtime Configuration استفاده شود.

---

## 39.26 Reverse Proxy پیشنهادی

بهتر است Browser فقط با یک Origin ارتباط داشته باشد:

```text
https://monitor.example.com
├── /             -> Next.js
└── /api/*        -> Control Plane API
```

این روش مزایای زیر را دارد:

- ساده‌تر شدن CORS
- مدیریت یکپارچه TLS
- عدم نمایش آدرس داخلی API
- پشتیبانی بهتر از Cookie و Authentication آینده

نمونه Nginx:

```nginx
server {
    listen 80;

    location /api/ {
        proxy_pass http://api:8080/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location / {
        proxy_pass http://web:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## 39.27 Dockerfile فرانت‌اند

```dockerfile
FROM node:22-alpine AS dependencies

WORKDIR /app

COPY package.json pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile

FROM node:22-alpine AS builder

WORKDIR /app

COPY --from=dependencies /app/node_modules ./node_modules
COPY . .

RUN corepack enable && pnpm build

FROM node:22-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
ENV PORT=3000

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

COPY --from=builder /app/public ./public
COPY --from=builder \
  --chown=nextjs:nodejs \
  /app/.next/standalone ./
COPY --from=builder \
  --chown=nextjs:nodejs \
  /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000

CMD ["node", "server.js"]
```

---

## 39.28 اضافه‌شدن Web به Docker Compose

```yaml
  web:
    build:
      context: ./web
      dockerfile: Dockerfile
    environment:
      NEXT_PUBLIC_API_BASE_URL: http://localhost:8080
    ports:
      - "3000:3000"
    depends_on:
      api:
        condition: service_started
```

برای محیط Production بهتر است Nginx یا Traefik در جلوی `web` و `api` قرار بگیرد.

---

## 39.29 CORS در API

اگر UI و API روی Originهای مختلف اجرا شوند، API باید CORS را محدود و مشخص تنظیم کند.

نمونه Go:

```go
corsHandler := cors.Handler(cors.Options{
	AllowedOrigins: []string{
		"http://localhost:3000",
	},
	AllowedMethods: []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
	},
	AllowedHeaders: []string{
		"Accept",
		"Authorization",
		"Content-Type",
	},
	MaxAge: 300,
})
```

استفاده از `AllowedOrigins: ["*"]` برای نسخه Production توصیه نمی‌شود.

---

## 39.30 تست فرانت‌اند

### Unit Test

- Status Badge
- Date Formatter
- Form Validation
- Probe Config Builder
- API Error Parser

### Component Test

- Monitor Table
- Monitor Form
- Result Detail Drawer
- Dashboard Summary Cards

### E2E Test

سناریوهای Playwright:

1. باز شدن Dashboard
2. ساخت HTTP Monitor
3. مشاهده Monitor در جدول
4. باز شدن جزئیات
5. Pause کردن Monitor
6. Resume کردن Monitor
7. ویرایش Interval
8. حذف Monitor
9. نمایش Resultهای موفق
10. نمایش Error Result

---

## 39.31 Accessibility

موارد ضروری:

- استفاده از Semantic HTML
- Label برای تمام Inputها
- Focus State قابل مشاهده
- Keyboard Navigation
- ARIA Label برای Icon Buttonها
- Contrast مناسب
- نمایش Status با Text، نه فقط رنگ
- Dialog و Drawer با Focus Trap
- Error Message متصل به Input

---

## 39.32 امنیت فرانت‌اند

- Tokenهای حساس در Local Storage نگهداری نشوند.
- در آینده Authentication مبتنی بر HttpOnly Cookie ترجیح داده شود.
- Errorهای Backend بدون Sanitize مستقیم نمایش داده نشوند.
- Headerهای محرمانه Monitor در UI Mask شوند.
- Request Bodyهای حساس فقط با دسترسی مناسب نمایش داده شوند.
- از تزریق HTML در Error Message جلوگیری شود.
- مقدار Target قبل از نمایش Encode شود.
- برای عملیات Delete و Pause از CSRF Protection در معماری Cookie-based استفاده شود.

---

## 39.33 تغییرات لازم در Backend برای UI

برای پشتیبانی مناسب از UI، Endpointهای زیر به Backend اضافه یا تکمیل شوند:

```text
GET    /api/v1/dashboard/summary
GET    /api/v1/monitors
POST   /api/v1/monitors
GET    /api/v1/monitors/{id}
PUT    /api/v1/monitors/{id}
DELETE /api/v1/monitors/{id}
POST   /api/v1/monitors/{id}/pause
POST   /api/v1/monitors/{id}/resume
GET    /api/v1/monitors/{id}/results
GET    /api/v1/monitors/{id}/metrics
GET    /api/v1/probe-locations
GET    /api/v1/system/health
```

تمام List Endpointها باید از موارد زیر پشتیبانی کنند:

- Pagination
- Filtering
- Sorting
- Search
- Stable Response Schema

---

## 39.34 قرارداد خطای API

فرمت یکپارچه:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "The request contains invalid fields",
    "fields": {
      "target": [
        "HTTP target must start with http:// or https://"
      ]
    },
    "request_id": "req_123"
  }
}
```

فرانت‌اند باید خطاهای Field-level را در کنار Input و خطاهای عمومی را به شکل Alert یا Toast نمایش دهد.

---

## 39.35 برنامه پیاده‌سازی فرانت‌اند

### Sprint UI-1: Foundation

- ایجاد Next.js Project
- Tailwind
- shadcn/ui
- Layout
- Sidebar
- Header
- Query Provider
- API Client
- Error Boundary

### Sprint UI-2: Monitor Management

- Monitor List
- Filters
- Pagination
- Create Monitor
- Edit Monitor
- Dynamic Probe Fields
- Pause و Resume
- Delete Dialog

### Sprint UI-3: Monitor Detail

- Summary Cards
- Recent Results
- Result Drawer
- Latency Chart
- Uptime Summary
- Metrics API Integration

### Sprint UI-4: Dashboard و System

- Dashboard Summary
- Recent Failures
- Slowest Monitors
- Probe Locations
- System Health
- Polling

### Sprint UI-5: Quality

- RTL
- Responsive
- Accessibility
- Component Tests
- Playwright E2E
- Docker
- Production Build

---

## 39.36 معیار پذیرش فرانت‌اند

فرانت‌اند فاز اول زمانی تکمیل است که:

- Dashboard وضعیت Monitorها را نمایش دهد.
- فهرست Monitorها دارای Search و Filter باشد.
- کاربر بتواند هر چهار نوع Monitor را ایجاد کند.
- فرم بر اساس Type به‌صورت Dynamic تغییر کند.
- Validation سمت Client و Server نمایش داده شود.
- Monitor قابل Edit، Pause، Resume و Delete باشد.
- جزئیات Monitor شامل Uptime، Latency و Results باشد.
- Result ناموفق Error Code و Error Message را نمایش دهد.
- Polling آخرین وضعیت را بدون Refresh کامل صفحه به‌روزرسانی کند.
- UI در Desktop، Tablet و Mobile قابل استفاده باشد.
- UI از RTL پشتیبانی کند.
- Loading، Empty و Error State وجود داشته باشد.
- Docker Compose بتواند Web، API، Scheduler، Worker و Dependencyها را اجرا کند.
- تست E2E مسیر Create تا مشاهده Result را پوشش دهد.

---

## 40. ساختار نهایی Repository

پس از اضافه‌شدن فرانت‌اند:

```text
monitoring-platform/
├── cmd/
├── internal/
├── migrations/
├── deployments/
├── web/
│   ├── app/
│   ├── components/
│   ├── hooks/
│   ├── lib/
│   ├── types/
│   ├── tests/
│   ├── Dockerfile
│   └── package.json
├── docker-compose.yml
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

---

## 41. خروجی نهایی فاز اول

فاز اول نهایی شامل این بخش‌ها است:

### Backend

- REST API
- Monitor Management
- Scheduler
- Redis Queue
- Probe Worker
- HTTP Probe
- TCP Probe
- DNS Probe
- Ping Probe
- Result Ingestion
- PostgreSQL
- VictoriaMetrics
- Security و SSRF Protection

### Frontend

- Dashboard
- Monitor List
- Create و Edit Monitor
- Monitor Detail
- Probe Results
- Latency و Uptime Charts
- Probe Locations
- System Health
- RTL و Responsive Design

### Infrastructure

- Docker Compose
- Containerهای مستقل
- Health Checks
- Migration
- Internal Metrics
- Logging
- Test Plan

---

# 42. معماری نهایی وب: Vite برای کنسول و Next.js برای لندینگ

## تصمیم معماری

وب محصول به دو اپلیکیشن مستقل تقسیم می‌شود:

### Monitoring Console

```text
React + Vite
TypeScript
TanStack Router
TanStack Query
Tailwind CSS
shadcn/ui
React Hook Form
Zod
Apache ECharts
uPlot
Server-Sent Events
```

این بخش برای کاربران واردشده است و شامل Dashboard، Monitorها، نمودارها، Resultها، Probe Locationها و System Health می‌شود.

### Public Website

```text
Next.js
TypeScript
Tailwind CSS
Static Generation
Server Rendering در صورت نیاز
MDX یا Headless CMS
```

این بخش شامل Landing Page، Pricing، Features، Docs، Blog، صفحات SEO و Public Status Page است.

## ساختار Monorepo

```text
monitoring-platform/
├── apps/
│   ├── console/
│   │   ├── src/
│   │   │   ├── routes/
│   │   │   ├── features/
│   │   │   ├── components/
│   │   │   ├── hooks/
│   │   │   ├── lib/
│   │   │   └── main.tsx
│   │   ├── vite.config.ts
│   │   ├── Dockerfile
│   │   └── package.json
│   └── website/
│       ├── app/
│       ├── components/
│       ├── content/
│       ├── public/
│       ├── next.config.ts
│       ├── Dockerfile
│       └── package.json
├── packages/
│   ├── ui/
│   ├── api-client/
│   ├── types/
│   └── config/
├── cmd/
├── internal/
├── migrations/
├── deployments/
├── docker-compose.yml
├── pnpm-workspace.yaml
└── go.mod
```

## صفحات وب‌سایت عمومی

```text
/
├── /features
├── /pricing
├── /integrations
├── /probe-locations
├── /security
├── /docs
├── /blog
├── /about
├── /contact
├── /login
├── /signup
└── /status/{slug}
```

### بخش‌های Landing Page

1. Header و Navigation
2. Hero و پیام اصلی محصول
3. تصویر یا Demo داشبورد
4. معرفی HTTP، Ping، TCP و DNS Monitoring
5. Multi-location Monitoring
6. کاهش False Positive
7. Live Charts و Alerting
8. امنیت و SSRF Protection
9. Pricing Preview
10. FAQ
11. CTA نهایی
12. Footer

## Wireframe لندینگ

```text
┌──────────────────────────────────────────────────────────────┐
│ Logo  Product  Pricing  Docs  Security   Login  Start Free   │
├──────────────────────────────────────────────────────────────┤
│ Monitor every endpoint before your users notice             │
│ HTTP, Ping, TCP and DNS monitoring from multiple locations  │
│ [Start Monitoring] [View Demo]                              │
│                    Product Screenshot                       │
├──────────────────────────────────────────────────────────────┤
│ HTTP          Ping          TCP           DNS                │
├──────────────────────────────────────────────────────────────┤
│ Multi-location monitoring and failure confirmation          │
├──────────────────────────────────────────────────────────────┤
│ Live charts, alerts and uptime reports                       │
├──────────────────────────────────────────────────────────────┤
│ Pricing                                                      │
├──────────────────────────────────────────────────────────────┤
│ FAQ                                                          │
├──────────────────────────────────────────────────────────────┤
│ Final CTA                                                    │
└──────────────────────────────────────────────────────────────┘
```

## ساختار Console مبتنی بر Vite

```text
apps/console/src/
├── app/
│   ├── router.tsx
│   ├── providers.tsx
│   └── query-client.ts
├── routes/
│   ├── dashboard/
│   ├── monitors/
│   ├── locations/
│   ├── system/
│   └── settings/
├── features/
│   ├── monitors/
│   ├── results/
│   ├── metrics/
│   ├── dashboard/
│   └── live-events/
├── components/
│   ├── layout/
│   ├── charts/
│   └── ui/
├── lib/
│   ├── api-client.ts
│   ├── event-stream.ts
│   └── formatters.ts
├── types/
└── main.tsx
```

## Vite Configuration

```ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 3000,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
      "/events": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
```

## Live Data با SSE

```text
Probe Worker
    │
    ▼
Result Ingestion
    │
    ├── PostgreSQL
    ├── VictoriaMetrics
    └── Event Bus
           │
           ▼
      SSE Gateway
           │
           ▼
      Vite Console
```

Endpoint:

```http
GET /events/v1/stream
Accept: text/event-stream
```

نمونه Event:

```text
event: probe-result
id: evt_123
data: {"monitor_id":"mon_123","status":"up","duration_ms":184,"timestamp":"2026-07-16T12:00:00Z"}
```

## انتخاب کتابخانه نمودار

```text
نمودارهای عمومی و تعاملی       → Apache ECharts
Time Series بسیار پرتراکم      → uPlot
```

## Downsampling API

```http
GET /api/v1/monitors/{id}/metrics
  ?from=...
  &to=...
  &step=auto
```

| بازه | Step |
|---|---:|
| تا 15 دقیقه | 5 ثانیه |
| تا 1 ساعت | 15 ثانیه |
| تا 6 ساعت | 1 دقیقه |
| تا 24 ساعت | 5 دقیقه |
| تا 7 روز | 30 دقیقه |
| تا 30 روز | 2 ساعت |
| تا 1 سال | 1 روز |

محدودیت پیشنهادی:

```text
Historical chart points: 1500
Live sliding window:     300
Result table page size:  50
Dashboard time series:   500
```

## Public Status Page

```text
/status/{slug}
```

قابلیت‌ها:

- وضعیت Componentها
- Incident فعال
- تاریخچه Incident
- Uptime
- Branding
- Custom Domain در مراحل بعد

## دامنه‌های پیشنهادی

```text
www.example.com       Next.js Website
app.example.com       Vite Monitoring Console
api.example.com       REST API و SSE
status.example.com    Public Status Pages
docs.example.com      Documentation
```

## Docker Compose تکمیلی

```yaml
services:
  website:
    build:
      context: ./apps/website
    ports:
      - "3001:3000"
    depends_on:
      api:
        condition: service_started

  console:
    build:
      context: ./apps/console
    ports:
      - "3000:80"
    depends_on:
      api:
        condition: service_started
```

## معیار پذیرش معماری وب

- Console با React و Vite اجرا شود.
- Website عمومی با Next.js اجرا شود.
- Landing Page دارای Metadata، Sitemap و SEO مناسب باشد.
- Console از TanStack Router استفاده کند.
- Live Resultها از SSE دریافت شوند.
- Cache به‌صورت Incremental Update شود.
- Historical Chartها Downsample شوند.
- ECharts برای نمودارهای عمومی استفاده شود.
- uPlot برای نمودارهای پرتراکم قابل استفاده باشد.
- Website و Console مستقل Build و Deploy شوند.
- Public Status Page در معماری پیش‌بینی شود.

# 43. جمع‌بندی معماری نهایی

```text
Public Experience
└── Next.js
    ├── Landing
    ├── Pricing
    ├── Docs
    ├── Blog
    └── Public Status

Monitoring Experience
└── React + Vite
    ├── Dashboard
    ├── Monitors
    ├── Charts
    ├── Live Results
    └── System Health

Backend
└── Go
    ├── REST API
    ├── SSE Gateway
    ├── Scheduler
    ├── Probe Workers
    └── Result Ingestion

Storage and Messaging
├── PostgreSQL
├── Redis Streams
└── VictoriaMetrics
```

---

# 44. توسعه فاز اول به هشت نوع مانیتور

فاز اول نهایی شامل این Monitorها است:

1. HTTP/HTTPS
2. ICMP Ping
3. TCP Port
4. DNS
5. SSL/TLS Certificate
6. Domain Expiration
7. SMTP Handshake
8. NTP

## 44.1 تغییر نوع Monitor

```go
const (
    MonitorHTTP             MonitorType = "http"
    MonitorTCP              MonitorType = "tcp"
    MonitorDNS              MonitorType = "dns"
    MonitorPing             MonitorType = "ping"
    MonitorTLS              MonitorType = "tls"
    MonitorDomainExpiration MonitorType = "domain_expiration"
    MonitorSMTP             MonitorType = "smtp"
    MonitorNTP              MonitorType = "ntp"
)
```

Migration پیشنهادی:

```sql
ALTER TYPE monitor_type ADD VALUE IF NOT EXISTS 'tls';
ALTER TYPE monitor_type ADD VALUE IF NOT EXISTS 'domain_expiration';
ALTER TYPE monitor_type ADD VALUE IF NOT EXISTS 'smtp';
ALTER TYPE monitor_type ADD VALUE IF NOT EXISTS 'ntp';
```

برای توسعه بلندمدت، بهتر است `monitor_type` در آینده از ENUM به Lookup Table یا `VARCHAR` همراه Validation اپلیکیشن تبدیل شود.

---

# 45. SSL/TLS Certificate Monitoring

## قابلیت‌ها

- TLS handshake روی Port دلخواه
- اعتبار Hostname
- اعتبار Chain
- تاریخ شروع و انقضا
- روزهای باقی‌مانده
- Issuer و Subject
- SANها
- TLS Version و Cipher
- Self-signed Detection
- SHA-256 Fingerprint
- تشخیص تغییر Certificate
- Threshold هشدار و Critical

## Config

```json
{
  "server_name": "example.com",
  "port": 443,
  "verify_chain": true,
  "verify_hostname": true,
  "minimum_tls_version": "1.2",
  "warning_days": 30,
  "critical_days": 7,
  "expected_issuer_contains": "",
  "expected_fingerprint_sha256": ""
}
```

## Metrics

```text
monitor_tls_handshake_duration_seconds
monitor_tls_certificate_days_remaining
monitor_tls_certificate_valid
monitor_tls_hostname_valid
monitor_tls_chain_valid
monitor_tls_certificate_changed
```

## Error Codeها

```text
tls_connect_failed
tls_handshake_failed
tls_chain_invalid
tls_hostname_invalid
tls_certificate_expired
tls_certificate_expiring
tls_version_too_old
tls_issuer_mismatch
tls_fingerprint_mismatch
```

---

# 46. Domain Expiration Monitoring

## قابلیت‌ها

- RDAP به‌عنوان روش اصلی
- WHOIS به‌عنوان Fallback
- Registered بودن دامنه
- Expiration Date
- Days Remaining
- Registrar
- Domain Statusها
- Name Serverها
- Creation و Updated Date
- تشخیص تغییر Registrar و Name Server
- IDN/Punycode
- Cache و Rate Limit

## Config

```json
{
  "warning_days": 45,
  "critical_days": 15,
  "check_nameservers": true,
  "expected_registrar_contains": "",
  "expected_nameservers": []
}
```

## Metrics

```text
monitor_domain_registered
monitor_domain_days_remaining
monitor_domain_lookup_duration_seconds
monitor_domain_nameserver_match
monitor_domain_registrar_match
```

## Error Codeها

```text
invalid_domain
domain_not_registered
rdap_lookup_failed
whois_lookup_failed
domain_expired
domain_expiring
registrar_mismatch
nameserver_mismatch
expiration_date_missing
```

Interval پیش‌فرض این Monitor بهتر است 12 ساعت باشد تا Rate Limit سرویس‌های Registry رعایت شود.

---

# 47. SMTP Handshake Monitoring

## قابلیت‌ها

- Modeهای Plain، STARTTLS و Implicit TLS
- TCP Connect
- دریافت Banner و بررسی کد 220
- ارسال EHLO
- استخراج Capabilityها
- STARTTLS
- TLS Validation
- NOOP و QUIT
- اندازه‌گیری زمان هر مرحله
- بدون Login و بدون ارسال ایمیل واقعی

## Config

```json
{
  "port": 587,
  "mode": "starttls",
  "ehlo_domain": "monitor.example.com",
  "expected_banner_contains": "",
  "require_starttls": true,
  "verify_tls": true,
  "expected_capabilities": ["STARTTLS"]
}
```

## Metrics

```text
monitor_smtp_connect_duration_seconds
monitor_smtp_banner_duration_seconds
monitor_smtp_ehlo_duration_seconds
monitor_smtp_starttls_duration_seconds
monitor_smtp_total_duration_seconds
monitor_smtp_starttls_available
```

## Error Codeها

```text
smtp_connect_failed
smtp_banner_timeout
smtp_invalid_banner
smtp_ehlo_failed
smtp_starttls_unavailable
smtp_starttls_failed
smtp_tls_invalid
smtp_capability_missing
smtp_noop_failed
smtp_quit_failed
```

---

# 48. NTP Monitoring

## قابلیت‌ها

- NTP Response
- Round-trip Time
- Clock Offset
- Stratum
- Leap Indicator
- Reference ID
- Root Delay
- Root Dispersion
- Version و Mode
- Threshold برای Offset و RTT

## Config

```json
{
  "port": 123,
  "version": 4,
  "max_offset_millis": 1000,
  "max_round_trip_millis": 2000,
  "allowed_stratum_min": 1,
  "allowed_stratum_max": 15
}
```

## Metrics

```text
monitor_ntp_offset_seconds
monitor_ntp_round_trip_seconds
monitor_ntp_stratum
monitor_ntp_root_delay_seconds
monitor_ntp_root_dispersion_seconds
monitor_ntp_response_valid
```

## Error Codeها

```text
ntp_timeout
ntp_invalid_response
ntp_unsynchronized
ntp_offset_too_high
ntp_round_trip_too_high
ntp_stratum_invalid
ntp_kiss_of_death
```

برای امنیت، NTP عمومی فقط اجازه ارسال Client Request استاندارد به UDP Port 123 را دارد.

---

# 49. Registry نهایی Probeها

```go
registry := probe.NewRegistry(
    probe.NewHTTPExecutor(),
    probe.NewTCPExecutor(),
    probe.NewDNSExecutor(),
    probe.NewPingExecutor(),
    probe.NewTLSExecutor(),
    probe.NewDomainExpirationExecutor(),
    probe.NewSMTPExecutor(),
    probe.NewNTPExecutor(),
)
```

---

# 50. نمونه API مانیتورهای جدید

## TLS

```json
{
  "name": "Main TLS Certificate",
  "type": "tls",
  "target": "example.com",
  "interval_seconds": 3600,
  "timeout_millis": 10000,
  "retries": 1,
  "config": {
    "server_name": "example.com",
    "port": 443,
    "verify_chain": true,
    "verify_hostname": true,
    "minimum_tls_version": "1.2",
    "warning_days": 30,
    "critical_days": 7
  }
}
```

## Domain Expiration

```json
{
  "name": "Main Domain",
  "type": "domain_expiration",
  "target": "example.com",
  "interval_seconds": 43200,
  "timeout_millis": 15000,
  "retries": 1,
  "config": {
    "warning_days": 45,
    "critical_days": 15,
    "check_nameservers": true
  }
}
```

## SMTP

```json
{
  "name": "Mail Submission",
  "type": "smtp",
  "target": "mail.example.com",
  "interval_seconds": 60,
  "timeout_millis": 10000,
  "retries": 1,
  "config": {
    "port": 587,
    "mode": "starttls",
    "ehlo_domain": "monitor.example.com",
    "require_starttls": true,
    "verify_tls": true
  }
}
```

## NTP

```json
{
  "name": "Primary Time Server",
  "type": "ntp",
  "target": "time.example.com",
  "interval_seconds": 300,
  "timeout_millis": 5000,
  "retries": 1,
  "config": {
    "port": 123,
    "version": 4,
    "max_offset_millis": 1000,
    "max_round_trip_millis": 2000
  }
}
```

---

# 51. Interval پیشنهادی بر اساس نوع

| Monitor | حداقل پیشنهادی | پیش‌فرض |
|---|---:|---:|
| HTTP | 10s | 60s |
| Ping | 10s | 60s |
| TCP | 10s | 60s |
| DNS | 30s | 60s |
| TLS | 5m | 1h |
| Domain Expiration | 1h | 12h |
| SMTP | 30s | 1m |
| NTP | 1m | 5m |

---

# 52. UI مانیتورهای جدید

## Monitor Type Selector

```text
Web and API
├── HTTP/HTTPS
└── SSL/TLS Certificate

Network
├── Ping
├── TCP Port
├── DNS
└── NTP

Domain and Email
├── Domain Expiration
└── SMTP Handshake
```

ساختار کامپوننت‌ها:

```text
apps/console/src/features/monitors/components/
├── monitor-form.tsx
├── monitor-type-selector.tsx
├── common-monitor-fields.tsx
├── http-config-fields.tsx
├── tcp-config-fields.tsx
├── dns-config-fields.tsx
├── ping-config-fields.tsx
├── tls-config-fields.tsx
├── domain-expiration-config-fields.tsx
├── smtp-config-fields.tsx
├── ntp-config-fields.tsx
└── config-summary.tsx
```

## فرم TLS

فیلدها:

- Host
- Port
- Server Name
- Verify Chain
- Verify Hostname
- Minimum TLS Version
- Warning Days
- Critical Days
- Expected Issuer
- Expected Fingerprint

صفحه جزئیات:

- Certificate Health
- Days Remaining
- Expiration Date
- Issuer و Subject
- SANها
- TLS Version و Cipher
- Fingerprint
- Certificate Change Timeline
- Days Remaining Chart

## فرم Domain Expiration

فیلدها:

- Domain
- Warning Days
- Critical Days
- Check Name Servers
- Expected Registrar
- Expected Name Servers

صفحه جزئیات:

- Days Remaining
- Expiration Date
- Registrar
- Creation و Updated Date
- Name Serverها
- Domain Statusها
- Change History
- RDAP یا WHOIS Source

## فرم SMTP

فیلدها:

- Host
- Port
- Connection Mode
- EHLO Domain
- Require STARTTLS
- Verify TLS
- Expected Banner
- Expected Capabilities

صفحه جزئیات:

- Banner
- EHLO Response
- Capabilityها
- STARTTLS Availability
- TLS Version
- Connect، Banner، EHLO و STARTTLS Duration
- Timeline نتایج SMTP

## فرم NTP

فیلدها:

- Server
- Port
- NTP Version
- Maximum Clock Offset
- Maximum Round-trip
- Minimum و Maximum Stratum

صفحه جزئیات:

- Current Offset
- Round-trip Time
- Stratum
- Leap Indicator
- Reference ID
- Root Delay
- Root Dispersion
- Offset Chart
- Round-trip Chart

---

# 53. TypeScript Typeهای جدید

```ts
export type MonitorType =
  | "http"
  | "tcp"
  | "dns"
  | "ping"
  | "tls"
  | "domain_expiration"
  | "smtp"
  | "ntp";
```

```ts
export interface TLSMonitorConfig {
  server_name: string;
  port: number;
  verify_chain: boolean;
  verify_hostname: boolean;
  minimum_tls_version: "1.2" | "1.3";
  warning_days: number;
  critical_days: number;
  expected_issuer_contains?: string;
  expected_fingerprint_sha256?: string;
}

export interface DomainExpirationMonitorConfig {
  warning_days: number;
  critical_days: number;
  check_nameservers: boolean;
  expected_registrar_contains?: string;
  expected_nameservers?: string[];
}

export interface SMTPMonitorConfig {
  port: number;
  mode: "plain" | "starttls" | "implicit_tls";
  ehlo_domain: string;
  expected_banner_contains?: string;
  require_starttls: boolean;
  verify_tls: boolean;
  expected_capabilities?: string[];
}

export interface NTPMonitorConfig {
  port: number;
  version: 3 | 4;
  max_offset_millis: number;
  max_round_trip_millis: number;
  allowed_stratum_min: number;
  allowed_stratum_max: number;
}
```

---

# 54. Dashboard و فیلترها

فیلتر Type باید هر هشت نوع را پشتیبانی کند.

Cardهای جدید:

- Certificates Expiring Soon
- Domains Expiring Soon
- SMTP STARTTLS Failures
- NTP Servers with High Offset

نمونه API:

```json
{
  "attention_required": {
    "certificates_expiring_30d": 3,
    "domains_expiring_45d": 2,
    "smtp_starttls_failures": 1,
    "ntp_high_offset": 1
  }
}
```

نمودارهای اصلی:

| Monitor | نمودار |
|---|---|
| TLS | Days Remaining و Handshake Duration |
| Domain | Days Remaining |
| SMTP | Connect، EHLO، STARTTLS و Total Duration |
| NTP | Clock Offset و Round-trip |

---

# 55. آیکن‌ها و Labelها

| Type | Label فارسی | Icon |
|---|---|---|
| http | HTTP/HTTPS | Globe |
| ping | پینگ | Radio |
| tcp | پورت TCP | Network |
| dns | DNS | Waypoints |
| tls | گواهی SSL/TLS | ShieldCheck |
| domain_expiration | انقضای دامنه | CalendarClock |
| smtp | سرویس SMTP | MailCheck |
| ntp | سرور زمان NTP | Clock3 |

---

# 56. معیار پذیرش

## TLS

- Certificate از Port دلخواه خوانده شود.
- Chain و Hostname بررسی شوند.
- Days Remaining ثبت شود.
- TLS Version و Cipher نمایش داده شوند.
- تغییر Fingerprint تشخیص داده شود.

## Domain Expiration

- RDAP Lookup انجام شود.
- Expiration Date و Days Remaining ثبت شود.
- Registrar و Name Serverها نمایش داده شوند.
- Cache و Rate Limit رعایت شوند.

## SMTP

- Banner، EHLO و STARTTLS بررسی شوند.
- Modeهای Plain، STARTTLS و Implicit TLS پشتیبانی شوند.
- Capabilityها ذخیره شوند.
- هیچ ایمیل واقعی ارسال نشود.

## NTP

- NTP Response معتبر دریافت شود.
- Offset، RTT و Stratum ثبت شوند.
- Thresholdها اعمال شوند.
- UDP فقط روی Port 123 محدود شود.

---

# 57. برنامه توسعه تکمیلی

## Sprint 7: TLS و Domain

- TLS Executor
- Certificate Parsing
- Fingerprint Change Detection
- RDAP Client
- WHOIS Fallback
- Cache و Rate Limit
- فرم‌ها و صفحات Detail
- Integration Test

## Sprint 8: SMTP و NTP

- SMTP State Machine
- STARTTLS
- SMTP Capability Parsing
- NTP Client
- Offset Validation
- فرم‌ها و نمودارها
- E2E Test

## Sprint 9: Hardening

- SSRF Validation
- Load Test
- Queue Backpressure
- Metric Cardinality Review
- Error UX
- Documentation
- Production Images

---

# 58. محدوده نهایی فاز اول

## Monitorها

```text
HTTP/HTTPS
Ping
TCP
DNS
SSL/TLS
Domain Expiration
SMTP Handshake
NTP
```

## خارج از فاز اول

```text
WebSocket
gRPC
API Scenario
Browser Synthetic
Traceroute
Email Delivery Test
Error Tracking
APM
Logs
Distributed Tracing
SNMP
BGP
```

---

# 59. الزامات نهایی UI/UX، تم و چندزبانه‌بودن

## 59.1 سبک طراحی

تمام بخش‌های محصول باید با سبک زیر طراحی شوند:

```text
Modern
Minimal
Professional
Data-dense
Accessible
Responsive
Consistent
```

این الزام شامل Landing Page، Monitoring Console، Public Status Page، Login، Signup، Documentation، Pricing و تمام Loading، Empty و Error Stateها است.

اصول طراحی:

- استفاده محدود و هدفمند از رنگ
- تمرکز روی خوانایی داده
- فاصله‌گذاری منظم
- سلسله‌مراتب بصری واضح
- Cardهای ساده با Border و Shadow کنترل‌شده
- حذف تزئینات غیرضروری
- نمایش وضعیت با رنگ، Text و Icon
- اجتناب از Gradientهای سنگین و UI شلوغ
- اولویت با سرعت درک اطلاعات

## 59.2 تکنولوژی UI

برای Website و Console:

```text
Tailwind CSS
shadcn/ui
TypeScript
Lucide Icons
Shared Design Tokens
```

Console:

```text
React + Vite
TanStack Router
TanStack Query
React Hook Form
Zod
Apache ECharts
uPlot
```

Website:

```text
Next.js
Tailwind CSS
shadcn/ui
MDX یا Headless CMS
```

## 59.3 Light Theme و Dark Theme

محصول باید سه حالت رسمی داشته باشد:

```text
system
light
dark
```

الزامات:

- Theme Toggle در Header یا User Menu
- پشتیبانی از System Theme
- ذخیره انتخاب کاربر
- جلوگیری از Flash اشتباه Theme
- هماهنگی تمام Componentها
- خوانایی نمودارها در هر دو Theme
- Contrast مناسب
- پشتیبانی کامل Dialog، Drawer، Table، Tooltip و Chart
- هماهنگی Landing، Console و Status Page

اولویت:

```text
User preference
→ Stored preference
→ System preference
→ Light fallback
```

## 59.4 Theme Provider

ساختار پیشنهادی Console:

```text
apps/console/src/features/theme/
├── theme-provider.tsx
├── theme-toggle.tsx
├── use-theme.ts
└── theme-storage.ts
```

```ts
export type ThemeMode = "system" | "light" | "dark";
```

در Website، Theme باید پیش از Paint اعمال شود تا Flash ایجاد نشود.

## 59.5 Design Tokenها

Tokenها در Package مشترک نگهداری شوند:

```text
packages/ui/
├── tokens/
│   ├── colors.css
│   ├── spacing.css
│   ├── radius.css
│   ├── typography.css
│   └── shadows.css
```

Tokenهای ضروری:

- Background
- Foreground
- Card
- Border
- Muted
- Primary
- Secondary
- Destructive
- Success
- Warning
- Info
- Chart Series
- Focus Ring

## 59.6 زبان‌های رسمی

نسخه اول محصول باید دو زبان رسمی داشته باشد:

```text
فارسی
English
```

موارد زیر باید ترجمه شوند:

- Landing Page
- Console
- Formها
- Validation Messageها
- Empty Stateها
- Error Messageها
- Dashboard
- Public Status Page
- Login و Signup
- Settings
- Documentation Navigation

## 59.7 RTL و LTR

```text
fa → rtl
en → ltr
```

نمونه:

```tsx
<html
  lang={locale}
  dir={locale === "fa" ? "rtl" : "ltr"}
>
```

الزامات:

- Sidebar در هر دو جهت درست باشد.
- Arrow و Chevronهای Directional Mirror شوند.
- Table و Pagination صحیح باشند.
- Drawer از سمت مناسب باز شود.
- Chart و Tooltip با RTL سازگار باشند.
- فرم‌ها و Error Messageها جهت درست داشته باشند.
- اعداد و Timestampها خوانا باقی بمانند.

## 59.8 ساختار Internationalization

```text
packages/i18n/
├── locales/
│   ├── fa/
│   │   ├── common.json
│   │   ├── monitors.json
│   │   ├── dashboard.json
│   │   ├── validation.json
│   │   └── website.json
│   └── en/
│       ├── common.json
│       ├── monitors.json
│       ├── dashboard.json
│       ├── validation.json
│       └── website.json
├── config.ts
└── formatters.ts
```

Namespaceها:

```text
common
navigation
dashboard
monitors
results
settings
validation
website
status
```

## 59.9 Language Switcher

Language Switcher در این نقاط باشد:

- Header وب‌سایت
- Login و Signup
- User Menu کنسول
- Settings
- Public Status Page

رفتار:

- زبان ذخیره شود.
- Context صفحه حفظ شود.
- داده‌های کاربر ترجمه نشوند.
- فقط UI و Copy محصول ترجمه شوند.

## 59.10 Formatting

Formatting بر اساس Locale:

- Date
- Time
- Relative Time
- Number
- Percentage
- Duration
- Byte Size

Timestamp Backend باید UTC بماند و در Client با Timezone کاربر نمایش داده شود.

## 59.11 Typography

الزامات:

- Font Stack مناسب فارسی
- Font Stack مناسب انگلیسی
- Line Height مناسب فارسی
- وزن‌های محدود
- خوانایی بالا در Table و Chart
- عدم استفاده از Font Decorative
- Monospace برای Code و Metric

## 59.12 تنظیمات Appearance

در Settings:

```text
Appearance
├── Theme
│   ├── System
│   ├── Light
│   └── Dark
└── Language
    ├── فارسی
    └── English
```

این تنظیمات ابتدا در Local Storage ذخیره و بعداً با Profile کاربر Sync شوند.

## 59.13 Chart Design

- رنگ Seriesها از Design Token گرفته شود.
- Grid Lineها در Dark Mode کنترل شوند.
- Tooltip Contrast مناسب داشته باشد.
- Success و Failure فقط با رنگ تفکیک نشوند.
- Zoom و Selection قابل استفاده باشد.
- Labelهای فارسی و انگلیسی Overlap نداشته باشند.
- Empty Chart State طراحی شود.

## 59.14 Component Stateها

```text
Default
Hover
Focus
Active
Disabled
Loading
Error
Success
Empty
Selected
```

Componentهای ضروری:

- Button
- Input
- Select
- Checkbox
- Switch
- Radio Group
- Tabs
- Table
- Badge
- Alert
- Toast
- Dialog
- Drawer
- Dropdown Menu
- Tooltip
- Skeleton
- Pagination
- Date Range Picker
- Command/Search Palette

## 59.15 Responsive Design

Desktop:

- Sidebar ثابت یا Collapsible
- Data Table کامل
- Chart چندستونه

Tablet:

- Sidebar جمع‌شونده
- Table با Scroll افقی
- Grid دو یا تک‌ستونه

Mobile:

- Navigation Drawer یا Bottom Navigation
- Monitor Card به‌جای Table
- Chart تمام‌عرض
- Action Menu فشرده
- Form تک‌ستونه

هر دو زبان و هر دو Theme باید در تمام Breakpointها تست شوند.

## 59.16 Landing Page Visual Direction

Landing Page باید:

- مدرن و مینیمال باشد.
- Product-led باشد.
- Screenshot یا Demo واقعی محصول را نمایش دهد.
- Copy کوتاه و واضح داشته باشد.
- CTA اصلی و ثانویه داشته باشد.
- Featureها را با مثال واقعی نمایش دهد.
- از Illustration شلوغ اجتناب کند.
- در Dark و Light Theme قابل نمایش باشد.
- فارسی و انگلیسی طبیعی و مستقل داشته باشد.

## 59.17 Accessibility

حداقل الزامات:

- WCAG AA
- Keyboard Navigation
- Focus Visible
- Screen Reader Label
- Semantic HTML
- Status با Text و Icon
- Form Error متصل به Input
- Skip Navigation
- Reduced Motion
- عدم وابستگی کامل به Hover
- عدم استفاده از رنگ به‌تنهایی

## 59.18 معیار پذیرش UI/UX

- Light، Dark و System Theme کامل کار کنند.
- Theme بدون Flash اشتباه Load شود.
- فارسی و انگلیسی کامل پشتیبانی شوند.
- RTL و LTR بدون Layout Bug کار کنند.
- Landing، Console و Status Page هماهنگ باشند.
- Tailwind و shadcn/ui مبنای Componentها باشند.
- Design Token مشترک استفاده شود.
- نمودارها در هر دو Theme خوانا باشند.
- فرم‌های هر هشت Monitor در هر دو زبان کار کنند.
- Mobile، Tablet و Desktop پشتیبانی شوند.
- Accessibility حداقل AA رعایت شود.
- طراحی مدرن، مینیمال و Data-focused باقی بماند.

---

# 60. تصمیم نهایی فرانت‌اند: Next.js یکپارچه

تمام تجربه وب محصول در یک اپلیکیشن Next.js پیاده‌سازی می‌شود:

```text
Next.js
├── Landing Page
├── Features
├── Pricing
├── Docs
├── Blog
├── Login و Signup
├── Monitoring Console
├── Public Status Page
└── Settings
```

Stack نهایی:

```text
Next.js App Router
TypeScript
Tailwind CSS
shadcn/ui
TanStack Query
React Hook Form
Zod
Apache ECharts
uPlot
Server-Sent Events
next-intl
next-themes
Lucide React
```

Backend اصلی همچنان Go است و شامل API، Scheduler، Worker و Result Ingestion می‌شود.

## ساختار پروژه

```text
apps/web/
├── app/
│   ├── [locale]/
│   │   ├── (marketing)/
│   │   ├── (auth)/
│   │   ├── (console)/
│   │   └── status/[slug]/
│   ├── globals.css
│   └── providers.tsx
├── components/
│   ├── ui/
│   ├── layout/
│   ├── marketing/
│   ├── monitors/
│   └── charts/
├── features/
├── lib/
├── messages/
│   ├── fa.json
│   └── en.json
├── components.json
├── next.config.ts
├── tailwind.config.ts
└── package.json
```

## Server و Client Component

Server Component برای Landing، Pricing، Docs، Blog، Metadata و داده اولیه استفاده شود.

Client Component فقط برای Dashboard، Formها، Table، Theme Toggle، Language Switcher، نمودارها، TanStack Query و SSE استفاده شود.

نباید تمام صفحات بدون نیاز با `"use client"` ساخته شوند.

---

# 61. الزام قطعی استفاده از shadcn/ui CLI

## قانون اصلی

تیم توسعه نباید Componentهای استاندارد UI را از ابتدا بسازد.

تمام Componentهای موجود در shadcn/ui باید با CLI رسمی نصب شوند:

```bash
pnpm dlx shadcn@latest init
```

نمونه نصب Componentها:

```bash
pnpm dlx shadcn@latest add button
pnpm dlx shadcn@latest add input
pnpm dlx shadcn@latest add select
pnpm dlx shadcn@latest add form
pnpm dlx shadcn@latest add card
pnpm dlx shadcn@latest add table
pnpm dlx shadcn@latest add dialog
pnpm dlx shadcn@latest add drawer
pnpm dlx shadcn@latest add sheet
pnpm dlx shadcn@latest add tabs
pnpm dlx shadcn@latest add dropdown-menu
pnpm dlx shadcn@latest add tooltip
pnpm dlx shadcn@latest add alert
pnpm dlx shadcn@latest add badge
pnpm dlx shadcn@latest add skeleton
pnpm dlx shadcn@latest add command
pnpm dlx shadcn@latest add checkbox
pnpm dlx shadcn@latest add switch
pnpm dlx shadcn@latest add radio-group
pnpm dlx shadcn@latest add pagination
pnpm dlx shadcn@latest add breadcrumb
pnpm dlx shadcn@latest add separator
```

نصب گروهی:

```bash
pnpm dlx shadcn@latest add   button input select form card table dialog   drawer sheet tabs dropdown-menu tooltip   alert badge skeleton command checkbox switch   radio-group pagination breadcrumb separator
```

## موارد ممنوع

- ساخت دستی Button، Dialog، Select، Table و Componentهای موجود در shadcn/ui
- کپی از Blog یا Repositoryهای نامعتبر
- ایجاد Component موازی بدون دلیل
- تغییر API پایه به‌شکل ناسازگار
- استفاده هم‌زمان از UI Library مشابه بدون تصمیم معماری

## موارد مجاز

بعد از نصب با CLI:

- تغییر Style با Tailwind
- اتصال به Design Tokenها
- اضافه‌کردن Variant
- ساخت Wrapper تخصصی
- پشتیبانی RTL
- افزودن Accessibility Label
- اتصال به React Hook Form

مثال:

```tsx
import { Button } from "@/components/ui/button";

export function MonitorPauseButton() {
  return (
    <Button variant="outline">
      Pause monitor
    </Button>
  );
}
```

## Component Inventory فاز اول

### پایه

- Button
- Input
- Textarea
- Label
- Select
- Checkbox
- Switch
- Radio Group
- Form
- Card
- Badge
- Alert

### Navigation

- Breadcrumb
- Tabs
- Dropdown Menu
- Command
- Sheet
- Pagination

### Feedback

- Dialog
- Alert Dialog
- Drawer
- Tooltip
- Toast یا Sonner
- Skeleton
- Progress

### Data

- Table
- Scroll Area
- Separator
- Popover
- Calendar

## ترتیب راه‌اندازی

```bash
pnpm create next-app@latest apps/web
cd apps/web
pnpm dlx shadcn@latest init
```

تنظیمات پیشنهادی:

```text
Style: New York
Base color: Neutral یا Slate
CSS variables: Yes
RSC: Yes
TypeScript: Yes
```

فایل `components.json` باید Commit شود.

## معیار پذیرش shadcn/ui

- `components.json` در Repository وجود داشته باشد.
- Componentهای پایه با CLI رسمی اضافه شده باشند.
- Button، Form، Dialog، Table و Select از صفر ساخته نشده باشند.
- Customization با Tailwind و Design Token انجام شده باشد.
- Componentها در Light/Dark و RTL/LTR تست شوند.
- Accessibility پایه حفظ شده باشد.

---

# 62. Theme و i18n

Theme با `next-themes`:

```text
system
light
dark
```

زبان با `next-intl`:

```text
/fa/...
/en/...
```

الزامات:

- فارسی RTL
- انگلیسی LTR
- Metadata ترجمه‌شده
- Landing و Console دوزبانه
- حفظ مسیر هنگام تغییر زبان
- Locale-aware Date و Number Formatting

---

# 63. نمودارها و Live Data

ECharts و uPlot به‌صورت Client Component و Lazy-loaded استفاده شوند.

```tsx
"use client";

import dynamic from "next/dynamic";

const LatencyChart = dynamic(
  () => import("./latency-chart"),
  {
    ssr: false,
    loading: () => <ChartSkeleton />,
  },
);
```

```text
Historical Data → REST API
Live Data       → SSE
```

Downsampling، Sliding Window، Incremental Cache Update و محدودیت تعداد Point همچنان الزامی هستند.

---

# 64. Routeهای نهایی

```text
/[locale]
├── /
├── /features
├── /pricing
├── /docs
├── /blog
├── /security
├── /login
├── /signup
├── /app/dashboard
├── /app/monitors
├── /app/monitors/new
├── /app/monitors/[monitorId]
├── /app/monitors/[monitorId]/edit
├── /app/locations
├── /app/system
├── /app/settings
└── /status/[slug]
```

---

# 65. معیار پذیرش معماری Next.js

- Website و Console در یک اپلیکیشن Next.js باشند.
- صفحات عمومی SEO و Metadata داشته باشند.
- Client Componentها محدود و هدفمند باشند.
- Live Data از SSE دریافت شود.
- نمودارها Lazy-loaded باشند.
- Tailwind و shadcn/ui مبنای UI باشند.
- Componentهای shadcn فقط با CLI رسمی نصب شوند.
- Component استاندارد از صفر بازنویسی نشود.
- Light، Dark و System Theme کامل باشند.
- فارسی و انگلیسی کامل باشند.
- RTL و LTR بدون Layout Bug کار کنند.
- Landing، Console و Status Page Design System مشترک داشته باشند.
- Backend اصلی Go مستقل باقی بماند.

---

# 66. تکمیل سند به مشخصات جامع محصول

این سند از این بخش به بعد فقط فاز اول فنی نیست و بخش‌های کامل محصول، فرانت‌اند، UI/UX، دسترسی، هشدار، Incident، Billing، امنیت، عملیات و مسیر توسعه آینده را نیز پوشش می‌دهد.

## 66.1 دامنه کلی محصول

```text
Observability Platform
├── Agentless Monitoring
├── Alerting
├── Incident Management
├── Public Status Pages
├── Organization and Projects
├── Authentication and RBAC
├── Billing and Quotas
├── Administration
├── Product Analytics
├── Security and Compliance
├── Deployment and Operations
└── Future Modules
    ├── Error Tracking
    ├── APM
    ├── Logs
    └── Distributed Tracing
```

---

# 67. Organization، Project و Tenant Isolation

## ساختار

```text
User
└── Organization
    ├── Members
    ├── Projects
    │   ├── Monitors
    │   ├── Alert Policies
    │   ├── Incidents
    │   ├── Status Pages
    │   └── API Keys
    ├── Billing
    └── Audit Logs
```

تمام Queryها باید با `organization_id` محدود شوند و شناسه Resource به‌تنهایی قابل اعتماد نیست.

---

# 68. Authentication و Session

قابلیت‌ها:

- Signup
- Login
- Logout
- Email Verification
- Forgot Password
- Reset Password
- Session Revocation
- Multi-device Sessions
- MFA در مراحل بعد
- SSO برای Plan سازمانی

پیشنهاد:

```text
Secure HttpOnly Cookie
Short-lived Access Session
Rotating Refresh Session
CSRF Protection
Argon2id Password Hashing
```

صفحات UI:

```text
/[locale]/login
/[locale]/signup
/[locale]/verify-email
/[locale]/forgot-password
/[locale]/reset-password
```

تمام صفحات باید فارسی/انگلیسی، RTL/LTR و Light/Dark باشند.

---

# 69. RBAC

Roleهای پایه:

```text
Owner
Admin
Editor
Viewer
Billing
```

Permissionهای اصلی:

```text
organization.read
organization.update
members.manage
projects.manage
monitors.read
monitors.create
monitors.update
monitors.delete
alerts.manage
incidents.manage
status_pages.manage
api_keys.manage
billing.read
billing.manage
audit_logs.read
```

Backend منبع نهایی Permission است و مخفی‌کردن Button در UI کافی نیست.

---

# 70. Member و Invitation Management

- دعوت با Email
- پذیرش دعوت
- لغو Invitation
- تغییر Role
- حذف Member
- Session Revocation
- Expiration و Single-use Token
- Audit Log

UI:

```text
/app/settings/members
/app/settings/members/invitations
```

---

# 71. API Key Management

انواع:

- Personal API Key
- Project API Key
- Probe Registration Token
- Read-only Integration Key

الزامات:

- Secret فقط یک‌بار نمایش داده شود.
- فقط Hash ذخیره شود.
- Scope داشته باشد.
- Expiration اختیاری
- Last Used At
- Revoke
- Audit Log

---

# 72. Alerting Engine

جریان:

```text
Probe Result
→ State Evaluator
→ Alert Policy
→ Deduplication
→ Notification Router
→ Channel
```

Conditionهای پایه:

- Monitor Down
- Consecutive Failures
- High Latency
- Packet Loss
- SSL Expiration
- Domain Expiration
- NTP Offset
- SMTP STARTTLS Failure
- DNS Mismatch
- Missing Results
- Region Failure

Anti-flapping:

```text
Open after 3 failures
Resolve after 2 successes
```

Stateها:

```text
pending
firing
recovering
resolved
suppressed
```

UI:

```text
/app/alerts
/app/alerts/policies
/app/alerts/policies/new
```

---

# 73. Notification Channels

فاز اصلی:

- Email
- Webhook
- Telegram

فاز بعد:

- Slack
- Microsoft Teams
- PagerDuty
- SMS
- Opsgenie

Webhook باید HMAC Signature، Retry، Backoff، Delivery Log و Secret Rotation داشته باشد.

UI:

```text
/app/settings/notifications
/app/settings/notifications/new
/app/settings/notifications/[id]
```

---

# 74. Incident Management

Stateها:

```text
open
acknowledged
investigating
identified
monitoring
resolved
```

Severity:

```text
SEV-1
SEV-2
SEV-3
SEV-4
```

صفحات:

```text
/app/incidents
/app/incidents/[incidentId]
```

صفحه Incident:

- Header
- Severity
- Status
- Timeline
- Related Monitors
- Related Alerts
- Assignee
- Public Updates
- Resolution Summary

---

# 75. Maintenance Window و Muting

قابلیت‌ها:

- One-time
- Recurring
- Monitor-specific
- Project-wide
- Tag-based
- Alert Suppression
- Pause Probe Execution

تفاوت:

```text
Mute Alerts:
Check continues.

Pause Monitoring:
Check execution stops.
```

---

# 76. Tags، Search و Bulk Actions

Tag نمونه:

```text
environment:production
team:payments
region:eu
criticality:high
```

Bulk Actionها:

- Pause
- Resume
- Delete
- Add Tag
- Remove Tag
- Change Interval
- Attach Alert Policy

---

# 77. Multi-region Monitoring

هر Monitor می‌تواند از یک یا چند Probe Location اجرا شود.

Aggregation:

```text
Any Failure
Majority
All Regions
Custom Quorum
```

UI:

- Location Selector
- Quorum Policy
- Region Matrix
- Latency by Region
- Region Timeline

---

# 78. Uptime، SLA و SLO

Metricها:

- Uptime %
- Downtime
- Incident Count
- MTTD
- MTTR
- p50
- p95
- p99
- Error Budget

Windowها:

```text
24h
7d
30d
90d
Custom
```

Maintenance می‌تواند از SLA حذف شود.

---

# 79. Data Retention و Downsampling

نمونه Retention:

```text
Free: 7 days
Starter: 30 days
Pro: 90 days
Business: 365 days
```

Tierها:

```text
Raw
1-minute
5-minute
1-hour
```

Rollup:

- count
- success_count
- failure_count
- min
- max
- avg
- p50
- p95
- p99

---

# 80. Billing، Plans و Quotas

Planها:

```text
Free
Starter
Pro
Business
Enterprise
```

Quotaها:

- Monitor Count
- Minimum Interval
- Probe Regions
- Retention
- Members
- Notification Channels
- Status Pages
- API Rate Limit
- Monthly Check Volume

UI:

```text
/app/settings/billing
/app/settings/usage
/app/settings/invoices
```

Enforcement باید در Backend انجام شود.

---

# 81. Onboarding

Flow:

```text
Create Account
→ Create Organization
→ Create Project
→ Create First Monitor
→ Add Notification Channel
→ Run First Check
→ View Result
```

Checklist داخل Dashboard:

- Add Monitor
- Add Notification Channel
- Invite Member
- Create Status Page
- Enable Multi-region

---

# 82. Public Status Page کامل

قابلیت‌ها:

- Custom Slug
- Component Groups
- Current Status
- Incident History
- Scheduled Maintenance
- Uptime History
- Subscribe to Updates
- Branding
- Custom Domain
- فارسی و انگلیسی
- Light و Dark

Statusها:

```text
Operational
Degraded Performance
Partial Outage
Major Outage
Maintenance
```

---

# 83. Admin Panel داخلی

قابلیت‌ها:

- Search Organizations
- Search Users
- Suspend/Unsuspend
- Plan Override
- Usage Inspection
- Probe Health
- Queue Health
- Failed Notifications
- Audit Events
- Feature Flags
- Support Impersonation با Audit کامل

MFA برای Admin اجباری است.

---

# 84. Audit Log

Eventها:

- Login
- Logout
- Password Change
- Member Invite
- Role Change
- Monitor Create/Delete
- API Key Create/Revoke
- Alert Policy Change
- Billing Change
- Organization Suspension

Audit Log نباید Secret ذخیره کند.

---

# 85. Feature Flags

Scope:

```text
global
plan
organization
user
```

کاربرد:

- Rollout تدریجی
- Beta
- Emergency Disable
- Organization-specific Feature

---

# 86. Product Analytics

Eventها:

- signup_completed
- organization_created
- monitor_created
- first_check_completed
- notification_channel_created
- alert_fired
- status_page_published
- upgrade_started
- upgrade_completed

نباید Secret یا Target حساس بدون نیاز ارسال شود.

---

# 87. Security Specification

تهدیدهای اصلی:

- SSRF
- Cross-tenant Access
- Credential Theft
- API Key Leakage
- Queue Poisoning
- Webhook Abuse
- XSS
- CSRF
- Brute Force
- Supply-chain Attack
- Probe Abuse
- Data Exfiltration

الزامات:

- KMS یا Secret Manager
- Encryption at Rest
- CSP
- HSTS
- Secure Cookies
- CSRF Protection
- Rate Limit
- Input Validation
- Output Encoding
- DNS Rebinding Protection
- Metadata Endpoint Blocking
- Redirect Re-validation

---

# 88. Privacy و Compliance

صفحات:

```text
/privacy
/terms
/security
/subprocessors
/dpa
```

موضوعات:

- Data Collection
- Retention
- Customer Content
- Data Export
- Account Deletion
- Incident Disclosure
- Security Contact

---

# 89. Backup و Disaster Recovery

- PostgreSQL Backup
- PITR
- Object Storage Versioning
- Restore Test
- Encryption
- DR Runbook

اهداف پیشنهادی:

```text
RPO: 15 minutes
RTO: 2 hours
```

---

# 90. Infrastructure و Deployment

محیط‌ها:

```text
local
development
staging
production
```

Componentها:

```text
Load Balancer
Next.js Web
Go API
Scheduler
Probe Workers
Result Consumers
Alert Workers
Notification Workers
PostgreSQL
Redis
VictoriaMetrics
Object Storage
Observability Stack
```

Development با Docker Compose و Production با Managed Containers یا Kubernetes انجام شود.

IaC با Terraform یا OpenTofu.

---

# 91. CI/CD

```text
Lint
→ Type Check
→ Unit Test
→ Integration Test
→ Security Scan
→ Build
→ Container Scan
→ Deploy Staging
→ E2E
→ Approval
→ Production
```

الزامات:

- Migration Check
- Rollback Plan
- Immutable Images
- Version Tag
- Changelog
- Preview Deployment
- Branch Protection

---

# 92. Observability خود پلتفرم

Metricها:

- API Latency
- API Error Rate
- Queue Lag
- Scheduler Lag
- Probe Duration
- Probe Failure Rate
- Result Ingestion Rate
- Alert Evaluation Latency
- Notification Failure
- Database Connections
- Redis Memory
- Metrics Ingestion Lag

Logها باید Structured JSON و دارای Correlation ID باشند.

---

# 93. Capacity Planning

فرمول:

```text
Checks per second =
monitors × regions ÷ interval_seconds
```

سناریوهای تست:

- 10k Monitor
- 100k Monitor
- 1M Result/hour
- Notification Burst
- Region Failure
- Redis Restart
- Database Failover

---

# 94. Rate Limiting و Abuse Prevention

Rate Limit:

- Login
- Signup
- Reset Password
- API
- Monitor Creation
- Test Notification
- Webhook Retry
- Status Subscription

Probe Abuse:

- Block prohibited targets
- Limit ports
- Limit payload
- Detect scan behavior
- Suspend abusive accounts

---

# 95. QA Strategy

Test Levelها:

- Unit
- Integration
- Contract
- E2E
- Load
- Security
- Accessibility
- Visual Regression
- RTL/LTR
- Light/Dark

Browser Matrix:

- Chrome
- Firefox
- Safari
- Edge
- Mobile Safari
- Chrome Android

---

# 96. Release Management

مراحل:

```text
Development
→ Internal Alpha
→ Private Beta
→ Public Beta
→ General Availability
```

Release Checklist:

- Migration Reviewed
- Monitoring Enabled
- Alerting Enabled
- Rollback Tested
- Documentation Updated
- Security Review
- Support Briefing
- Status Page Ready

---

# 97. Support و Operations Runbook

Support Tools:

- Organization Lookup
- Monitor Lookup
- Delivery Logs
- Probe Result Debug
- Re-run Check
- Session Revoke
- Usage View

Runbookها:

- API Outage
- Probe Region Outage
- Redis Failure
- PostgreSQL Degradation
- Notification Failure
- Metrics Lag
- Security Incident
- Billing Failure

---

# 98. Documentation

Public:

- Getting Started
- Monitor Types
- Alerting
- Status Pages
- API
- Webhooks
- Security
- Troubleshooting

Internal:

- ADR
- Database Schema
- Queue Contracts
- Runbooks
- Deployment
- Incident Response
- Access Management

---

# 99. Future Modules

- Error Tracking
- APM
- Logs
- Distributed Tracing
- Browser Synthetic
- Infrastructure Monitoring

---

# 100. Roadmap نهایی

## Phase 1

- هشت Monitor
- Dashboard
- Result History
- Next.js UI
- Theme
- i18n
- Basic Status Page

## Phase 2

- Organization
- Project
- Auth
- RBAC
- API Keys
- Tags
- Multi-region

## Phase 3

- Alerting
- Email/Webhook/Telegram
- Incident
- Maintenance
- SLA

## Phase 4

- Billing
- Quotas
- Usage
- Admin
- Audit

## Phase 5

- IaC
- CI/CD
- DR
- Security Hardening
- Load Testing
- Runbooks

## Phase 6

- Error Tracking
- APM
- Logs
- Tracing
- Browser Synthetic

---

# 101. تکمیل فرانت‌اند و UI/UX محصول

## 101.1 معماری نهایی فرانت‌اند

کل محصول وب در یک اپلیکیشن Next.js:

```text
apps/web/
├── app/[locale]/
│   ├── (marketing)/
│   ├── (auth)/
│   ├── (console)/
│   └── status/[slug]/
├── components/
│   ├── ui/
│   ├── layout/
│   ├── marketing/
│   ├── dashboard/
│   ├── monitors/
│   ├── alerts/
│   ├── incidents/
│   ├── billing/
│   ├── settings/
│   └── charts/
├── features/
├── lib/
├── messages/
│   ├── fa.json
│   └── en.json
└── components.json
```

## 101.2 صفحات Marketing

```text
/
├── /features
├── /pricing
├── /docs
├── /blog
├── /security
├── /status
├── /privacy
└── /terms
```

Landing Page Sections:

- Hero
- Live Product Preview
- Monitor Types
- Alerting
- Multi-region
- Incident Workflow
- Public Status Page
- Pricing
- FAQ
- CTA
- Footer

## 101.3 صفحات Console

```text
/app/dashboard
/app/monitors
/app/monitors/new
/app/monitors/[id]
/app/alerts
/app/incidents
/app/maintenance
/app/status-pages
/app/locations
/app/settings/general
/app/settings/members
/app/settings/notifications
/app/settings/api-keys
/app/settings/billing
/app/settings/usage
/app/settings/audit
```

## 101.4 Dashboard

Widgetها:

- Overall Uptime
- Active Monitors
- Down Monitors
- Open Alerts
- Open Incidents
- Certificates Expiring
- Domains Expiring
- SMTP Failures
- NTP High Offset
- Latest Events
- Latency Trend
- Region Health
- Usage Summary

## 101.5 Monitor List

- Search
- Type Filter
- Status Filter
- Tag Filter
- Project Filter
- Region Filter
- Bulk Actions
- Table/Card Toggle
- Pagination
- Saved Views در مراحل بعد

Columns:

- Name
- Type
- Target
- Status
- Latency
- Uptime
- Regions
- Last Check
- Alert State
- Actions

## 101.6 Monitor Detail

Tabs:

```text
Overview
Results
Metrics
Regions
Alerts
Incidents
Configuration
Activity
```

Overview:

- Status Header
- Current Latency
- Uptime
- Last Check
- Region Matrix
- Main Chart
- Recent Failures
- Related Alerts
- Related Incidents

## 101.7 فرم ساخت Monitor

Flow:

```text
Select Type
→ Basic Configuration
→ Assertions
→ Locations
→ Schedule
→ Alerts
→ Review
→ Create
```

برای هر هشت Monitor فرم اختصاصی موجود باشد.

## 101.8 Alert UI

- Policy List
- Policy Builder
- Condition Editor
- Channel Selector
- Test Notification
- Alert Timeline
- Recovery State
- Suppression Badge

## 101.9 Incident UI

- Incident List
- Severity Filter
- Status Filter
- Timeline
- Related Resources
- Assignee
- Internal Notes
- Public Update Composer
- Resolve Modal

## 101.10 Billing UI

- Current Plan
- Usage Meters
- Limits
- Upgrade CTA
- Payment Method
- Invoice Table
- Billing Email
- Cancel Plan
- Quota Warning Banner

## 101.11 Settings UI

Sections:

```text
General
Appearance
Language
Members
Roles
Notifications
API Keys
Billing
Usage
Audit Logs
Security
```

## 101.12 Design System

الزام:

```text
Tailwind CSS
shadcn/ui
Lucide Icons
Design Tokens
```

کامپوننت‌های shadcn فقط با CLI رسمی نصب شوند:

```bash
pnpm dlx shadcn@latest init
pnpm dlx shadcn@latest add button
pnpm dlx shadcn@latest add input
pnpm dlx shadcn@latest add select
pnpm dlx shadcn@latest add form
pnpm dlx shadcn@latest add card
pnpm dlx shadcn@latest add table
pnpm dlx shadcn@latest add dialog
pnpm dlx shadcn@latest add drawer
pnpm dlx shadcn@latest add tabs
pnpm dlx shadcn@latest add tooltip
pnpm dlx shadcn@latest add alert
pnpm dlx shadcn@latest add badge
pnpm dlx shadcn@latest add skeleton
pnpm dlx shadcn@latest add command
pnpm dlx shadcn@latest add checkbox
pnpm dlx shadcn@latest add switch
pnpm dlx shadcn@latest add radio-group
pnpm dlx shadcn@latest add pagination
pnpm dlx shadcn@latest add breadcrumb
pnpm dlx shadcn@latest add separator
```

ساخت دستی نسخه موازی از Componentهای موجود ممنوع است.

## 101.13 Theme

```text
system
light
dark
```

- Theme Toggle
- No Flash
- Chart Theme
- Persisted Preference
- Shared Tokens

## 101.14 i18n

```text
fa → rtl
en → ltr
```

- next-intl
- Locale Routing
- Language Switcher
- Localized Metadata
- Date/Number Formatting
- RTL-safe Components
- Direction-aware Drawer و Icon

## 101.15 Responsive

Desktop:

- Collapsible Sidebar
- Full Data Tables
- Multi-column Dashboard

Tablet:

- Compact Sidebar
- Horizontal Table Scroll
- Two-column Grid

Mobile:

- Drawer Navigation
- Cards instead of dense Tables
- Full-width Charts
- Sticky Primary Action
- Single-column Forms

## 101.16 Accessibility

- WCAG AA
- Keyboard Navigation
- Focus Visible
- Screen Reader Labels
- Semantic HTML
- Error Association
- Reduced Motion
- Color-independent Status

## 101.17 Stateهای UI

```text
Default
Hover
Focus
Active
Disabled
Loading
Error
Success
Empty
Selected
```

## 101.18 Live Data و Charts

```text
Historical → REST
Live → SSE
```

- ECharts برای Dashboard
- uPlot برای Dense Time-series
- Lazy Load
- Downsampling
- Sliding Window
- Incremental Update
- Chart Skeleton
- Empty State
- Error State

## 101.19 UX Acceptance Criteria

- تمام صفحات فارسی و انگلیسی باشند.
- Light/Dark/System کامل باشند.
- RTL/LTR بدون Bug باشند.
- Landing، Console و Status Page Design System مشترک داشته باشند.
- تمام فرم‌های هشت Monitor کامل باشند.
- Alert، Incident، Billing و Settings UI کامل باشند.
- Mobile، Tablet و Desktop تست شوند.
- shadcn فقط با CLI نصب شود.
- Accessibility حداقل AA باشد.

---

# 102. Definition of Done نهایی

محصول برای عرضه زمانی آماده است که:

- Tenant Isolation تست شده باشد.
- Auth و RBAC کامل باشد.
- هشت Monitor پایدار باشند.
- Alert و Recovery کار کنند.
- Email و Webhook فعال باشند.
- Incident قابل مدیریت باشد.
- Multi-region پایه فعال باشد.
- Billing یا Quota Enforcement مشخص باشد.
- Backup/Restore تست شده باشد.
- CI/CD و Rollback وجود داشته باشد.
- Observability خود پلتفرم فعال باشد.
- Security Review انجام شده باشد.
- Runbook و Documentation آماده باشد.
- Landing، Console و Status Page کامل باشند.
- فارسی/انگلیسی و Light/Dark تست شده باشند.

---

# 103. تصمیم نهایی UX بخش مانیتورها

## 103.1 ساختار Navigation

در Sidebar:

```text
Monitors
├── Monitor List
└── Add Monitor
```

Routeها:

```text
/app/monitors
/app/monitors/new
/app/monitors/[monitorId]
/app/monitors/[monitorId]/edit
```

صفحه `Monitor List` شامل Search، Filter، Bulk Action و دکمه `Add Monitor` است.  
صفحه `Add Monitor` مستقل و Section-based است و داخل Modal ساخته نمی‌شود.

## 103.2 فرم ساخت Monitor

```text
Monitor Type
Basic Information
Target Configuration
Assertions / Thresholds
Schedule
Probe Locations
Alerting
Advanced Settings
```

Action Bar:

```text
[Test Check] [Save Draft] [Cancel] [Create Monitor]
```

`Test Check` باید بدون ساخت Monitor دائمی، نتیجه واقعی یک Probe را نمایش دهد.

## 103.3 Monitor Detail

Tabs:

```text
Overview
Results
Metrics
Regions
Alerts
Incidents
Configuration
Activity
```

تمام نمودارها باید Time Range، Region Filter، Threshold، Tooltip، Missing Data، Downsampling، Export CSV، Light/Dark، RTL/LTR و Live Update با SSE داشته باشند.

---

# 104. Ping Monitor

## پارامترها

- Name
- Target Host/IP
- IPv4 / IPv6 / Auto
- Packet Count
- Packet Size
- Packet Interval
- Timeout
- Check Interval
- Retry Count
- Maximum TTL
- Source Interface/IP
- Resolve Hostname per Check
- Maximum Average RTT
- Maximum Packet Loss
- Maximum Jitter
- Failures Before Down
- Successes Before Recovery
- Locations، Project و Tags

## خروجی Check

```json
{
  "resolved_ip": "8.8.8.8",
  "packets_sent": 4,
  "packets_received": 4,
  "packet_loss_percent": 0,
  "rtt_min_ms": 18.2,
  "rtt_avg_ms": 22.6,
  "rtt_max_ms": 29.4,
  "rtt_stddev_ms": 4.1,
  "jitter_ms": 5.3,
  "ttl": 117
}
```

## کارت‌ها

- Status
- Average RTT
- Packet Loss
- Jitter
- Uptime
- Last Check

## نمودارها

- Latency: Min / Avg / Max
- Packet Loss
- Jitter
- Region Comparison

## جدول

| Time | Location | Status | Avg RTT | Min | Max | Loss | Jitter |
|---|---|---|---:|---:|---:|---:|---:|

---

# 105. HTTP/HTTPS Monitor

## پارامترها

- Name و URL
- Method
- Query Parameters
- Headers
- Request Body
- Content Type
- Basic Auth
- Bearer Token
- Custom Auth Header
- Follow Redirects
- Maximum Redirects
- User Agent
- HTTP Version Preference
- Timeout و Interval
- Verify TLS
- SNI Override
- Host Header Override
- Expected Status / Range
- Body Contains / Does Not Contain
- Regex Assertion
- JSONPath Assertion
- Header Assertions
- Maximum Total Time
- Maximum DNS / Connect / TLS / TTFB
- Response Size Limits
- Locations، Project و Tags

Secretها باید Encrypt و Mask شوند.

## خروجی Check

```json
{
  "resolved_ip": "203.0.113.10",
  "status_code": 200,
  "http_version": "HTTP/2",
  "redirect_count": 1,
  "dns_ms": 18.4,
  "connect_ms": 22.1,
  "tls_ms": 31.8,
  "ttfb_ms": 74.6,
  "download_ms": 12.7,
  "total_ms": 159.6,
  "response_size_bytes": 2451,
  "certificate_days_remaining": 63,
  "assertions_passed": 4,
  "assertions_failed": 0
}
```

## کارت‌ها

- Status
- Response Time
- Status Code
- Uptime
- TTFB
- Certificate Days
- Last Check
- Assertions

## نمودارها

- Total Response Time
- Timing Breakdown: DNS / Connect / TLS / TTFB / Download
- Status Code Distribution
- Response Size
- Availability
- Region Latency
- Certificate Days برای HTTPS

## جدول

| Time | Location | Status | Code | Total | DNS | Connect | TLS | TTFB | Size |
|---|---|---|---:|---:|---:|---:|---:|---:|---:|

UI باید Waterfall خلاصه Request Timing داشته باشد.

---

# 106. TCP Port Monitor

## پارامترها

- Name
- Host
- Port
- IPv4 / IPv6
- Connect Timeout
- Check Interval
- Retry Count
- Send Payload اختیاری
- Expected Response یا Regex
- Read Timeout
- Maximum Response Bytes
- Enable TLS
- TLS SNI
- Verify Certificate
- Minimum TLS Version
- Maximum Connect / Handshake Time
- Source Interface/IP
- Locations، Project و Tags

## خروجی Check

```json
{
  "resolved_ip": "203.0.113.20",
  "port": 5432,
  "connect_ms": 18.7,
  "tls_ms": 0,
  "bytes_sent": 0,
  "bytes_received": 0,
  "banner": null
}
```

## کارت‌ها

- Port Status
- Connect Time
- Uptime
- Last Check
- Resolved IP
- Active Regions

## نمودارها

- TCP Connect Time
- TLS Handshake Time
- Availability
- Region Comparison
- Failure Reason Distribution

## جدول

| Time | Location | Status | IP | Port | Connect | TLS | Error |
|---|---|---|---|---:|---:|---:|---|

---

# 107. DNS Monitor

## پارامترها

- Name
- Domain
- Record Type: A / AAAA / CNAME / MX / NS / TXT / SOA / PTR / SRV / CAA
- Resolver Mode: System / Public / Custom
- Resolver IP و Port
- UDP / TCP
- Recursion Desired
- DNSSEC Validation
- Timeout و Retry
- Expected Record Count
- Exact / Contains / Not Contains Value
- TTL Min / Max
- Expected Authoritative Server
- Expected RCODE
- MX Priority
- SOA Serial Change
- Locations، Project و Tags

## خروجی Check

```json
{
  "query_name": "example.com",
  "record_type": "A",
  "resolver": "1.1.1.1",
  "rcode": "NOERROR",
  "response_ms": 24.5,
  "answer_count": 2,
  "answers": [{"value": "93.184.216.34", "ttl": 300}],
  "authoritative": false,
  "truncated": false,
  "dnssec_valid": true
}
```

## کارت‌ها

- DNS Status
- Response Time
- Record Count
- Current Value
- Minimum TTL
- DNSSEC
- Last Change

## نمودارها

- DNS Response Time
- TTL
- Record Count
- RCODE Distribution
- Record Value Change Timeline
- Resolver Comparison
- Region Response Time

## جدول

| Time | Location | Resolver | Status | RCODE | Response | Answers | Min TTL |
|---|---|---|---|---|---:|---:|---:|

---

# 108. SSL/TLS Certificate Monitor

## پارامترها

- Name
- Host و Port
- SNI
- Timeout و Interval
- Verify Hostname
- Verify Chain
- Minimum TLS Version
- Cipher Policy
- Alert Before Expiration
- Alert on Issuer / Fingerprint / Chain Change
- Expected Common Name
- Expected SAN
- Allow Self-signed
- Locations، Project و Tags

## خروجی Check

```json
{
  "tls_version": "TLS1.3",
  "cipher_suite": "TLS_AES_128_GCM_SHA256",
  "handshake_ms": 41.2,
  "valid_from": "2026-06-01T00:00:00Z",
  "valid_to": "2026-09-01T23:59:59Z",
  "days_remaining": 44,
  "issuer": "Example CA",
  "subject": "CN=example.com",
  "sha256_fingerprint": "AA:BB:CC",
  "san": ["example.com", "www.example.com"],
  "chain_valid": true,
  "hostname_valid": true,
  "self_signed": false
}
```

## کارت‌ها

- Certificate Status
- Days Remaining
- Expiration Date
- Issuer
- TLS Version
- Handshake Time
- Chain Validity

## نمودارها

- Days Remaining
- TLS Handshake Time
- Certificate Change Timeline
- TLS Version by Region
- Chain Validation
- Availability

## جدول

| Time | Location | Status | Days Left | TLS | Handshake | Issuer | Changed |
|---|---|---|---:|---|---:|---|---|

---

# 109. Domain Expiration Monitor

## پارامترها

- Name
- Domain
- WHOIS / RDAP Preference
- Check Interval
- Timeout
- Alert Before Expiration
- Alert on Registrar Change
- Alert on Nameserver Change
- Alert on Status Change
- Expected Registrar
- Expected Nameservers
- Retry Policy
- Project و Tags

## خروجی Check

```json
{
  "domain": "example.com",
  "source": "rdap",
  "registered_at": "1995-08-14T00:00:00Z",
  "expires_at": "2027-08-13T23:59:59Z",
  "updated_at": "2026-06-10T00:00:00Z",
  "days_remaining": 390,
  "registrar": "Example Registrar",
  "statuses": ["clientTransferProhibited"],
  "nameservers": ["a.iana-servers.net", "b.iana-servers.net"]
}
```

## کارت‌ها

- Domain Status
- Days Remaining
- Expiration Date
- Registrar
- Last Updated
- Nameserver Count

## نمودارها

- Days Remaining
- Registration Timeline
- Registrar Change Timeline
- Nameserver Change Timeline
- Lookup Success/Failure
- WHOIS/RDAP Source Distribution

## جدول

| Time | Status | Days Left | Expires At | Registrar | Nameservers | Source |
|---|---|---:|---|---|---:|---|

---

# 110. SMTP Handshake Monitor

## پارامترها

- Name
- SMTP Host و Port
- Plain SMTP / STARTTLS / Implicit TLS
- EHLO Domain
- Verify Certificate
- SNI
- Minimum TLS Version
- Expected Banner
- Banner Regex
- Expected Capability
- Require STARTTLS
- Require AUTH Capability
- Send NOOP
- Send RSET
- Graceful QUIT
- Timeout
- Maximum Connect / Banner / EHLO / STARTTLS / Total Time
- Locations، Project و Tags

Authentication و ارسال Mail در فاز اصلی غیرفعال باشد.

## خروجی Check

```json
{
  "resolved_ip": "203.0.113.30",
  "port": 587,
  "connect_ms": 22.8,
  "banner_ms": 18.1,
  "ehlo_ms": 12.6,
  "starttls_ms": 44.2,
  "total_ms": 101.4,
  "banner": "220 mail.example.com ESMTP",
  "capabilities": ["SIZE", "STARTTLS", "AUTH PLAIN LOGIN"],
  "starttls_supported": true,
  "tls_version": "TLS1.3",
  "certificate_days_remaining": 70,
  "smtp_code": 220
}
```

## کارت‌ها

- SMTP Status
- Total Handshake Time
- SMTP Code
- STARTTLS
- Certificate Days
- Banner
- Last Check

## نمودارها

- Total Handshake Time
- Connect / Banner / EHLO / STARTTLS Breakdown
- SMTP Code Distribution
- STARTTLS Availability
- Certificate Days
- Region Comparison

## جدول

| Time | Location | Status | Code | Connect | Banner | EHLO | STARTTLS | Total |
|---|---|---|---:|---:|---:|---:|---:|---:|

---

# 111. NTP Monitor

## پارامترها

- Name
- NTP Host و Port
- NTP Version Preference
- Packet Count
- Timeout و Interval
- Maximum Clock Offset
- Maximum Round-trip Delay
- Maximum Jitter
- Allowed Stratum Range
- Leap Indicator Policy
- Require Synchronized Server
- Expected Reference ID
- Locations، Project و Tags

## خروجی Check

```json
{
  "resolved_ip": "192.0.2.10",
  "ntp_version": 4,
  "stratum": 2,
  "leap_indicator": 0,
  "reference_id": "GPS",
  "offset_ms": 1.8,
  "delay_ms": 24.4,
  "dispersion_ms": 3.1,
  "jitter_ms": 0.9,
  "synchronized": true
}
```

## کارت‌ها

- NTP Status
- Clock Offset
- Round-trip Delay
- Jitter
- Stratum
- Synchronization
- Last Check

## نمودارها

- Clock Offset با Zero Line و Threshold مثبت/منفی
- Round-trip Delay
- Jitter
- Stratum
- Synchronization State
- Region Offset Comparison

## جدول

| Time | Location | Status | Offset | Delay | Jitter | Stratum | Sync |
|---|---|---|---:|---:|---:|---:|---|

---

# 112. Error Codeهای UI

## عمومی

```text
TIMEOUT
DNS_RESOLUTION_FAILED
CONNECTION_REFUSED
NETWORK_UNREACHABLE
INVALID_TARGET
PERMISSION_DENIED
PROBE_OFFLINE
CONFIGURATION_ERROR
ASSERTION_FAILED
```

## Type-specific

```text
HTTP_STATUS_MISMATCH
HTTP_BODY_MISMATCH
HTTP_HEADER_MISMATCH
HTTP_REDIRECT_LIMIT
ICMP_TIMEOUT
PACKET_LOSS_THRESHOLD
LATENCY_THRESHOLD
JITTER_THRESHOLD
TCP_CONNECT_TIMEOUT
TCP_RESPONSE_MISMATCH
DNS_NXDOMAIN
DNS_SERVFAIL
DNS_RECORD_MISMATCH
DNSSEC_VALIDATION_FAILED
TLS_HANDSHAKE_FAILED
TLS_HOSTNAME_MISMATCH
TLS_CHAIN_INVALID
TLS_EXPIRED
DOMAIN_LOOKUP_FAILED
DOMAIN_EXPIRING
SMTP_BANNER_INVALID
SMTP_EHLO_FAILED
SMTP_STARTTLS_FAILED
NTP_UNSYNCHRONIZED
NTP_OFFSET_THRESHOLD
NTP_STRATUM_INVALID
```

UI باید Error Code را به پیام قابل‌فهم و راه‌حل عملی تبدیل کند.

---

# 113. مدل نهایی Probe Infrastructure

Probeها بخشی از زیرساخت خود پلتفرم هستند و روی سرورهای متعلق به تیم پلتفرم اجرا می‌شوند.

```text
Platform Infrastructure
├── Probe Tehran
├── Probe Frankfurt
├── Probe Singapore
└── Probe New York
```

کاربر نهایی فقط Locationها را در فرم Monitor انتخاب می‌کند و به مدیریت Probe دسترسی ندارد.

مدیریت Probeها فقط در Admin Panel انجام می‌شود.

---

# 114. Probe Agent باینری

## 114.1 هدف

روی هر Probe Server یک Agent باینری نصب می‌شود که:

- خود را به Control Plane معرفی کند.
- Pending Registration ایجاد کند.
- Heartbeat ارسال کند.
- Job دریافت کند.
- Check اجرا کند.
- Result ارسال کند.
- نسخه و ظرفیت خود را گزارش دهد.
- در قطع ارتباط Backoff و Local Queue داشته باشد.

Agent روی Targetهای مانیتورشونده نصب نمی‌شود؛ فقط روی Probe Serverهای خود پلتفرم نصب می‌شود.

## 114.2 فناوری

Agent با Go ساخته شود و Single Binary باشد.

Artifactها:

```text
probe-agent-linux-amd64
probe-agent-linux-arm64
probe-agent-windows-amd64.exe
probe-agent-darwin-amd64
probe-agent-darwin-arm64
```

برای فاز اول Linux AMD64 و ARM64 کافی است.

## 114.3 Commandها

```bash
probe-agent install
probe-agent uninstall
probe-agent start
probe-agent stop
probe-agent restart
probe-agent status
probe-agent logs
probe-agent enroll
probe-agent diagnose
probe-agent version
probe-agent update
probe-agent run
```

نمونه نصب:

```bash
sudo probe-agent install \
  --control-plane https://control.example.com \
  --token enr_xxxxx \
  --name probe-frankfurt-01
```

## 114.4 کارهای خودکار Installer

- Copy Binary
- ساخت System User
- ساخت Directoryها
- ساخت Config
- نصب systemd Service
- تنظیم Restart Policy
- تنظیم Log
- اعطای Capability محدود ICMP
- اجرای Service
- Enrollment اولیه

مسیرها:

```text
/usr/local/bin/probe-agent
/etc/probe-agent/config.yaml
/var/lib/probe-agent/
/var/log/probe-agent/
```

برای ICMP:

```bash
sudo setcap cap_net_raw+ep /usr/local/bin/probe-agent
```

Agent بعد از نصب با User غیر Root اجرا شود.

---

# 115. Enrollment و تأیید ادمین

Flow:

```text
Admin creates enrollment token
→ Agent installed on platform probe server
→ Agent connects
→ Probe appears as Pending
→ Platform Admin reviews
→ Admin approves
→ mTLS certificate issued
→ Probe becomes Active
```

Token باید:

- Single-use
- Short-lived
- Hash‌شده
- متصل به Location
- پس از Enrollment باطل
- غیرقابل استفاده به‌عنوان Credential دائمی

وضعیت‌ها:

```text
pending
approved
active
offline
disabled
rejected
revoked
updating
```

## اطلاعات Registration

```json
{
  "hostname": "probe-frankfurt-01",
  "agent_version": "1.0.0",
  "os": "linux",
  "architecture": "amd64",
  "private_ips": ["10.10.1.20"],
  "public_ip": "203.0.113.10",
  "capabilities": ["http", "icmp", "tcp", "dns", "tls", "domain", "smtp", "ntp"],
  "cpu_count": 4,
  "memory_bytes": 8589934592
}
```

---

# 116. Identity و mTLS

Agent باید Key Pair را محلی بسازد.

```text
Agent generates private key
→ Sends CSR
→ Admin approves
→ Control Plane signs certificate
→ Agent receives client certificate
→ Enrollment token revoked
→ Future requests use mTLS
```

Private Key نباید از Probe خارج شود.

Certificateها باید:

- Expiration
- Rotation
- Revocation
- Probe ID
- Environment
- Allowed Scope

داشته باشند.

---

# 117. ارتباط Probe با Control Plane

تمام ارتباط‌ها از Agent به Control Plane آغاز شوند.

```text
Probe Agent → Control Plane
```

نیازی به Port ورودی روی Probe نیست.

برای فاز اول:

```text
Enrollment: HTTPS REST
Heartbeat: HTTPS
Job Fetch: Long Polling
Result Upload: HTTPS Batch
Authentication: mTLS after approval
```

برای مقیاس بالاتر:

```text
Control Channel: gRPC Stream
Results: Batched HTTPS or stream
```

---

# 118. Heartbeat و ظرفیت

Heartbeat هر 15 تا 30 ثانیه:

```json
{
  "probe_id": "prb_123",
  "agent_version": "1.0.0",
  "running_jobs": 8,
  "available_slots": 92,
  "cpu_percent": 15,
  "memory_percent": 28,
  "queue_depth": 3,
  "sent_at": "2026-07-19T10:00:00Z"
}
```

اگر Heartbeat از Threshold عبور کند:

```text
active → offline
```

Scheduler به Probe Offline Job جدید نمی‌دهد.

---

# 119. Job Model

نمونه Job:

```json
{
  "job_id": "job_123",
  "monitor_id": "mon_456",
  "type": "http",
  "target": "https://example.com/health",
  "timeout_millis": 10000,
  "attempt": 1,
  "location_id": "loc_frankfurt",
  "required_capabilities": ["http"],
  "config": {}
}
```

انتخاب Probe:

```text
status = active
location matches
capability matches
capacity available
recent heartbeat
supported version
```

Agent فقط Executorهای از پیش تعریف‌شده را اجرا کند و نباید Shell یا Arbitrary Code اجرا کند.

---

# 120. Local Queue و قطع ارتباط

در قطع Control Plane:

- Agent Crash نکند.
- Exponential Backoff داشته باشد.
- Resultها را در Local Queue محدود ذخیره کند.
- Queue Disk Limit داشته باشد.
- پس از اتصال Batch Upload کند.
- Job منقضی را اجرا نکند.
- Duplicate Result با Idempotency Key کنترل شود.

---

# 121. Update Agent

Update Policy:

```text
manual
stable
beta
```

الزامات:

- Signed Artifact
- SHA-256 Verification
- Version Compatibility
- Previous Binary Backup
- Atomic Replace
- Service Restart
- Health Check
- Rollback on Failure

برای فاز اول Update دستی پیشنهاد می‌شود.

---

# 122. Admin UI مدیریت Probeها

مسیر:

```text
/admin/probes
/admin/probes/new
/admin/probes/[probeId]
```

Tabs:

```text
All
Pending
Active
Offline
Disabled
Revoked
```

جدول:

| Name | Location | IP | Status | Version | Capacity | Running Jobs | Last Seen |
|---|---|---|---|---|---:|---:|---|

صفحه Pending:

- Hostname
- Public/Private IP
- OS/Architecture
- Agent Version
- Capabilities
- CPU/RAM
- Enrollment Time
- Token Source
- Approve
- Reject

هنگام Approve:

- Display Name
- Location
- Tags
- Concurrent Job Limit
- Allowed Monitor Types
- Update Channel

صفحه Detail:

- Health
- Heartbeat Timeline
- CPU/RAM
- Running Jobs
- Queue Depth
- Version
- Certificate
- Capabilities
- Recent Errors
- Disable
- Revoke
- Rotate Certificate
- Trigger Update

---

# 123. Public Location UI برای کاربران

کاربر نهایی فقط این اطلاعات را می‌بیند:

- Location Name
- Country/Region
- Availability
- Supported Monitor Types
- Status کلی
- Plan Availability

کاربر نباید این موارد را ببیند:

- Probe Hostname
- IP داخلی
- Certificate
- Agent Logs
- Queue
- Infrastructure Metrics
- Enrollment Token

---

# 124. Security Agent

- Non-root runtime
- Minimal Linux capabilities
- mTLS
- Certificate Rotation
- Signed Binary
- No Arbitrary Command
- No Remote Script
- Strict Job Schema
- Payload Limits
- Protocol Allowlist
- Port Policy
- Secret Masking
- Local Queue Protection
- Immediate Revocation
- Audit Log
- Rate Limit
- Binary provenance

---

# 125. معیار پذیرش نهایی

- هر هشت Monitor فرم اختصاصی داشته باشد.
- Metric، کارت، نمودار و جدول هر Type مشخص باشد.
- Test Check در فرم وجود داشته باشد.
- Region Comparison پشتیبانی شود.
- Error Code و Troubleshooting نمایش داده شود.
- Probe Agent باینری Go ساخته شود.
- Installer خودکار systemd نصب کند.
- Probe بعد از اتصال Pending باشد.
- فقط Platform Admin بتواند Probe را تأیید کند.
- ارتباط بعد از تأیید با mTLS باشد.
- Heartbeat، Capacity، Job Fetch و Result Upload فعال باشد.
- Agent فقط Checkهای تعریف‌شده را اجرا کند.
- کاربر فقط Locationها را انتخاب کند.
- Admin UI مدیریت کامل Probeها را داشته باشد.

---

# 126. سیستم Alerting و Notification نهایی

این بخش مرجع اجرایی پیاده‌سازی Alerting، Notification Routing، Delivery، Retry، Suppression و UI مدیریتی است.

## 126.1 معماری

```text
Probe Result
→ Result Ingestion
→ Monitor State Evaluator
→ Alert Rule Evaluator
→ Deduplication
→ Alert State Machine
→ Notification Router
→ Delivery Queue
→ Delivery Worker
→ Email / Webhook / Telegram / Slack
```

Probe Agent فقط Result تولید می‌کند و نباید داخل Probe درباره Alert یا Notification تصمیم‌گیری شود.

---

# 127. Monitor State Evaluator

Stateهای Monitor:

```text
UP
DEGRADED
DOWN
UNKNOWN
PAUSED
```

State Evaluator براساس Result خام، Thresholdها، Regionها و Policy نهایی تصمیم می‌گیرد.

نمونه Ping:

```text
packet_loss = 100% → DOWN candidate
packet_loss > threshold → DEGRADED candidate
avg_rtt > threshold → DEGRADED candidate
otherwise → UP candidate
```

نمونه HTTP:

```text
timeout → DOWN candidate
connection failure → DOWN candidate
critical assertion failed → DOWN candidate
latency threshold exceeded → DEGRADED candidate
all required assertions passed → UP candidate
```

State نهایی باید از Raw Result جدا ذخیره شود.

---

# 128. Alert Policy

فیلدهای اصلی:

```text
id
organization_id
project_id
name
description
enabled
scope
conditions
severity
failure_count
recovery_count
cooldown_seconds
renotify_interval_seconds
recovery_notification_enabled
channels
schedule
created_by
created_at
updated_at
```

Scopeها:

```text
all_monitors
project
monitor_type
monitor_ids
tags
regions
```

Conditionها:

- Monitor Down
- Monitor Degraded
- Consecutive Failures
- Latency Threshold
- Packet Loss Threshold
- Jitter Threshold
- HTTP Status Mismatch
- Assertion Failure
- DNS Record Mismatch
- DNSSEC Failure
- TLS Expiring
- TLS Invalid
- Domain Expiring
- SMTP Handshake Failure
- NTP Offset Threshold
- Probe Offline
- Missing Results
- Region Quorum Failure

---

# 129. Anti-flapping و Trigger Rules

قانون پایه:

```text
Open alert after 3 consecutive failures.
Resolve alert after 2 consecutive successes.
```

تنظیمات:

```text
failure_count
recovery_count
evaluation_window
minimum_regions
quorum_mode
```

Quorum Mode:

```text
any
majority
all
custom
```

مثال:

```text
Monitor in 3 regions
→ Alert only if 2 of 3 regions fail
```

---

# 130. Alert State Machine

Stateهای Alert:

```text
pending
firing
acknowledged
recovering
resolved
suppressed
```

Transitionها:

```text
healthy → pending
pending + threshold met → firing
firing + user action → acknowledged
firing + partial recovery → recovering
recovering + recovery threshold met → resolved
firing + maintenance/mute → suppressed
suppressed + suppression end + issue active → firing
```

Resolved Alert نباید دوباره باز شود؛ در رخداد جدید Alert جدید ساخته شود.

---

# 131. Deduplication

برای هر مسئله فقط یک Alert باز وجود داشته باشد.

Deduplication Key:

```text
organization_id
project_id
monitor_id
alert_policy_id
condition_type
region_scope
```

تا زمانی که Alert در یکی از حالت‌های زیر است، Alert جدید ساخته نشود:

```text
pending
firing
acknowledged
recovering
suppressed
```

Resultهای جدید باید Timeline همان Alert را به‌روز کنند.

---

# 132. Cooldown و Renotify

Cooldown:

```text
cooldown_seconds
```

هدف:

- جلوگیری از ارسال چند Notification در فاصله کوتاه
- کنترل Burst

Renotify:

```text
renotify_interval_seconds
```

مثال:

```text
Initial alert immediately
Reminder every 30 minutes while still firing
```

Renotify باید قابل خاموش‌کردن باشد.

---

# 133. Recovery Notification

Recovery Notification باید مستقل قابل فعال یا غیرفعال‌کردن باشد.

Payload نمونه:

```json
{
  "event": "alert.resolved",
  "alert_id": "alt_123",
  "monitor_id": "mon_123",
  "started_at": "2026-07-19T10:20:00Z",
  "resolved_at": "2026-07-19T10:31:24Z",
  "duration_seconds": 684,
  "failed_checks": 14,
  "recovered_regions": ["frankfurt"]
}
```

در UI:

```text
Resolved
Downtime: 11m 24s
Failed checks: 14
Recovered from: Frankfurt
```

---

# 134. Notification Router

Router براساس موارد زیر Channel نهایی را انتخاب می‌کند:

- Severity
- Organization
- Project
- Tags
- Monitor Type
- Alert Policy
- Region
- Schedule
- Maintenance Window
- Mute Rule
- Channel Status
- Escalation Step

Routing Rule نمونه:

```text
severity = critical
AND project = production
→ Email + Telegram
```

---

# 135. Notification Channels

## 135.1 Email

Eventها:

- Alert Fired
- Alert Resolved
- Alert Acknowledged
- Incident Update
- Probe Offline
- TLS Expiring
- Domain Expiring
- Billing Warning

الزامات:

- HTML version
- Plain-text version
- Localized subject/body
- Link مستقیم به Alert
- Unsubscribe فقط برای Notificationهای غیرعملیاتی
- Delivery Log
- Provider Message ID
- Retry

## 135.2 Webhook

Payload باید Versioned باشد:

```json
{
  "version": "1",
  "event": "alert.fired",
  "id": "evt_123",
  "occurred_at": "2026-07-19T10:20:00Z",
  "organization_id": "org_123",
  "project_id": "prj_123",
  "data": {
    "alert_id": "alt_123",
    "monitor_id": "mon_123",
    "monitor_name": "Production API",
    "monitor_type": "http",
    "status": "down",
    "severity": "critical",
    "reason": "HTTP_STATUS_MISMATCH",
    "expected": "200-299",
    "actual": 500
  }
}
```

الزامات:

- HMAC Signature
- Timestamp Header
- Event ID
- Idempotency Key
- Retry
- Exponential Backoff
- Timeout
- Delivery Log
- Secret Rotation
- Replay Protection
- Disable خودکار پس از Failure زیاد

Headerهای پیشنهادی:

```text
X-Monitoring-Event
X-Monitoring-Event-ID
X-Monitoring-Timestamp
X-Monitoring-Signature
```

Signature:

```text
HMAC-SHA256(secret, timestamp + "." + raw_body)
```

## 135.3 Telegram

قابلیت‌ها:

- Bot Token رمزنگاری‌شده
- Chat ID
- Test Message
- Markdown-safe Formatting
- Rate Limit
- Message Template

نمونه:

```text
🔴 Production API is DOWN

Reason: HTTP 500
Location: Frankfurt
Failed checks: 3
Started: 10:20

Open alert
```

## 135.4 Slack و سایر Channelها

برای مراحل بعد:

- Slack
- Microsoft Teams
- PagerDuty
- Opsgenie
- SMS

Interface داخلی Channel باید از ابتدا قابل توسعه باشد.

---

# 136. Notification Delivery Pipeline

Queueها:

```text
alert-events
notification-jobs
notification-retries
notification-dead-letter
```

جریان:

```text
Alert Event
→ Notification Router
→ Notification Job
→ Delivery Worker
→ Provider
→ Delivery Result
```

هر Job باید داشته باشد:

```text
id
event_id
alert_id
channel_id
attempt
scheduled_at
idempotency_key
payload_version
status
```

---

# 137. Retry Policy

Policy پیشنهادی:

```text
Attempt 1: immediately
Attempt 2: after 30 seconds
Attempt 3: after 2 minutes
Attempt 4: after 10 minutes
Attempt 5: after 30 minutes
```

پس از آخرین Failure:

```text
delivery_failed
```

Job باید وارد Dead-letter Queue شود.

Retry فقط برای Errorهای قابل Retry انجام شود:

```text
timeout
network_error
provider_5xx
rate_limited
```

برای Errorهای غیرقابل Retry:

```text
invalid_recipient
invalid_token
malformed_payload
permission_denied
```

Delivery باید بلافاصله Failed شود.

---

# 138. Maintenance، Mute و Suppression

پیش از Notification باید بررسی شود:

```text
monitor paused?
maintenance active?
alert muted?
project suppressed?
organization suspended?
channel disabled?
```

Mute:

```text
Check continues
Alert may continue
Notification suppressed
```

Pause:

```text
Check execution stops
No new result
Monitor state = PAUSED
```

Maintenance Policy:

- Record alert silently
- Suppress notification
- Exclude from SLA
- Optional pause checks

Suppression باید در Alert Timeline ثبت شود.

---

# 139. Escalation Policy

ساختار پیشنهادی:

```text
0 min  → Telegram
5 min  → Email
15 min → Slack
30 min → PagerDuty
```

فیلدها:

```text
policy_id
step_order
delay_seconds
channels
stop_on_acknowledge
stop_on_resolve
```

برای MVP الزامی نیست، اما مدل داده باید آماده باشد.

---

# 140. مدل داده Alerting

Tableهای اصلی:

```text
alert_policies
alerts
alert_events
notification_channels
notification_deliveries
notification_attempts
maintenance_windows
mute_rules
escalation_policies
escalation_steps
```

## alerts

```text
id
organization_id
project_id
monitor_id
policy_id
condition_type
deduplication_key
severity
status
started_at
acknowledged_at
resolved_at
suppressed_at
last_event_at
failure_count
recovery_count
metadata
```

## alert_events

```text
id
alert_id
event_type
monitor_result_id
region_id
message
metadata
created_at
```

## notification_deliveries

```text
id
alert_id
channel_id
event_type
status
attempt_count
provider_message_id
last_error_code
last_error_message
next_retry_at
created_at
updated_at
```

---

# 141. UI Alert Policies

مسیر:

```text
/app/alerts/policies
```

جدول:

| Name | Scope | Condition | Severity | Channels | Status |
|---|---|---|---|---|---|

صفحه ساخت:

```text
1. Scope
2. Conditions
3. Trigger Rules
4. Recovery
5. Notification Channels
6. Schedule
7. Review
```

فرم می‌تواند Section-based باشد، ولی Step Indicator داشته باشد.

Actionها:

```text
Test Rule
Save Draft
Enable
Disable
Duplicate
Delete
```

---

# 142. UI Notification Channels

مسیر:

```text
/app/settings/notifications
```

Card/Table هر Channel:

- Name
- Type
- Enabled
- Last Delivery
- Last Error
- Test
- Edit
- Disable
- Delete

فرم ساخت:

```text
Channel Type
Configuration
Template
Test Notification
Save
```

Secretها Mask شوند.

---

# 143. UI Alert List و Detail

## Alert List

مسیر:

```text
/app/alerts
```

Filterها:

- Status
- Severity
- Project
- Monitor
- Type
- Region
- Acknowledged
- Suppressed
- Date Range

Columns:

| Alert | Monitor | Severity | Status | Started | Duration | Notifications |
|---|---|---|---|---|---|---|

## Alert Detail

مسیر:

```text
/app/alerts/[alertId]
```

محتوا:

- Current Status
- Severity
- Monitor
- Started At
- Duration
- Reason
- Region Breakdown
- Timeline
- Notification Deliveries
- Related Incident
- Acknowledge
- Mute
- Create Incident
- Resolve Manually در صورت مجازبودن

---

# 144. Notification Delivery UI

در Alert Detail:

| Channel | Event | Status | Attempts | Last Attempt | Error |
|---|---|---|---:|---|---|

در Admin Panel:

```text
/admin/notifications
```

Filterها:

- Provider
- Status
- Error Code
- Organization
- Date Range
- Retryable
- Dead-letter

Actionها:

- Retry Now
- Disable Channel
- View Payload
- View Response
- Move from Dead-letter

Payload و Response باید Sanitized باشند.

---

# 145. Template System

Templateها:

```text
alert_fired
alert_resolved
alert_acknowledged
incident_updated
probe_offline
tls_expiring
domain_expiring
```

هر Template:

- Locale
- Channel Type
- Subject
- Body
- Variables
- Version

Variable نمونه:

```text
monitor.name
monitor.type
monitor.target
alert.severity
alert.reason
alert.started_at
alert.duration
region.name
dashboard_url
```

Template Rendering باید Escape-safe باشد.

---

# 146. Error Codeهای Notification

```text
NOTIFICATION_TIMEOUT
PROVIDER_UNAVAILABLE
RATE_LIMITED
INVALID_RECIPIENT
INVALID_TOKEN
SIGNATURE_FAILED
PAYLOAD_REJECTED
TEMPLATE_RENDER_FAILED
CHANNEL_DISABLED
DELIVERY_RETRIES_EXHAUSTED
```

UI باید پیام قابل‌فهم و اقدام پیشنهادی نمایش دهد.

---

# 147. Security Alerting

- Encrypt Channel Secrets
- Never Log Secret
- HMAC Webhook
- Replay Protection
- Rate Limit Test Notification
- Permission Check
- Audit Log
- Payload Sanitization
- PII Minimization
- Provider Credential Rotation
- Dead-letter Access فقط برای Admin مجاز

---

# 148. Observability Alerting

Metricها:

```text
alerts_opened_total
alerts_resolved_total
alert_evaluation_latency_seconds
notification_jobs_total
notification_delivery_success_total
notification_delivery_failure_total
notification_retry_total
notification_dead_letter_total
notification_queue_lag_seconds
```

Logها:

- alert_id
- event_id
- channel_id
- provider
- attempt
- error_code
- correlation_id

---

# 149. معیار پذیرش سیستم Alerting

- Probe فقط Result تولید کند.
- State Evaluator جدا باشد.
- Consecutive Failure/Recovery وجود داشته باشد.
- Deduplication فعال باشد.
- Alert State Machine کامل باشد.
- Cooldown و Renotify پشتیبانی شوند.
- Recovery Notification وجود داشته باشد.
- Email، Webhook و Telegram پیاده‌سازی شوند.
- Retry Queue و Dead-letter وجود داشته باشد.
- Delivery Log قابل مشاهده باشد.
- Test Notification وجود داشته باشد.
- Maintenance و Mute Notification را Suppress کنند.
- Webhook HMAC و Replay Protection داشته باشد.
- UI Policy، Channel، Alert List و Alert Detail کامل باشد.
- Alert و Notification در Light/Dark و فارسی/انگلیسی تست شوند.

---

# 150. تصمیم نهایی معماری محصول و UX مبتنی بر Node

این بخش مرجع نهایی نام‌گذاری، مدل دامنه و طراحی رابط کاربری است و تصمیم‌های Monitor-centric قبلی را در UI کاربر جایگزین می‌کند.

## 150.1 واژگان رسمی

```text
Node
= تارگت، سرور، دامنه، API، وب‌سایت یا سرویس متعلق به کاربر.

Monitor
= Check خارجی و Agentless فعال روی Node؛ مانند Ping، HTTP، DNS، TCP، TLS، Domain، SMTP و NTP.

Node Agent
= Agent آینده که روی Node کاربر نصب می‌شود و CPU، RAM، Disk، Network، Process، Log و Telemetry داخلی را جمع‌آوری می‌کند.

Probe
= سرور اجرایی متعلق به زیرساخت پلتفرم که Checkهای Agentless را اجرا می‌کند و فقط Platform Admin آن را مدیریت می‌کند.
```

Probe در UI عادی کاربر نمایش داده نمی‌شود؛ کاربر فقط Probe Location عمومی را می‌بیند.

# 151. مدل دامنه Node-centric

```text
Node
├── Agentless Monitors
│   ├── Ping
│   ├── HTTP
│   ├── DNS
│   ├── TCP
│   ├── TLS
│   ├── Domain
│   ├── SMTP
│   └── NTP
├── Agent-based Telemetry
│   ├── CPU
│   ├── Memory
│   ├── Disk
│   ├── Network
│   ├── Processes
│   ├── Services
│   ├── Containers
│   └── Logs
├── Alerts
├── Incidents
├── Maintenance
├── Status Page Components
└── SLA / SLO
```

هر Monitor به یک Node تعلق دارد ولی Target مستقل خودش را دارد.

# 152. Navigation نهایی

```text
Dashboard

Nodes
├── Node List
└── Add Node

Monitoring
├── All Monitors
└── Alert Policies

Alerts
Incidents
Maintenance
Status Pages
Service Levels
Notifications
Settings
```

بخش‌های آینده با Feature Flag:

```text
Infrastructure
├── Agents
├── Processes
├── Services
└── Containers

Logs
Errors
Traces
```

# 153. معماری سه‌سطحی UX

```text
Level 1: Node Inventory
Level 2: Node Overview
Level 3: Signal-specific Detail
```

Routeها:

```text
/app/nodes
/app/nodes/[nodeId]
/app/nodes/[nodeId]/monitors/[monitorId]
```

Routeهای آینده:

```text
/app/nodes/[nodeId]/infrastructure
/app/nodes/[nodeId]/logs
/app/nodes/[nodeId]/errors
/app/nodes/[nodeId]/traces
```

# 154. صفحه Node List

مسیر:

```text
/app/nodes
```

- Desktop: Table View پیش‌فرض
- Mobile: Card View پیش‌فرض
- انتخاب کاربر ذخیره شود

## Header و Summary

```text
Nodes                                      [+ Add Node]

Total Nodes | Healthy | Degraded | Down | Unknown | Active Alerts
```

کلیک روی Summary Card باید Filter متناظر را اعمال کند.

## Toolbar

```text
[Search name or target]
[Node Status] [Node Type] [Monitor Type] [Monitor Status]
[Project] [Environment] [Tags] [Active Alert] [Maintenance]
[Table] [Cards] [Group By] [Sort]
```

Group By:

```text
Project
Environment
Node Type
Status
Tag
```

# 155. جدول نهایی Nodes

| Node | Target | Monitor Health | Overall Status | Alerts | Last Seen |
|---|---|---|---|---:|---|

نسخه توسعه‌یافته اختیاری:

| Node | Target | Collection | Monitor Health | Overall Status | Uptime | Alerts | Last Seen |
|---|---|---|---|---|---:|---:|---|

## ستون Node

```text
Production API
API · Production
```

شامل Icon نوع Node، نام، نوع، Environment/Project و Tagهای مهم.

## ستون Target

Primary Target نمایش داده شود. Targetهای اضافی در Tooltip یا Popover دیده شوند.

## ستون Monitor Health

Monitorهای فعال به شکل Badge رنگی و قابل کلیک نمایش داده شوند:

```text
[Ping ✓] [HTTP ×] [DNS ✓] [TLS !]
```

وضعیت‌ها:

```text
UP           → سبز
DEGRADED     → زرد/نارنجی
DOWN         → قرمز
UNKNOWN      → خاکستری
MAINTENANCE  → آبی/بنفش
PAUSED       → خنثی
```

فقط رنگ کافی نیست؛ Badge باید Icon یا علامت وضعیت و متن قابل دسترس داشته باشد.

اگر تعداد Badgeها زیاد بود:

```text
[Ping ✓] [HTTP ×] [DNS ✓] [TLS !] [+4]
```

`+4` یک Popover کامل باز کند.

## Tooltip Badge

نمونه HTTP:

```text
HTTP Monitor
Status: Down
Target: https://api.example.com/health
Result: HTTP 500
Response time: 284 ms
Last check: 8 seconds ago
```

نمونه Ping:

```text
Ping Monitor
Status: Up
Average RTT: 24 ms
Packet loss: 0%
Last check: 12 seconds ago
```

نمونه TLS:

```text
TLS Monitor
Status: Degraded
Days remaining: 12
Expires at: 2026-08-01
```

## رفتار کلیک

```text
Click Row → Node Detail
Click Monitor Badge → Monitor Detail
Click Alert Count → Node Alerts
```

## Row Actions

```text
Run All Checks
Add Monitor
Pause All
Resume All
Mute Alerts
Create Maintenance
Edit Node
Duplicate Node
Export
Delete
```

# 156. Card View

```text
┌──────────────────────────────────────────┐
│ Production API               ● Degraded │
│ api.example.com                         │
│ Production · API                        │
│                                          │
│ [Ping ✓] [HTTP ×] [DNS ✓] [TLS !]      │
│                                          │
│ Healthy monitors        3 / 4           │
│ Uptime                  99.95%           │
│ Active alerts           1               │
│ Last seen               8 sec ago       │
│                                          │
│ [View Node]                       [⋯]    │
└──────────────────────────────────────────┘
```

- Large Desktop: سه ستون
- Laptop/Tablet: دو ستون
- Mobile: یک ستون

Card نباید نمودارها و Metricهای تخصصی را شلوغ کند.

# 157. محاسبه Overall Status

هر Monitor یا Signal دارای Criticality است:

```text
critical
important
informational
```

Policy پایه:

```text
Critical monitor DOWN → Node DOWN
Important monitor DOWN → Node DEGRADED
Any monitor DEGRADED → Node DEGRADED
All enabled monitors UP → Node UP
All signals PAUSED → Node PAUSED
No recent data → Node UNKNOWN
Maintenance active → Node MAINTENANCE
```

در آینده وضعیت Node از این Health Domainها محاسبه شود:

```text
External Health
Host Health
Application Health
Data Collection Health
```

# 158. آمادگی برای Node Agent آینده

Collection Mode:

```text
Agentless
Agent
Hybrid
Not configured
```

در Node List ستون ثابت CPU، RAM، Disk و Network ساخته نشود، چون همه Nodeها Host نیستند.

Host Telemetry در لیست به شکل Signal تجمیعی نمایش داده شود:

```text
[Ping ✓] [HTTP ✓] [DNS ✓] [Host !]
```

Tooltip Host:

```text
Node Agent      Connected
CPU             82% · Warning
Memory          61% · Healthy
Disk            44% · Healthy
Network         Healthy
```

Metricهای کامل Host فقط در Node Detail نمایش داده شوند.

# 159. صفحه Node Detail

مسیر:

```text
/app/nodes/[nodeId]
```

Tabs فعلی:

```text
Overview
Monitors
Alerts
Incidents
Maintenance
Configuration
Activity
```

Tabs آینده با Feature Flag:

```text
Infrastructure
Processes
Services
Containers
Logs
Errors
Traces
```

# 160. Node Overview

## Header

```text
Production API                         DEGRADED
api.example.com
Production · API

[Run All Checks] [Add Monitor] [Edit Node] [More]
```

## Summary Cards

```text
Overall Status
Enabled Monitors
Healthy Monitors
Availability
Active Alerts
Last Seen
```

در آینده:

```text
Node Agent Status
CPU
Memory
Disk
```

## Monitoring Coverage

```text
External checks
✓ Ping
✓ HTTP
✓ DNS
✓ TLS

Host telemetry
○ CPU
○ Memory
○ Disk
○ Network
○ Processes
○ Logs
```

## Monitor Health Matrix

| Monitor | Status | Primary Metric | Uptime | Locations | Last Check |
|---|---|---:|---:|---:|---|
| Ping | UP | 24 ms | 99.99% | 3 | 8s ago |
| HTTP | DOWN | HTTP 500 | 99.81% | 4 | 5s ago |
| TCP 443 | UP | 18 ms | 99.99% | 2 | 9s ago |
| TLS | DEGRADED | 12 days | 100% | 2 | 4h ago |
| DNS A | UP | 31 ms | 100% | 2 | 14s ago |

# 161. نمودارهای Node Overview

صفحه Node فقط نمودارهای تجمیعی را نشان دهد.

## Overall Availability Timeline

```text
UP / DEGRADED / DOWN / MAINTENANCE / PAUSED / NO DATA
```

## Monitor Status Swimlane

```text
Ping  ───────────── UP ─────────────
HTTP  ───── UP ── DOWN ─── UP ─────
TCP   ───────────── UP ─────────────
TLS   ───── UP ── DEGRADED ─────────
DNS   ───────────── UP ─────────────
```

## Primary Metric Sparklines

```text
Ping RTT          24 ms       ▁▂▂▃▂▁
HTTP Latency      184 ms      ▂▃▅▂▃▂
DNS Response      31 ms       ▁▂▁▁▃▂
TCP Connect       18 ms       ▁▁▂▁▂▁
TLS Remaining     12 days     ▆▅▅▄▃▂
```

Metricهای مختلف روی یک محور مشترک رسم نشوند.

## Availability by Monitor

| Monitor | Availability |
|---|---:|
| Ping | 99.99% |
| HTTP | 99.81% |
| TCP | 99.99% |
| TLS | 100% |
| DNS | 100% |

## Alert / Incident Timeline

Markerها:

```text
Alert Fired
Incident Started
Acknowledged
Maintenance Started
Resolved
```

# 162. Monitor Detail

مسیر:

```text
/app/nodes/[nodeId]/monitors/[monitorId]
```

نمودارهای تخصصی فقط اینجا نمایش داده شوند:

```text
Ping → RTT، Packet Loss، Jitter، Region Comparison
HTTP → Response Time، Timing Breakdown، Status Codes، Response Size
DNS → Response Time، TTL، Record Count، RCODE، Record Changes
TCP → Connect Time، TLS Handshake، Failure Reasons
TLS → Days Remaining، Handshake، Certificate Timeline
Domain → Days Remaining، Registrar/Nameserver Changes
SMTP → Protocol Breakdown، SMTP Codes، STARTTLS
NTP → Offset، Delay، Jitter، Stratum
```

# 163. Add Node Flow

مسیر:

```text
/app/nodes/new
```

مرحله اول:

```text
Name
Node Type
Primary Target
Project
Environment
Tags
Description
```

مرحله دوم:

```text
Monitoring Mode
- External Monitoring
- Install Node Agent
- Use Both
```

در MVP فقط External Monitoring فعال باشد.

مرحله سوم:

```text
Enable Monitors
Ping
HTTP
DNS
TCP
TLS
Domain
SMTP
NTP
```

برای هر Monitor فرم اختصاصی خودش نمایش داده شود.

# 164. Empty، Loading و Error State

Empty:

```text
No nodes yet
Add your first server, service, website, domain or endpoint.
[Add First Node]
```

Templateها:

```text
Website
API
Mail Server
DNS Server
Database
Custom Node
```

Filter Empty:

```text
No nodes match the selected filters.
[Clear Filters]
```

Error:

```text
We couldn't load your nodes.
[Retry]
```

# 165. Accessibility و Responsive

- Badgeها علاوه بر رنگ، Icon و متن وضعیت داشته باشند.
- Tooltip با Keyboard قابل استفاده باشد.
- Row Focus State واضح باشد.
- در Mobile هر Row به Card تبدیل شود.
- Contrast حداقل WCAG AA باشد.
- RTL/LTR کامل باشد.
- Targetهای طولانی Ellipsis و Tooltip داشته باشند.
- Actionهای خطرناک Confirmation داشته باشند.

# 166. معیار پذیرش تصمیم Node-centric

- Node موجودیت اصلی UI کاربر باشد.
- Probe فقط در Admin Panel نمایش داده شود.
- Table View نمای پیش‌فرض Desktop باشد.
- Card View نمای پیش‌فرض Mobile باشد.
- نام و Target هر Node نمایش داده شود.
- Monitorهای فعال به شکل Badge رنگی و قابل کلیک نمایش داده شوند.
- Badge فقط به رنگ متکی نباشد.
- Overall Status مستقل نمایش داده شود.
- کلیک روی Node به Node Detail برود.
- کلیک روی Badge به Monitor Detail برود.
- Node Detail همه Monitorهای فعال را نشان دهد.
- نمودار Node Overview تجمیعی باشد.
- نمودارهای تخصصی فقط در Monitor Detail نمایش داده شوند.
- معماری برای Node Agent، CPU، RAM، Disk، Logs، Errors و Traces آماده باشد.
- ستون ثابت CPU/RAM در Node List ساخته نشود.
- Host Telemetry در لیست به شکل Signal تجمیعی نمایش داده شود.
- Search، Filter، Grouping، Bulk Action و Saved View پشتیبانی شوند.
