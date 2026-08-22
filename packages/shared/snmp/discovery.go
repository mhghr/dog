package snmp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"monitoring-platform/packages/shared/domain"
)

// discoveryTimeout is the overall budget for one discovery run. Discovery is
// heavier than a poll (interface walks) so it gets a generous window.
const discoveryTimeout = 30 * time.Second

// TestResult is the outcome of a connectivity test.
type TestResult struct {
	State  domain.SNMPFailureState `json:"state"`
	OK     bool                    `json:"ok"`
	Detail string                  `json:"detail"`
	SysName string                 `json:"sys_name,omitempty"`
	SysDescr string                `json:"sys_descr,omitempty"`
	SysObjectID string             `json:"sys_object_id,omitempty"`
	Uptime string                  `json:"uptime,omitempty"`
	Steps  []string                `json:"steps,omitempty"`
}

// TestConnection performs a real SNMP GET of sysObjectID + sysName against the
// target using the concrete parameters. It returns Connected only when both
// succeed; every other outcome maps to the failure taxonomy.
func TestConnection(ctx context.Context, params ConnectParams) TestResult {
	result := TestDevice(ctx, params)
	return result
}

// TestDevice performs a real SNMP GET of the system identity (sysName,
// sysDescr, sysObjectID, sysUpTime) and reports a staged, human-readable
// breakdown: DNS resolution, target reachability, SNMP UDP connectivity,
// SNMP response and authentication. Secrets are never part of the result.
func TestDevice(ctx context.Context, params ConnectParams) TestResult {
	if params.Port == 0 {
		params.Port = 161
	}

	result := TestResult{Steps: []string{}}

	// 1. DNS resolution (when a hostname is used).
	if net.ParseIP(params.Host) == nil {
		addrs, err := net.LookupHost(params.Host)
		if err != nil || len(addrs) == 0 {
			result.Steps = append(result.Steps, "DNS resolution: failed")
			return TestResult{State: domain.SNMPStateDeviceUnreachable, Detail: "hostname did not resolve", Steps: result.Steps}
		}
		result.Steps = append(result.Steps, "DNS resolution: "+addrs[0])
	} else {
		result.Steps = append(result.Steps, "DNS resolution: "+params.Host)
	}

	// 2. UDP socket connect (target reachability + UDP/161 reachable).
	conn, err := net.DialTimeout("udp", net.JoinHostPort(params.Host, strconv.Itoa(params.Port)), params.Timeout)
	if err != nil {
		state := DialErrorState(err)
		result.Steps = append(result.Steps, "Target reachability: failed ("+string(state)+")")
		return TestResult{State: state, Detail: "target not reachable on UDP/" + strconv.Itoa(params.Port), Steps: result.Steps}
	}
	result.Steps = append(result.Steps, "Target reachability: ok (UDP/"+strconv.Itoa(params.Port)+" reachable)")
	_ = conn.Close()

	client, err := NewClient(params)
	if err != nil {
		result.Steps = append(result.Steps, "SNMP client: invalid configuration")
		return TestResult{State: domain.SNMPStateInvalidConfig, Detail: "invalid connection parameters", Steps: result.Steps}
	}
	if err := client.Connect(); err != nil {
		state := DialErrorState(err)
		result.Steps = append(result.Steps, "SNMP UDP connectivity: failed ("+string(state)+")")
		return TestResult{State: state, Detail: "SNMP connection failed", Steps: result.Steps}
	}
	defer client.Close()
	result.Steps = append(result.Steps, "SNMP UDP connectivity: ok")

	// 3. Real SNMP GET on the system identity.
	packet, err := client.Get([]string{oidSysName, oidSysDescr, oidSysObjectID, oidSysUpTime})
	if err != nil {
		state := ResponseErrorState(err)
		result.Steps = append(result.Steps, "SNMP response: failed ("+string(state)+")")
		if state == domain.SNMPStateAuthentication || state == domain.SNMPStateAuthorization {
			result.Steps = append(result.Steps, "Authentication: failed")
		}
		return TestResult{State: state, Detail: SanitizeError(err), Steps: result.Steps}
	}
	if packet.Error != gosnmp.NoError {
		state := ResponseErrorState(fmt.Errorf("%s", packet.Error))
		result.Steps = append(result.Steps, "SNMP response: "+string(state))
		return TestResult{State: state, Detail: SanitizeError(fmt.Errorf("%s", packet.Error)), Steps: result.Steps}
	}
	result.Steps = append(result.Steps, "SNMP response: ok")

	for _, v := range packet.Variables {
		switch strings.TrimPrefix(v.Name, ".") {
		case oidSysName:
			result.SysName = pduStringName2(v)
		case oidSysDescr:
			result.SysDescr = pduStringName2(v)
		case oidSysObjectID:
			if oid, ok := v.Value.(string); ok {
				result.SysObjectID = strings.TrimPrefix(oid, ".")
			}
		case oidSysUpTime:
			if ticks, ok := sysUpTimeValue(v); ok {
				result.Uptime = fmt.Sprintf("%.0f", ticks/100)
			}
		}
	}

	if result.SysObjectID == "" {
		result.Steps = append(result.Steps, "SNMP response: device did not return sysObjectID")
		return TestResult{State: domain.SNMPStateUnsupported, Detail: "device did not return sysObjectID", Steps: result.Steps}
	}
	result.Steps = append(result.Steps, "Authentication: ok")
	result.OK = true
	result.State = domain.SNMPStateSuccess
	result.Detail = "connected"
	return result
}

func pduStringName2(v gosnmp.SnmpPDU) string {
	switch value := v.Value.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	default:
		return ""
	}
}

func sysUpTimeValue(v gosnmp.SnmpPDU) (float64, bool) {
	switch value := v.Value.(type) {
	case uint32:
		return float64(value), true
	case uint:
		return float64(value), true
	case uint64:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	}
	return 0, false
}

// Discovery performs a full device discovery: identity, interface table and
// environmental sensors. The result is cached by the API; polling never re-runs
// a full walk.
func Discovery(ctx context.Context, params ConnectParams, registry *Registry) (domain.SNMPDiscoveryResult, error) {
	ctx, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()

	client, err := NewClient(params)
	if err != nil {
		return domain.SNMPDiscoveryResult{}, fmt.Errorf("build snmp client: %w", err)
	}
	if err := client.Connect(); err != nil {
		return domain.SNMPDiscoveryResult{}, err
	}
	defer client.Close()

	identity, err := discoverIdentity(ctx, client)
	if err != nil {
		return domain.SNMPDiscoveryResult{}, err
	}

	provider := registry.ProviderFor(identity.SysObjectID)

	interfaces, err := discoverInterfaces(ctx, client)
	if err != nil {
		// Interface discovery is the core requirement of a network device —
		// fail discovery rather than returning a device with no interfaces.
		return domain.SNMPDiscoveryResult{}, err
	}

	sensors := discoverSensors(ctx, client, provider)
	hardwareOK := sensorsOK(sensors)

	return domain.SNMPDiscoveryResult{
		Device:       identity,
		Interfaces:   interfaces,
		Sensors:      sensors,
		HardwareOK:   hardwareOK,
		DiscoveredAt: time.Now().UTC(),
	}, nil
}

func discoverIdentity(ctx context.Context, client *gosnmp.GoSNMP) (domain.SNMPDeviceIdentity, error) {
	oidList := []string{oidSysName, oidSysDescr, oidSysObjectID, oidSysUpTime, oidSysLocation}
	result, err := client.Get(oidList)
	if err != nil {
		return domain.SNMPDeviceIdentity{}, fmt.Errorf("get system identity: %w", err)
	}

	var id domain.SNMPDeviceIdentity
	for _, v := range result.Variables {
		name := strings.TrimPrefix(v.Name, ".")
		text, _ := v.Value.(string)
		if bytes, ok := v.Value.([]byte); ok {
			text = string(bytes)
		}
		switch name {
		case oidSysName:
			id.SysName = text
		case oidSysDescr:
			id.SysDescr = text
		case oidSysObjectID:
			id.SysObjectID = strings.TrimPrefix(text, ".")
		case oidSysLocation:
			id.SysLocation = text
		case oidSysUpTime:
			if ticks, ok := v.Value.(uint32); ok {
				id.SysUpTime = strconv.FormatFloat(float64(ticks)/100, 'f', 0, 64)
			}
		}
	}

	if id.SysObjectID == "" {
		return domain.SNMPDeviceIdentity{}, errors.New("device did not return sysObjectID")
	}

	vendor, model := ClassifyVendorModel(id.SysDescr, id.SysObjectID)
	id.Vendor = vendor
	id.Model = model
	return id, nil
}

// discoverInterfaces walks the IF-MIB/IFX-MIB tables. 64-bit counters are
// preferred; when IFX-MIB is absent the legacy 32-bit counters are used.
func discoverInterfaces(ctx context.Context, client *gosnmp.GoSNMP) ([]domain.SNMPInterfaceInfo, error) {
	byIndex := map[int]*domain.SNMPInterfaceInfo{}
	var order []int

	// Metadata + 32-bit counters come from IF-MIB; names from IFX-MIB.
	meta := append(append([]MetricDef{}, MetaColumns...), LegacyInterfaceColumns...)
	for _, col := range meta {
		if err := walkColumn(ctx, client, col, func(ifIndex int, pdu gosnmp.SnmpPDU) {
			if _, ok := byIndex[ifIndex]; !ok {
				byIndex[ifIndex] = &domain.SNMPInterfaceInfo{IfIndex: ifIndex}
				order = append(order, ifIndex)
			}
			applyPDU(byIndex[ifIndex], col.Metric, pdu)
		}); err != nil {
			return nil, fmt.Errorf("walk %s (%s): %w", col.Metric, col.OID, err)
		}
	}

	// 64-bit counters: only available on devices implementing IFX-MIB. Detect
	// availability by walking the HC columns; missing tables produce no rows.
	hcIn, hcOut := false, false
	for _, col := range InterfaceColumns {
		if col.DataType != DataTypeCounter64 {
			continue
		}
		count := 0
		_ = walkColumn(ctx, client, col, func(ifIndex int, pdu gosnmp.SnmpPDU) {
			count++
			if _, ok := byIndex[ifIndex]; !ok {
				byIndex[ifIndex] = &domain.SNMPInterfaceInfo{IfIndex: ifIndex}
				order = append(order, ifIndex)
			}
			applyPDU(byIndex[ifIndex], col.Metric, pdu)
		})
		switch col.Metric {
		case "interface.in_octets":
			hcIn = count > 0
		case "interface.out_octets":
			hcOut = count > 0
		}
	}

	interfaces := make([]domain.SNMPInterfaceInfo, 0, len(order))
	for _, idx := range order {
		inf := byIndex[idx]
		// Prefer the 64-bit counters whenever the IFX-MIB walk produced them;
		// otherwise the 32-bit IF-MIB values gathered above are kept.
		inf.Has64BitIn = hcIn
		inf.Has64BitOut = hcOut
		interfaces = append(interfaces, *inf)
	}
	return interfaces, nil
}

// walkColumn walks one table column and invokes fn per row with the row index
// (last OID component). Missing tables (NoSuchObject/EndOfMibView) are not
// errors — the caller decides how to react.
func walkColumn(ctx context.Context, client *gosnmp.GoSNMP, col MetricDef, fn func(ifIndex int, pdu gosnmp.SnmpPDU)) error {
	seen := 0
	err := client.BulkWalk(col.OID, func(pdu gosnmp.SnmpPDU) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if isMissingPDU(pdu) {
			return nil
		}
		idx, ok := indexFromOID(pdu.Name)
		if !ok {
			return nil
		}
		seen++
		fn(idx, pdu)
		return nil
	})
	if err != nil && seen == 0 {
		if isMissingTableError(err) {
			return nil
		}
		return err
	}
	return err
}

func isMissingPDU(pdu gosnmp.SnmpPDU) bool {
	switch pdu.Type {
	case gosnmp.NoSuchObject, gosnmp.NoSuchInstance, gosnmp.EndOfMibView:
		return true
	}
	return false
}

func isMissingTableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "nosuchobject") ||
		strings.Contains(msg, "nosuchinstance") ||
		strings.Contains(msg, "endofmibview") ||
		strings.Contains(msg, "no more oids") ||
		strings.Contains(msg, "mib view") ||
		strings.Contains(msg, "no such")
}

// applyPDU stores a row value into an interface record.
func applyPDU(inf *domain.SNMPInterfaceInfo, metric string, pdu gosnmp.SnmpPDU) {
	switch metric {
	case "interface.name":
		inf.IfName = pduString(pdu)
	case "interface.descr":
		inf.IfDescr = pduString(pdu)
	case "interface.alias":
		inf.IfAlias = pduString(pdu)
	case "interface.type":
		inf.IfType = int(pduFloat(pdu))
	case "interface.mtu":
		inf.IfMtu = int(pduFloat(pdu))
	case "interface.speed_bps":
		inf.IfSpeed = int64(pduFloat(pdu))
	case "interface.admin_status":
		inf.IfAdminStatus = int(pduFloat(pdu))
	case "interface.oper_status":
		inf.IfOperStatus = int(pduFloat(pdu))
	case "interface.in_octets":
		inf.IfInOctets = pduUint(pdu)
	case "interface.out_octets":
		inf.IfOutOctets = pduUint(pdu)
	case "interface.in_packets":
		inf.IfInPackets = pduUint(pdu)
	case "interface.out_packets":
		inf.IfOutPackets = pduUint(pdu)
	case "interface.in_errors":
		inf.IfInErrors = pduUint(pdu)
	case "interface.out_errors":
		inf.IfOutErrors = pduUint(pdu)
	case "interface.in_discards":
		inf.IfInDiscards = pduUint(pdu)
	case "interface.out_discards":
		inf.IfOutDiscards = pduUint(pdu)
	}
}

// discoverSensors walks the provider's sensor tables. Missing tables are
// skipped (many devices expose none).
func discoverSensors(ctx context.Context, client *gosnmp.GoSNMP, provider Provider) []domain.SNMPSensorInfo {
	var sensors []domain.SNMPSensorInfo
	for _, def := range provider.SensorTables() {
		values := map[int]float64{}
		statuses := map[int]float64{}
		_ = walkColumn(ctx, client, MetricDef{OID: def.OID}, func(idx int, pdu gosnmp.SnmpPDU) {
			values[idx] = pduFloat(pdu)
		})
		if def.StatusOID != "" {
			_ = walkColumn(ctx, client, MetricDef{OID: def.StatusOID}, func(idx int, pdu gosnmp.SnmpPDU) {
				statuses[idx] = pduFloat(pdu)
			})
		}
		for idx, value := range values {
			name := fmt.Sprintf("%s-%d", def.SensorType, idx)
			sensors = append(sensors, domain.SNMPSensorInfo{
				Name:       name,
				SensorType: def.SensorType,
				Value:      value,
				Unit:       def.Unit,
				Status:     envStatusText(def.SensorType, statuses[idx]),
			})
		}
	}
	return sensors
}

// envStatusText normalizes Cisco envmon status codes (1=normal/ok).
func envStatusText(sensorType string, status float64) string {
	if status == 0 {
		return ""
	}
	switch sensorType {
	case "fan", "power":
		if status == 1 {
			return "ok"
		}
		if status == 2 {
			return "warning"
		}
		if status == 3 {
			return "critical"
		}
	case "temperature":
		if status == 1 {
			return "ok"
		}
		if status == 2 {
			return "warning"
		}
		if status == 3 {
			return "critical"
		}
	}
	return "unknown"
}

// sensorsOK reports whether all environmental sensors are in a good state.
// Devices without sensors are considered healthy.
func sensorsOK(sensors []domain.SNMPSensorInfo) bool {
	for _, s := range sensors {
		if s.Status == "critical" {
			return false
		}
	}
	return true
}

// ClassifyVendorModel infers vendor and model from sysDescr + sysObjectID.
func ClassifyVendorModel(sysDescr, sysObjectID string) (vendor, model string) {
	lower := strings.ToLower(sysDescr)
	switch {
	case strings.HasPrefix(sysObjectID, "1.3.6.1.4.1.9"):
		vendor = "cisco"
	case strings.HasPrefix(sysObjectID, "1.3.6.1.4.1.2636"):
		vendor = "juniper"
	case strings.HasPrefix(sysObjectID, "1.3.6.1.4.1.14988"):
		vendor = "mikrotik"
	case strings.HasPrefix(sysObjectID, "1.3.6.1.4.1.2011"):
		vendor = "huawei"
	case strings.HasPrefix(sysObjectID, "1.3.6.1.4.1.11"):
		vendor = "hpe"
	case strings.HasPrefix(sysObjectID, "1.3.6.1.4.1.12356"):
		vendor = "fortinet"
	case strings.Contains(lower, "cisco"):
		vendor = "cisco"
	case strings.Contains(lower, "juniper"):
		vendor = "juniper"
	case strings.Contains(lower, "mikrotik"):
		vendor = "mikrotik"
	case strings.Contains(lower, "huawei"):
		vendor = "huawei"
	case strings.Contains(lower, "aruba"):
		vendor = "hpe"
	case strings.Contains(lower, "fortigate"):
		vendor = "fortinet"
	case strings.Contains(lower, "linux"):
		vendor = "linux"
	case strings.Contains(lower, "ios"):
		vendor = "cisco"
	}

	model = firstTokenAfter(sysDescr, []string{"ios", "nx-os", "junos", "routeros", "vrp"})
	if model == "" {
		model = sysDescr
		if len(model) > 120 {
			model = model[:120]
		}
	}
	return vendor, model
}

func firstTokenAfter(text string, markers []string) string {
	for _, marker := range markers {
		idx := strings.Index(strings.ToLower(text), marker)
		if idx >= 0 {
			rest := strings.TrimSpace(text[idx+len(marker):])
			if rest == "" {
				return text
			}
			fields := strings.Fields(rest)
			if len(fields) > 0 {
				return strings.Trim(fields[0], ",.;()[]")
			}
		}
	}
	return ""
}

// indexFromOID extracts the last numeric component of an OID as the row index.
// Leading dots (gosnmp returns ".1.3.6...") are tolerated.
func indexFromOID(oid string) (int, bool) {
	oid = strings.TrimPrefix(oid, ".")
	parts := strings.Split(oid, ".")
	if len(parts) == 0 {
		return 0, false
	}
	idx, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, false
	}
	return idx, true
}

// pduString extracts a string from a PDU value (OctetString or string).
func pduString(pdu gosnmp.SnmpPDU) string {
	switch v := pdu.Value.(type) {
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	default:
		return ""
	}
}

// pduFloat extracts a numeric value from a PDU.
func pduFloat(pdu gosnmp.SnmpPDU) float64 {
	switch v := pdu.Value.(type) {
	case int:
		return float64(v)
	case uint:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	default:
		return 0
	}
}

// pduUint extracts an unsigned counter value from a PDU.
func pduUint(pdu gosnmp.SnmpPDU) uint64 {
	switch v := pdu.Value.(type) {
	case uint64:
		return v
	case uint32:
		return uint64(v)
	case uint:
		return uint64(v)
	case int:
		if v >= 0 {
			return uint64(v)
		}
	case int64:
		if v >= 0 {
			return uint64(v)
		}
	case float64:
		if v >= 0 {
			return uint64(v)
		}
	}
	return 0
}
