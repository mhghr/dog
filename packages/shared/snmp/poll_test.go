package snmp

import (
	"testing"

	"github.com/gosnmp/gosnmp"

	"monitoring-platform/packages/shared/domain"
)

func TestIdentityFromPacket(t *testing.T) {
	pkt := &gosnmp.SnmpPacket{Variables: []gosnmp.SnmpPDU{
		{Name: "." + oidSysObjectID, Value: ".1.3.6.1.4.1.9.1.1"},
		{Name: "." + oidSysName, Value: []byte("switch-a")},
		{Name: "." + oidSysUpTime, Value: uint32(1234500)},
	}}

	id, sysObjectID, uptime := identityFromPacket(pkt)
	if sysObjectID != "1.3.6.1.4.1.9.1.1" {
		t.Fatalf("sysObjectID = %q", sysObjectID)
	}
	if id.SysObjectID != sysObjectID {
		t.Fatalf("identity sysObjectID = %q", id.SysObjectID)
	}
	if id.SysName != "switch-a" {
		t.Fatalf("sysName = %q", id.SysName)
	}
	if id.SysUpTime != "12345" {
		t.Fatalf("sysUpTime = %q", id.SysUpTime)
	}
	if uptime != 12345 {
		t.Fatalf("uptime seconds = %v", uptime)
	}
	if id.Vendor == "" {
		t.Fatal("expected a vendor to be classified")
	}
}

func TestIdentityFromPacketMissingObjectID(t *testing.T) {
	pkt := &gosnmp.SnmpPacket{Variables: []gosnmp.SnmpPDU{
		{Name: "." + oidSysName, Value: []byte("x")},
	}}
	id, sysObjectID, uptime := identityFromPacket(pkt)
	if sysObjectID != "" {
		t.Fatalf("expected empty sysObjectID, got %q", sysObjectID)
	}
	if id.SysName != "x" {
		t.Fatalf("sysName = %q", id.SysName)
	}
	if uptime != 0 {
		t.Fatalf("uptime = %v", uptime)
	}
}

func TestBuildInterfaceSnapshot(t *testing.T) {
	d := &polledInterface{
		operStatus:  1,
		adminStatus: 1,
		speed:       100_000_000,
		counters:    map[string]float64{"in_octets": 5000, "out_octets": 9000},
		counterBits: map[string]Bits{"in_octets": Bits64, "out_octets": Bits32},
		name:        "Gi0/1",
	}
	snap := buildInterfaceSnapshot(7, d, domain.SNMPInterfaceInfo{})

	if snap.IfIndex != 7 {
		t.Fatalf("IfIndex = %v", snap.IfIndex)
	}
	if snap.IfName != "Gi0/1" {
		t.Fatalf("IfName = %q", snap.IfName)
	}
	if snap.IfSpeed != 100_000_000 {
		t.Fatalf("IfSpeed = %v", snap.IfSpeed)
	}
	if snap.IfInOctets != 5000 {
		t.Fatalf("IfInOctets = %v", snap.IfInOctets)
	}
	if !snap.Has64BitIn {
		t.Fatal("expected Has64BitIn from counter bits")
	}
	if snap.Has64BitOut {
		t.Fatal("expected no Has64BitOut for 32-bit counter")
	}
}

func TestBuildInterfaceSnapshotHighSpeed(t *testing.T) {
	d := &polledInterface{
		highSpeed:  1000,
		counters:   map[string]float64{},
		counterBits: map[string]Bits{},
	}
	snap := buildInterfaceSnapshot(1, d, domain.SNMPInterfaceInfo{})
	if snap.IfSpeed != 1_000_000_000 {
		t.Fatalf("IfSpeed from highSpeed = %v", snap.IfSpeed)
	}
}

func TestBuildInterfaceSnapshotCachedFlags(t *testing.T) {
	d := &polledInterface{counters: map[string]float64{}, counterBits: map[string]Bits{}}
	cached := domain.SNMPInterfaceInfo{IfName: "Fa0/0", IfSpeed: 10_000_000, Has64BitIn: true}
	snap := buildInterfaceSnapshot(2, d, cached)

	if snap.IfName != "Fa0/0" {
		t.Fatalf("IfName from cache = %q", snap.IfName)
	}
	if snap.IfSpeed != 10_000_000 {
		t.Fatalf("IfSpeed from cache = %v", snap.IfSpeed)
	}
	if !snap.Has64BitIn {
		t.Fatal("expected Has64BitIn inherited from cache")
	}
}

func TestMonitoredInterface(t *testing.T) {
	monitored := map[int]bool{1: true}

	if !monitoredInterface(1, domain.SNMPInterfaceInfo{}, monitored, nil, 1) {
		t.Fatal("index 1 should be monitored when present in the set")
	}
	if monitoredInterface(2, domain.SNMPInterfaceInfo{}, monitored, nil, 1) {
		t.Fatal("index 2 should be excluded when not in the set")
	}
	if monitoredInterface(1, domain.SNMPInterfaceInfo{IfName: "lo"}, nil, []domain.SNMPInterfaceSettings{
		{IfIndex: 1, Ignore: true},
	}, 1) {
		t.Fatal("ignored interface should be excluded")
	}
	if !monitoredInterface(1, domain.SNMPInterfaceInfo{}, nil, nil, 1) {
		t.Fatal("no policy should default to monitored")
	}
	if monitoredInterface(1, domain.SNMPInterfaceInfo{}, nil, nil, OperDown) {
		t.Fatal("administratively down interface should be skipped by default")
	}
}

func TestCounterColumnsPrefer64Bit(t *testing.T) {
	if counterColumns[0].metric != "in_octets" || counterColumns[0].hc == "" || counterColumns[0].legacy == "" {
		t.Fatalf("unexpected in_octets spec: %+v", counterColumns[0])
	}
}
