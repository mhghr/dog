// Package snmpfake implements a minimal SNMPv2c agent used by integration
// tests to exercise the collector against a Cisco-like endpoint. It supports
// the system scalars and IF-MIB tables with 32/64-bit counters, optional
// authentication failures, and optional interface state control.
package snmpfake

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
)

// oidString marks a value that must be encoded as an ObjectIdentifier, not an
// OctetString (e.g. sysObjectID).
type oidString string

// OIDString returns a value encoded as an ObjectIdentifier scalar.
func OIDString(oid string) any { return oidString(oid) }

// Scalar holds a device scalar value.
type Scalar struct {
	OID   string
	Value any // string | int64 | uint32 | uint64 | oidString
}

// TableColumn is a table column served by the agent.
type TableColumn struct {
	OID    string // column root, e.g. 1.3.6.1.2.1.2.2.1.10
	Values map[int]any
}

// Agent is a configurable fake SNMPv2c agent.
type Agent struct {
	mu        sync.Mutex
	Community string
	Scalars   []Scalar
	Columns   []TableColumn
	// AuthFail, when set, makes the agent reject requests whose community
	// does not match.
	AuthFail bool
	// OmitColumns, when set, returns no rows for every table column (used to
	// simulate partial collection).
	OmitColumns bool
	// SysObjectID overrides the device's sysObjectID (default: Cisco IOS).
	SysObjectID string
	// OmitSystemColumns, when set, returns no rows for HOST-RESOURCES CPU/RAM
	// columns (system scalars unavailable) while the interface table works.
	OmitSystemColumns bool

	conn net.PacketConn
	// DebugLog, when set, receives one line per request for diagnostics.
	DebugLog func(msg string)
}

// Start binds the agent on 127.0.0.1:0 and returns its UDP address.
func (a *Agent) Start() (*net.UDPAddr, error) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	a.conn = conn
	go a.serve()
	return conn.LocalAddr().(*net.UDPAddr), nil
}

func (a *Agent) Close() {
	if a.conn != nil {
		_ = a.conn.Close()
	}
}

// SysObjectID of a Cisco-like device (Cisco IOS router).
const CiscoIOSObjectID = "1.3.6.1.4.1.9.1.1245"

// LinuxObjectID identifies a generic SNMP host (net-snmp).
const LinuxObjectID = "1.3.6.1.4.1.8072.3.2.10"

func (a *Agent) serve() {
	buf := make([]byte, 65535)
	for {
		n, addr, err := a.conn.ReadFrom(buf)
		if err != nil {
			return
		}
		resp := a.handle(buf[:n])
		if resp != nil {
			_, _ = a.conn.WriteTo(resp, addr)
		}
	}
}

func (a *Agent) handle(req []byte) []byte {
	message, ok := parseSequence(req)
	if !ok {
		return nil
	}
	fields, ok := parseFieldsTagged(message)
	if !ok || len(fields) < 3 {
		return nil
	}
	version, ok := parseInt(fields[0].content)
	if !ok || version != 1 { // v2c
		return nil
	}
	community, ok := parseString(fields[1].content)
	if !ok {
		return nil
	}
	if a.AuthFail && community != a.Community {
		return a.buildResponse(community, requestIDFrom(fields[2].content), 16, 0, nil) // SNMP_ERROR_AUTHORIZATIONERROR
	}

	pdu := fields[2]
	pduFields, ok := parseFieldsTagged(pdu.content)
	if !ok || len(pduFields) < 4 {
		return nil
	}
	requestID, _ := parseInt(pduFields[0].content)
	pduTag := pdu.tag
	varbinds, ok := parseVarbinds(pduFields[3].content)
	if !ok {
		return nil
	}

	if pduTag == 0xA5 { // GetBulk
		maxReps, _ := parseInt(pduFields[2].content)
		if a.DebugLog != nil {
			roots := make([]string, 0, len(varbinds))
			for _, vb := range varbinds {
				roots = append(roots, vb.oid)
			}
			a.DebugLog(fmt.Sprintf("GetBulk roots=%v maxReps=%d", roots, maxReps))
		}
		var out []varbind
		for _, vb := range varbinds {
			if a.OmitColumns {
				out = append(out, varbind{vb.oid, []byte{0x82, 0x00}}) // EndOfMibView
				continue
			}
			rows := a.nextRows(vb.oid, int(maxReps))
			if len(rows) == 0 {
				out = append(out, varbind{vb.oid, []byte{0x82, 0x00}})
				continue
			}
			out = append(out, rows...)
		}
		return a.buildResponse(community, requestID, 0, 0, out)
	}

	// GetRequest
	var out []varbind
	for _, vb := range varbinds {
		if value, ok := a.scalarValue(vb.oid); ok {
			out = append(out, varbind{vb.oid, encodeValue(value)})
		} else {
			out = append(out, varbind{vb.oid, []byte{0x82, 0x00}}) // EndOfMibView/NoSuchObject
		}
	}
	return a.buildResponse(community, requestID, 0, 0, out)
}

func (a *Agent) scalarValue(oid string) (any, bool) {
	for _, s := range a.Scalars {
		if s.OID == oid {
			return s.Value, true
		}
		// SNMP scalar instances are conventionally indexed by .0.
		if s.OID == oid+".0" {
			return s.Value, true
		}
	}
	return nil, false
}

// nextRows returns up to max rows whose OID is lexicographically after the
// given OID within the matching column.
func (a *Agent) nextRows(root string, max int) []varbind {
	for _, col := range a.Columns {
		if !strings.HasPrefix(col.OID, root) && !strings.HasPrefix(root, col.OID) {
			continue
		}
		if a.OmitSystemColumns && strings.HasPrefix(col.OID, "1.3.6.1.2.1.25.") {
			continue
		}
		// Collect matching row OIDs (col.OID.idx) ordered lexicographically.
		var keys []int
		for idx := range col.Values {
			rowOID := col.OID + "." + strconv.Itoa(idx)
			if lexGreater(rowOID, root) {
				keys = append(keys, idx)
			}
		}
		sortInts(keys)
		var out []varbind
		for _, idx := range keys {
			if len(out) >= max {
				break
			}
			out = append(out, varbind{col.OID + "." + strconv.Itoa(idx), encodeValue(col.Values[idx])})
		}
		return out
	}
	return nil
}

func (a *Agent) buildResponse(community string, requestID int64, errorStatus, errorIndex int, varbinds []varbind) []byte {
	var vbContent []byte
	for _, vb := range varbinds {
		vbContent = append(vbContent, encodeSequence(append(encodeOID(vb.oid), vb.value...))...)
	}
	resp := encodeTLV(0x02, encodeIntContent(requestID))
	resp = append(resp, encodeTLV(0x02, encodeIntContent(int64(errorStatus)))...)
	resp = append(resp, encodeTLV(0x02, encodeIntContent(int64(errorIndex)))...)
	resp = append(resp, encodeTLV(0x30, vbContent)...)
	pduBody := encodeTLV(0xA2, resp)

	msg := encodeTLV(0x02, encodeIntContent(1)) // version v2c
	msg = append(msg, encodeTLV(0x04, []byte(community))...)
	msg = append(msg, pduBody...)
	return encodeTLV(0x30, msg)
}

// requestIDFrom extracts the request-id from a PDU content byte slice.
func requestIDFrom(pdu []byte) int64 {
	fields, ok := parseFields(pdu)
	if !ok || len(fields) < 1 {
		return 0
	}
	id, _ := parseInt(fields[0])
	return id
}

// ── BER helpers (subset) ───────────────────────────────────────────────────

type varbind struct {
	oid   string
	value []byte
}

func parseSequence(b []byte) ([]byte, bool) {
	if len(b) < 2 || b[0] != 0x30 {
		return nil, false
	}
	content, _, ok := readTLVContent(b)
	return content, ok
}

func parseFields(content []byte) ([][]byte, bool) {
	var fields [][]byte
	rest := content
	for len(rest) > 0 {
		content, n, ok := readTLVContent(rest)
		if !ok {
			return nil, false
		}
		fields = append(fields, content)
		rest = rest[n:]
	}
	return fields, true
}

type taggedField struct {
	tag     byte
	content []byte
}

// parseFieldsTagged is like parseFields but also returns each TLV tag (needed
// to distinguish GetBulk from Get on the PDU).
func parseFieldsTagged(content []byte) ([]taggedField, bool) {
	var fields []taggedField
	rest := content
	for len(rest) > 0 {
		if len(rest) < 2 {
			return nil, false
		}
		tag := rest[0]
		content, n, ok := readTLVContent(rest)
		if !ok {
			return nil, false
		}
		fields = append(fields, taggedField{tag: tag, content: content})
		rest = rest[n:]
	}
	return fields, true
}

func parseVarbinds(content []byte) ([]varbind, bool) {
	fields, ok := parseFields(content)
	if !ok {
		return nil, false
	}
	var out []varbind
	for _, field := range fields {
		vbs, ok := parseFields(field)
		if !ok || len(vbs) < 2 {
			continue
		}
		oid, ok := parseOID(vbs[0])
		if !ok {
			continue
		}
		out = append(out, varbind{oid: oid, value: vbs[1]})
	}
	return out, true
}

func parseInt(b []byte) (int64, bool) {
	if len(b) == 0 {
		return 0, false
	}
	var v int64
	for _, x := range b {
		v = v<<8 | int64(x)
	}
	return v, true
}

func parseString(b []byte) (string, bool) {
	return string(b), true
}

func parseOID(b []byte) (string, bool) {
	if len(b) == 0 {
		return "", false
	}
	first := int(b[0])
	var parts []string
	if first < 40 {
		parts = []string{"0", strconv.Itoa(first)}
	} else if first < 80 {
		parts = []string{"1", strconv.Itoa(first - 40)}
	} else {
		parts = []string{"2", strconv.Itoa(first - 80)}
	}
	value := 0
	for _, x := range b[1:] {
		value = (value << 7) | int(x&0x7f)
		if x&0x80 == 0 {
			parts = append(parts, strconv.Itoa(value))
			value = 0
		}
	}
	return strings.TrimPrefix(strings.Join(parts, "."), "."), true
}

// readTLVContent returns (content bytes, total bytes consumed, ok).
func readTLVContent(b []byte) ([]byte, int, bool) {
	if len(b) < 2 {
		return nil, 0, false
	}
	length := int(b[1])
	n := 2
	if length&0x80 != 0 {
		num := length & 0x7f
		if num > 4 || 2+num > len(b) {
			return nil, 0, false
		}
		length = 0
		for _, x := range b[2 : 2+num] {
			length = length<<8 | int(x)
		}
		n = 2 + num
	}
	if n+length > len(b) {
		return nil, 0, false
	}
	return b[n : n+length], n + length, true
}

func encodeTLV(tag byte, content []byte) []byte {
	out := []byte{tag}
	switch {
	case len(content) < 128:
		out = append(out, byte(len(content)))
	default:
		lenBytes := make([]byte, 0, 4)
		l := len(content)
		for l > 0 {
			lenBytes = append([]byte{byte(l & 0xff)}, lenBytes...)
			l >>= 8
		}
		out = append(out, byte(0x80|len(lenBytes)))
		out = append(out, lenBytes...)
	}
	return append(out, content...)
}

func encodeSequence(content []byte) []byte { return encodeTLV(0x30, content) }

func encodeOID(oid string) []byte {
	parts := strings.Split(oid, ".")
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n, _ := strconv.Atoi(p)
		nums = append(nums, n)
	}
	if len(nums) < 2 {
		return encodeTLV(0x06, nil)
	}
	var content []byte
	switch {
	case nums[0] == 0:
		content = append(content, byte(nums[1]))
	case nums[0] == 1:
		content = append(content, byte(40+nums[1]))
	default:
		content = append(content, byte(80+nums[1]))
	}
	for _, n := range nums[2:] {
		if n < 128 {
			content = append(content, byte(n))
			continue
		}
		var buf []byte
		for n > 0 {
			buf = append([]byte{byte(n & 0x7f)}, buf...)
			n >>= 7
		}
		for i := 0; i < len(buf)-1; i++ {
			buf[i] |= 0x80
		}
		content = append(content, buf...)
	}
	return encodeTLV(0x06, content)
}

func encodeIntContent(v int64) []byte {
	switch {
	case v == 0:
		return []byte{0}
	case v > 0:
		var out []byte
		for v > 0 {
			out = append([]byte{byte(v & 0xff)}, out...)
			v >>= 8
		}
		if out[0]&0x80 != 0 {
			out = append([]byte{0}, out...)
		}
		return out
	default:
		return []byte{0xff}
	}
}

func encodeValue(value any) []byte {
	switch v := value.(type) {
	case oidString:
		return encodeOID(string(v))
	case string:
		return encodeTLV(0x04, []byte(v))
	case int:
		return encodeTLV(0x02, encodeIntContent(int64(v)))
	case int64:
		return encodeTLV(0x02, encodeIntContent(v))
	case uint32:
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, v)
		return encodeTLV(0x41, b) // Counter32
	case uint64:
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, v)
		return encodeTLV(0x46, b) // Counter64
	default:
		return encodeTLV(0x05, nil) // NULL
	}
}

func lexGreater(a, b string) bool {
	return strings.Compare(a, b) > 0
}

func sortInts(keys []int) {
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
}

// DefaultAgent builds a Cisco-like device with two interfaces.
func DefaultAgent() *Agent {
	sysObjectID := CiscoIOSObjectID
	return &Agent{
		Community:   "public",
		SysObjectID: sysObjectID,
		Scalars: []Scalar{
			{OID: "1.3.6.1.2.1.1.1.0", Value: "Cisco IOS Software, C880 Software (C880DATA-UNIVERSALK9-M), Version 15.2"},
			{OID: "1.3.6.1.2.1.1.2.0", Value: oidString(sysObjectID)},
			{OID: "1.3.6.1.2.1.1.3.0", Value: uint32(360000)}, // sysUpTime in ticks (1h)
			{OID: "1.3.6.1.2.1.1.5.0", Value: "router-core-1"},
		},
		Columns: []TableColumn{
			{OID: "1.3.6.1.2.1.2.2.1.1", Values: map[int]any{1: 1, 2: 2, 3: 3}},
			{OID: "1.3.6.1.2.1.2.2.1.2", Values: map[int]any{1: "GigabitEthernet0/0", 2: "GigabitEthernet0/1", 3: "Loopback0"}},
			{OID: "1.3.6.1.2.1.2.2.1.3", Values: map[int]any{1: 6, 2: 6, 3: 24}},
			{OID: "1.3.6.1.2.1.2.2.1.5", Values: map[int]any{1: int64(1000000000), 2: int64(1000000000), 3: int64(0)}},
			{OID: "1.3.6.1.2.1.2.2.1.7", Values: map[int]any{1: 1, 2: 1, 3: 1}},
			{OID: "1.3.6.1.2.1.2.2.1.8", Values: map[int]any{1: 1, 2: 2, 3: 1}}, // if2 is DOWN
			{OID: "1.3.6.1.2.1.2.2.1.10", Values: map[int]any{1: uint64(1000000), 2: uint64(500000), 3: uint64(0)}},
			{OID: "1.3.6.1.2.1.2.2.1.16", Values: map[int]any{1: uint64(2000000), 2: uint64(600000), 3: uint64(0)}},
			{OID: "1.3.6.1.2.1.2.2.1.11", Values: map[int]any{1: uint64(1000), 2: uint64(500), 3: uint64(0)}},
			{OID: "1.3.6.1.2.1.2.2.1.17", Values: map[int]any{1: uint64(2000), 2: uint64(600), 3: uint64(0)}},
			{OID: "1.3.6.1.2.1.2.2.1.14", Values: map[int]any{1: uint32(0), 2: uint32(3), 3: uint32(0)}},
			{OID: "1.3.6.1.2.1.2.2.1.20", Values: map[int]any{1: uint32(1), 2: uint32(0), 3: uint32(0)}},
			{OID: "1.3.6.1.2.1.2.2.1.13", Values: map[int]any{1: uint32(2), 2: uint32(0), 3: uint32(0)}},
			{OID: "1.3.6.1.2.1.2.2.1.19", Values: map[int]any{1: uint32(0), 2: uint32(1), 3: uint32(0)}},
			{OID: "1.3.6.1.2.1.31.1.1.1.6", Values: map[int]any{1: uint64(1000000), 2: uint64(500000), 3: uint64(0)}},
			{OID: "1.3.6.1.2.1.31.1.1.1.10", Values: map[int]any{1: uint64(2000000), 2: uint64(600000), 3: uint64(0)}},
			{OID: "1.3.6.1.2.1.31.1.1.1.1", Values: map[int]any{1: "Gi0/0", 2: "Gi0/1", 3: "Lo0"}},
			{OID: "1.3.6.1.2.1.31.1.1.1.18", Values: map[int]any{1: "uplink", 2: "downlink", 3: ""}},
			{OID: "1.3.6.1.2.1.25.3.3.1.2", Values: map[int]any{1: 45, 2: 62}},
			{OID: "1.3.6.1.2.1.25.2.3.1.2", Values: map[int]any{1: "1.3.6.1.2.1.25.2.1.7", 2: "1.3.6.1.2.1.25.2.1.2"}},
			{OID: "1.3.6.1.2.1.25.2.3.1.5", Values: map[int]any{1: int64(100000), 2: int64(500000)}},
			{OID: "1.3.6.1.2.1.25.2.3.1.6", Values: map[int]any{1: int64(40000), 2: int64(100000)}},
			// Cisco PROCESS / MEMORY-POOL / ENVMON MIBs for the Cisco provider.
			{OID: "1.3.6.1.4.1.9.9.109.1.1.1.1.6", Values: map[int]any{1: 45, 2: 62}},
			{OID: "1.3.6.1.4.1.9.9.48.1.1.1.2", Values: map[int]any{1: "processor"}},
			{OID: "1.3.6.1.4.1.9.9.48.1.1.1.5", Values: map[int]any{1: int64(40000000)}},
			{OID: "1.3.6.1.4.1.9.9.48.1.1.1.6", Values: map[int]any{1: int64(60000000)}},
			{OID: "1.3.6.1.4.1.9.9.13.1.3.1.3", Values: map[int]any{1: 55, 2: 47}},
			{OID: "1.3.6.1.4.1.9.9.13.1.3.1.5", Values: map[int]any{1: 1, 2: 1}},
			{OID: "1.3.6.1.4.1.9.9.13.1.4.1.2", Values: map[int]any{1: 1}},
			{OID: "1.3.6.1.4.1.9.9.13.1.4.1.3", Values: map[int]any{1: 1}},
			{OID: "1.3.6.1.4.1.9.9.13.1.5.1.2", Values: map[int]any{1: 1}},
			{OID: "1.3.6.1.4.1.9.9.13.1.5.1.3", Values: map[int]any{1: 1}},
		},
	}
}
