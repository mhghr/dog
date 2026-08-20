package probe

import (
	"context"
	"errors"
	"net"
	"strings"
	"syscall"

	"monitoring-platform/packages/shared/security"
)

// isBlockedError reports whether the Guard rejected the destination as
// private, loopback, link-local, or otherwise non-routable address space.
func isBlockedError(err error) bool {
	var blocked *security.BlockedTargetError
	return errors.As(err, &blocked)
}

// classifyDialError maps a guarded dial failure to a deterministic,
// machine-readable error code. Every executor that opens its own connection
// (TCP, TLS) funnels its dial failures through this function so the health
// rules and the UI can distinguish the common failure modes.
func classifyDialError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		switch {
		case errors.Is(err, syscall.ECONNREFUSED):
			return "connection_refused"
		case errors.Is(err, syscall.ENETUNREACH), errors.Is(err, syscall.EHOSTUNREACH):
			return "network_unreachable"
		case errors.Is(err, syscall.ETIMEDOUT):
			return "timeout"
		case errors.Is(err, syscall.EADDRNOTAVAIL):
			return "network_unreachable"
		}
	}

	if isDNSError(err) {
		return "dns_resolution_failed"
	}

	// String fallbacks cover wrapped errors whose sentinel checks above miss
	// (e.g. a dial error flattened into an opaque message).
	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "refused the network connection"),
		strings.Contains(lower, "actively refused"):
		return "connection_refused"
	case strings.Contains(lower, "network is unreachable"),
		strings.Contains(lower, "no route to host"),
		strings.Contains(lower, "host is unreachable"):
		return "network_unreachable"
	case strings.Contains(lower, "i/o timeout"), strings.Contains(lower, "timed out"):
		return "timeout"
	}

	return "connection_failed"
}

// isDNSError reports whether the wrapped error is a DNS resolution failure
// (either a *net.DNSError or the Guard's own resolution wrapper).
func isDNSError(err error) bool {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "dns resolution failed")
}
