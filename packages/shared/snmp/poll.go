package snmp

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"monitoring-platform/packages/shared/domain"
)

// InterfaceSnapshot is the per-interface state collected in one poll cycle,
// including computed rates and utilization.
type InterfaceSnapshot struct {
	domain.SNMPInterfaceInfo
	InBps       float64 `json:"in_bps"`
	OutBps      float64 `json:"out_bps"`
	Utilization float64 `json:"utilization_percent"`
}

// PollOptions configures a single poll cycle.
type PollOptions struct {
	Params   ConnectParams
	Registry *Registry
	// Discovery carries the cached interface identity (names/aliases) so the
	// poll only fetches the OIDs it needs. May be empty on first contact.
	Discovery domain.SNMPDiscoveryResult
	// InterfaceSettings is the per-interface monitoring policy.
	InterfaceSettings []domain.SNMPInterfaceSettings
	// MonitoredInterfaceIDs restricts polling to these ifIndex values.
	MonitoredInterfaceIDs []int
	// KeyPrefix scopes the counter store (typically the monitor id).
	KeyPrefix string
	Counters  *CounterState
	Now       time.Time
}

// CollectResult is the normalized outcome of one poll cycle.
type CollectResult struct {
	Device          domain.SNMPDeviceIdentity `json:"device"`
	Interfaces      []InterfaceSnapshot       `json:"interfaces"`
	Sensors         []domain.SNMPSensorInfo   `json:"sensors"`
	Metrics         map[string]any            `json:"metrics"`
	Attributes      map[string]any            `json:"attributes"`
	State           domain.SNMPFailureState   `json:"state"`
	PartialFailures []string                  `json:"partial_failures,omitempty"`
	PacketsSent     int                       `json:"-"`
	PacketsReceived int                       `json:"-"`
}

// Collect runs one asynchronous poll cycle: scalar system metrics via the
// vendor provider, then the interface table (status + counters) with GETBULK,
// computing rates and utilization from counter deltas. A slow or down device
// only fails its own poll — the caller never blocks.
func Collect(ctx context.Context, opts PollOptions) (CollectResult, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if opts.Counters == nil {
		opts.Counters = NewCounterState()
	}

	client, err := NewClient(opts.Params)
	if err != nil {
		return CollectResult{State: domain.SNMPStateInvalidConfig, Metrics: map[string]any{}, Attributes: map[string]any{}},
			fmt.Errorf("build snmp client: %w", err)
	}
	if err := client.Connect(); err != nil {
		return CollectResult{State: DialErrorState(err), Metrics: map[string]any{}, Attributes: map[string]any{}}, err
	}
	defer client.Close()

	result := CollectResult{
		Metrics:    map[string]any{},
		Attributes: map[string]any{},
		State:      domain.SNMPStateSuccess,
	}

	// System identity + provider selection.
	identity, sysObjectID, uptimeSeconds, err := pollIdentity(ctx, client)
	if err != nil {
		result.State = ResponseErrorState(err)
		return result, err
	}
	result.Device = identity
	if uptimeSeconds > 0 {
		result.Metrics["device.uptime_seconds"] = uptimeSeconds
	}
	provider := opts.Registry.ProviderFor(sysObjectID)

	// System scalars (uptime, CPU, memory). A failure here must not take the
	// whole device down — the session works, the metric is just unavailable.
	if err := pollSystemScalars(ctx, client, provider, identity, &result); err != nil {
		result.State = domain.SNMPStatePartial
		result.PartialFailures = append(result.PartialFailures, "system_scalars")
	}

	// Interface table.
	snapshots, state, partial, sent, recv := pollInterfaces(ctx, client, opts, now)
	result.Interfaces = snapshots
	result.PacketsSent += sent
	result.PacketsReceived += recv
	if state != domain.SNMPStateSuccess {
		result.PartialFailures = append(result.PartialFailures, partial...)
		if state == domain.SNMPStateTimeout || state == domain.SNMPStateDeviceUnreachable {
			result.State = state
		} else {
			result.State = domain.SNMPStatePartial
		}
	}

	// Sensor values ride along when the device exposes them.
	result.Sensors = discoverSensors(ctx, client, provider)

	reachable := result.State != domain.SNMPStateDeviceUnreachable &&
		result.State != domain.SNMPStateTimeout &&
		result.State != domain.SNMPStateAuthentication &&
		result.State != domain.SNMPStateAuthorization
	result.Metrics["snmp.reachability"] = float64(boolToIntFn(reachable))
	result.Metrics["snmp.partial_failures"] = float64(len(result.PartialFailures))

	return result, nil
}

// pollIdentity fetches sysObjectID + sysName + sysUpTime cheaply.
func pollIdentity(ctx context.Context, client *gosnmp.GoSNMP) (domain.SNMPDeviceIdentity, string, float64, error) {
	packet, err := client.Get([]string{oidSysObjectID, oidSysName, oidSysUpTime})
	if err != nil {
		return domain.SNMPDeviceIdentity{}, "", 0, err
	}
	// gosnmp surfaces SNMP error-status in the packet, not as a Go error.
	if packet.Error != gosnmp.NoError {
		return domain.SNMPDeviceIdentity{}, "", 0, fmt.Errorf("%s", packet.Error)
	}

	id, sysObjectID, uptimeSeconds := identityFromPacket(packet)
	if sysObjectID == "" {
		return id, "", 0, fmt.Errorf("device did not return sysObjectID")
	}
	return id, sysObjectID, uptimeSeconds, nil
}

// identityFromPacket maps the sys* variable bindings into the device identity.
func identityFromPacket(packet *gosnmp.SnmpPacket) (domain.SNMPDeviceIdentity, string, float64) {
	var id domain.SNMPDeviceIdentity
	var sysObjectID string
	var uptimeSeconds float64
	for _, v := range packet.Variables {
		name := strings.TrimPrefix(v.Name, ".")
		if oid := objectIDFromPDU(v); name == oidSysObjectID {
			sysObjectID = oid
			id.SysObjectID = oid
			id.Vendor, _ = ClassifyVendorModel("", oid)
		} else if name == oidSysName {
			id.SysName = pduStringName(v)
		} else if name == oidSysUpTime {
			uptimeSeconds = uptimeFromPDU(v, &id)
		}
	}
	return id, sysObjectID, uptimeSeconds
}

// objectIDFromPDU extracts the dot-stripped OID string, or "" when absent.
func objectIDFromPDU(v gosnmp.SnmpPDU) string {
	oid, ok := v.Value.(string)
	if !ok {
		return ""
	}
	return strings.TrimPrefix(oid, ".")
}

// uptimeFromPDU records sysUpTime in both numeric seconds and string form.
func uptimeFromPDU(v gosnmp.SnmpPDU, id *domain.SNMPDeviceIdentity) float64 {
	ticks, ok := sysUpTimeTicks(v)
	if !ok {
		return 0
	}
	id.SysUpTime = fmt.Sprintf("%d", uint64(ticks/100))
	return ticks / 100
}

// sysUpTimeTicks extracts the sysUpTime TimeTicks value from a PDU, tolerating
// the numeric types gosnmp may produce (TimeTicks and Counter32 both decode to
// uint32, but agents vary).
func sysUpTimeTicks(v gosnmp.SnmpPDU) (float64, bool) {
	switch value := v.Value.(type) {
	case uint32:
		return float64(value), true
	case uint:
		return float64(value), true
	case uint64:
		return float64(value), true
	case int:
		if value >= 0 {
			return float64(value), true
		}
	case int64:
		if value >= 0 {
			return float64(value), true
		}
	case float64:
		return value, true
	}
	return 0, false
}

// pduStringName extracts a string from either OctetString ([]byte) or string
// PDU values.
func pduStringName(v gosnmp.SnmpPDU) string {
	switch value := v.Value.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	default:
		return ""
	}
}

// pollSystemScalars collects CPU and memory from the provider's MIBs. Uptime
// is already captured by pollIdentity.
func pollSystemScalars(ctx context.Context, client *gosnmp.GoSNMP, provider Provider, identity domain.SNMPDeviceIdentity, result *CollectResult) error {
	cpu, cpuErr := pollCPU(ctx, client, provider)
	if cpuErr == nil && cpu >= 0 {
		result.Metrics["device.cpu_percent"] = cpu
	}

	mem, memErr := pollMemory(ctx, client, provider)
	if memErr == nil && mem >= 0 {
		result.Metrics["device.memory_percent"] = mem
	}

	if cpuErr != nil || memErr != nil {
		return fmt.Errorf("cpu=%v mem=%v", cpuErr, memErr)
	}
	return nil
}

func pollCPU(ctx context.Context, client *gosnmp.GoSNMP, provider Provider) (float64, error) {
	root := oidHRProcessorLoad
	if provider.Name() == "cisco" {
		root = oidCpmCPUTotal5secRev
	}
	values := []float64{}
	err := client.BulkWalk(root, func(pdu gosnmp.SnmpPDU) error {
		if !isMissingPDU(pdu) {
			values = append(values, pduFloat(pdu))
		}
		return nil
	})
	if err != nil && len(values) == 0 {
		return -1, err
	}
	return average(values), nil
}

func pollMemory(ctx context.Context, client *gosnmp.GoSNMP, provider Provider) (float64, error) {
	if provider.Name() == "cisco" {
		return pollCiscoMemory(ctx, client)
	}
	return pollHostResourcesMemory(ctx, client)
}

// pollHostResourcesMemory sums used/size across hrStorage RAM entries
// (hrStorageType = 1.3.6.1.2.1.25.2.1.7).
func pollHostResourcesMemory(ctx context.Context, client *gosnmp.GoSNMP) (float64, error) {
	type row struct{ size, used uint64 }
	rows := map[string]*row{}

	err := client.BulkWalk(oidHRStorageType, func(pdu gosnmp.SnmpPDU) error {
		if isMissingPDU(pdu) {
			return nil
		}
		key := oidSuffixOf(pdu.Name)
		if pduString(pdu) == hrStorageRAM {
			rows[key] = &row{}
		}
		return nil
	})
	if err != nil && len(rows) == 0 {
		return -1, err
	}

	if err := client.BulkWalk(oidHRStorageSize, func(pdu gosnmp.SnmpPDU) error {
		if r, ok := rows[oidSuffixOf(pdu.Name)]; ok {
			r.size = pduUint(pdu)
		}
		return nil
	}); err != nil {
		return -1, err
	}
	if err := client.BulkWalk(oidHRStorageUsed, func(pdu gosnmp.SnmpPDU) error {
		if r, ok := rows[oidSuffixOf(pdu.Name)]; ok {
			r.used = pduUint(pdu)
		}
		return nil
	}); err != nil {
		return -1, err
	}

	var used, size uint64
	for _, r := range rows {
		used += r.used
		size += r.size
	}
	if size == 0 {
		return -1, fmt.Errorf("no RAM storage found")
	}
	return MemoryPercent(used, size), nil
}

func pollCiscoMemory(ctx context.Context, client *gosnmp.GoSNMP) (float64, error) {
	usedByPool := map[string]float64{}
	freeByPool := map[string]float64{}
	if err := client.BulkWalk(oidCiscoMemoryPoolUsed, func(pdu gosnmp.SnmpPDU) error {
		if !isMissingPDU(pdu) {
			usedByPool[oidSuffixOf(pdu.Name)] = pduFloat(pdu)
		}
		return nil
	}); err != nil {
		return -1, err
	}
	if err := client.BulkWalk(oidCiscoMemoryPoolFree, func(pdu gosnmp.SnmpPDU) error {
		if !isMissingPDU(pdu) {
			freeByPool[oidSuffixOf(pdu.Name)] = pduFloat(pdu)
		}
		return nil
	}); err != nil {
		return -1, err
	}
	if len(usedByPool) == 0 {
		return -1, fmt.Errorf("no cisco memory pools found")
	}
	var used, total float64
	for key, u := range usedByPool {
		used += u
		total += u + freeByPool[key]
	}
	if total == 0 {
		return -1, fmt.Errorf("cisco memory pools empty")
	}
	return math.Min(100, (used/total)*100), nil
}

// polledInterface accumulates the numeric poll data for one interface.
type polledInterface struct {
	operStatus  float64
	adminStatus float64
	speed       float64
	highSpeed   float64
	counters    map[string]float64
	counterBits map[string]Bits
	name        string
	descr       string
	alias       string
	ifType      float64
	mtu         float64
}

// pollInterfaces collects status + counters for the monitored interface set
// using GETBULK. 64-bit IFX-MIB counters are preferred, 32-bit IF-MIB used as
// fallback.
func pollInterfaces(ctx context.Context, client *gosnmp.GoSNMP, opts PollOptions, now time.Time) ([]InterfaceSnapshot, domain.SNMPFailureState, []string, int, int) {
	cache := map[int]domain.SNMPInterfaceInfo{}
	for _, inf := range opts.Discovery.Interfaces {
		cache[inf.IfIndex] = inf
	}
	settings := opts.InterfaceSettings

	monitored := map[int]bool{}
	for _, idx := range opts.MonitoredInterfaceIDs {
		monitored[idx] = true
	}

	data := map[int]*polledInterface{}
	var order []int
	ensure := func(idx int) *polledInterface {
		if _, ok := data[idx]; !ok {
			data[idx] = &polledInterface{counters: map[string]float64{}, counterBits: map[string]Bits{}}
			order = append(order, idx)
		}
		return data[idx]
	}

	sent := 0
	recv := 0
	partial := []string{}

	walkNum := func(root, metric string) {
		n, err := walkNumericColumn(ctx, client, root, func(idx int) *polledInterface { return ensure(idx) }, data, metric)
		sent++
		recv += n
		_ = err
	}

	// Status + admin + speed.
	walkNum(oidIfOperStatus, "oper_status")
	walkNum(oidIfAdminStatus, "admin_status")
	walkNum(oidIfSpeed, "speed")

	// Counter columns — 64-bit preferred, 32-bit fallback.
	partial = append(partial, pollCounterColumns(ctx, client, ensure, data, &sent, &recv)...)

	// Meta columns only when no discovery cache exists (first contact).
	if len(cache) == 0 {
		sent += pollMetaColumns(ctx, client, ensure)
	}

	snapshots := make([]InterfaceSnapshot, 0, len(order))
	for _, idx := range order {
		d := data[idx]
		cached := cache[idx]

		// Apply the monitoring policy (defaults skip loopback/down interfaces).
		if !monitoredInterface(idx, cached, monitored, settings, d.operStatus) {
			continue
		}

		snap := buildInterfaceSnapshot(idx, d, cached)
		snap.InBps = counterRate(opts, idx, "in_octets", snap.IfInOctets, rateBits(snap.Has64BitIn), now)
		snap.OutBps = counterRate(opts, idx, "out_octets", snap.IfOutOctets, rateBits(snap.Has64BitOut), now)
		counterRate(opts, idx, "in_packets", snap.IfInPackets, Bits32, now)
		counterRate(opts, idx, "out_packets", snap.IfOutPackets, Bits32, now)
		counterRate(opts, idx, "in_errors", snap.IfInErrors, Bits32, now)
		counterRate(opts, idx, "out_errors", snap.IfOutErrors, Bits32, now)
		counterRate(opts, idx, "in_discards", snap.IfInDiscards, Bits32, now)
		counterRate(opts, idx, "out_discards", snap.IfOutDiscards, Bits32, now)
		snap.Utilization = Utilization(snap.InBps, snap.OutBps, snap.IfSpeed)
		snapshots = append(snapshots, snap)
	}

	if len(snapshots) == 0 && len(order) == 0 {
		return snapshots, domain.SNMPStateTimeout, append(partial, "no interface data"), sent, recv
	}
	return snapshots, domain.SNMPStateSuccess, partial, sent, recv
}

// counterColumnSpec identifies the OIDs to walk for one counter metric.
type counterColumnSpec struct {
	metric string
	hc     string
	legacy string
}

var counterColumns = []counterColumnSpec{
	{"in_octets", oidIfHCInOctets, oidIfInOctets},
	{"out_octets", oidIfHCOutOctets, oidIfOutOctets},
	{"in_packets", oidIfHCInUcastPkts, oidIfInUcastPkts},
	{"out_packets", oidIfHCOutUcastPkts, oidIfOutUcastPkts},
	{"in_errors", "", oidIfInErrors},
	{"out_errors", "", oidIfOutErrors},
	{"in_discards", "", oidIfInDiscards},
	{"out_discards", "", oidIfOutDiscards},
}

// pollCounterColumns walks the counter columns, preferring 64-bit IFX-MIB OIDs
// and falling back to 32-bit IF-MIB on absence. Returns any fallback markers.
func pollCounterColumns(ctx context.Context, client *gosnmp.GoSNMP, ensure func(int) *polledInterface, data map[int]*polledInterface, sent, recv *int) []string {
	partial := []string{}
	for _, cc := range counterColumns {
		bits := Bits32
		if cc.hc != "" {
			// Try the 64-bit column first; fall back to 32-bit on absence.
			n, _ := walkNumericColumn(ctx, client, cc.hc, ensure, data, cc.metric)
			*sent++
			*recv += n
			if n > 0 {
				bits = Bits64
			} else {
				walkNumericColumn(ctx, client, cc.legacy, ensure, data, cc.metric)
				*sent++
				partial = append(partial, "fallback:"+cc.metric)
			}
		} else {
			walkNumericColumn(ctx, client, cc.legacy, ensure, data, cc.metric)
			*sent++
		}
		for _, d := range data {
			if _, ok := d.counters[cc.metric]; ok {
				d.counterBits[cc.metric] = bits
			}
		}
	}
	return partial
}

// pollMetaColumns walks name/descr/alias columns on first contact (no cache).
func pollMetaColumns(ctx context.Context, client *gosnmp.GoSNMP, ensure func(int) *polledInterface) int {
	sent := 0
	walkMeta := func(root, field string) {
		_ = client.BulkWalk(root, func(pdu gosnmp.SnmpPDU) error {
			if isMissingPDU(pdu) {
				return nil
			}
			idx, ok := indexFromOID(pdu.Name)
			if !ok {
				return nil
			}
			d := ensure(idx)
			text := pduString(pdu)
			switch field {
			case "name":
				d.name = text
			case "descr":
				d.descr = text
			case "alias":
				d.alias = text
			}
			return nil
		})
		sent++
	}
	walkMeta(oidIfName, "name")
	walkMeta(oidIfDescr, "descr")
	walkMeta(oidIfAlias, "alias")
	return sent
}

// buildInterfaceSnapshot assembles a snapshot from the polled data and the
// discovery cache, reconciling 64/32-bit counter flags and highSpeed.
func buildInterfaceSnapshot(idx int, d *polledInterface, cached domain.SNMPInterfaceInfo) InterfaceSnapshot {
	snap := InterfaceSnapshot{}
	snap.IfIndex = idx
	snap.IfName = firstNonEmpty(d.name, cached.IfName)
	snap.IfDescr = firstNonEmpty(d.descr, cached.IfDescr)
	snap.IfAlias = firstNonEmpty(d.alias, cached.IfAlias)
	snap.IfType = int(firstNonZero(d.ifType, float64(cached.IfType)))
	snap.IfMtu = int(firstNonZero(d.mtu, float64(cached.IfMtu)))
	snap.IfOperStatus = int(firstNonZero(d.operStatus, float64(cached.IfOperStatus)))
	snap.IfAdminStatus = int(firstNonZero(d.adminStatus, float64(cached.IfAdminStatus)))
	snap.IfSpeed = resolveInterfaceSpeed(d, cached)
	applyInterfaceCounters(&snap, d)
	snap.Has64BitIn = snap.Has64BitIn || cached.Has64BitIn
	snap.Has64BitOut = snap.Has64BitOut || cached.Has64BitOut
	return snap
}

// resolveInterfaceSpeed prefers the polled speed, falling back to highSpeed
// (Gbps) and then the cached value when the polled one is absent or a 32-bit
// overflow sentinel.
func resolveInterfaceSpeed(d *polledInterface, cached domain.SNMPInterfaceInfo) int64 {
	speed := int64(firstNonZero(d.speed, float64(cached.IfSpeed)))
	if speed == 0 || speed >= 4294967295 {
		if d.highSpeed > 0 {
			return int64(d.highSpeed) * 1_000_000
		}
		if cached.IfSpeed > 0 && cached.IfSpeed < 4294967295 {
			return cached.IfSpeed
		}
	}
	return speed
}

// applyInterfaceCounters copies the polled counter values into the snapshot,
// setting the 64-bit flags where the matching column was walked as 64-bit.
func applyInterfaceCounters(snap *InterfaceSnapshot, d *polledInterface) {
	for metric, value := range d.counters {
		switch metric {
		case "in_octets":
			snap.IfInOctets = uint64(value)
			if d.counterBits[metric] == Bits64 {
				snap.Has64BitIn = true
			}
		case "out_octets":
			snap.IfOutOctets = uint64(value)
			if d.counterBits[metric] == Bits64 {
				snap.Has64BitOut = true
			}
		case "in_packets":
			snap.IfInPackets = uint64(value)
		case "out_packets":
			snap.IfOutPackets = uint64(value)
		case "in_errors":
			snap.IfInErrors = uint64(value)
		case "out_errors":
			snap.IfOutErrors = uint64(value)
		case "in_discards":
			snap.IfInDiscards = uint64(value)
		case "out_discards":
			snap.IfOutDiscards = uint64(value)
		}
	}
}

func rateBits(is64 bool) Bits {
	if is64 {
		return Bits64
	}
	return Bits32
}

func counterRate(opts PollOptions, idx int, suffix string, value uint64, bits Bits, now time.Time) float64 {
	rate, ok := opts.Counters.Rate(fmt.Sprintf("%s:%d:%s", opts.KeyPrefix, idx, suffix), float64(value), bits, now)
	if !ok {
		return 0
	}
	return rate
}

// monitoredInterface applies the discovery-cache + per-interface policy.
func monitoredInterface(idx int, cached domain.SNMPInterfaceInfo, monitored map[int]bool, settings []domain.SNMPInterfaceSettings, operStatus float64) bool {
	if len(monitored) > 0 && !monitored[idx] {
		return false
	}
	if isIgnoredInterface(settings, idx, cached.IfName, cached.IfDescr, cached.IfAlias, int(operStatus)) {
		return false
	}
	return true
}

// walkNumericColumn walks a numeric column into data[idx][metric].
func walkNumericColumn(ctx context.Context, client *gosnmp.GoSNMP, root string, ensure func(int) *polledInterface, data map[int]*polledInterface, metric string) (int, error) {
	count := 0
	err := client.BulkWalk(root, func(pdu gosnmp.SnmpPDU) error {
		if isMissingPDU(pdu) {
			return nil
		}
		idx, ok := indexFromOID(pdu.Name)
		if !ok {
			return nil
		}
		d := ensure(idx)
		if metric == "oper_status" {
			d.operStatus = pduFloat(pdu)
		} else if metric == "admin_status" {
			d.adminStatus = pduFloat(pdu)
		} else if metric == "speed" {
			d.speed = pduFloat(pdu)
		} else if metric == "high_speed" {
			d.highSpeed = pduFloat(pdu)
		} else if metric == "if_type" {
			d.ifType = pduFloat(pdu)
		} else if metric == "mtu" {
			d.mtu = pduFloat(pdu)
		} else {
			d.counters[metric] = pduFloat(pdu)
		}
		count++
		return nil
	})
	if err != nil && count == 0 && isMissingTableError(err) {
		return count, nil
	}
	return count, err
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func firstNonZero(a, b float64) float64 {
	if a > 0 {
		return a
	}
	return b
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func boolToIntFn(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// oidSuffixOf returns everything after the last dot of an OID.
func oidSuffixOf(oid string) string {
	idx := strings.LastIndex(oid, ".")
	if idx < 0 {
		return oid
	}
	return oid[idx+1:]
}

// StateIsHealthy reports whether the collection is healthy enough to mark the
// device up (success or partial with a working SNMP session).
func (c CollectResult) StateIsHealthy() bool {
	return c.State == domain.SNMPStateSuccess || c.State == domain.SNMPStatePartial
}

// MarshalInterfaces serializes interface snapshots for result attributes.
func MarshalInterfaces(snapshots []InterfaceSnapshot) any {
	if len(snapshots) == 0 {
		return nil
	}
	data, err := json.Marshal(snapshots)
	if err != nil {
		return nil
	}
	return string(data)
}
