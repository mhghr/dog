package probe

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// stringConn is an in-memory net.Conn that returns a fixed response then EOF.
type stringConn struct {
	reader io.Reader
}

func newStringConn(response string) net.Conn {
	return &stringConn{reader: strings.NewReader(response)}
}

func (c *stringConn) Read(b []byte) (int, error)         { return c.reader.Read(b) }
func (c *stringConn) Write(b []byte) (int, error)        { return len(b), nil }
func (c *stringConn) Close() error                       { return nil }
func (c *stringConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (c *stringConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (c *stringConn) SetDeadline(t time.Time) error      { return nil }
func (c *stringConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *stringConn) SetWriteDeadline(t time.Time) error { return nil }

func TestParseVcardFullName(t *testing.T) {
	fn := `["vcard",[["version",{},"text","4.0"],["fn",{},"text","Example Registrar"]]]`
	if got := parseVcardFullName(json.RawMessage(fn)); got != "Example Registrar" {
		t.Errorf("expected 'Example Registrar', got %q", got)
	}

	if got := parseVcardFullName(nil); got != "" {
		t.Errorf("expected empty for nil, got %q", got)
	}
	if got := parseVcardFullName(json.RawMessage(`{}`)); got != "" {
		t.Errorf("expected empty for malformed, got %q", got)
	}
	if got := parseVcardFullName(json.RawMessage(`["vcard",[["email",{},"text","x@y.z"]]]`)); got != "" {
		t.Errorf("expected empty when no fn property, got %q", got)
	}
}

func TestRdapRegistrarName(t *testing.T) {
	entity := rdapEntity{
		Roles:      []string{"registrar"},
		VcardArray: json.RawMessage(`["vcard",[["fn",{},"text","Acme Registrar"]]]`),
	}
	if got := rdapRegistrarName(entity); got != "Acme Registrar" {
		t.Errorf("expected 'Acme Registrar', got %q", got)
	}

	if got := rdapRegistrarName(rdapEntity{Roles: []string{"registrant"}, VcardArray: entity.VcardArray}); got != "" {
		t.Errorf("expected empty for non-registrar role, got %q", got)
	}
}

func TestRdapInfoFromPayload(t *testing.T) {
	expiration := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
	payload := rdapDomain{
		Events: []struct {
			EventAction string    `json:"eventAction"`
			EventDate   time.Time `json:"eventDate"`
		}{
			{EventAction: "expiration", EventDate: expiration},
			{EventAction: "registration", EventDate: time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC)},
			{EventAction: "last changed", EventDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)},
		},
		Status: []string{"active"},
		Entities: []rdapEntity{{
			Roles:      []string{"registrar"},
			VcardArray: json.RawMessage(`["vcard",[["fn",{},"text","Acme Registrar"]]]`),
		}},
		Nameservers: []struct {
			LdhName string `json:"ldhName"`
		}{
			{LdhName: "ns1.example.com."},
			{LdhName: "NS2.EXAMPLE.COM."},
		},
	}

	info := rdapInfoFromPayload(payload)

	if !info.Registered {
		t.Error("expected registered=true")
	}
	if info.Source != "rdap" {
		t.Errorf("expected source rdap, got %q", info.Source)
	}
	if info.ExpiresAt == nil || !info.ExpiresAt.Equal(expiration) {
		t.Errorf("expected expiration %v, got %v", expiration, info.ExpiresAt)
	}
	if info.CreatedAt == nil || info.UpdatedAt == nil {
		t.Error("expected created and updated dates")
	}
	if info.Registrar != "Acme Registrar" {
		t.Errorf("expected 'Acme Registrar', got %q", info.Registrar)
	}
	if len(info.Nameservers) != 2 || info.Nameservers[0] != "ns1.example.com" || info.Nameservers[1] != "ns2.example.com" {
		t.Errorf("unexpected nameservers %v", info.Nameservers)
	}
}

func TestApplyWhoisExpiry(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"Registry Expiry Date: 2027-01-15T23:59:59Z", "2027-01-15T23:59:59Z"},
		{"Expiry Date: 2026-12-31", "2026-12-31T00:00:00Z"},
		{"paid-till: 2030.01.01 00:00:00", "2030-01-01T00:00:00Z"},
		{"Some other field: not a date", ""},
	}

	for _, tc := range cases {
		info := applyWhoisExpiry(domainInfo{}, tc.line)
		if tc.want == "" {
			if info.ExpiresAt != nil {
				t.Errorf("line %q: expected no expiry, got %v", tc.line, info.ExpiresAt)
			}
			continue
		}
		if info.ExpiresAt == nil {
			t.Errorf("line %q: expected expiry, got nil", tc.line)
			continue
		}
		if got := info.ExpiresAt.Format(time.RFC3339); got != tc.want {
			t.Errorf("line %q: expected %s, got %s", tc.line, tc.want, got)
		}
	}

	alreadySet := time.Now()
	if got := applyWhoisExpiry(domainInfo{ExpiresAt: &alreadySet}, "Registry Expiry Date: 2099-01-01"); got.ExpiresAt != &alreadySet {
		t.Error("expected existing expiry to be preserved")
	}
}

func TestApplyWhoisRegistrar(t *testing.T) {
	info := applyWhoisRegistrar(domainInfo{}, "Registrar: Example LLC")
	if info.Registrar != "Example LLC" {
		t.Errorf("expected 'Example LLC', got %q", info.Registrar)
	}

	info = applyWhoisRegistrar(domainInfo{}, "Name Server: NS1.EXAMPLE.COM")
	if info.Registrar != "" {
		t.Errorf("expected no registrar for nameserver line, got %q", info.Registrar)
	}

	if got := applyWhoisRegistrar(domainInfo{Registrar: "Kept"}, "Registrar: Overwrite"); got.Registrar != "Kept" {
		t.Error("expected existing registrar to be preserved")
	}
}

func TestApplyWhoisNameserver(t *testing.T) {
	info := applyWhoisNameserver(domainInfo{}, "Name Server: NS1.Example.com.")
	if len(info.Nameservers) != 1 || info.Nameservers[0] != "ns1.example.com" {
		t.Errorf("unexpected nameservers %v", info.Nameservers)
	}

	info = applyWhoisNameserver(info, "Nserver: NS2.EXAMPLE.COM")
	if len(info.Nameservers) != 2 || info.Nameservers[1] != "ns2.example.com" {
		t.Errorf("expected second nameserver, got %v", info.Nameservers)
	}

	info = applyWhoisNameserver(info, "Name Server: ns1.example.com")
	if len(info.Nameservers) != 2 {
		t.Errorf("expected duplicate nameserver to be dropped, got %v", info.Nameservers)
	}
}

func TestWhoisInfoFromResponse(t *testing.T) {
	response := strings.Join([]string{
		"Domain Name: EXAMPLE.COM",
		"Registry Expiry Date: 2027-01-15T23:59:59Z",
		"Registrar: Example Registrar, Inc.",
		"Name Server: NS1.EXAMPLE.COM",
		"Name Server: NS2.EXAMPLE.COM",
		"",
	}, "\n")

	info := whoisInfoFromResponse(response)

	if info.Source != "whois" {
		t.Errorf("expected source whois, got %q", info.Source)
	}
	if info.ExpiresAt == nil || info.ExpiresAt.Format(time.RFC3339) != "2027-01-15T23:59:59Z" {
		t.Errorf("unexpected expiry %v", info.ExpiresAt)
	}
	if info.Registrar != "Example Registrar, Inc." {
		t.Errorf("unexpected registrar %q", info.Registrar)
	}
	if len(info.Nameservers) != 2 {
		t.Errorf("unexpected nameservers %v", info.Nameservers)
	}
}

func TestParseWhoisDate(t *testing.T) {
	cases := []struct {
		value string
		want  string
	}{
		{"2027-01-15T23:59:59Z", "2027-01-15T23:59:59Z"},
		{"2027-01-15T15:04:05+07:00", "2027-01-15T15:04:05+07:00"},
		{"2027-01-15 23:59:59", "2027-01-15T23:59:59Z"},
		{"2027-01-15", "2027-01-15T00:00:00Z"},
		{"15-Jan-2027", "2027-01-15T00:00:00Z"},
		{"2027.01.15 23:59:59", "2027-01-15T23:59:59Z"},
		{"2027.01.15", "2027-01-15T00:00:00Z"},
		{"not a date", ""},
	}

	for _, tc := range cases {
		got, ok := parseWhoisDate(tc.value)
		if tc.want == "" {
			if ok {
				t.Errorf("value %q: expected parse failure", tc.value)
			}
			continue
		}
		if !ok {
			t.Errorf("value %q: expected parse success", tc.value)
			continue
		}
		if formatted := got.Format(time.RFC3339); formatted != tc.want {
			t.Errorf("value %q: expected %s, got %s", tc.value, tc.want, formatted)
		}
	}
}

func TestDomainLookupCache(t *testing.T) {
	cache := newDomainLookupCache(time.Hour)
	info := domainInfo{Registered: true, Registrar: "Acme"}

	if _, _, ok := cache.get("example.com"); ok {
		t.Error("expected cache miss on first get")
	}

	cache.set("example.com", info, nil)
	got, err, ok := cache.get("example.com")
	if !ok || err != nil || got.Registrar != "Acme" {
		t.Errorf("expected cached info, got ok=%v err=%v info=%+v", ok, err, got)
	}

	expired := newDomainLookupCache(-time.Minute)
	expired.set("example.com", info, nil)
	if _, _, ok := expired.get("example.com"); ok {
		t.Error("expected expired entry to miss")
	}
}

func TestLookupWHOIS(t *testing.T) {
	iana := strings.Join([]string{
		"refer: whois.example-registrar.net",
		"",
	}, "\n")
	registrar := strings.Join([]string{
		"Domain Name: EXAMPLE.COM",
		"Registry Expiry Date: 2027-01-15T23:59:59Z",
		"Registrar: Example Registrar, Inc.",
		"Name Server: NS1.EXAMPLE.COM",
		"",
	}, "\n")

	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		return newStringConn(map[string]string{
			"whois.iana.org:43":                 iana,
			"whois.example-registrar.net:43":    registrar,
		}[address]), nil
	}

	info, err := lookupWHOIS(context.Background(), dial, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.Registered {
		t.Error("expected registered=true")
	}
	if info.Registrar != "Example Registrar, Inc." {
		t.Errorf("unexpected registrar %q", info.Registrar)
	}
	if info.ExpiresAt == nil || info.ExpiresAt.Format(time.RFC3339) != "2027-01-15T23:59:59Z" {
		t.Errorf("unexpected expiry %v", info.ExpiresAt)
	}
}

func TestLookupWHOISNotRegistered(t *testing.T) {
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		return newStringConn("no match for \"EXAMPLE.COM\"\n"), nil
	}

	_, err := lookupWHOIS(context.Background(), dial, "example.com")
	if _, ok := err.(*errDomainNotRegistered); !ok {
		t.Errorf("expected errDomainNotRegistered, got %v", err)
	}
}
