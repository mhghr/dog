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
	result.Attributes["packets_sent"] = stats.PacketsSent
	result.Attributes["packets_received"] = stats.PacketsRecv
	result.Attributes["resolved_ip"] = ips[0].String()

	if stats.PacketsRecv == 0 {
		return finishFailure(result, "packet_loss_100", fmt.Errorf("no ICMP reply received"))
	}

	result.Metrics["packet_loss_percent"] = stats.PacketLoss
	result.Metrics["rtt_ms"] = float64(stats.AvgRtt.Microseconds()) / 1000
	result.Metrics["min_rtt_ms"] = float64(stats.MinRtt.Microseconds()) / 1000
	result.Metrics["max_rtt_ms"] = float64(stats.MaxRtt.Microseconds()) / 1000
	result.Metrics["jitter_ms"] = float64(stats.StdDevRtt.Microseconds()) / 1000

	return finishSuccess(result)
}
