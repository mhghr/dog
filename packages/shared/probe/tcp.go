package probe

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/security"
)

// TCPExecutor verifies reachability of a host:port by establishing a real
// TCP connection. A check only succeeds when the connection is actually
// established — no banner reads, no pooling (each check is an independent
// connection by design, mirroring how a client would connect).
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

	host, port, err := parseTCPTarget(job)
	if err != nil {
		result.Metrics["reachability"] = 0.0
		result.Attributes["error_type"] = "invalid_target"
		return finishFailure(result, "invalid_target", err)
	}

	family := security.ParseIPFamily(stringConfig(job.Config, "ip_version", string(security.IPFamilyAuto)))
	guard := e.deps.Guard.WithIPFamily(family)

	if timeout := intConfig(job.Config, "timeout_ms", 0); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
		defer cancel()
	}

	result.Attributes["target"] = host
	result.Attributes["port"] = port
	result.Attributes["protocol"] = "tcp"
	result.Attributes["ip_version"] = string(family)

	connectStartedAt := time.Now()
	connection, err := guard.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return finishTCPFailure(result, err)
	}
	defer connection.Close()

	result.Metrics["reachability"] = 1.0
	result.Metrics["connect_time_ms"] = float64(time.Since(connectStartedAt).Milliseconds())
	result.Attributes["remote_address"] = connection.RemoteAddr().String()
	if remoteIP, _, splitErr := net.SplitHostPort(connection.RemoteAddr().String()); splitErr == nil {
		result.Attributes["resolved_ip"] = remoteIP
		result.Attributes["ip_family"] = ipFamilyOf(remoteIP)
	}

	return finishSuccess(result)
}

// parseTCPTarget resolves the target host and port. The resource target may
// already carry a port (host:port); otherwise the port comes from the
// monitor configuration, which is authoritative for TCP port checks.
func parseTCPTarget(job domain.ProbeJob) (string, int, error) {
	if host, portRaw, err := net.SplitHostPort(job.Target); err == nil {
		port, err := parsePort(portRaw)
		if err != nil {
			return "", 0, err
		}
		return host, port, nil
	}

	host := job.Target
	if configuredHost := stringConfig(job.Config, "host", ""); configuredHost != "" {
		host = configuredHost
	}
	if host == "" {
		return "", 0, fmt.Errorf("TCP target host is required")
	}

	port, err := parsePort(strconv.Itoa(intConfig(job.Config, "port", 0)))
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}

func parsePort(raw string) (int, error) {
	port, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("TCP port must be an integer")
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("TCP port must be between 1 and 65535")
	}
	return port, nil
}

// ipFamilyOf reports the address family of a textual IP address.
func ipFamilyOf(address string) string {
	if net.ParseIP(address).To4() != nil {
		return "ipv4"
	}
	return "ipv6"
}

// finishTCPFailure classifies a guarded dial failure into the deterministic
// error taxonomy. Reachability is always 0; connect time is never fabricated
// for a connection that never completed.
func finishTCPFailure(result domain.ProbeResult, err error) domain.ProbeResult {
	result.Metrics["reachability"] = 0.0

	code := classifyDialError(err)
	if isBlockedError(err) {
		code = "blocked_target"
	}
	result.Attributes["error_type"] = code

	return finishFailure(result, code, err)
}
