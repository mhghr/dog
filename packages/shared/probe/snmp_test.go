package probe

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/security"
	"monitoring-platform/packages/shared/snmp/snmpfake"
)

// execSNMPJob runs the SNMP executor against the fake agent with the given
// configuration.
func execSNMPJob(t *testing.T, config map[string]any, key string) domain.ProbeResult {
	t.Helper()
	executor := NewSNMPExecutor(Deps{SNMPKey: key})
	job := domain.ProbeJob{
		ID:         uuid.NewString(),
		MonitorID:  uuid.NewString(),
		ResourceID: uuid.NewString(),
		Type:       domain.MonitorSNMP,
		Config:     config,
	}
	return executor.Execute(context.Background(), job)
}

func encryptTest(t *testing.T, key, plaintext string) string {
	t.Helper()
	enc, err := security.EncryptSecret(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	return enc
}

func snmpJobConfig(t *testing.T, addr string) map[string]any {
	t.Helper()
	key := "test-master-key-0123456789abcdef"
	return map[string]any{
		"host":            "127.0.0.1",
		"port":            addrPort(addr),
		"version":         "2c",
		"community":       encryptTest(t, key, "public"),
		"timeout_seconds": 2,
		"retries":         1,
		"discovery": map[string]any{
			"device": map[string]any{},
		},
	}
}

func addrPort(addr string) int {
	// "127.0.0.1:PORT" → port int
	var port int
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			for _, ch := range addr[i+1:] {
				port = port*10 + int(ch-'0')
			}
			break
		}
	}
	return port
}

func TestSNMPExecutor_SuccessfulPoll(t *testing.T) {
	agent := snmpfake.DefaultAgent()
	addr, err := agent.Start()
	if err != nil {
		t.Fatalf("start fake agent: %v", err)
	}
	defer agent.Close()

	key := "test-master-key-0123456789abcdef"
	config := snmpJobConfig(t, addr.String())
	// Cached discovery: only ifIndex 1 is a monitored production interface.
	// ifIndex 2 is operationally down and ifIndex 3 is a loopback — both are
	// excluded by the default policy.
	config["discovery"] = map[string]any{
		"interfaces": []map[string]any{
			{"if_index": 1, "if_name": "Gi0/0", "if_descr": "GigabitEthernet0/0", "if_alias": "uplink", "if_oper_status": 1},
			{"if_index": 2, "if_name": "Gi0/1", "if_descr": "GigabitEthernet0/1", "if_alias": "downlink", "if_oper_status": 2},
			{"if_index": 3, "if_name": "Lo0", "if_descr": "Loopback0", "if_alias": "", "if_oper_status": 1},
		},
	}

	// First poll: rates unavailable (baseline).
	first := execSNMPJob(t, config, key)
	if !first.Success {
		t.Fatalf("first poll should succeed, got code=%s msg=%s", first.ErrorCode, first.ErrorMessage)
	}
	if first.Status != domain.StatusUp {
		t.Fatalf("expected up status, got %s", first.Status)
	}

	// Second poll 10ms later: counter rates are computed.
	time.Sleep(10 * time.Millisecond)
	second := execSNMPJob(t, config, key)
	if !second.Success {
		t.Fatalf("second poll should succeed, got %s", second.ErrorCode)
	}

	// Device metrics normalized to stable names.
	if _, ok := second.Metrics["device.cpu_percent"]; !ok {
		t.Fatal("missing device.cpu_percent")
	}
	mem, ok := second.Metrics["device.memory_percent"].(float64)
	if !ok || mem < 39 || mem > 41 {
		t.Fatalf("expected memory_percent ≈ 40, got %v", mem)
	}
	if _, ok := second.Metrics["device.uptime_seconds"]; !ok {
		t.Fatal("missing device.uptime_seconds")
	}
	if v := second.Metrics["snmp.reachability"]; v != 1.0 {
		t.Fatalf("expected reachability 1, got %v", v)
	}

	// Interface status + flat per-interface keys for the monitored interface.
	if v := second.Metrics["if_1_oper_status"]; v != 1.0 {
		t.Fatalf("expected if_1_oper_status 1, got %v", v)
	}
	if _, ok := second.Metrics["if_1_in_octets"]; !ok {
		t.Fatal("missing if_1_in_octets")
	}
	if _, ok := second.Metrics["if_1_in_bps"]; !ok {
		t.Fatal("missing if_1_in_bps")
	}
	// Down + loopback interfaces are excluded by default.
	if _, ok := second.Metrics["if_2_oper_status"]; ok {
		t.Fatal("down interface should be excluded by default")
	}
	if _, ok := second.Metrics["if_3_oper_status"]; ok {
		t.Fatal("loopback interface should be excluded by default")
	}
	// All monitored interfaces up → aggregate healthy.
	if v := second.Metrics["snmp.interface_oper_status"]; v != 1.0 {
		t.Fatalf("expected interface_oper_status 1, got %v", v)
	}

	// Interface metadata rides in attributes for the detail UI (only the
	// monitored interface).
	raw := second.Attributes["snmp.interfaces"]
	if raw == nil {
		t.Fatal("missing snmp.interfaces attribute")
	}
	var ifaces []map[string]any
	if err := json.Unmarshal([]byte(raw.(string)), &ifaces); err != nil {
		t.Fatalf("unmarshal interfaces: %v", err)
	}
	if len(ifaces) != 1 {
		t.Fatalf("expected 1 monitored interface, got %d", len(ifaces))
	}

	// Device identity is Cisco.
	deviceRaw := second.Attributes["snmp.device"].(string)
	var device map[string]any
	_ = json.Unmarshal([]byte(deviceRaw), &device)
	if device["vendor"] != "cisco" {
		t.Fatalf("expected cisco vendor, got %v", device["vendor"])
	}
}

func TestSNMPExecutor_AuthenticationFailure(t *testing.T) {
	agent := snmpfake.DefaultAgent()
	agent.AuthFail = true
	agent.Community = "public"
	addr, err := agent.Start()
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer agent.Close()

	key := "test-master-key-0123456789abcdef"
	config := snmpJobConfig(t, addr.String())
	// Wrong community.
	config["community"] = encryptTest(t, key, "wrong")

	result := execSNMPJob(t, config, key)
	if result.Success {
		t.Fatal("expected credential failure")
	}
	// A v2c agent that answers a community mismatch with an authorization
	// error surfaces as authorization; wrong-credential wording maps to
	// authentication (covered by unit tests). Either is a hard failure that
	// never leaks the secret.
	state := result.Attributes["snmp.state"].(string)
	if state != string(domain.SNMPStateAuthorization) && state != string(domain.SNMPStateAuthentication) {
		t.Fatalf("expected auth/authorization state, got %v", state)
	}
	// No secret material in the error.
	if containsSecret(result.ErrorMessage) {
		t.Fatalf("error leaked sensitive material: %q", result.ErrorMessage)
	}
}

func TestSNMPExecutor_DeviceUnreachable(t *testing.T) {
	key := "test-master-key-0123456789abcdef"
	// Port is closed → connection refused.
	config := map[string]any{
		"host":            "127.0.0.1",
		"port":            6553,
		"version":         "2c",
		"community":       encryptTest(t, key, "public"),
		"timeout_seconds": 1,
		"retries":         0,
	}

	result := execSNMPJob(t, config, key)
	if result.Success {
		t.Fatal("expected failure for unreachable device")
	}
	if result.Attributes["snmp.state"] != string(domain.SNMPStateDeviceUnreachable) &&
		result.Attributes["snmp.state"] != string(domain.SNMPStateTimeout) {
		t.Fatalf("expected unreachable/timeout state, got %v", result.Attributes["snmp.state"])
	}
}

func TestSNMPExecutor_PartialCollection(t *testing.T) {
	agent := snmpfake.DefaultAgent()
	// A generic host so the core provider (HOST-RESOURCES) is used; the agent
	// omits its CPU/RAM columns → system scalars unavailable, interfaces fine.
	agent.Scalars[1].Value = snmpfake.OIDString(snmpfake.LinuxObjectID)
	agent.OmitSystemColumns = true
	addr, err := agent.Start()
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer agent.Close()

	key := "test-master-key-0123456789abcdef"
	result := execSNMPJob(t, snmpJobConfig(t, addr.String()), key)

	// A partial collection must NOT take the device down.
	if !result.Success {
		t.Fatalf("partial collection should keep device up, got code=%s", result.ErrorCode)
	}
	if result.Status != domain.StatusUp {
		t.Fatalf("expected up status, got %s", result.Status)
	}
	if result.Attributes["snmp.state"] != string(domain.SNMPStatePartial) {
		t.Fatalf("expected partial state, got %v", result.Attributes["snmp.state"])
	}
	// Interface metrics still flow.
	if _, ok := result.Metrics["if_1_oper_status"]; !ok {
		t.Fatal("interface metrics missing in partial collection")
	}
}

func TestSNMPExecutor_InvalidConfig(t *testing.T) {
	executor := NewSNMPExecutor(Deps{SNMPKey: ""})
	job := domain.ProbeJob{ID: uuid.NewString(), MonitorID: uuid.NewString(), Type: domain.MonitorSNMP, Config: map[string]any{}}
	result := executor.Execute(context.Background(), job)
	if result.Success {
		t.Fatal("expected failure for invalid config")
	}
	if result.Attributes["snmp.state"] != string(domain.SNMPStateInvalidConfig) {
		t.Fatalf("expected invalid_config state, got %v", result.Attributes["snmp.state"])
	}
}

func containsSecret(s string) bool {
	secrets := []string{"public", "wrong", "s3cret", "p3riv"}
	for _, sec := range secrets {
		if len(s) > 0 && indexOf(s, sec) >= 0 {
			return true
		}
	}
	return false
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
