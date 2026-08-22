package trap

import (
	"fmt"
	"strings"

	"monitoring-platform/packages/shared/domain"
)

// Standard trap OIDs (SNMPv2-MIB).
const (
	oidTrapColdStart          = "1.3.6.1.6.3.1.1.5.1"
	oidTrapWarmStart          = "1.3.6.1.6.3.1.1.5.2"
	oidTrapLinkDown           = "1.3.6.1.6.3.1.1.5.3"
	oidTrapLinkUp             = "1.3.6.1.6.3.1.1.5.4"
	oidTrapAuthenticationFail = "1.3.6.1.6.3.1.1.5.5"
	oidTrapEGPNeighborLoss    = "1.3.6.1.6.3.1.1.5.6"

	// Interface varbind OIDs inside linkDown/linkUp traps.
	oidVarIfIndex       = "1.3.6.1.2.1.2.2.1.1"
	oidVarIfDescr       = "1.3.6.1.2.1.2.2.1.2"
	oidVarIfAdminStatus = "1.3.6.1.2.1.2.2.1.7"
	oidVarIfOperStatus  = "1.3.6.1.2.1.2.2.1.8"
)

// StandardNormalizer maps well-known traps (v1 generic + v2 SNMPv2-MIB) into
// normalized events. Enterprise-specific traps are passed to the registered
// vendor normalizers; unknown enterprises produce a generic enterprise event.
type StandardNormalizer struct {
	// Vendors is the extension point for vendor-specific trap mapping.
	Vendors []VendorNormalizer
}

// VendorNormalizer maps an enterprise-specific trap to a normalized event.
// It returns (event, true) when it recognizes the trap, (zero, false) when the
// trap is not this vendor's concern.
type VendorNormalizer interface {
	Normalize(trap RawTrap) (domain.SNMPEvent, bool)
}

var _ Normalizer = StandardNormalizer{}

// Normalize converts a raw trap into a normalized event.
func (s StandardNormalizer) Normalize(trap RawTrap) domain.SNMPEvent {
	event := domain.SNMPEvent{
		Kind:      domain.SNMPEventTrap,
		Severity:  "info",
		EventType: "unknown",
		Source:    trap.AgentAddress,
		Details:   map[string]string{},
	}

	if trap.GenericTrap > 0 && trap.GenericTrap < 7 && trap.Version == "v1" {
		return s.normalizeV1Generic(trap, event)
	}

	switch trap.Enterprise {
	case oidTrapColdStart:
		event.EventType = "coldStart"
		event.Summary = "device cold start"
		event.Severity = "info"
		return event
	case oidTrapWarmStart:
		event.EventType = "warmStart"
		event.Summary = "device warm start"
		event.Severity = "info"
		return event
	case oidTrapLinkDown:
		return s.normalizeLink(trap, event, "linkDown", "interface went down", "critical")
	case oidTrapLinkUp:
		return s.normalizeLink(trap, event, "linkUp", "interface came up", "info")
	case oidTrapAuthenticationFail:
		event.EventType = "authenticationFailure"
		event.Summary = "SNMP authentication failure"
		event.Severity = "critical"
		return event
	case oidTrapEGPNeighborLoss:
		event.EventType = "egpNeighborLoss"
		event.Summary = "EGP neighbor lost"
		event.Severity = "warning"
		return event
	}

	for _, vendor := range s.Vendors {
		if event, ok := vendor.Normalize(trap); ok {
			return event
		}
	}

	// Enterprise-specific, unrecognized — keep the OID and a generic label.
	event.EventType = "enterpriseSpecific"
	event.Summary = fmt.Sprintf("enterprise trap %s (%d)", trap.Enterprise, trap.SpecificTrap)
	event.Details["enterprise_oid"] = trap.Enterprise
	event.Details["specific_trap"] = fmt.Sprintf("%d", trap.SpecificTrap)
	event.Severity = "warning"
	return event
}

func (s StandardNormalizer) normalizeV1Generic(trap RawTrap, event domain.SNMPEvent) domain.SNMPEvent {
	switch trap.GenericTrap {
	case 0:
		event.EventType = "coldStart"
		event.Summary = "device cold start"
	case 1:
		event.EventType = "warmStart"
		event.Summary = "device warm start"
	case 2:
		return s.normalizeLink(trap, event, "linkDown", "interface went down", "critical")
	case 3:
		return s.normalizeLink(trap, event, "linkUp", "interface came up", "info")
	case 4:
		event.EventType = "authenticationFailure"
		event.Summary = "SNMP authentication failure"
		event.Severity = "critical"
	case 5:
		event.EventType = "egpNeighborLoss"
		event.Summary = "EGP neighbor lost"
		event.Severity = "warning"
	case 6:
		for _, vendor := range s.Vendors {
			if normalized, ok := vendor.Normalize(trap); ok {
				return normalized
			}
		}
		event.EventType = "enterpriseSpecific"
		event.Summary = fmt.Sprintf("enterprise trap %s (%d)", trap.Enterprise, trap.SpecificTrap)
		event.Severity = "warning"
	}
	return event
}

func (s StandardNormalizer) normalizeLink(trap RawTrap, event domain.SNMPEvent, eventType, summary, severity string) domain.SNMPEvent {
	event.EventType = eventType
	event.Summary = summary
	event.Severity = severity
	for _, v := range trap.Variables {
		switch {
		case strings.HasPrefix(v.OID, oidVarIfIndex):
			if idx, ok := toInt(v.Value); ok {
				event.IfIndex = idx
				event.Details["interface_index"] = fmt.Sprintf("%d", idx)
			}
		case strings.HasPrefix(v.OID, oidVarIfDescr):
			event.IfName = toStr(v.Value)
			event.Details["interface_descr"] = event.IfName
		case strings.HasPrefix(v.OID, oidVarIfOperStatus):
			event.Details["oper_status"] = fmt.Sprintf("%v", v.Value)
		case strings.HasPrefix(v.OID, oidVarIfAdminStatus):
			event.Details["admin_status"] = fmt.Sprintf("%v", v.Value)
		}
	}
	if event.IfIndex > 0 {
		event.Summary = fmt.Sprintf("%s (ifIndex %d)", summary, event.IfIndex)
	}
	return event
}

func toInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int32:
		return int(val), true
	case int64:
		return int(val), true
	case uint:
		return int(val), true
	case uint32:
		return int(val), true
	case uint64:
		return int(val), true
	default:
		return 0, false
	}
}

func toStr(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", v)
	}
}
