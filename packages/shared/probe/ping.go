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
		result.Metrics["packet_loss_percent"] = 100.0
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
