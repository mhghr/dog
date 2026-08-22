package trap

import (
	"fmt"
	"strings"

	"monitoring-platform/packages/shared/domain"
)

// Cisco enterprise-specific trap OIDs (CISCO-SYSLOG-MIB, CISCO-ENVMON-MIB,
// CISCO-CONFIG-MAN-MIB, CISCO-MEMORY-POOL-MIB).
const (
	ciscoSyslogMsgGenerated      = "1.3.6.1.4.1.9.9.41.1.2.7"
	ciscoConfigManEvent          = "1.3.6.1.4.1.9.9.43.2.0.1"
	ciscoEnvMonTemperatureNotify = "1.3.6.1.4.1.9.9.13.3.0.1"
	ciscoEnvMonFanNotify         = "1.3.6.1.4.1.9.9.13.3.0.2"
	ciscoEnvMonSupplyNotify      = "1.3.6.1.4.1.9.9.13.3.0.3"
	ciscoMemoryPoolMiscThreshold = "1.3.6.1.4.1.9.9.48.2.0.1"
)

// CiscoVendorNormalizer maps well-known Cisco enterprise traps to normalized
// events. Unknown Cisco traps fall through to the generic enterprise event.
type CiscoVendorNormalizer struct{}

var _ VendorNormalizer = CiscoVendorNormalizer{}

// Normalize implements VendorNormalizer.
func (c CiscoVendorNormalizer) Normalize(trap RawTrap) (domain.SNMPEvent, bool) {
	event := domain.SNMPEvent{
		Kind:     domain.SNMPEventTrap,
		Severity: "warning",
		Source:   trap.AgentAddress,
		Details:  map[string]string{},
	}

	switch trap.Enterprise {
	case ciscoSyslogMsgGenerated:
		event.EventType = "cisco.syslog"
		event.Summary = "device syslog message"
		event.Severity = "info"
		for _, v := range trap.Variables {
			if strings.Contains(v.OID, "clogMessage") {
				event.Summary = fmt.Sprintf("syslog: %s", toStr(v.Value))
			}
		}
		return event, true

	case ciscoConfigManEvent:
		event.EventType = "cisco.configChange"
		event.Summary = "device configuration changed"
		event.Severity = "info"
		return event, true

	case ciscoEnvMonTemperatureNotify:
		event.EventType = "cisco.envMon.temperature"
		event.Summary = "environmental temperature alarm"
		event.Severity = "critical"
		return event, true

	case ciscoEnvMonFanNotify:
		event.EventType = "cisco.envMon.fan"
		event.Summary = "environmental fan alarm"
		event.Severity = "critical"
		return event, true

	case ciscoEnvMonSupplyNotify:
		event.EventType = "cisco.envMon.powerSupply"
		event.Summary = "environmental power supply alarm"
		event.Severity = "critical"
		return event, true

	case ciscoMemoryPoolMiscThreshold:
		event.EventType = "cisco.memoryPool.threshold"
		event.Summary = "memory pool threshold reached"
		event.Severity = "warning"
		return event, true

	default:
		return domain.SNMPEvent{}, false
	}
}

// DefaultVendors returns the built-in vendor normalizers.
func DefaultVendors() []VendorNormalizer {
	return []VendorNormalizer{CiscoVendorNormalizer{}}
}
