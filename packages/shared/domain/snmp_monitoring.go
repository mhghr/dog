package domain

import (
	"encoding/json"
	"time"
)

// SNMP monitoring domain model — the resource-level representation of a
// network device monitored via SNMP. The Monitor record (type "snmp") owns the
// connection configuration; discovery artefacts (interfaces, sensors) and the
// event stream are persisted separately so polling stays stateless and the
// UI can browse discovery + interface settings without re-walking the MIB.

// SNMPVersion is the SNMP protocol version used to talk to a device.
type SNMPVersion string

const (
	SNMPv1  SNMPVersion = "1"
	SNMPv2c SNMPVersion = "2c"
	SNMPv3  SNMPVersion = "3"
)

// SNMPDeviceConfig is the normalized, probe-side representation of the SNMP
// monitor configuration. Secrets arrive already encrypted (AES-256-GCM) and
// are decrypted by the executor at poll time — never logged, never serialized
// back to clients, never stored in plaintext.
type SNMPDeviceConfig struct {
	Host           string      `json:"host"`
	Port           int         `json:"port"`
	Version        SNMPVersion `json:"version"`
	Transport      string      `json:"transport"`
	TimeoutSeconds int         `json:"timeout_seconds"`
	Retries        int         `json:"retries"`
	MaxRepetitions int         `json:"max_repetitions"`

	// Community is the AES-256-GCM encrypted v1/v2c community string.
	Community string `json:"community,omitempty"`

	// v3 security. Passphrases are AES-256-GCM encrypted values.
	Username             string            `json:"username,omitempty"`
	SecurityLevel        SNMPSecurityLevel `json:"security_level,omitempty"`
	AuthenticationProto  string            `json:"authentication_protocol,omitempty"`
	AuthenticationSecret string            `json:"authentication_secret,omitempty"`
	PrivacyProto         string            `json:"privacy_protocol,omitempty"`
	PrivacySecret        string            `json:"privacy_secret,omitempty"`
	ContextName          string            `json:"context_name,omitempty"`

	// Interfaces is the per-interface monitoring policy applied on top of
	// discovered interfaces (which interfaces to monitor/ignore, display
	// names and utilization thresholds).
	Interfaces []SNMPInterfaceSettings `json:"interfaces,omitempty"`

	// MonitoredInterfaceIDs restricts polling to the given ifIndex values
	// (empty = poll every discovered, non-ignored interface). Avoids walking
	// tables on devices where only a subset matters.
	MonitoredInterfaceIDs []int `json:"monitored_interface_ids,omitempty"`
}

// SNMPInterfaceSettings is the per-interface monitoring policy. A discovered
// interface can be ignored (excluded from alerts and polling), its display
// name customized, and its utilization thresholds overridden.
type SNMPInterfaceSettings struct {
	IfIndex      int      `json:"if_index"`
	IfName       string   `json:"if_name,omitempty"`
	DisplayName  string   `json:"display_name,omitempty"`
	Ignore       bool     `json:"ignore,omitempty"`
	Monitor      bool     `json:"monitor"`
	UtilizationW *float64 `json:"utilization_warning,omitempty"`
	UtilizationC *float64 `json:"utilization_critical,omitempty"`
	// OperDownCritical triggers a critical health state when the interface is
	// administratively up but operationally down.
	OperDownCritical *bool `json:"oper_down_critical,omitempty"`
}

// SNMPFailureState is the deterministic failure taxonomy for a poll attempt.
// A successful collection where individual OIDs failed is Partial; the device
// is only marked unreachable/critical when the SNMP session itself cannot be
// established.
type SNMPFailureState string

const (
	SNMPStateSuccess           SNMPFailureState = "success"
	SNMPStatePartial           SNMPFailureState = "partial"
	SNMPStateDeviceUnreachable SNMPFailureState = "device_unreachable"
	SNMPStateTimeout           SNMPFailureState = "snmp_timeout"
	SNMPStateAuthentication    SNMPFailureState = "authentication_failed"
	SNMPStateAuthorization     SNMPFailureState = "authorization_failed"
	SNMPStateInvalidOID        SNMPFailureState = "invalid_oid"
	SNMPStateUnsupported       SNMPFailureState = "unsupported_oid"
	SNMPStateInvalidConfig     SNMPFailureState = "invalid_config"
	SNMPStateInternalError     SNMPFailureState = "internal_error"
	SNMPStateNoResponse        SNMPFailureState = "no_response"
)

// SNMPDeviceIdentity is the generic device identity captured by discovery.
type SNMPDeviceIdentity struct {
	SysName      string `json:"sys_name"`
	SysDescr     string `json:"sys_descr"`
	SysObjectID  string `json:"sys_object_id"`
	SysUpTime    string `json:"sys_uptime"`
	SysLocation  string `json:"sys_location,omitempty"`
	Vendor       string `json:"vendor"`
	Model        string `json:"model"`
	SerialNumber string `json:"serial_number,omitempty"`
	OS           string `json:"os,omitempty"`
	Firmware     string `json:"firmware,omitempty"`
	MACAddress   string `json:"mac_address,omitempty"`
	IPAddress    string `json:"ip_address,omitempty"`
}

// SNMPInterfaceInfo is one discovered interface (IF-MIB). Counters are stored
// as raw values; rates (in_bps/out_bps, utilization) are computed by the
// metric processor from counter deltas.
type SNMPInterfaceInfo struct {
	IfIndex       int    `json:"if_index"`
	IfName        string `json:"if_name,omitempty"`
	IfDescr       string `json:"if_descr,omitempty"`
	IfAlias       string `json:"if_alias,omitempty"`
	IfType        int    `json:"if_type"`
	IfMtu         int    `json:"if_mtu"`
	IfSpeed       int64  `json:"if_speed_bps"`
	IfAdminStatus int    `json:"if_admin_status"` // 1=up 2=down 3=testing
	IfOperStatus  int    `json:"if_oper_status"`  // 1=up 2=down 3=testing 4=unknown...
	IfInOctets    uint64 `json:"if_in_octets"`
	IfOutOctets   uint64 `json:"if_out_octets"`
	IfInPackets   uint64 `json:"if_in_packets"`
	IfOutPackets  uint64 `json:"if_out_packets"`
	IfInErrors    uint64 `json:"if_in_errors"`
	IfOutErrors   uint64 `json:"if_out_errors"`
	IfInDiscards  uint64 `json:"if_in_discards"`
	IfOutDiscards uint64 `json:"if_out_discards"`
	LastChange    string `json:"last_change,omitempty"`
	// High capacity flags record whether 64-bit counters were available.
	Has64BitIn  bool `json:"has_64_bit_in"`
	Has64BitOut bool `json:"has_64_bit_out"`
}

// SNMPSensorInfo is a discovered environmental sensor (temperature, fan,
// power supply). Vendor-specific mappings (e.g. CISCO-ENVMON-MIB) are resolved
// by the registry's provider layer.
type SNMPSensorInfo struct {
	Name       string  `json:"name"`
	SensorType string  `json:"sensor_type"` // temperature | fan | power | health
	Value      float64 `json:"value"`
	Unit       string  `json:"unit"`
	Status     string  `json:"status,omitempty"` // ok | warning | critical | notPresent
}

// SNMPDiscoveryResult is the cached artefact of a full discovery run. It is
// refreshed on enrollment, on config change, and on manual rediscovery — never
// on every poll.
type SNMPDiscoveryResult struct {
	Device       SNMPDeviceIdentity  `json:"device"`
	Interfaces   []SNMPInterfaceInfo `json:"interfaces"`
	Sensors      []SNMPSensorInfo    `json:"sensors"`
	HardwareOK   bool                `json:"hardware_ok"`
	DiscoveredAt time.Time           `json:"discovered_at"`
}

// SNMPEventKind distinguishes trap-derived events from poll-derived state
// changes.
type SNMPEventKind string

const (
	SNMPEventTrap      SNMPEventKind = "trap"
	SNMPEventPollState SNMPEventKind = "poll_state"
	SNMPEventDiscovery SNMPEventKind = "discovery"
)

// SNMPEvent is a normalized SNMP event (trap or poll state change) bound to a
// resource and, when relevant, to an interface.
type SNMPEvent struct {
	ID          string        `json:"id"`
	WorkspaceID string        `json:"workspace_id,omitempty"`
	ResourceID  string        `json:"resource_id"`
	MonitorID   string        `json:"monitor_id,omitempty"`
	ProbeID     string        `json:"probe_id,omitempty"`
	Kind        SNMPEventKind `json:"kind"`
	// EventType is the normalized trap type (linkDown, linkUp,
	// authenticationFailure, coldStart, warmStart, ...) or poll failure code.
	EventType   string            `json:"event_type"`
	Severity    string            `json:"severity"` // info | warning | critical
	Source      string            `json:"source"`
	Summary     string            `json:"summary"`
	InterfaceID string            `json:"interface_id,omitempty"`
	IfIndex     int               `json:"if_index,omitempty"`
	IfName      string            `json:"if_name,omitempty"`
	Details     map[string]string `json:"details,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// SNMPInterfaceRow is the persisted per-interface policy + last-known state.
// It backs the device detail page and lets users enable/disable/ignore
// interfaces, rename them, and override utilization thresholds.
type SNMPInterfaceRow struct {
	ID                    string     `json:"id"`
	MonitorID             string     `json:"monitor_id"`
	IfIndex               int        `json:"if_index"`
	IfName                string     `json:"if_name,omitempty"`
	IfDescr               string     `json:"if_descr,omitempty"`
	IfAlias               string     `json:"if_alias,omitempty"`
	DisplayName           string     `json:"display_name,omitempty"`
	Ignore                bool       `json:"ignore"`
	Monitor               bool       `json:"monitor"`
	UtilizationWarning    *float64   `json:"utilization_warning,omitempty"`
	UtilizationCritical   *float64   `json:"utilization_critical,omitempty"`
	OperDownCritical      *bool      `json:"oper_down_critical,omitempty"`
	LastOperStatus        *int       `json:"last_oper_status,omitempty"`
	LastInBps             *float64   `json:"last_in_bps,omitempty"`
	LastOutBps            *float64   `json:"last_out_bps,omitempty"`
	LastUtilizationPercent *float64  `json:"last_utilization_percent,omitempty"`
	LastCheckAt           *time.Time `json:"last_check_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// SNMPTaskKind distinguishes the on-demand SNMP operations executed by the
// collector (through the worker in NATS mode, inline otherwise).
type SNMPTaskKind string

const (
	SNMPTaskTest      SNMPTaskKind = "test"
	SNMPTaskDiscovery SNMPTaskKind = "discovery"
)

// SNMPTaskStatus tracks an on-demand SNMP task through its lifecycle.
type SNMPTaskStatus string

const (
	SNMPTaskPending  SNMPTaskStatus = "pending"
	SNMPTaskRunning  SNMPTaskStatus = "running"
	SNMPTaskSuccess  SNMPTaskStatus = "success"
	SNMPTaskFailed   SNMPTaskStatus = "failed"
)

// SNMPTask is an on-demand operation (test connection / discovery) that runs
// on the SNMP collector. In NATS mode the task is published to the worker;
// the worker executes it and publishes the result back. Inline fallback runs
// the identical collector code inside the API when no bus is configured.
type SNMPTask struct {
	TaskID       string          `json:"task_id"`
	WorkspaceID  string          `json:"workspace_id,omitempty"`
	ResourceID   string          `json:"resource_id"`
	MonitorID    string          `json:"monitor_id,omitempty"`
	Kind         SNMPTaskKind    `json:"kind"`
	// Config carries the monitor configuration (encrypted secrets included)
	// so the worker can execute without DB access.
	Config       map[string]any  `json:"config,omitempty"`
	ReplySubject string          `json:"reply_subject,omitempty"`
	Status       SNMPTaskStatus  `json:"status"`
	Result       json.RawMessage `json:"result,omitempty"`
	Error        string          `json:"error,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	FinishedAt   *time.Time      `json:"finished_at,omitempty"`
}

// SNMPTaskResult is the normalized outcome of a test or discovery task.
type SNMPTaskResult struct {
	OK          bool   `json:"ok"`
	State       string `json:"state"`
	Kind        string `json:"kind"`
	Detail      string `json:"detail,omitempty"`
	// Test fields.
	SysName    string `json:"sys_name,omitempty"`
	SysDescr   string `json:"sys_descr,omitempty"`
	SysObjectID string `json:"sys_object_id,omitempty"`
	Uptime     string `json:"uptime,omitempty"`
	// Discovery fields (present for discovery tasks).
	Discovery json.RawMessage `json:"discovery,omitempty"`
	// Diagnostics.
	DurationMillis int64    `json:"duration_millis,omitempty"`
	Steps          []string `json:"steps,omitempty"`
}
