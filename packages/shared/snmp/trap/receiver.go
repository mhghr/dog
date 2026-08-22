// Package trap implements the SNMP trap receiver (UDP 162) that turns traps
// into normalized events bound to resources. Polling stays the primary metric
// source; traps accelerate incident detection (linkDown, linkUp, auth
// failures, device restarts) and feed the event stream.
package trap

import (
	"net"

	"github.com/gosnmp/gosnmp"

	"monitoring-platform/packages/shared/domain"
)

// Handler receives a normalized SNMP event. Implementations persist the event
// and route it to the matching resource/interface.
type Handler func(event domain.SNMPEvent)

// RawTrap is the parsed, de-sensitized representation of an SNMP trap. Raw
// packet bytes and credentials never leave the parser.
type RawTrap struct {
	Enterprise   string
	GenericTrap  int
	SpecificTrap int
	AgentAddress string
	SysUpTime    uint32
	Version      string
	Variables    []Variable
}

// Variable is one trap varbind (OID + value).
type Variable struct {
	OID   string
	Value any
}

// Normalizer converts a raw trap into a normalized domain event. Vendor
// providers can extend this (e.g. Cisco-specific traps).
type Normalizer interface {
	// Normalize returns the event for the trap. Resource/Monitor binding is
	// left to the Handler (which may look up the device by address/name).
	Normalize(trap RawTrap) domain.SNMPEvent
}

// Receiver listens on a UDP socket and forwards normalized events.
type Receiver struct {
	listener *gosnmp.TrapListener
	handler  Handler
	norm     Normalizer
	addr     string
}

// NewReceiver builds a trap receiver. addr is the UDP bind address (default
// ":162"); norm defaults to the standard normalizer when nil.
func NewReceiver(addr string, handler Handler, norm Normalizer) *Receiver {
	if norm == nil {
		norm = StandardNormalizer{}
	}
	listener := gosnmp.NewTrapListener()
	listener.Params = gosnmp.Default
	listener.Params.Community = "public"
	listener.OnNewTrap = func(packet *gosnmp.SnmpPacket, addr *net.UDPAddr) {
		handler(norm.Normalize(packetToRaw(packet, addr)))
	}

	return &Receiver{listener: listener, handler: handler, norm: norm, addr: addr}
}

// Start begins listening on the configured address. Binding happens in a
// goroutine; the socket is UDP so Listen returns once bound.
func (r *Receiver) Start() error {
	addr := r.addr
	if addr == "" {
		addr = ":162"
	}
	go func() {
		_ = r.listener.Listen(addr)
	}()
	return nil
}

// Close shuts the receiver down.
func (r *Receiver) Close() {
	r.listener.Close()
}

// packetToRaw extracts the safe fields from a trap packet.
func packetToRaw(packet *gosnmp.SnmpPacket, addr *net.UDPAddr) RawTrap {
	raw := RawTrap{
		Enterprise:   packet.Enterprise,
		GenericTrap:  packet.GenericTrap,
		SpecificTrap: packet.SpecificTrap,
		AgentAddress: packet.AgentAddress,
		SysUpTime:    uint32(packet.Timestamp),
		Version:      "v2c",
	}
	if packet.Version == gosnmp.Version1 {
		raw.Version = "v1"
	} else if packet.Version == gosnmp.Version3 {
		raw.Version = "v3"
	}
	if raw.AgentAddress == "" && addr != nil {
		raw.AgentAddress = addr.IP.String()
	}
	for _, v := range packet.Variables {
		raw.Variables = append(raw.Variables, Variable{OID: v.Name, Value: v.Value})
	}
	return raw
}
