package snmp

import (
	"testing"

	"github.com/gosnmp/gosnmp"

	"monitoring-platform/packages/shared/domain"
)

func TestIdentityFromSystemGet(t *testing.T) {
	vars := []gosnmp.SnmpPDU{
		{Name: "." + oidSysName, Value: []byte("core-sw1")},
		{Name: "." + oidSysDescr, Value: "Cisco IOS XR"},
		{Name: "." + oidSysObjectID, Value: ".1.3.6.1.4.1.9.1.1"},
		{Name: "." + oidSysLocation, Value: "DC-1"},
		{Name: "." + oidSysUpTime, Value: uint32(8640000)},
	}

	id := identityFromSystemGet(vars)
	if id.SysName != "core-sw1" {
		t.Fatalf("SysName = %q", id.SysName)
	}
	if id.SysDescr != "Cisco IOS XR" {
		t.Fatalf("SysDescr = %q", id.SysDescr)
	}
	if id.SysObjectID != "1.3.6.1.4.1.9.1.1" {
		t.Fatalf("SysObjectID = %q", id.SysObjectID)
	}
	if id.SysLocation != "DC-1" {
		t.Fatalf("SysLocation = %q", id.SysLocation)
	}
	if id.SysUpTime != "86400" {
		t.Fatalf("SysUpTime = %q", id.SysUpTime)
	}
}

func TestIdentityFromSystemGetMissingFields(t *testing.T) {
	id := identityFromSystemGet([]gosnmp.SnmpPDU{
		{Name: "." + oidSysName, Value: "x"},
	})
	if id.SysObjectID != "" {
		t.Fatalf("expected empty SysObjectID, got %q", id.SysObjectID)
	}
	if id.SysName != "x" {
		t.Fatalf("SysName = %q", id.SysName)
	}
}

func TestApplyTestIdentity(t *testing.T) {
	var result TestResult
	applyTestIdentity([]gosnmp.SnmpPDU{
		{Name: "." + oidSysName, Value: []byte("sw-a")},
		{Name: "." + oidSysDescr, Value: "SWITCH"},
		{Name: "." + oidSysObjectID, Value: ".1.3.6.1.4.1.9.1.22"},
		{Name: "." + oidSysUpTime, Value: uint32(100000)},
	}, &result)

	if result.SysName != "sw-a" {
		t.Fatalf("SysName = %q", result.SysName)
	}
	if result.SysObjectID != "1.3.6.1.4.1.9.1.22" {
		t.Fatalf("SysObjectID = %q", result.SysObjectID)
	}
	if result.Uptime != "1000" {
		t.Fatalf("Uptime = %q", result.Uptime)
	}
}

func TestEnvStatusText(t *testing.T) {
	if got := envStatusText("temperature", 0); got != "" {
		t.Fatalf("status 0 should be empty, got %q", got)
	}
	if got := envStatusText("fan", 1); got != "ok" {
		t.Fatalf("fan 1 = %q", got)
	}
	if got := envStatusText("power", 3); got != "critical" {
		t.Fatalf("power 3 = %q", got)
	}
	if got := envStatusText("temperature", 2); got != "warning" {
		t.Fatalf("temperature 2 = %q", got)
	}
	if got := envStatusText("unknown", 5); got != "unknown" {
		t.Fatalf("unknown type = %q", got)
	}
}

func TestSensorsOK(t *testing.T) {
	okSensors := []domain.SNMPSensorInfo{{Name: "fan-1", Status: "ok"}}
	if !sensorsOK(okSensors) {
		t.Fatal("ok sensors should be healthy")
	}
	critSensors := []domain.SNMPSensorInfo{{Name: "power-1", Status: "critical"}}
	if sensorsOK(critSensors) {
		t.Fatal("critical sensor should fail health")
	}
	if !sensorsOK(nil) {
		t.Fatal("no sensors should be healthy")
	}
}
