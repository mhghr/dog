package trap

import (
	"testing"

	"github.com/gosnmp/gosnmp"

	"monitoring-platform/packages/shared/domain"
)

func TestNormalizeLinkDownV2(t *testing.T) {
	norm := StandardNormalizer{Vendors: DefaultVendors()}
	event := norm.Normalize(RawTrap{
		Enterprise:   "1.3.6.1.6.3.1.1.5.3",
		GenericTrap:  0,
		AgentAddress: "192.168.1.1",
		Variables: []Variable{
			{OID: "1.3.6.1.2.1.2.2.1.1", Value: 5},
			{OID: "1.3.6.1.2.1.2.2.1.2", Value: []byte("GigabitEthernet0/1")},
			{OID: "1.3.6.1.2.1.2.2.1.7", Value: 1},
			{OID: "1.3.6.1.2.1.2.2.1.8", Value: 2},
		},
	})

	if event.EventType != "linkDown" {
		t.Fatalf("expected linkDown, got %s", event.EventType)
	}
	if event.Severity != "critical" {
		t.Fatalf("expected critical, got %s", event.Severity)
	}
	if event.IfIndex != 5 {
		t.Fatalf("expected ifIndex 5, got %d", event.IfIndex)
	}
	if event.IfName != "GigabitEthernet0/1" {
		t.Fatalf("expected ifName, got %q", event.IfName)
	}
	if event.Kind != domain.SNMPEventTrap {
		t.Fatalf("expected trap kind, got %s", event.Kind)
	}
}

func TestNormalizeLinkUpV1Generic(t *testing.T) {
	norm := StandardNormalizer{Vendors: DefaultVendors()}
	event := norm.Normalize(RawTrap{
		Version:      "v1",
		GenericTrap:  3,
		SpecificTrap: 0,
		Variables: []Variable{
			{OID: "1.3.6.1.2.1.2.2.1.1", Value: 2},
		},
	})
	if event.EventType != "linkUp" {
		t.Fatalf("expected linkUp, got %s", event.EventType)
	}
	if event.Severity != "info" {
		t.Fatalf("expected info, got %s", event.Severity)
	}
	if event.IfIndex != 2 {
		t.Fatalf("expected ifIndex 2, got %d", event.IfIndex)
	}
}

func TestNormalizeAuthFailure(t *testing.T) {
	norm := StandardNormalizer{}
	event := norm.Normalize(RawTrap{Enterprise: "1.3.6.1.6.3.1.1.5.5"})
	if event.EventType != "authenticationFailure" {
		t.Fatalf("expected authenticationFailure, got %s", event.EventType)
	}
	if event.Severity != "critical" {
		t.Fatalf("expected critical, got %s", event.Severity)
	}
}

func TestNormalizeColdWarmStart(t *testing.T) {
	norm := StandardNormalizer{}
	if got := norm.Normalize(RawTrap{Enterprise: "1.3.6.1.6.3.1.1.5.1"}); got.EventType != "coldStart" {
		t.Fatalf("expected coldStart, got %s", got.EventType)
	}
	if got := norm.Normalize(RawTrap{Enterprise: "1.3.6.1.6.3.1.1.5.2"}); got.EventType != "warmStart" {
		t.Fatalf("expected warmStart, got %s", got.EventType)
	}
}

func TestNormalizeCiscoVendorTrap(t *testing.T) {
	norm := StandardNormalizer{Vendors: DefaultVendors()}
	event := norm.Normalize(RawTrap{Enterprise: "1.3.6.1.4.1.9.9.13.3.0.1", AgentAddress: "10.1.1.1"})
	if event.EventType != "cisco.envMon.temperature" {
		t.Fatalf("expected cisco temperature trap, got %s", event.EventType)
	}
	if event.Severity != "critical" {
		t.Fatalf("expected critical, got %s", event.Severity)
	}
}

func TestNormalizeUnknownEnterprise(t *testing.T) {
	norm := StandardNormalizer{Vendors: DefaultVendors()}
	event := norm.Normalize(RawTrap{Enterprise: "1.3.6.1.4.1.9999.1.1", SpecificTrap: 7, AgentAddress: "10.0.0.9"})
	if event.EventType != "enterpriseSpecific" {
		t.Fatalf("expected enterpriseSpecific, got %s", event.EventType)
	}
	if event.Source != "10.0.0.9" {
		t.Fatalf("expected source 10.0.0.9, got %s", event.Source)
	}
}

func TestPacketToRaw(t *testing.T) {
	raw := packetToRaw(&gosnmp.SnmpPacket{
		Version: gosnmp.Version2c,
		SnmpTrap: gosnmp.SnmpTrap{
			Enterprise:   "x",
			GenericTrap:  4,
			SpecificTrap: 0,
			Timestamp:    123,
		},
	}, nil)
	if raw.Enterprise != "x" || raw.GenericTrap != 4 {
		t.Fatalf("raw extraction failed: %+v", raw)
	}
	if raw.Version != "v2c" || raw.SysUpTime != 123 {
		t.Fatalf("raw version/uptime extraction failed: %+v", raw)
	}
}
