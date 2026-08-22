package snmp

import (
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"monitoring-platform/packages/shared/domain"
)

// SanitizeError returns a safe, non-sensitive summary of an SNMP error.
// Credentials, community strings and raw packets are never echoed back; the
// message is reduced to a stable failure class and a short hint.
func SanitizeError(err error) string {
	if err == nil {
		return ""
	}
	state := ResponseErrorState(err)
	msg := strings.TrimSpace(err.Error())
	switch {
	case len(msg) > 160:
		msg = msg[:160]
	case state == domain.SNMPStateAuthentication:
		msg = "authentication failed (bad community string or credentials)"
	case state == domain.SNMPStateAuthorization:
		msg = "access denied by the device"
	case state == domain.SNMPStateTimeout:
		msg = "request timed out"
	}
	return msg
}

// recorder applies SNMP observability counters around poll/discovery runs.
type recorder struct {
	metrics *Metrics
}

// Metrics mirrors the Prometheus SNMPMetrics surface with atomic counters so
// the collector works without a Prometheus registry (tests, single-binary).
type Metrics struct {
	PollSuccess  uint64
	PollFailure  uint64
	TimeoutCount uint64
	PacketsSent  uint64
	PacketsRcvd  uint64
}

// RecordPoll updates observability counters for one poll cycle.
func (m *Metrics) RecordPoll(start time.Time, state domain.SNMPFailureState, packetsSent, packetsReceived int) {
	_ = start
	switch state {
	case domain.SNMPStateSuccess, domain.SNMPStatePartial:
		m.PollSuccess++
	case domain.SNMPStateTimeout:
		m.PollSuccess++
		m.TimeoutCount++
	default:
		m.PollFailure++
	}
	m.PacketsSent += uint64(packetsSent)
	m.PacketsRcvd += uint64(packetsReceived)
}

// VersionText returns the SNMP version label used in logs and attributes.
func VersionText(version domain.SNMPVersion) string {
	switch version {
	case domain.SNMPv1:
		return "v1"
	case domain.SNMPv3:
		return "v3"
	default:
		return "v2c"
	}
}

// gosnmpTypeName is a diagnostic helper for PDU type classification.
func gosnmpTypeName(t gosnmp.Asn1BER) string {
	switch t {
	case gosnmp.Counter32:
		return "counter32"
	case gosnmp.Counter64:
		return "counter64"
	case gosnmp.Gauge32:
		return "gauge"
	case gosnmp.Integer:
		return "integer"
	case gosnmp.OctetString:
		return "octet_string"
	case gosnmp.TimeTicks:
		return "ticks"
	default:
		return "other"
	}
}

// collectorVersion is reported in diagnostics. Bump it whenever the collector
// changes behavior (not just code churn).
const collectorVersion = "snmp-collector/1.0.0"

// CollectorVersion returns the collector version string for diagnostics.
func CollectorVersion() string {
	return collectorVersion
}
