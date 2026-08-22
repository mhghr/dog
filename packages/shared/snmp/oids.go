// Package snmp implements SNMP network-device collection for the Dog
// monitoring platform. It is a collector, not a simple check: the package
// discovers devices, polls interfaces and system sensors through a
// vendor-aware OID/MIB registry, computes rates from raw counters, and
// normalizes everything into stable metric names independent of OIDs.
package snmp

import (
	"strings"
)

// Standard MIB OIDs (SNMPv2-MIB, IF-MIB, IFX-MIB, HOST-RESOURCES-MIB).
const (
	oidSysDescr    = "1.3.6.1.2.1.1.1"
	oidSysObjectID = "1.3.6.1.2.1.1.2"
	oidSysUpTime   = "1.3.6.1.2.1.1.3"
	oidSysName     = "1.3.6.1.2.1.1.5"
	oidSysLocation = "1.3.6.1.2.1.1.6"
	oidSysServices = "1.3.6.1.2.1.1.7"

	// IF-MIB (1.3.6.1.2.1.2.2.1.x)
	oidIfNumber       = "1.3.6.1.2.1.2.1.0"
	oidIfTable        = "1.3.6.1.2.1.2.2.1"
	oidIfIndex        = oidIfTable + ".1"
	oidIfDescr        = oidIfTable + ".2"
	oidIfType         = oidIfTable + ".3"
	oidIfMtu          = oidIfTable + ".4"
	oidIfSpeed        = oidIfTable + ".5"
	oidIfPhysAddress  = oidIfTable + ".6"
	oidIfAdminStatus  = oidIfTable + ".7"
	oidIfOperStatus   = oidIfTable + ".8"
	oidIfLastChange   = oidIfTable + ".9"
	oidIfInOctets     = oidIfTable + ".10"
	oidIfInUcastPkts  = oidIfTable + ".11"
	oidIfInDiscards   = oidIfTable + ".13"
	oidIfInErrors     = oidIfTable + ".14"
	oidIfOutOctets    = oidIfTable + ".16"
	oidIfOutUcastPkts = oidIfTable + ".17"
	oidIfOutDiscards  = oidIfTable + ".19"
	oidIfOutErrors    = oidIfTable + ".20"

	// IFX-MIB (1.3.6.1.2.1.31.1.1.1.x) — 64-bit high-capacity counters.
	oidIfXTable         = "1.3.6.1.2.1.31.1.1.1"
	oidIfName           = oidIfXTable + ".1"
	oidIfHCInOctets     = oidIfXTable + ".6"
	oidIfHCInUcastPkts  = oidIfXTable + ".7"
	oidIfHCOutOctets    = oidIfXTable + ".10"
	oidIfHCOutUcastPkts = oidIfXTable + ".11"
	oidIfHighSpeed      = oidIfXTable + ".15"
	oidIfAlias          = oidIfXTable + ".18"

	// HOST-RESOURCES-MIB.
	oidHRSystemUptime  = "1.3.6.1.2.1.25.1.1"
	oidHRMemorySize    = "1.3.6.1.2.1.25.2.2.0"
	oidHRStorageIndex  = "1.3.6.1.2.1.25.2.3.1.1"
	oidHRStorageType   = "1.3.6.1.2.1.25.2.3.1.2"
	oidHRStorageDescr  = "1.3.6.1.2.1.25.2.3.1.3"
	oidHRStorageAlloc  = "1.3.6.1.2.1.25.2.3.1.4"
	oidHRStorageSize   = "1.3.6.1.2.1.25.2.3.1.5"
	oidHRStorageUsed   = "1.3.6.1.2.1.25.2.3.1.6"
	oidHRProcessorLoad = "1.3.6.1.2.1.25.3.3.1.2"

	// hrStorageType value for RAM.
	hrStorageRAM = "1.3.6.1.2.1.25.2.1.7"
)

// Cisco enterprise MIB OIDs.
const (
	ciscoEnterpriseRoot = "1.3.6.1.4.1.9"
	// Cisco MIBs live under ciscoMgmt (1.3.6.1.4.1.9.9).
	ciscoMgmtRoot = ciscoEnterpriseRoot + ".9"

	// CISCO-ENVMON-MIB.
	oidCiscoEnvMonTemperature  = ciscoMgmtRoot + ".13.1.3.1.3"
	oidCiscoEnvMonTempStatus   = ciscoMgmtRoot + ".13.1.3.1.5"
	oidCiscoEnvMonFanStatus    = ciscoMgmtRoot + ".13.1.4.1.2"
	oidCiscoEnvMonFanState     = ciscoMgmtRoot + ".13.1.4.1.3"
	oidCiscoEnvMonSupplyStatus = ciscoMgmtRoot + ".13.1.5.1.2"
	oidCiscoEnvMonSupplyState  = ciscoMgmtRoot + ".13.1.5.1.3"

	// CISCO-PROCESS-MIB — per-CPU utilization over 5s window.
	oidCpmCPUTotal5secRev = ciscoMgmtRoot + ".109.1.1.1.1.6"
	oidCpmCPUTotal1minRev = ciscoMgmtRoot + ".109.1.1.1.1.5"

	// CISCO-MEMORY-POOL-MIB.
	oidCiscoMemoryPoolName = ciscoMgmtRoot + ".48.1.1.1.2"
	oidCiscoMemoryPoolUsed = ciscoMgmtRoot + ".48.1.1.1.5"
	oidCiscoMemoryPoolFree = ciscoMgmtRoot + ".48.1.1.1.6"

	// CISCO-SYSTEM-MIB — hardware inventory serial number.
	oidCiscoChassisSerial = ciscoMgmtRoot + ".1.1.1.1.1"
)

// DataType describes how an OID value must be interpreted.
type DataType string

const (
	DataTypeGauge     DataType = "gauge"
	DataTypeCounter32 DataType = "counter32"
	DataTypeCounter64 DataType = "counter64"
	DataTypeInt       DataType = "int"
	DataTypeString    DataType = "string"
	DataTypeTicks     DataType = "ticks"
)

// MetricKind distinguishes a scalar from a column in a table.
type MetricKind string

const (
	KindScalar MetricKind = "scalar"
	KindColumn MetricKind = "column"
)

// MetricDef maps one normalized metric name to an OID. The normalized name is
// stable and OID-independent — this is the contract the UI, health engine and
// VictoriaMetrics import all rely on.
type MetricDef struct {
	Metric   string `json:"metric"`
	OID      string `json:"oid"`
	MIB      string `json:"mib"`
	Kind     MetricKind
	DataType DataType
	Unit     string   `json:"unit"`
	Labels   []string `json:"labels,omitempty"`
}

// SensorDef describes an environmental sensor row to walk during discovery.
type SensorDef struct {
	OID        string
	SensorType string // temperature | fan | power
	StatusOID  string
	Unit       string
}

// Provider is a vendor-aware mapping layer. Core MIBs (IF-MIB, IP-MIB,
// SNMPv2-MIB, HOST-RESOURCES-MIB) live in the core provider; vendor-specific
// sensors and system metrics are added by vendor providers (Cisco, Juniper,
// MikroTik, Huawei, HPE, Fortinet, ...).
type Provider interface {
	Name() string
	Detect(sysObjectID string) bool
	// SystemScalars returns scalar system metrics to poll (CPU, memory, ...).
	SystemScalars() []MetricDef
	// SensorTables returns environmental sensor table definitions to walk.
	SensorTables() []SensorDef
	// RAMIndicator returns the hrStorageType value (or OID) that identifies
	// physical RAM in HOST-RESOURCES-MIB. Empty string means "use core RAM".
	RAMIndicator() string
}

// coreProvider implements the MIB-independent baseline: system identity,
// HOST-RESOURCES CPU/memory, and IF-MIB/IFX-MIB interface columns.
type coreProvider struct{}

func (coreProvider) Name() string         { return "core" }
func (coreProvider) Detect(_ string) bool { return true }

func (coreProvider) SystemScalars() []MetricDef {
	return []MetricDef{
		{Metric: "device.cpu_percent", OID: oidHRProcessorLoad, MIB: "HOST-RESOURCES-MIB", Kind: KindColumn, DataType: DataTypeGauge, Unit: "%"},
		{Metric: "device.uptime_seconds", OID: oidHRSystemUptime, MIB: "HOST-RESOURCES-MIB", Kind: KindScalar, DataType: DataTypeTicks, Unit: "s"},
	}
}

func (coreProvider) SensorTables() []SensorDef { return nil }

func (coreProvider) RAMIndicator() string { return "" }

// ciscoProvider maps Cisco routers/switches. Detected by the Cisco enterprise
// OID prefix 1.3.6.1.4.1.9.
type ciscoProvider struct{}

func (ciscoProvider) Name() string { return "cisco" }

func (ciscoProvider) Detect(sysObjectID string) bool {
	return strings.HasPrefix(sysObjectID, ciscoEnterpriseRoot)
}

func (ciscoProvider) SystemScalars() []MetricDef {
	return []MetricDef{
		{Metric: "device.cpu_percent", OID: oidCpmCPUTotal5secRev, MIB: "CISCO-PROCESS-MIB", Kind: KindColumn, DataType: DataTypeGauge, Unit: "%"},
		{Metric: "device.cpu_percent_1min", OID: oidCpmCPUTotal1minRev, MIB: "CISCO-PROCESS-MIB", Kind: KindColumn, DataType: DataTypeGauge, Unit: "%"},
		{Metric: "device.memory_used_bytes", OID: oidCiscoMemoryPoolUsed, MIB: "CISCO-MEMORY-POOL-MIB", Kind: KindColumn, DataType: DataTypeGauge, Unit: "bytes"},
		{Metric: "device.memory_free_bytes", OID: oidCiscoMemoryPoolFree, MIB: "CISCO-MEMORY-POOL-MIB", Kind: KindColumn, DataType: DataTypeGauge, Unit: "bytes"},
	}
}

func (ciscoProvider) SensorTables() []SensorDef {
	return []SensorDef{
		{OID: oidCiscoEnvMonTemperature, StatusOID: oidCiscoEnvMonTempStatus, SensorType: "temperature", Unit: "celsius"},
		{OID: oidCiscoEnvMonFanStatus, StatusOID: oidCiscoEnvMonFanState, SensorType: "fan", Unit: ""},
		{OID: oidCiscoEnvMonSupplyStatus, StatusOID: oidCiscoEnvMonSupplyState, SensorType: "power", Unit: ""},
	}
}

func (ciscoProvider) RAMIndicator() string { return "cisco-memory-pool" }

// InterfaceColumns returns the interface table columns to poll, in a stable
// order. 64-bit columns are preferred and marked as such; the executor falls
// back to 32-bit counters when high-capacity columns are absent.
var InterfaceColumns = []MetricDef{
	{Metric: "interface.oper_status", OID: oidIfOperStatus, MIB: "IF-MIB", Kind: KindColumn, DataType: DataTypeInt},
	{Metric: "interface.admin_status", OID: oidIfAdminStatus, MIB: "IF-MIB", Kind: KindColumn, DataType: DataTypeInt},
	{Metric: "interface.speed_bps", OID: oidIfSpeed, MIB: "IF-MIB", Kind: KindColumn, DataType: DataTypeGauge, Unit: "bps"},
	{Metric: "interface.in_octets", OID: oidIfHCInOctets, MIB: "IFX-MIB", Kind: KindColumn, DataType: DataTypeCounter64, Unit: "bytes"},
	{Metric: "interface.out_octets", OID: oidIfHCOutOctets, MIB: "IFX-MIB", Kind: KindColumn, DataType: DataTypeCounter64, Unit: "bytes"},
	{Metric: "interface.in_packets", OID: oidIfHCInUcastPkts, MIB: "IFX-MIB", Kind: KindColumn, DataType: DataTypeCounter64, Unit: "packets"},
	{Metric: "interface.out_packets", OID: oidIfHCOutUcastPkts, MIB: "IFX-MIB", Kind: KindColumn, DataType: DataTypeCounter64, Unit: "packets"},
	{Metric: "interface.in_errors", OID: oidIfInErrors, MIB: "IF-MIB", Kind: KindColumn, DataType: DataTypeCounter32, Unit: "errors"},
	{Metric: "interface.out_errors", OID: oidIfOutErrors, MIB: "IF-MIB", Kind: KindColumn, DataType: DataTypeCounter32, Unit: "errors"},
	{Metric: "interface.in_discards", OID: oidIfInDiscards, MIB: "IF-MIB", Kind: KindColumn, DataType: DataTypeCounter32, Unit: "packets"},
	{Metric: "interface.out_discards", OID: oidIfOutDiscards, MIB: "IF-MIB", Kind: KindColumn, DataType: DataTypeCounter32, Unit: "packets"},
}

// LegacyInterfaceColumns are the 32-bit fallback columns used when IFX-MIB
// high-capacity counters are not available on the device.
var LegacyInterfaceColumns = []MetricDef{
	{Metric: "interface.in_octets", OID: oidIfInOctets, MIB: "IF-MIB", Kind: KindColumn, DataType: DataTypeCounter32, Unit: "bytes"},
	{Metric: "interface.out_octets", OID: oidIfOutOctets, MIB: "IF-MIB", Kind: KindColumn, DataType: DataTypeCounter32, Unit: "bytes"},
	{Metric: "interface.in_packets", OID: oidIfInUcastPkts, MIB: "IF-MIB", Kind: KindColumn, DataType: DataTypeCounter32, Unit: "packets"},
	{Metric: "interface.out_packets", OID: oidIfOutUcastPkts, MIB: "IF-MIB", Kind: KindColumn, DataType: DataTypeCounter32, Unit: "packets"},
}

// MetaColumns are non-metric columns walked once per interface to build the
// interface identity (name, description, alias, type, MTU, speed).
var MetaColumns = []MetricDef{
	{Metric: "interface.name", OID: oidIfName, MIB: "IFX-MIB", Kind: KindColumn, DataType: DataTypeString},
	{Metric: "interface.descr", OID: oidIfDescr, MIB: "IF-MIB", Kind: KindColumn, DataType: DataTypeString},
	{Metric: "interface.alias", OID: oidIfAlias, MIB: "IFX-MIB", Kind: KindColumn, DataType: DataTypeString},
	{Metric: "interface.type", OID: oidIfType, MIB: "IF-MIB", Kind: KindColumn, DataType: DataTypeInt},
	{Metric: "interface.mtu", OID: oidIfMtu, MIB: "IF-MIB", Kind: KindColumn, DataType: DataTypeGauge},
	{Metric: "interface.speed_bps", OID: oidIfSpeed, MIB: "IF-MIB", Kind: KindColumn, DataType: DataTypeGauge, Unit: "bps"},
	{Metric: "interface.high_speed", OID: oidIfHighSpeed, MIB: "IFX-MIB", Kind: KindColumn, DataType: DataTypeGauge, Unit: "Mbps"},
}

// Registry is the vendor-aware OID mapping layer. The core provider is the
// fallback; vendor providers are consulted first in registration order.
type Registry struct {
	providers []Provider
}

// NewRegistry builds a registry with the given vendor providers plus the core
// provider appended as the fallback.
func NewRegistry(vendors ...Provider) *Registry {
	return &Registry{
		providers: append(append([]Provider{}, vendors...), coreProvider{}),
	}
}

// DefaultRegistry returns the registry with the built-in Cisco provider.
func DefaultRegistry() *Registry {
	return NewRegistry(ciscoProvider{})
}

// ProviderFor selects the first vendor provider that detects the device,
// falling back to the core provider.
func (r *Registry) ProviderFor(sysObjectID string) Provider {
	// The core provider is always the last entry and must only be returned as
	// a fallback so vendor providers get first say.
	for i := 0; i < len(r.providers)-1; i++ {
		if r.providers[i].Detect(sysObjectID) {
			return r.providers[i]
		}
	}
	if len(r.providers) == 0 {
		return coreProvider{}
	}
	return r.providers[len(r.providers)-1]
}
