package probe

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"monitoring-platform/packages/shared/domain"
)

type TCPExecutor struct {
	deps Deps
}

func NewTCPExecutor(deps Deps) *TCPExecutor {
	return &TCPExecutor{deps: deps}
}

func (e *TCPExecutor) Type() domain.MonitorType {
	return domain.MonitorTCP
}

func (e *TCPExecutor) Execute(ctx context.Context, job domain.ProbeJob) domain.ProbeResult {
	result := newBaseResult(job)

	target := job.Target
	if _, _, err := net.SplitHostPort(target); err != nil {
		port := intConfig(job.Config, "port", 0)
		if port < 1 || port > 65535 {
			return finishFailure(
				result,
				"invalid_target",
				fmt.Errorf("TCP port is required via target host:port or config.port"),
			)
		}

		target = net.JoinHostPort(target, strconv.Itoa(port))
	}

	connection, err := e.deps.Guard.DialContext(ctx, "tcp", target)
	if err != nil {
		return finishFailure(result, "tcp_connect_failed", err)
	}
	defer connection.Close()

	result.Attributes["remote_address"] = connection.RemoteAddr().String()
	result = finishSuccess(result)
	result.Metrics["connect_duration_ms"] = result.DurationMillis

	return result
}
