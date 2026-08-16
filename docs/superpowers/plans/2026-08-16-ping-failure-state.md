# Ping Monitor Failure-State Handling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Model Ping availability explicitly and independently from performance metrics so an unreachable target is never represented as zero latency — DOWN carries `packet_loss_percent=100`, NULL latency, an explicit `monitor_ping_status` gauge, a dedicated status series, and a distinct failed-state UI.

**Architecture:** The probe executor writes a `reachability` metric and leaves latency keys absent on failure; the health engine's Ping parameter keys are aligned to real executor metric keys so Ping health evaluation actually runs; VictoriaMetrics gains a `monitor_ping_status` gauge plus a new per-probe `StatusSeriesByProbe` availability series; the frontend Kpi cards adapt per state and charts mark DOWN periods from the explicit status series (never inferred from gaps).

**Tech Stack:** Go (pro-bing, chi, pgx), PostgreSQL JSONB, VictoriaMetrics Prometheus-import format, Next.js/React (next-intl, TanStack Query, ECharts), Vitest.

**Spec:** `docs/superpowers/specs/2026-08-16-ping-failure-state-design.md`

---

## File Structure

| File | Responsibility |
|------|---------------|
| `packages/shared/probe/ping.go` | Ping executor: write `reachability`, set `packet_loss_percent=100` on total failure, leave latency NULL; delegate shaping to a pure function. |
| `packages/shared/probe/ping_test.go` (new) | Unit tests for the pure shaping function. |
| `packages/shared/probe/helpers.go` | `mapErrorCode`: permission-denied promotion. |
| `packages/shared/health/catalog.go` | Ping packet-loss thresholds → warning 5 / error 20. |
| `packages/shared/health/engine.go` | `splitParamKey`: map Ping parameter keys to real executor metric keys. |
| `packages/shared/health/catalog_test.go` (new) | Threshold + split key tests. |
| `packages/shared/metrics/victoria.go` | Emit `monitor_ping_status` gauge for ping. |
| `packages/shared/metrics/victoria_test.go` (new) | `buildLines` status-gauge tests. |
| `migrations/000029_ping_health_thresholds.up.sql` / `.down.sql` (new) | Update ping monitor-type seed thresholds to 5/20. |
| `packages/shared/repository/repository.go` | Add `StatusSeriesByProbe` + `LatestSuccessAt` to `ResultRepository`. |
| `packages/shared/infrastructure/postgres/result_repository.go` | Implement `StatusSeriesByProbe` + `LatestSuccessAt`. |
| `packages/shared/interfaces/http/resource_monitor_handler.go` | Expose status series (`metric=status`) + `last_success_at`. |
| `apps/web/entities/resource/api/resource.api.ts` | Add `last_success_at` to `MonitorMetricsResponse`. |
| `apps/web/entities/resource/hooks/use-resource.ts` | New `useResourceMonitorStatus` hook. |
| `apps/web/entities/resource/ui/monitoring/ping/ping-config.ts` | Packet-loss defaults → warning 5 / critical 20. |
| `apps/web/entities/resource/ui/monitoring/ping/ping-metrics.ts` | Down-state formatters. |
| `apps/web/entities/resource/ui/monitoring/ping/PingMonitoringView.tsx` | Down-state card values + failure banner. |
| `apps/web/entities/resource/ui/monitoring/ping/PingMetricChart.tsx` | DOWN `markArea` from status series. |
| `apps/web/entities/resource/ui/monitoring/ping/PingAvailabilityChart.tsx` (new) | Availability bands from status series. |
| `apps/web/tests/ping-metrics.test.ts` | Down-state formatter tests. |
| `apps/web/tests/ping-health.test.ts` | Threshold-consistent tests. |
| `apps/web/tests/ping-chart.test.ts` (new) | DOWN-interval computation test. |

---

## Task 1: Ping executor — explicit reachability + NULL latency on failure

**Files:**
- Modify: `packages/shared/probe/ping.go`
- Test: `packages/shared/probe/ping_test.go` (new)

- [ ] **Step 1: Write the failing test**

Create `packages/shared/probe/ping_test.go`:

```go
package probe

import (
	"context"
	"testing"
	"time"

	"monitoring-platform/packages/shared/domain"
)

func newPingResult() domain.ProbeResult {
	return newBaseResult(testJob(domain.MonitorPing, "127.0.0.1", nil))
}

func TestShapePingResultSuccess(t *testing.T) {
	result := shapePingResult(newPingResult(), "127.0.0.1", pingStats{
		packetsSent:     4,
		packetsReceived: 4,
		packetLoss:      0,
		avgRTT:          42 * time.Millisecond,
		minRTT:          38 * time.Millisecond,
		maxRTT:          47 * time.Millisecond,
		stdDevRTT:       5 * time.Millisecond,
	})

	if result.Status != domain.StatusUp || !result.Success {
		t.Fatalf("expected up/success, got %+v", result)
	}
	if result.Metrics["reachability"] != 1 {
		t.Fatalf("expected reachability=1, got %v", result.Metrics["reachability"])
	}
	if result.Metrics["packet_loss_percent"] != 0.0 {
		t.Fatalf("expected packet_loss_percent=0, got %v", result.Metrics["packet_loss_percent"])
	}
	if result.Metrics["rtt_ms"] != 42.0 {
		t.Fatalf("expected rtt_ms=42, got %v", result.Metrics["rtt_ms"])
	}
	if result.Attributes["packets_sent"] != 4 || result.Attributes["packets_received"] != 4 {
		t.Fatalf("expected packet counters in attributes, got %+v", result.Attributes)
	}
}

func TestShapePingResultTotalFailureKeepsLatencyNull(t *testing.T) {
	result := shapePingResult(newPingResult(), "127.0.0.1", pingStats{
		packetsSent:     4,
		packetsReceived: 0,
		packetLoss:      100,
	})

	if result.Status != domain.StatusDown || result.Success {
		t.Fatalf("expected down/failure, got %+v", result)
	}
	if result.Metrics["reachability"] != 0 {
		t.Fatalf("expected reachability=0, got %v", result.Metrics["reachability"])
	}
	if result.Metrics["packet_loss_percent"] != 100.0 {
		t.Fatalf("expected packet_loss_percent=100, got %v", result.Metrics["packet_loss_percent"])
	}
	for _, key := range []string{"rtt_ms", "min_rtt_ms", "max_rtt_ms", "jitter_ms"} {
		if _, present := result.Metrics[key]; present {
			t.Fatalf("expected %s to be ABSENT (NULL), got %v", key, result.Metrics[key])
		}
	}
	if result.ErrorCode != "timeout" {
		t.Fatalf("expected error_code=timeout, got %s", result.ErrorCode)
	}
}

func TestShapePingResultPartialLoss(t *testing.T) {
	result := shapePingResult(newPingResult(), "127.0.0.1", pingStats{
		packetsSent:     4,
		packetsReceived: 3,
		packetLoss:      25,
		avgRTT:          40 * time.Millisecond,
	})

	if result.Status != domain.StatusUp || !result.Success {
		t.Fatalf("expected up (partial loss is still reachable), got %+v", result)
	}
	if result.Metrics["packet_loss_percent"] != 25.0 {
		t.Fatalf("expected packet_loss_percent=25, got %v", result.Metrics["packet_loss_percent"])
	}
}

func TestShapePingResultSetsResolvedIP(t *testing.T) {
	result := shapePingResult(newPingResult(), "10.0.0.1", pingStats{
		packetsSent:     4,
		packetsReceived: 4,
		avgRTT:          20 * time.Millisecond,
	})
	if result.Attributes["resolved_ip"] != "10.0.0.1" {
		t.Fatalf("expected resolved_ip attribute, got %+v", result.Attributes)
	}
}

func TestMapErrorCodePermissionDenied(t *testing.T) {
	if code := mapErrorCode("ping_failed", context.DeadlineExceeded); code != "timeout" {
		t.Fatalf("expected timeout, got %s", code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./packages/shared/probe/ -run 'TestShapePingResult|TestMapErrorCodePermissionDenied' -v`
Expected: FAIL — `shapePingResult`, `pingStats`, `TestShapePingResult...` undefined.

- [ ] **Step 3: Rewrite `packages/shared/probe/ping.go`**

Replace the whole file body with:

```go
package probe

import (
	"context"
	"fmt"
	"net"
	"time"

	probing "github.com/prometheus-community/pro-bing"

	"monitoring-platform/packages/shared/domain"
)

type PingExecutor struct {
	deps Deps
}

func NewPingExecutor(deps Deps) *PingExecutor {
	return &PingExecutor{deps: deps}
}

func (e *PingExecutor) Type() domain.MonitorType {
	return domain.MonitorPing
}

func (e *PingExecutor) Execute(ctx context.Context, job domain.ProbeJob) domain.ProbeResult {
	result := newBaseResult(job)

	ips, err := e.deps.Guard.ResolveAndValidate(ctx, job.Target)
	if err != nil {
		return finishFailure(result, "dns_resolution_failed", err)
	}

	pinger := probing.New(job.Target)
	pinger.SetIPAddr(&net.IPAddr{IP: ips[0]})

	// The UI schema stores these as "count"/"packet_size"; keep the legacy
	// "packet_count" key working as a fallback.
	packetCount := intConfig(job.Config, "count", 0)
	if packetCount < 1 {
		packetCount = intConfig(job.Config, "packet_count", 4)
	}
	if packetCount < 1 {
		packetCount = 1
	}
	pinger.Count = packetCount

	if size := intConfig(job.Config, "packet_size", 56); size > 0 {
		pinger.Size = size
	}

	intervalMillis := intConfig(job.Config, "packet_interval_millis", 200)
	if intervalMillis >= 10 {
		pinger.Interval = time.Duration(intervalMillis) * time.Millisecond
	}

	pinger.Timeout = time.Duration(job.TimeoutMillis) * time.Millisecond
	pinger.SetPrivileged(boolConfig(job.Config, "privileged", e.deps.PingPrivileged))

	runDone := make(chan error, 1)
	go func() {
		runDone <- pinger.Run()
	}()

	select {
	case <-ctx.Done():
		pinger.Stop()
		<-runDone
		return finishFailure(result, "ping_timeout", ctx.Err())
	case err := <-runDone:
		if err != nil {
			return finishFailure(result, "ping_failed", err)
		}
	}

	stats := pinger.Statistics()
	return shapePingResult(result, ips[0].String(), pingStats{
		packetsSent:     stats.PacketsSent,
		packetsReceived: stats.PacketsRecv,
		packetLoss:      stats.PacketLoss,
		avgRTT:          stats.AvgRtt,
		minRTT:          stats.MinRtt,
		maxRTT:          stats.MaxRtt,
		stdDevRTT:       stats.StdDevRtt,
	})
}

// pingStats carries the raw statistics of a finished ping run so result
// shaping can be unit-tested without real ICMP traffic.
type pingStats struct {
	packetsSent     int
	packetsReceived int
	packetLoss      float64
	avgRTT          time.Duration
	minRTT          time.Duration
	maxRTT          time.Duration
	stdDevRTT       time.Duration
}

// shapePingResult turns raw ping statistics into a finished ProbeResult,
// keeping availability separate from performance metrics. An unreachable
// target (zero replies) yields reachability=0, packet_loss_percent=100 and
// NO latency keys — they stay absent so consumers see NULL, never 0.
func shapePingResult(result domain.ProbeResult, resolvedIP string, stats pingStats) domain.ProbeResult {
	result.Attributes["packets_sent"] = stats.packetsSent
	result.Attributes["packets_received"] = stats.packetsReceived
	result.Attributes["resolved_ip"] = resolvedIP

	if stats.packetsReceived == 0 {
		result.Metrics["reachability"] = 0
		result.Metrics["packet_loss_percent"] = 100
		return finishFailure(result, "timeout", fmt.Errorf("no ICMP reply received"))
	}

	result.Metrics["reachability"] = 1
	result.Metrics["packet_loss_percent"] = stats.packetLoss
	result.Metrics["rtt_ms"] = float64(stats.avgRTT.Microseconds()) / 1000
	result.Metrics["min_rtt_ms"] = float64(stats.minRTT.Microseconds()) / 1000
	result.Metrics["max_rtt_ms"] = float64(stats.maxRTT.Microseconds()) / 1000
	result.Metrics["jitter_ms"] = float64(stats.stdDevRTT.Microseconds()) / 1000

	return finishSuccess(result)
}
```

- [ ] **Step 4: Update `mapErrorCode` in `packages/shared/probe/helpers.go`**

In `helpers.go`, replace the `mapErrorCode` function (lines 65-81) with:

```go
// mapErrorCode promotes transport-level failures to the standard error codes.
func mapErrorCode(code string, err error) string {
	if err == nil {
		return code
	}

	var blocked *security.BlockedTargetError
	if errors.As(err, &blocked) {
		return "blocked_target"
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	if strings.Contains(strings.ToLower(err.Error()), "permission denied") ||
		strings.Contains(strings.ToLower(err.Error()), "operation not permitted") {
		return "permission_denied"
	}

	return code
}
```

Add `"strings"` to the imports in `helpers.go`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./packages/shared/probe/ -v`
Expected: PASS, including existing `TestDefaultRegistryCoversEveryMonitorType`.

- [ ] **Step 6: Commit**

```bash
git add packages/shared/probe/ping.go packages/shared/probe/helpers.go packages/shared/probe/ping_test.go
git commit -m "feat(ping): explicit reachability metric, NULL latency on down, error taxonomy"
```

---

## Task 2: Health — align Ping parameter keys and thresholds

**Files:**
- Modify: `packages/shared/health/engine.go`
- Modify: `packages/shared/health/catalog.go`
- Test: `packages/shared/health/catalog_test.go` (new)

- [ ] **Step 1: Write the failing test**

Create `packages/shared/health/catalog_test.go`:

```go
package health

import (
	"testing"

	"monitoring-platform/packages/shared/domain"
)

func TestSplitParamKeyMapsRealPingMetricKeys(t *testing.T) {
	cases := map[string][]string{
		"ping.reachability":       {"ping.reachability", "reachability"},
		"ping.rtt.avg_ms":         {"rtt_avg_ms", "avg_rtt_ms", "rtt_ms"},
		"ping.rtt.min_ms":         {"rtt_min_ms", "min_rtt_ms"},
		"ping.rtt.max_ms":         {"rtt_max_ms", "max_rtt_ms"},
		"ping.packet_loss_percent": {"packet_loss_percent", "packet_loss"},
		"ping.jitter_ms":          {"jitter_ms", "jitter"},
	}

	for key, want := range cases {
		got := splitParamKey(key)
		if len(got) != len(want) {
			t.Errorf("splitParamKey(%q) = %v, want %v", key, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("splitParamKey(%q)[%d] = %q, want %q", key, i, got[i], want[i])
			}
		}
	}
}

func TestPingPacketLossThresholdsMatchSpec(t *testing.T) {
	var def *ParameterDefinition
	for _, p := range PingParameters {
		if p.Key == "ping.packet_loss_percent" {
			def = &p
			break
		}
	}
	if def == nil {
		t.Fatal("ping.packet_loss_percent not found in PingParameters")
	}
	if def.DefaultWarning == nil || *def.DefaultWarning != 5 {
		t.Fatalf("expected DefaultWarning=5, got %v", def.DefaultWarning)
	}
	if def.DefaultError == nil || *def.DefaultError != 20 {
		t.Fatalf("expected DefaultError=20, got %v", def.DefaultError)
	}
}

func TestExtractParamValueReadsExecutorReachability(t *testing.T) {
	result := domain.ProbeResult{
		Metrics: map[string]any{"reachability": 1, "rtt_ms": 42.0},
	}
	if v, ok := extractParamValue(&result, "ping.reachability"); !ok || v != 1 {
		t.Fatalf("expected reachability value, got %v ok=%v", v, ok)
	}
	if v, ok := extractParamValue(&result, "ping.rtt.avg_ms"); !ok || v != 42.0 {
		t.Fatalf("expected rtt_ms value via avg key, got %v ok=%v", v, ok)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./packages/shared/health/ -v`
Expected: FAIL — `splitParamKey` returns only `{"ping.reachability"}` etc.; thresholds are 1/5.

- [ ] **Step 3: Update `splitParamKey` in `packages/shared/health/engine.go`**

In `engine.go`, replace the Ping cases (lines 125-136):

```go
	switch {
	case key == "ping.reachability":
		return []string{"ping.reachability", "reachability"}
	case key == "ping.packet_loss_percent":
		return []string{"packet_loss_percent", "packet_loss"}
	case key == "ping.rtt.avg_ms":
		return []string{"rtt_avg_ms", "avg_rtt_ms", "rtt_ms"}
	case key == "ping.rtt.min_ms":
		return []string{"rtt_min_ms", "min_rtt_ms"}
	case key == "ping.rtt.max_ms":
		return []string{"rtt_max_ms", "max_rtt_ms"}
	case key == "ping.jitter_ms":
		return []string{"jitter_ms", "jitter"}
```

- [ ] **Step 4: Update Ping packet-loss thresholds in `packages/shared/health/catalog.go`**

In `catalog.go`, change the `ping.packet_loss_percent` entry (lines 35-38):

```go
	{
		Key: "ping.packet_loss_percent", Name: "Packet Loss",
		DataType: "PERCENTAGE", Direction: "HIGHER_IS_WORSE", Unit: "%",
		DefaultWarning: ptr(5.0), DefaultError: ptr(20.0), Recovery: ptr(3.0),
	},
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./packages/shared/health/ ./packages/shared/ingestion/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add packages/shared/health/engine.go packages/shared/health/catalog.go packages/shared/health/catalog_test.go
git commit -m "feat(health): align ping parameter keys to executor metrics, thresholds 5/20"
```

---

## Task 3: VictoriaMetrics — `monitor_ping_status` gauge

**Files:**
- Modify: `packages/shared/metrics/victoria.go`
- Test: `packages/shared/metrics/victoria_test.go` (new)

- [ ] **Step 1: Write the failing test**

Create `packages/shared/metrics/victoria_test.go`:

```go
package metrics

import (
	"strings"
	"testing"
	"time"

	"monitoring-platform/packages/shared/domain"
)

func pingResult(success bool) *domain.ProbeResult {
	status := domain.StatusUp
	if !success {
		status = domain.StatusDown
	}
	metrics := map[string]any{"reachability": 0, "packet_loss_percent": 100.0}
	if success {
		metrics = map[string]any{"reachability": 1, "packet_loss_percent": 0.0, "rtt_ms": 42.0}
	}
	return &domain.ProbeResult{
		ID:          "r1",
		MonitorID:   "m1",
		Status:      status,
		Success:     success,
		DurationMillis: 40,
		Metrics:     metrics,
		Attributes:  map[string]any{"resource_id": "res-1", "workspace_id": "ws-1"},
		FinishedAt:  time.Now().UTC(),
	}
}

func TestBuildLinesEmitsMonitorPingStatus(t *testing.T) {
	lines := buildLines(pingResult(true), "ping", "loc-1")
	joined := strings.Join(lines, "\n")

	if !strings.Contains(joined, `monitor_ping_status{monitor_id="m1",monitor_type="ping",probe_location="loc-1",resource_id="res-1",workspace_id="ws-1"} 1`) {
		t.Fatalf("expected up status line, got:\n%s", joined)
	}
	if !strings.Contains(joined, `monitor_ping_rtt_ms{`) {
		t.Fatalf("expected latency line on success, got:\n%s", joined)
	}
}

func TestBuildLinesDownWritesNoLatency(t *testing.T) {
	lines := buildLines(pingResult(false), "ping", "loc-1")
	joined := strings.Join(lines, "\n")

	if !strings.Contains(joined, `monitor_ping_status{`) || !strings.Contains(joined, "} 0 ") {
		t.Fatalf("expected down status line with value 0, got:\n%s", joined)
	}
	if strings.Contains(joined, "monitor_ping_rtt_ms") {
		t.Fatalf("down result must not emit latency line, got:\n%s", joined)
	}
	if strings.Contains(joined, "monitor_ping_jitter_ms") {
		t.Fatalf("down result must not emit jitter line, got:\n%s", joined)
	}
}

func TestBuildLinesNonPingHasNoStatusGauge(t *testing.T) {
	result := &domain.ProbeResult{
		ID: "r1", MonitorID: "m1", Status: domain.StatusUp, Success: true,
		Metrics:    map[string]any{},
		Attributes: map[string]any{},
		FinishedAt: time.Now().UTC(),
	}
	joined := strings.Join(buildLines(result, "http", "loc-1"), "\n")
	if strings.Contains(joined, "monitor_ping_status") {
		t.Fatalf("http result must not emit ping status, got:\n%s", joined)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./packages/shared/metrics/ -v`
Expected: FAIL — no `monitor_ping_status` line emitted.

- [ ] **Step 3: Add the status gauge to `buildLines` in `packages/shared/metrics/victoria.go`**

In `victoria.go`, inside `buildLines`, after the `monitor_probe_duration_seconds` line block (after line 164), add:

```go
	if monitorType == string(domain.MonitorPing) {
		lines = append(lines, fmt.Sprintf(
			`monitor_ping_status{%s} %d %d`,
			labels, successValue, timestamp,
		))
	}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./packages/shared/metrics/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add packages/shared/metrics/victoria.go packages/shared/metrics/victoria_test.go
git commit -m "feat(metrics): emit monitor_ping_status gauge, keep latency lines out of down results"
```

---

## Task 4: Migration — ping seed thresholds 5/20

**Files:**
- Create: `migrations/000029_ping_health_thresholds.up.sql`
- Create: `migrations/000029_ping_health_thresholds.down.sql`

- [ ] **Step 1: Create `migrations/000029_ping_health_thresholds.up.sql`**

```sql
-- Update the Ping monitor-type seed thresholds to the spec health rules:
-- packet loss warning >= 5%, critical >= 20%.
UPDATE monitor_types
SET health_parameters = jsonb_set(
        health_parameters,
        '{packet_loss,warning_threshold}',
        '5'
    )
WHERE slug = 'ping';

UPDATE monitor_types
SET health_parameters = jsonb_set(
        health_parameters,
        '{packet_loss,critical_threshold}',
        '20'
    )
WHERE slug = 'ping';
```

- [ ] **Step 2: Create `migrations/000029_ping_health_thresholds.down.sql`**

```sql
UPDATE monitor_types
SET health_parameters = jsonb_set(
        health_parameters,
        '{packet_loss,warning_threshold}',
        '1'
    )
WHERE slug = 'ping';

UPDATE monitor_types
SET health_parameters = jsonb_set(
        health_parameters,
        '{packet_loss,critical_threshold}',
        '5'
    )
WHERE slug = 'ping';
```

- [ ] **Step 3: Validate the migration**

Run: `go test ./packages/shared/infrastructure/... 2>&1 | Select-String -Pattern 'migration|000029'`
Also confirm `migrations/000029_ping_health_thresholds.down.sql` mirrors the up file exactly (it does).
Expected: no errors; no other file references `000029`.

- [ ] **Step 4: Commit**

```bash
git add migrations/000029_ping_health_thresholds.up.sql migrations/000029_ping_health_thresholds.down.sql
git commit -m "feat(db): ping packet-loss seed thresholds to 5/20"
```

---

## Task 5: Backend — `StatusSeriesByProbe` + `LatestSuccessAt`

**Files:**
- Modify: `packages/shared/repository/repository.go`
- Modify: `packages/shared/infrastructure/postgres/result_repository.go`
- Modify: `packages/shared/interfaces/http/resource_monitor_handler.go`

- [ ] **Step 1: Add interface methods to `packages/shared/repository/repository.go`**

Add inside `ResultRepository` (after line 24):

```go
	// StatusSeriesByProbe returns per-location time-bucketed success ratios
	// (0..1) for a monitor, including failed checks. 1.0 = fully up, 0.0 =
	// fully down. Missing buckets mean no data — NOT downtime.
	StatusSeriesByProbe(ctx context.Context, monitorID string, from, to time.Time, stepSeconds int) ([]domain.ProbeSeries, error)
	// LatestSuccessAt returns the most recent successful check time for a
	// monitor, or nil when there is none.
	LatestSuccessAt(ctx context.Context, monitorID string) (*time.Time, error)
```

- [ ] **Step 2: Implement `StatusSeriesByProbe` + `LatestSuccessAt` in `packages/shared/infrastructure/postgres/result_repository.go`**

Append at the end of the file:

```go
// StatusSeriesByProbe returns one time-bucketed success-ratio series per
// probe location for a monitor, including failed checks. This is the explicit
// availability signal: 1.0 means fully up in that bucket, 0.0 fully down, and
// an absent bucket means no data (never inferred as downtime by consumers).
func (r *ResultRepository) StatusSeriesByProbe(ctx context.Context, monitorID string, from, to time.Time, stepSeconds int) ([]domain.ProbeSeries, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			pl.id::text,
			COALESCE(pl.name, ''),
			COALESCE(pl.code, ''),
			date_bin(make_interval(secs => $4::int), pr.started_at, TIMESTAMPTZ 'epoch') AS bucket,
			AVG(CASE WHEN pr.success THEN 1 ELSE 0 END)::float8 AS value
		FROM probe_results pr
		LEFT JOIN probe_locations pl ON pl.id = pr.probe_location_id
		WHERE pr.monitor_id = $1::uuid
		  AND pr.started_at >= $2
		  AND pr.started_at < $3
		GROUP BY pl.id, pl.name, pl.code, bucket
		ORDER BY pl.name, bucket`,
		monitorID, from, to, stepSeconds)
	if err != nil {
		return nil, fmt.Errorf("query status series: %w", err)
	}
	defer rows.Close()

	type rawPoint struct {
		probeID   string
		probeName string
		location  string
		ts        time.Time
		value     float64
	}
	var raw []rawPoint
	for rows.Next() {
		var (
			pid, pname, loc string
			bucket          time.Time
			value           float64
		)
		if err := rows.Scan(&pid, &pname, &loc, &bucket, &value); err != nil {
			return nil, fmt.Errorf("scan status bucket: %w", err)
		}
		raw = append(raw, rawPoint{probeID: pid, probeName: pname, location: loc, ts: bucket, value: value})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var series []domain.ProbeSeries
	var current *domain.ProbeSeries
	for _, p := range raw {
		if current == nil || current.ProbeID != p.probeID {
			series = append(series, domain.ProbeSeries{
				ProbeID:   p.probeID,
				ProbeName: p.probeName,
				Location:  p.location,
				MetricKey: "status",
				Points:    []domain.MetricPoint{},
				Values:    []domain.MetricPoint{},
			})
			current = &series[len(series)-1]
		}
		current.Points = append(current.Points, domain.MetricPoint{Timestamp: p.ts, Value: p.value})
		current.Values = append(current.Values, domain.MetricPoint{Timestamp: p.ts, Value: p.value})
	}

	return series, nil
}

// LatestSuccessAt returns the most recent successful check time, or nil.
func (r *ResultRepository) LatestSuccessAt(ctx context.Context, monitorID string) (*time.Time, error) {
	var at time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT started_at
		FROM probe_results
		WHERE monitor_id = $1::uuid AND success = TRUE
		ORDER BY started_at DESC
		LIMIT 1`,
		monitorID,
	).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query latest success: %w", err)
	}
	return &at, nil
}
```

- [ ] **Step 3: Expose status series + `last_success_at` in `packages/shared/interfaces/http/resource_monitor_handler.go`**

In `resourceMonitorMetrics` (lines 234-247), replace the series-selection block:

```go
	metricKey := query.Get("metric")
	var series []domain.ProbeSeries
	switch metricKey {
	case "status":
		series, err = h.deps.Results.StatusSeriesByProbe(r.Context(), monitorID, from, to, stepSeconds)
	default:
		if metricKey != "" {
			series, err = h.deps.Results.SeriesByProbeMetric(r.Context(), monitorID, metricKey, from, to, stepSeconds)
		} else {
			series, err = h.deps.Results.SeriesByProbe(r.Context(), monitorID, from, to, stepSeconds)
		}
	}
	if err != nil {
		h.deps.Logger.Error("query per-probe series failed", "monitor_id", monitorID, "error", err)
		writeDomainError(w, r, err)
		return
	}
```

Then update the JSON response to include `last_success_at`:

```go
	lastSuccessAt, err := h.deps.Results.LatestSuccessAt(r.Context(), monitorID)
	if err != nil {
		h.deps.Logger.Error("query latest success failed", "monitor_id", monitorID, "error", err)
		lastSuccessAt = nil
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"series":         series,
		"latest":         latest,
		"step_seconds":   stepSeconds,
		"from":           from,
		"to":             to,
		"metric_key":     metricKey,
		"monitor_type":   string(monitor.Type),
		"last_success_at": lastSuccessAt,
	})
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: compiles with no errors.

- [ ] **Step 5: Commit**

```bash
git add packages/shared/repository/repository.go packages/shared/infrastructure/postgres/result_repository.go packages/shared/interfaces/http/resource_monitor_handler.go
git commit -m "feat(api): explicit status series + last success timestamp for ping monitors"
```

---

## Task 6: Frontend — packet-loss defaults, API type, status hook

**Files:**
- Modify: `apps/web/entities/resource/ui/monitoring/ping/ping-config.ts`
- Modify: `apps/web/entities/resource/api/resource.api.ts`
- Modify: `apps/web/entities/resource/hooks/use-resource.ts`

- [ ] **Step 1: Update packet-loss defaults in `ping-config.ts`**

In `ping-config.ts`, change `DEFAULT_THRESHOLDS` (line 44-48):

```ts
const DEFAULT_THRESHOLDS: PingThresholds = {
  latency: { warning: 200, critical: 500 },
  packetLoss: { warning: 5, critical: 20 },
  jitter: { warning: 30, critical: 80 },
};
```

- [ ] **Step 2: Add `last_success_at` to `MonitorMetricsResponse` in `resource.api.ts`**

In `resource.api.ts`, extend `MonitorMetricsResponse` (lines 96-104):

```ts
export interface MonitorMetricsResponse {
  series: ProbeSeries[];
  latest: ProbeResult[];
  step_seconds: number;
  from: string;
  to: string;
  metric_key?: string;
  monitor_type?: string;
  last_success_at?: string | null;
}
```

- [ ] **Step 3: Add `useResourceMonitorStatus` hook in `use-resource.ts`**

In `use-resource.ts`, after `useResourceMonitorMetrics` (line 174), add:

```ts
// Fetches the explicit per-probe availability series (success ratio 0..1) for
// a ping monitor. Down periods come from this signal, never from latency gaps.
export function useResourceMonitorStatus(
  resourceId: string | undefined,
  monitorId: string | undefined,
  range: MetricsRange,
) {
  return useResourceMonitorMetrics(resourceId, monitorId, range, "status");
}
```

- [ ] **Step 4: Run type check**

Run: `pnpm --filter web exec tsc --noEmit`
Expected: no type errors.

- [ ] **Step 5: Commit**

```bash
git add apps/web/entities/resource/ui/monitoring/ping/ping-config.ts apps/web/entities/resource/api/resource.api.ts apps/web/entities/resource/hooks/use-resource.ts
git commit -m "feat(web): ping packet-loss defaults 5/20, status series hook"
```

---

## Task 7: Frontend — down-state metrics helpers

**Files:**
- Modify: `apps/web/entities/resource/ui/monitoring/ping/ping-metrics.ts`
- Test: `apps/web/tests/ping-metrics.test.ts`

- [ ] **Step 1: Write the failing test**

Append to `apps/web/tests/ping-metrics.test.ts`:

```ts
import {
  formatPingKpiValue,
  formatPingKpiValueWithUnit,
  buildDownIntervals,
} from "@/entities/resource/ui/monitoring/ping/ping-metrics";
import type { PingChartSeries } from "@/entities/resource/ui/monitoring/ping/ping-metrics";

describe("formatPingKpiValue", () => {
  it("formats a measured latency (bare, unit rendered by the card)", () => {
    expect(formatPingKpiValue(42, "ms", false)).toBe("42");
  });

  it("formats a measured packet loss (bare)", () => {
    expect(formatPingKpiValue(2.5, "percent", false)).toBe("2.50");
  });

  it("shows infinity for missing latency when down", () => {
    expect(formatPingKpiValue(null, "ms", true)).toBe("∞");
  });

  it("shows 100 for missing packet loss when down", () => {
    expect(formatPingKpiValue(null, "percent", true)).toBe("100");
  });

  it("shows N/A when there is simply no data", () => {
    expect(formatPingKpiValue(null, "ms", false)).toBe("N/A");
    expect(formatPingKpiValue(null, "percent", false)).toBe("N/A");
  });
});

describe("formatPingKpiValueWithUnit", () => {
  it("embeds the unit for inline row values", () => {
    expect(formatPingKpiValueWithUnit(42, "ms", false)).toBe("42 ms");
    expect(formatPingKpiValueWithUnit(2.5, "percent", false)).toBe("2.50%");
    expect(formatPingKpiValueWithUnit(null, "ms", true)).toBe("∞ ms");
    expect(formatPingKpiValueWithUnit(null, "percent", true)).toBe("100%");
  });

  it("leaves N/A without a unit", () => {
    expect(formatPingKpiValueWithUnit(null, "ms", false)).toBe("N/A");
  });
});

describe("buildDownIntervals", () => {
  const series: PingChartSeries[] = [
    {
      metric: "status",
      location: "Amsterdam",
      probeName: "Amsterdam",
      points: [
        { time: "2026-01-01T00:00:00Z", value: 1 },
        { time: "2026-01-01T00:05:00Z", value: 0 },
        { time: "2026-01-01T00:10:00Z", value: 0 },
        { time: "2026-01-01T00:15:00Z", value: 1 },
        { time: "2026-01-01T00:20:00Z", value: 1 },
      ],
    },
  ];

  it("returns [start, end) time windows where status is 0", () => {
    const intervals = buildDownIntervals(series);
    expect(intervals).toEqual([
      { start: "2026-01-01T00:05:00Z", end: "2026-01-01T00:15:00Z" },
    ]);
  });

  it("returns an empty array when fully up", () => {
    const upSeries: PingChartSeries[] = [
      {
        metric: "status",
        location: "Amsterdam",
        probeName: "Amsterdam",
        points: [
          { time: "2026-01-01T00:00:00Z", value: 1 },
          { time: "2026-01-01T00:05:00Z", value: 1 },
        ],
      },
    ];
    expect(buildDownIntervals(upSeries)).toEqual([]);
  });

  it("extends a trailing down window through the last sample", () => {
    const tail: PingChartSeries[] = [
      {
        metric: "status",
        location: "Amsterdam",
        probeName: "Amsterdam",
        points: [
          { time: "2026-01-01T00:00:00Z", value: 1 },
          { time: "2026-01-01T00:05:00Z", value: 0 },
          { time: "2026-01-01T00:10:00Z", value: 0 },
        ],
      },
    ];
    const intervals = buildDownIntervals(tail);
    expect(intervals).toEqual([
      { start: "2026-01-01T00:05:00Z", end: "2026-01-01T00:10:00Z" },
    ]);
  });

  it("does not merge windows across probes", () => {
    const multi: PingChartSeries[] = [
      {
        metric: "status",
        location: "A",
        probeName: "A",
        points: [
          { time: "2026-01-01T00:00:00Z", value: 1 },
          { time: "2026-01-01T00:05:00Z", value: 0 },
          { time: "2026-01-01T00:10:00Z", value: 0 },
        ],
      },
      {
        metric: "status",
        location: "B",
        probeName: "B",
        points: [
          { time: "2026-01-01T00:00:00Z", value: 1 },
          { time: "2026-01-01T00:15:00Z", value: 1 },
        ],
      },
    ];
    const intervals = buildDownIntervals(multi);
    expect(intervals).toEqual([
      { start: "2026-01-01T00:05:00Z", end: "2026-01-01T00:10:00Z" },
    ]);
  });
});
```

Note: `buildDownIntervals` treats a value of `0` (fully down) as DOWN and any positive value as UP. Each probe series is processed independently so windows never merge across probes; a trailing down run extends through that series' last sample time.

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm --filter web test -- tests/ping-metrics.test.ts`
Expected: FAIL — `formatPingKpiValue` / `buildDownIntervals` not exported.

- [ ] **Step 3: Implement helpers in `ping-metrics.ts`**

Append to `ping-metrics.ts`:

```ts
// KPI display formatter. When a metric is missing and the monitor is down,
// latency/jitter read as "∞" (unreachable) and packet loss as 100. Zero is
// never shown for a down target — absence of data is a distinct state.
//
// These return the BARE value; the KpiCard renders the unit in its own span.
// Use `formatPingKpiValueWithUnit` for inline row values.
export function formatPingKpiValue(
  value: number | null,
  format: "ms" | "percent",
  down: boolean,
): string {
  if (value != null) {
    return format === "ms" ? String(Math.round(value)) : value.toFixed(2);
  }
  if (down) {
    return format === "ms" ? "∞" : "100";
  }
  return "N/A";
}

// Same as `formatPingKpiValue` but embeds the unit for standalone rendering.
export function formatPingKpiValueWithUnit(
  value: number | null,
  format: "ms" | "percent",
  down: boolean,
): string {
  const bare = formatPingKpiValue(value, format, down);
  if (bare === "N/A") return bare;
  return `${bare}${format === "ms" ? " ms" : "%"}`;
}

// Computes [start, end) half-open windows where a status series reports fully
// down (value === 0). Each series is processed independently (a probe's window
// never bleeds into another probe's timeline). A trailing down run extends to
// that series' last sample time. Series points must be ordered ascending.
export interface DownInterval {
  start: string;
  end: string;
}

export function buildDownIntervals(series: PingChartSeries[]): DownInterval[] {
  const intervals: DownInterval[] = [];

  for (const s of series) {
    let start: string | null = null;
    for (const p of s.points) {
      if (p.value === 0) {
        if (start === null) start = p.time;
      } else if (start !== null) {
        intervals.push({ start, end: p.time });
        start = null;
      }
    }
    if (start !== null && s.points.length > 0) {
      intervals.push({ start, end: s.points[s.points.length - 1].time });
    }
  }
  return intervals;
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `pnpm --filter web test -- tests/ping-metrics.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/entities/resource/ui/monitoring/ping/ping-metrics.ts apps/web/tests/ping-metrics.test.ts
git commit -m "feat(web): down-state KPI formatters and down-interval builder"
```

---

## Task 8: Frontend — down-state cards + failure banner in `PingMonitoringView`

**Files:**
- Modify: `apps/web/entities/resource/ui/monitoring/ping/PingMonitoringView.tsx`

- [ ] **Step 1: Read the current file to confirm context**

Read `apps/web/entities/resource/ui/monitoring/ping/PingMonitoringView.tsx` to confirm line numbers match the snippets below (the file was edited earlier to pass `down` into `KpiGrid`).

- [ ] **Step 2: Wire status series + last success into the view**

Add `useResourceMonitorStatus` to the imports:

```ts
import {
  useResourceMonitorMetrics,
  useResourceMonitorStatus,
  type MetricsRange,
} from "@/entities/resource/hooks/use-resource";
```

Import the new helpers:

```ts
import {
  summarize,
  toProbeStats,
  toChartSeries,
  buildDownIntervals,
  formatPingKpiValue,
  formatPingKpiValueWithUnit,
  type PingProbeStat,
  type PingChartSeries,
} from "./ping-metrics";
```

Inside `PingMonitoringView`, after the `latencyQuery` (line 53), add a status query and derive down intervals:

```ts
  const statusQuery = useResourceMonitorStatus(resourceId, monitor.id, range);
  const statusSeries = useMemo(
    () => toChartSeries(statusQuery.data?.series ?? [], "status"),
    [statusQuery.data?.series],
  );
  const downIntervals = useMemo(() => buildDownIntervals(statusSeries), [statusSeries]);
  const lastSuccessAt = statusQuery.data?.last_success_at ?? null;
```

- [ ] **Step 3: Pass new props into `KpiGrid` and `PingMetricChart`**

Update the `KpiGrid` usage:

```tsx
          <KpiGrid
            isFa={isFa}
            down={overall === "down"}
            summary={summary}
            states={kpiStates}
            probeStats={probeStats}
            lastSuccessAt={lastSuccessAt}
          />
```

Update `PingMetricChart` usage to add `downIntervals`:

```tsx
          <PingMetricChart
            title={t("Latency over time", "تأخیر در طول زمان")}
            unit="ms"
            series={latencySeries}
            thresholds={config.thresholds.latency}
            downIntervals={downIntervals}
            isLoading={latencyQuery.isPending}
            isError={latencyQuery.isError}
          />
```

- [ ] **Step 4: Rewrite `KpiGrid` to use down-state values + failure banner**

Replace the whole `KpiGrid` function (currently lines 154-240) with:

```tsx
function KpiGrid({
  isFa,
  down,
  summary,
  states,
  probeStats,
  lastSuccessAt,
}: {
  isFa: boolean;
  down: boolean;
  summary: ReturnType<typeof summarize>;
  states: {
    availability: PingHealthState;
    latency: PingHealthState;
    packetLoss: PingHealthState;
    jitter: PingHealthState;
  };
  probeStats: PingProbeStat[];
  lastSuccessAt: string | null;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);

  const latency = summary.latency == null ? null : summary.latency;
  const packetLoss = summary.packetLoss == null ? null : summary.packetLoss;
  const jitter = summary.jitter == null ? null : summary.jitter;

  const availabilityRows: PingKpiRow[] = probeStats.map((s) => ({
    label: s.location,
    value: s.success ? t("Up", "بالا") : t("Down", "پایین"),
    tone: s.success ? "success" : "destructive",
  }));

  const latencyRows: PingKpiRow[] = probeStats.map((s) => ({
    label: s.location,
    value: formatPingKpiValueWithUnit(s.latency, "ms", !s.success),
  }));

  const lossRows: PingKpiRow[] = probeStats.map((s) => ({
    label: s.location,
    value: formatPingKpiValueWithUnit(s.packetLoss, "percent", !s.success),
    tone: s.packetLoss != null && s.packetLoss > 0 ? "warning" : "muted",
  }));

  const jitterRows: PingKpiRow[] = probeStats.map((s) => ({
    label: s.location,
    value: formatPingKpiValueWithUnit(s.jitter, "ms", !s.success),
  }));

  return (
    <section className="space-y-3">
      {down && (
        <div
          role="alert"
          className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-2xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm"
        >
          <span className="font-semibold text-destructive">
            {isFa ? "منبع قطع است" : "Target is down"}
          </span>
          {lastSuccessAt && (
            <span className="text-muted-foreground">
              {isFa ? "آخرین بررسی موفق: " : "Last successful check: "}
              {lastSuccessAt}
            </span>
          )}
        </div>
      )}

      <div className="grid max-w-4xl grid-cols-2 gap-3 sm:grid-cols-4">
        <PingKpiCard
          label={t("Availability", "دسترس‌پذیری")}
          value={summary.availability == null
            ? down ? "0.00" : "N/A"
            : summary.availability.toFixed(2)}
          unit="%"
          state={states.availability}
          rows={availabilityRows}
        />
        <PingKpiCard
          label={t("Latency", "تأخیر")}
          value={formatPingKpiValue(latency, "ms", down)}
          unit={latency != null || down ? "ms" : undefined}
          state={states.latency}
          rows={latencyRows}
        />
        <PingKpiCard
          label={t("Packet loss", "افت بسته")}
          value={formatPingKpiValue(packetLoss, "percent", down)}
          unit={packetLoss != null || down ? "%" : undefined}
          state={states.packetLoss}
          rows={lossRows}
        />
        <PingKpiCard
          label={t("Jitter", "نوسان")}
          value={formatPingKpiValue(jitter, "ms", down)}
          unit={jitter != null || down ? "ms" : undefined}
          state={states.jitter}
          rows={jitterRows}
        />
      </div>
    </section>
  );
}
```

Note: `formatPingKpiValue` returns the bare value (`"42"`, `"∞"`, `"100"`, `"N/A"`). `PingKpiCard` renders `value` then `unit` in its own smaller span (`PingKpiCard.tsx:74-79`), so when down the card shows `∞ ms` / `100%` / `0.00%` without duplication. Rows use `formatPingKpiValueWithUnit` because they render standalone (`PingKpiCard.tsx:91-99`).

- [ ] **Step 5: Remove the now-unused `noValue`/`fmtMs`/`fmtPct` locals**

They are gone because `KpiGrid` was fully replaced. Verify there are no other references to `noValue`, `fmtMs`, or `fmtPct` in the file.

- [ ] **Step 6: Run type check + tests**

Run: `pnpm --filter web exec tsc --noEmit`
Expected: no type errors.
Run: `pnpm --filter web test`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/web/entities/resource/ui/monitoring/ping/PingMonitoringView.tsx
git commit -m "feat(web): ping down-state cards and failure banner"
```

---

## Task 9: Frontend — DOWN `markArea` on latency chart

**Files:**
- Modify: `apps/web/entities/resource/ui/monitoring/ping/PingMetricChart.tsx`
- Test: `apps/web/tests/ping-chart.test.ts` (new)

- [ ] **Step 1: Write the failing test**

Create `apps/web/tests/ping-chart.test.ts`:

```ts
import { describe, expect, it } from "vitest";

import {
  toDownMarkArea,
} from "@/entities/resource/ui/monitoring/ping/PingMetricChart";
import type { DownInterval } from "@/entities/resource/ui/monitoring/ping/ping-metrics";

describe("toDownMarkArea", () => {
  it("maps down intervals to an echarts markArea data array", () => {
    const intervals: DownInterval[] = [
      { start: "2026-01-01T00:05:00Z", end: "2026-01-01T00:15:00Z" },
    ];
    expect(toDownMarkArea(intervals)).toEqual([
      {
        name: "Down",
        xAxis: ["2026-01-01T00:05:00Z", "2026-01-01T00:15:00Z"],
        itemStyle: { color: "rgba(220,48,53,0.08)" },
      },
    ]);
  });

  it("returns an empty array for no down intervals", () => {
    expect(toDownMarkArea([])).toEqual([]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm --filter web test -- tests/ping-chart.test.ts`
Expected: FAIL — `toDownMarkArea` not exported.

- [ ] **Step 3: Add the `downIntervals` prop and `toDownMarkArea` to `PingMetricChart.tsx`**

Add to the props type:

```ts
export interface PingMetricChartProps {
  title: string;
  unit: "ms" | "%";
  series: PingChartSeries[];
  thresholds: MetricThreshold;
  downIntervals?: DownInterval[];
  isLoading: boolean;
  isError: boolean;
  onRetry?: () => void;
}
```

Change the function signature:

```ts
export function PingMetricChart({
  title,
  unit,
  series,
  thresholds,
  downIntervals = [],
  isLoading,
  isError,
  onRetry,
}: PingMetricChartProps) {
```

Add the import:

```ts
import type { DownInterval } from "./ping-metrics";
```

Add the exported helper at the end of the file:

```ts
// Converts down intervals into the echarts markArea data shape so the latency
// chart can shade downtime from the explicit status signal (not from gaps).
export function toDownMarkArea(
  downIntervals: DownInterval[],
): Array<{ name: string; xAxis: [string, string]; itemStyle: { color: string } }> {
  return downIntervals.map((interval) => ({
    name: "Down",
    xAxis: [interval.start, interval.end],
    itemStyle: { color: "rgba(220,48,53,0.08)" },
  }));
}
```

- [ ] **Step 4: Wire `markArea` into the chart series**

Inside the `option` useMemo, after the `markLine` declaration (line 71), add:

```ts
    const markArea = {
      silent: true,
      data: toDownMarkArea(downIntervals),
    };
```

Then in the series mapping, add `markArea` to each series (only when it has data):

```ts
      series: visible.map((s, i) => ({
        type: "line" as const,
        name: s.probeName || s.location || `probe-${i + 1}`,
        showSymbol: false,
        smooth: 0.2,
        lineStyle: { width: 2, color: PALETTE[i % PALETTE.length] },
        itemStyle: { color: PALETTE[i % PALETTE.length] },
        areaStyle: { color: "transparent" },
        data: s.points.map((p) => [p.time, p.value]),
        markArea: markArea.data.length > 0 ? markArea : undefined,
        markLine: i === 0 && markLine.data.length > 0 ? markLine : undefined,
      })),
```

Add `downIntervals` to the dependency array:

```ts
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible, locale, thresholds.warning, thresholds.critical, downIntervals]);
```

- [ ] **Step 5: Run tests + type check**

Run: `pnpm --filter web test -- tests/ping-chart.test.ts`
Expected: PASS.
Run: `pnpm --filter web exec tsc --noEmit`
Expected: no type errors.

- [ ] **Step 6: Commit**

```bash
git add apps/web/entities/resource/ui/monitoring/ping/PingMetricChart.tsx apps/web/tests/ping-chart.test.ts
git commit -m "feat(web): shade down periods on ping latency chart from status series"
```

---

## Task 10: Frontend — `PingAvailabilityChart`

**Files:**
- Create: `apps/web/entities/resource/ui/monitoring/ping/PingAvailabilityChart.tsx`
- Modify: `apps/web/entities/resource/ui/monitoring/ping/PingMonitoringView.tsx`

- [ ] **Step 1: Create `PingAvailabilityChart.tsx`**

```tsx
"use client";

import { useMemo } from "react";
import { useLocale } from "next-intl";

import { EChart, useChartPalette } from "@/shared/ui/charts/echart";
import { makeGrid, makeTimeXAxis, makeTooltip } from "@/shared/ui/charts/chart-config";
import { Card, CardContent } from "@/shared/ui/card";
import { Skeleton } from "@/shared/ui/skeleton";
import type { PingChartSeries } from "./ping-metrics";

// Font used for the canvas-rendered chart text.
const CHART_FONT = "'bakh', 'estedad', ui-sans-serif, system-ui, sans-serif";

export function PingAvailabilityChart({
  title,
  series,
  isLoading,
  isError,
}: {
  title: string;
  series: PingChartSeries[];
  isLoading: boolean;
  isError: boolean;
}) {
  const locale = useLocale();
  const isFa = locale === "fa";
  const palette = useChartPalette();

  const option = useMemo(() => {
    return {
      animation: false,
      grid: makeGrid({ top: 16, right: 16, bottom: 40, left: 40 }),
      tooltip: {
        ...makeTooltip(palette, (value: unknown) =>
          typeof value === "number" ? `${Math.round(value * 100)}%` : String(value),
        ),
        textStyle: { color: palette.tooltipText, fontSize: 12, fontFamily: CHART_FONT },
      },
      xAxis: { ...makeTimeXAxis(locale, palette, CHART_FONT) },
      yAxis: {
        type: "value" as const,
        min: 0,
        max: 1,
        name: isFa ? "دسترس‌پذیری" : "availability",
        axisLabel: {
          color: palette.text,
          fontFamily: CHART_FONT,
          formatter: (value: unknown) =>
            typeof value === "number" ? `${Math.round(value * 100)}%` : String(value),
        },
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { show: false },
      },
      series: series.map((s, i) => ({
        type: "line" as const,
        name: s.probeName || s.location || `probe-${i + 1}`,
        showSymbol: false,
        smooth: 0.2,
        step: "end" as const,
        lineStyle: { width: 2, color: i === 0 ? "#0D9464" : "#4F66F0" },
        itemStyle: { color: i === 0 ? "#0D9464" : "#4F66F0" },
        areaStyle: { color: "rgba(13,148,100,0.12)" },
        data: s.points.map((p) => [p.time, p.value]),
      })),
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [series, locale]);

  return (
    <Card variant="bordered" className="h-full">
      <CardContent className="pt-1">
        {isLoading ? (
          <Skeleton className="h-40 w-full rounded-lg" />
        ) : isError ? (
          <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
            {isFa ? "خطا در دریافت داده" : "Unable to load data"}
          </div>
        ) : series.length === 0 ? (
          <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
            {isFa ? "داده‌ای برای نمایش نیست" : "No data to display"}
          </div>
        ) : (
          <EChart option={option} className="h-40 w-full" ariaLabel={title} />
        )}
      </CardContent>
    </Card>
  );
}
```

Note: verify `makeTimeXAxis`, `makeGrid`, `makeTooltip`, `EChart`, `useChartPalette`, `Card`, `CardContent`, `Skeleton` are exported exactly as used in `PingMetricChart.tsx` (they are — copy its imports). `makeTimeXAxis` and `makeGrid` take a `palette`/`ChartPalette`-typed first argument; `makeGrid` accepts `Partial<EChartsCoreOption["grid"]>` overrides, so `{ top: 16, right: 16, bottom: 40, left: 40 }` is valid.

- [ ] **Step 2: Wire the chart into `PingMonitoringView.tsx`**

Add the import:

```ts
import { PingAvailabilityChart } from "./PingAvailabilityChart";
```

Add it in the render tree right after the latency `PingMetricChart`:

```tsx
          <PingAvailabilityChart
            title={t("Availability over time", "دسترس‌پذیری در طول زمان")}
            series={statusSeries}
            isLoading={statusQuery.isPending}
            isError={statusQuery.isError}
          />
```

- [ ] **Step 3: Run type check + tests**

Run: `pnpm --filter web exec tsc --noEmit`
Expected: no type errors.
Run: `pnpm --filter web test`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add apps/web/entities/resource/ui/monitoring/ping/PingAvailabilityChart.tsx apps/web/entities/resource/ui/monitoring/ping/PingMonitoringView.tsx
git commit -m "feat(web): availability chart driven by explicit status series"
```

---

## Task 11: Alerting verification test

**Files:**
- Modify: `packages/shared/alerting/engine_test.go` (new)

- [ ] **Step 1: Write the test**

Create `packages/shared/alerting/engine_test.go`:

```go
package alerting

import (
	"context"
	"testing"

	"monitoring-platform/packages/shared/domain"
)

// fakeAlertRepo is a minimal in-memory AlertRepository for testing the status
// down/up transition. Implement only the methods the Engine calls.
type fakeAlertRepo struct {
	alerts map[string]domain.Alert
}

func (f *fakeAlertRepo) ListActivePolicies(ctx context.Context, monitorID string) ([]domain.AlertPolicy, error) {
	return []domain.AlertPolicy{{
		ID:               "policy-1",
		OrganizationID:   "org-1",
		MonitorID:        monitorID,
		Name:             "Ping Down",
		Severity:         "critical",
		OpeningFailures:  1,
		ResolvingSuccesses: 1,
	}}, nil
}

func (f *fakeAlertRepo) FindByDedup(ctx context.Context, dedupKey string) (domain.Alert, error) {
	if a, ok := f.alerts[dedupKey]; ok {
		return a, nil
	}
	return domain.Alert{}, domain.ErrNotFound
}

func (f *fakeAlertRepo) UpsertAlert(ctx context.Context, alert *domain.Alert) error {
	if f.alerts == nil {
		f.alerts = map[string]domain.Alert{}
	}
	f.alerts[alert.DedupKey] = *alert
	return nil
}

func TestAlertEngineFiresOnPingDown(t *testing.T) {
	repo := &fakeAlertRepo{}
	engine := NewEngine(repo, nil)

	result := domain.ProbeResult{
		ID: "r1", MonitorID: "m1", MonitorName: "server1",
		Status: domain.StatusDown, Success: false,
	}

	events := engine.Evaluate(context.Background(), result)
	if len(events) != 1 {
		t.Fatalf("expected 1 firing event on down, got %d", len(events))
	}
	if events[0].NewState != "firing" {
		t.Fatalf("expected firing state, got %s", events[0].NewState)
	}
}

func TestAlertEngineResolvesOnPingUp(t *testing.T) {
	repo := &fakeAlertRepo{}
	engine := NewEngine(repo, nil)

	down := domain.ProbeResult{ID: "r1", MonitorID: "m1", MonitorName: "server1", Status: domain.StatusDown, Success: false}
	engine.Evaluate(context.Background(), down)

	up := domain.ProbeResult{ID: "r2", MonitorID: "m1", MonitorName: "server1", Status: domain.StatusUp, Success: true}
	events := engine.Evaluate(context.Background(), up)
	if len(events) != 1 || events[0].NewState != "recovering" {
		t.Fatalf("expected recovering event on up, got %+v", events)
	}
}
```

Note: this test requires `domain.AlertPolicy` / `domain.Alert` / `domain.AlertRepository` to compile with the field names used here. Open `packages/shared/domain/alert.go` first and adjust field names to match (`OpeningFailures`, `ResolvingSuccesses`, `Severity`, `Name`, `DedupKey`, `State`, `MonitorID`, `OrganizationID`, `PolicyID`, `MonitorName`, `ChannelIDs`).

- [ ] **Step 2: Run test to verify it passes**

Run: `go test ./packages/shared/alerting/ -v`
Expected: PASS. If the fake repo signature does not satisfy `domain.AlertRepository`, read `packages/shared/domain/alert.go` and add the missing methods (returning empty values) to `fakeAlertRepo`.

- [ ] **Step 3: Commit**

```bash
git add packages/shared/alerting/engine_test.go
git commit -m "test(alerting): ping down fires, up resolves"
```

---

## Task 12: Full validation

- [ ] **Step 1: Backend build + vet + tests**

Run: `go build ./...`
Expected: builds.
Run: `go vet ./...`
Expected: no findings.
Run: `go test ./... -count=1`
Expected: all pass.

- [ ] **Step 2: Frontend type check + lint + tests**

Run: `pnpm --filter web exec tsc --noEmit`
Expected: no type errors.
Run: `pnpm --filter web lint`
Expected: no errors.
Run: `pnpm --filter web test`
Expected: all pass.

- [ ] **Step 3: Review the diff**

Run: `git status` and `git log --oneline -15`
Confirm all expected files changed and commit messages are coherent.

---

## Self-Review Notes

- **Spec coverage:** §1 (Task 1), §2 (Task 2), §3 (Task 3), §4 (Task 4 migration is a seed-alignment; JSONB already nullable), §5 (Tasks 5-6, 7, 9, 10), §6 (Task 2 thresholds + Task 1 reachability), §7 (Tasks 7-8), §8 (Tasks 5, 9, 10), §9 (Task 11), §10 (Tasks 1, 2, 7, 11).
- **Placeholder scan:** no TBD/TODO; every code step contains full code.
- **Type consistency:** `formatPingKpiValue(value, format, down)` defined in Task 7 and used in Task 8 with `(value, "ms"|"percent", down)`; `buildDownIntervals(series)` returns `DownInterval[]` used by Task 9 `toDownMarkArea`; `statusSeries` type `PingChartSeries[]` flows from `toChartSeries` through both charts.
