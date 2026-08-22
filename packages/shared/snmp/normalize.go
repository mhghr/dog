package snmp

import (
	"fmt"
	"math"
	"strings"

	"monitoring-platform/packages/shared/domain"
)

// Interface operational status values (IF-MIB ifOperStatus).
const (
	OperUp         = 1
	OperDown       = 2
	OperTesting    = 3
	OperUnknown    = 4
	OperDormant    = 5
	OperNotPresent = 6
	OperLowerDown  = 7
)

// OperStatusText maps an ifOperStatus value to a stable label.
func OperStatusText(status int) string {
	switch status {
	case OperUp:
		return "up"
	case OperDown:
		return "down"
	case OperTesting:
		return "testing"
	case OperUnknown:
		return "unknown"
	case OperDormant:
		return "dormant"
	case OperNotPresent:
		return "notPresent"
	case OperLowerDown:
		return "lowerLayerDown"
	default:
		return "unknown"
	}
}

// Interface admin status values (IF-MIB ifAdminStatus).
const (
	AdminUp      = 1
	AdminDown    = 2
	AdminTesting = 3
)

// AdminStatusText maps an ifAdminStatus value to a stable label.
func AdminStatusText(status int) string {
	switch status {
	case AdminUp:
		return "up"
	case AdminDown:
		return "down"
	case AdminTesting:
		return "testing"
	default:
		return "unknown"
	}
}

// Utilization computes link utilization percent from in/out bit rates and the
// interface speed. Speed below 1 yields 0 (no meaningful utilization).
func Utilization(inBps, outBps float64, speedBps int64) float64 {
	if speedBps <= 0 {
		return 0
	}
	return math.Min(100, ((inBps+outBps)/float64(speedBps))*100)
}

// MemoryPercent derives RAM utilization percent from used/total bytes.
func MemoryPercent(usedBytes, totalBytes uint64) float64 {
	if totalBytes == 0 {
		return 0
	}
	return math.Min(100, (float64(usedBytes)/float64(totalBytes))*100)
}

// interfaceMetricKey builds the flat, controlled metric key for a per-interface
// value, e.g. "if_3_in_bps". The interface index is encoded in the name rather
// than a free-form label to keep VM cardinality bounded and controlled.
func interfaceMetricKey(ifIndex int, suffix string) string {
	return fmt.Sprintf("if_%d_%s", ifIndex, suffix)
}

// InterfaceSuffixes maps the normalized metric names (interface.*) to the flat
// per-interface suffix used in probe results. Names are stable and OID-free.
var InterfaceSuffixes = map[string]string{
	"interface.oper_status":         "oper_status",
	"interface.admin_status":        "admin_status",
	"interface.in_octets":           "in_octets",
	"interface.out_octets":          "out_octets",
	"interface.in_bps":              "in_bps",
	"interface.out_bps":             "out_bps",
	"interface.in_packets":          "in_packets",
	"interface.out_packets":         "out_packets",
	"interface.in_errors":           "in_errors",
	"interface.out_errors":          "out_errors",
	"interface.in_discards":         "in_discards",
	"interface.out_discards":        "out_discards",
	"interface.utilization_percent": "utilization_percent",
	"interface.speed_bps":           "speed_bps",
}

// LabelKeys returns the controlled label set allowed on interface series.
func LabelKeys() []string {
	return []string{"interface_index", "interface_name", "interface_alias"}
}

// InterfaceLabels builds the controlled label set for an interface. Values are
// truncated to keep cardinality bounded; alias/name are bounded lengths.
func InterfaceLabels(info domain.SNMPInterfaceInfo) map[string]string {
	return map[string]string{
		"interface_index": fmt.Sprintf("%d", info.IfIndex),
		"interface_name":  truncateLabel(info.IfName, 64),
		"interface_alias": truncateLabel(info.IfAlias, 128),
	}
}

func truncateLabel(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) > max {
		return value[:max]
	}
	return value
}

// isIgnoredInterface decides whether a discovered interface is excluded from
// monitoring according to the per-interface policy.
func isIgnoredInterface(settings []domain.SNMPInterfaceSettings, ifIndex int, name, descr, alias string, operStatus int) bool {
	for _, s := range settings {
		if s.IfIndex == ifIndex {
			return s.Ignore || !s.Monitor
		}
	}
	// Default policy: ignore loopbacks and administratively-down interfaces
	// unless the user explicitly chose to monitor them.
	lower := strings.ToLower(name + " " + descr + " " + alias)
	if strings.Contains(lower, "loopback") {
		return true
	}
	if operStatus == OperDown {
		return true
	}
	return false
}

// monitoringSettingsFor returns the per-interface settings for an index, or a
// zero-value default.
func monitoringSettingsFor(settings []domain.SNMPInterfaceSettings, ifIndex int) domain.SNMPInterfaceSettings {
	for _, s := range settings {
		if s.IfIndex == ifIndex {
			return s
		}
	}
	return domain.SNMPInterfaceSettings{IfIndex: ifIndex, Monitor: true}
}
