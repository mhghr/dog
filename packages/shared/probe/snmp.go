package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"monitoring-platform/packages/shared/domain"
	snmplib "monitoring-platform/packages/shared/snmp"
)

// SNMPExecutor is the SNMP network-device collector. It polls a device's
// system metrics and interface table through a vendor-aware OID registry,
// normalizing every value into stable metric names. Discovery results cached
// in the monitor configuration prevent full MIB walks on routine polls.
type SNMPExecutor struct {
	deps     Deps
	counters *snmplib.CounterState
	obs      snmplib.Metrics
}

// NewSNMPExecutor builds an SNMP collector executor.
func NewSNMPExecutor(deps Deps) *SNMPExecutor {
	return &SNMPExecutor{
		deps:     deps,
		counters: snmplib.NewCounterState(),
	}
}

func (e *SNMPExecutor) Type() domain.MonitorType {
	return domain.MonitorSNMP
}

// Observability returns the internal SNMP counters (for tests/diagnostics).
func (e *SNMPExecutor) Observability() *snmplib.Metrics {
	return &e.obs
}

func (e *SNMPExecutor) Execute(ctx context.Context, job domain.ProbeJob) domain.ProbeResult {
	result := newBaseResult(job)
	start := time.Now().UTC()

	cfg, err := decodeConfig(job.Config)
	if err != nil {
		e.obs.RecordPoll(start, domain.SNMPStateInvalidConfig, 0, 0)
		return finishSnmpFailure(result, domain.SNMPStateInvalidConfig, err)
	}

	if cfg.Version == domain.SNMPv3 {
		// Prefer SNMPv3 is a security posture â€” surface it in attributes.
		result.Attributes["snmp.version"] = "v3"
	} else {
		result.Attributes["snmp.version"] = snmplib.VersionText(cfg.Version)
	}

	if err := snmplib.DecryptSecretValues(e.deps.SNMPKey, &cfg); err != nil {
		e.obs.RecordPoll(start, domain.SNMPStateInvalidConfig, 0, 0)
		return finishSnmpFailure(result, domain.SNMPStateInvalidConfig, err)
	}

	params, err := snmplib.BuildParams(&cfg)
	if err != nil {
		e.obs.RecordPoll(start, domain.SNMPStateInvalidConfig, 0, 0)
		return finishSnmpFailure(result, domain.SNMPStateInvalidConfig, err)
	}

	discovery := loadDiscoveryCache(job.Config)

	collected, err := snmplib.Collect(ctx, snmplib.PollOptions{
		Params:                params,
		Registry:              snmplib.DefaultRegistry(),
		Discovery:             discovery,
		InterfaceSettings:     cfg.Interfaces,
		MonitoredInterfaceIDs: cfg.MonitoredInterfaceIDs,
		KeyPrefix:             job.MonitorID,
		Counters:              e.counters,
		Now:                   time.Now().UTC(),
	})

	e.obs.RecordPoll(start, collected.State, collected.PacketsSent, collected.PacketsReceived)

	if err != nil && collected.State != domain.SNMPStatePartial {
		return finishSnmpFailure(result, collected.State, err)
	}

	// Merge collected metrics (device + per-interface flat keys).
	applyCollected(result, collected)

	// Device-level aggregates for the health engine: any monitored interface
	// operationally down, and the highest utilization across interfaces.
	if len(collected.Interfaces) > 0 {
		allUp := true
		maxUtil := 0.0
		for _, inf := range collected.Interfaces {
			if inf.IfOperStatus != snmplib.OperUp {
				allUp = false
			}
			if inf.Utilization > maxUtil {
				maxUtil = inf.Utilization
			}
		}
		if allUp {
			result.Metrics["snmp.interface_oper_status"] = 1.0
		} else {
			result.Metrics["snmp.interface_oper_status"] = 0.0
		}
		result.Metrics["snmp.interface_utilization_percent"] = maxUtil
	}

	// Structured device context for the UI â€” never secrets.
	deviceJSON, _ := json.Marshal(collected.Device)
	interfacesJSON, _ := json.Marshal(collected.Interfaces)
	sensorsJSON, _ := json.Marshal(collected.Sensors)
	result.Attributes["snmp.state"] = string(collected.State)
	result.Attributes["snmp.device"] = string(deviceJSON)
	result.Attributes["snmp.interfaces"] = string(interfacesJSON)
	result.Attributes["snmp.sensors"] = string(sensorsJSON)
	if len(collected.PartialFailures) > 0 {
		result.Attributes["snmp.partial_failures"] = collected.PartialFailures
	}

	// Device health = sensor state + interface health.
	deviceHealth := 1.0
	if !stateIsHealthy(collected.State) {
		deviceHealth = 0.0
	}
	result.Metrics["device.health"] = deviceHealth

	if collected.State == domain.SNMPStatePartial {
		result.Attributes["snmp.error_code"] = "partial"
		result.Success = true
		result.Status = domain.StatusUp
		result.FinishedAt = time.Now().UTC()
		result.DurationMillis = result.FinishedAt.Sub(result.StartedAt).Milliseconds()
		result.Metrics["total_duration_ms"] = result.DurationMillis
		return result
	}

	return finishSuccess(result)
}

// decodeConfig maps the flat monitor configuration onto the device config,
// including the encrypted secret fields as-is (decrypted later).
func decodeConfig(config map[string]any) (domain.SNMPDeviceConfig, error) {
	var cfg domain.SNMPDeviceConfig
	cfg.Host = stringConfig(config, "host", "")
	cfg.Port = intConfig(config, "port", 161)
	cfg.Version = domain.SNMPVersion(stringConfig(config, "version", "2c"))
	cfg.Transport = stringConfig(config, "transport", "udp")
	cfg.TimeoutSeconds = intConfig(config, "timeout_seconds", 3)
	cfg.Retries = intConfig(config, "retries", 1)
	cfg.MaxRepetitions = intConfig(config, "max_repetitions", 10)
	cfg.Community = stringConfig(config, "community", "")
	cfg.Username = stringConfig(config, "username", "")
	cfg.SecurityLevel = domain.SNMPSecurityLevel(stringConfig(config, "security_level", "noAuthNoPriv"))
	cfg.AuthenticationProto = stringConfig(config, "authentication_protocol", "")
	cfg.AuthenticationSecret = stringConfig(config, "authentication_secret", "")
	cfg.PrivacyProto = stringConfig(config, "privacy_protocol", "")
	cfg.PrivacySecret = stringConfig(config, "privacy_secret", "")
	cfg.ContextName = stringConfig(config, "context_name", "")

	if raw, ok := config["interfaces"]; ok {
		data, err := json.Marshal(raw)
		if err == nil {
			_ = json.Unmarshal(data, &cfg.Interfaces)
		}
	}
	if raw, ok := config["monitored_interface_ids"]; ok {
		data, err := json.Marshal(raw)
		if err == nil {
			_ = json.Unmarshal(data, &cfg.MonitoredInterfaceIDs)
		}
	}

	if cfg.Host == "" {
		return cfg, fmt.Errorf("snmp host is required")
	}
	return cfg, nil
}

// loadDiscoveryCache reads the cached discovery payload from the monitor
// configuration (populated by the API on enrollment / rediscovery).
func loadDiscoveryCache(config map[string]any) domain.SNMPDiscoveryResult {
	raw, ok := config["discovery"]
	if !ok {
		return domain.SNMPDiscoveryResult{}
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return domain.SNMPDiscoveryResult{}
	}
	var discovery domain.SNMPDiscoveryResult
	if err := json.Unmarshal(data, &discovery); err != nil {
		return domain.SNMPDiscoveryResult{}
	}
	return discovery
}

// applyCollected flattens device + per-interface values into the result
// metrics map with controlled, OID-independent names.
func applyCollected(result domain.ProbeResult, collected snmplib.CollectResult) {
	for key, value := range collected.Metrics {
		result.Metrics[key] = value
	}

	maxTemp := -1.0
	for _, sensor := range collected.Sensors {
		if sensor.SensorType == "temperature" && sensor.Value > maxTemp {
			maxTemp = sensor.Value
		}
	}
	if maxTemp >= 0 {
		result.Metrics["device.temperature_celsius"] = maxTemp
	}

	for _, inf := range collected.Interfaces {
		idx := inf.IfIndex
		set := func(suffix string, value any) {
			result.Metrics[interfaceFlatKey(idx, suffix)] = value
		}
		set("oper_status", float64(inf.IfOperStatus))
		set("admin_status", float64(inf.IfAdminStatus))
		set("speed_bps", float64(inf.IfSpeed))
		set("in_octets", float64(inf.IfInOctets))
		set("out_octets", float64(inf.IfOutOctets))
		set("in_packets", float64(inf.IfInPackets))
		set("out_packets", float64(inf.IfOutPackets))
		set("in_errors", float64(inf.IfInErrors))
		set("out_errors", float64(inf.IfOutErrors))
		set("in_discards", float64(inf.IfInDiscards))
		set("out_discards", float64(inf.IfOutDiscards))
		set("in_bps", inf.InBps)
		set("out_bps", inf.OutBps)
		set("utilization_percent", inf.Utilization)
	}
}

func interfaceFlatKey(ifIndex int, suffix string) string {
	return fmt.Sprintf("if_%d_%s", ifIndex, suffix)
}

// finishSnmpFailure records a deterministic SNMP failure without echoing
// sensitive details.
func finishSnmpFailure(result domain.ProbeResult, state domain.SNMPFailureState, err error) domain.ProbeResult {
	result.Metrics["snmp.reachability"] = 0.0
	result.Metrics["device.health"] = 0.0
	result.Attributes["snmp.state"] = string(state)
	result.Attributes["snmp.error_code"] = string(state)

	message := ""
	if err != nil {
		message = snmplib.SanitizeError(err)
	}
	result.Attributes["error_type"] = string(state)
	return finishFailure(result, string(state), fmt.Errorf("%s", message))
}

// StateIsHealthy reports whether the collection is healthy enough to mark the
// device up (success or partial with a working session).
func stateIsHealthy(state domain.SNMPFailureState) bool {
	return state == domain.SNMPStateSuccess || state == domain.SNMPStatePartial
}
