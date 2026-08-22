package snmp

import (
	"testing"

	"monitoring-platform/packages/shared/domain"
)

func TestRegistry_ProviderDetection(t *testing.T) {
	registry := DefaultRegistry()

	if got := registry.ProviderFor("1.3.6.1.4.1.9.12.3.1.3.9.20"); got.Name() != "cisco" {
		t.Fatalf("expected cisco provider for cisco sysObjectID, got %s", got.Name())
	}
	if got := registry.ProviderFor("1.3.6.1.4.1.2636.3.1.2.0.1"); got.Name() != "core" {
		t.Fatalf("expected core provider for juniper sysObjectID, got %s", got.Name())
	}
	if got := registry.ProviderFor(""); got.Name() != "core" {
		t.Fatalf("expected core fallback, got %s", got.Name())
	}
}

func TestClassifyVendorModel(t *testing.T) {
	cases := []struct {
		descr  string
		oid    string
		vendor string
	}{
		{"Cisco IOS Software, C880 Software (C880DATA-UNIVERSALK9-M)", "1.3.6.1.4.1.9.1.1245", "cisco"},
		{"Juniper Networks, Inc. mx960 internet router", "1.3.6.1.4.1.2636.1.1.1.2.20", "juniper"},
		{"RouterOS RB4011", "1.3.6.1.4.1.14988.1", "mikrotik"},
		{"Linux hostname 6.8.0", "1.3.6.1.4.1.8072.3.2.10", "linux"},
		{"", "", ""},
	}
	for _, tc := range cases {
		vendor, _ := ClassifyVendorModel(tc.descr, tc.oid)
		if vendor != tc.vendor {
			t.Errorf("ClassifyVendorModel(%q, %q) = %q, want %q", tc.descr, tc.oid, vendor, tc.vendor)
		}
	}
}

func TestOperStatusText(t *testing.T) {
	if got := OperStatusText(OperUp); got != "up" {
		t.Fatalf("expected up, got %s", got)
	}
	if got := OperStatusText(OperDown); got != "down" {
		t.Fatalf("expected down, got %s", got)
	}
	if got := OperStatusText(OperLowerDown); got != "lowerLayerDown" {
		t.Fatalf("expected lowerLayerDown, got %s", got)
	}
}

func TestUtilization(t *testing.T) {
	if got := Utilization(500_000_000, 300_000_000, 1_000_000_000); got != 80 {
		t.Fatalf("expected 80%%, got %v", got)
	}
	if got := Utilization(0, 0, 0); got != 0 {
		t.Fatalf("expected 0 with zero speed, got %v", got)
	}
	if got := Utilization(1e12, 1e12, 1e9); got > 100 {
		t.Fatalf("utilization must be capped at 100, got %v", got)
	}
}

func TestMemoryPercent(t *testing.T) {
	if got := MemoryPercent(80, 100); got != 80 {
		t.Fatalf("expected 80, got %v", got)
	}
	if got := MemoryPercent(0, 0); got != 0 {
		t.Fatalf("expected 0 for zero size, got %v", got)
	}
}

func TestInterfaceLabels_Controlled(t *testing.T) {
	info := domain.SNMPInterfaceInfo{
		IfIndex: 7,
		IfName:  "GigabitEthernet0/1",
		IfAlias: "uplink-to-core",
	}
	labels := InterfaceLabels(info)
	if labels["interface_index"] != "7" {
		t.Fatalf("wrong interface_index label: %v", labels)
	}
	if labels["interface_name"] != "GigabitEthernet0/1" {
		t.Fatalf("wrong interface_name label: %v", labels)
	}
	if labels["interface_alias"] != "uplink-to-core" {
		t.Fatalf("wrong interface_alias label: %v", labels)
	}
}

func TestIsIgnoredInterface_Defaults(t *testing.T) {
	// Loopback defaults to ignored (via name or description).
	if !isIgnoredInterface(nil, 2, "Loopback0", "", "", OperUp) {
		t.Fatal("loopback should be ignored by default")
	}
	if !isIgnoredInterface(nil, 2, "Lo0", "Loopback0", "", OperUp) {
		t.Fatal("loopback should be ignored via description")
	}
	// Down interface defaults to ignored.
	if !isIgnoredInterface(nil, 3, "GigabitEthernet0/2", "", "", OperDown) {
		t.Fatal("down interface should be ignored by default")
	}
	// Up non-loopback is monitored.
	if isIgnoredInterface(nil, 1, "GigabitEthernet0/1", "", "", OperUp) {
		t.Fatal("up production interface should be monitored")
	}
	// Explicit monitor=true overrides the loopback default.
	if isIgnoredInterface([]domain.SNMPInterfaceSettings{{IfIndex: 2, Monitor: true, Ignore: false}}, 2, "Loopback0", "", "", OperUp) {
		t.Fatal("explicit monitor=true should override the loopback default")
	}
}

func TestInterfaceMetricKey(t *testing.T) {
	if got := interfaceMetricKey(3, "in_bps"); got != "if_3_in_bps" {
		t.Fatalf("expected if_3_in_bps, got %s", got)
	}
}

func TestIndexFromOID(t *testing.T) {
	idx, ok := indexFromOID("1.3.6.1.2.1.2.2.1.10.3")
	if !ok || idx != 3 {
		t.Fatalf("expected index 3, got %d (%v)", idx, ok)
	}
	if _, ok := indexFromOID("1.3.6.1.2.1.2.2.1.1.abc"); ok {
		t.Fatal("non-numeric index should fail")
	}
}

func TestOIDSuffix(t *testing.T) {
	if got := oidSuffixOf("1.3.6.1.2.1.25.2.3.1.6.5"); got != "5" {
		t.Fatalf("expected 5, got %s", got)
	}
}
