package security

import (
	"context"
	"net"
	"testing"
)

func TestValidatePublicIPBlocksReservedRanges(t *testing.T) {
	blocked := []string{
		"127.0.0.1",
		"10.1.2.3",
		"192.168.1.10",
		"172.16.5.5",
		"169.254.169.254",
		"100.64.0.1",
		"0.0.0.0",
		"224.0.0.1",
		"::1",
		"fe80::1",
		"fc00::1",
	}

	for _, address := range blocked {
		if err := ValidatePublicIP(net.ParseIP(address)); err == nil {
			t.Errorf("expected %s to be blocked", address)
		}
	}
}

func TestValidatePublicIPAllowsPublicRanges(t *testing.T) {
	allowed := []string{
		"1.1.1.1",
		"8.8.8.8",
		"203.0.113.10",
		"2606:4700:4700::1111",
	}

	for _, address := range allowed {
		if err := ValidatePublicIP(net.ParseIP(address)); err != nil {
			t.Errorf("expected %s to be allowed, got %v", address, err)
		}
	}
}

func TestGuardAllowPrivateBypassesValidation(t *testing.T) {
	guard := NewGuard(true)

	if _, err := guard.ResolveAndValidate(context.Background(), "127.0.0.1"); err != nil {
		t.Fatalf("expected private IP to be allowed in dev mode, got %v", err)
	}
}

func TestGuardBlocksPrivateResolution(t *testing.T) {
	guard := NewGuard(false)

	if _, err := guard.ResolveAndValidate(context.Background(), "127.0.0.1"); err == nil {
		t.Fatal("expected loopback to be blocked")
	}

	var blockedErr *BlockedTargetError
	_, err := guard.ResolveAndValidate(context.Background(), "169.254.169.254")
	if err == nil {
		t.Fatal("expected metadata IP to be blocked")
	}
	if !asBlocked(err, &blockedErr) {
		t.Fatalf("expected BlockedTargetError, got %T", err)
	}
}

func TestValidateURL(t *testing.T) {
	guard := NewGuard(false)

	cases := []struct {
		url     string
		wantErr bool
	}{
		{"https://example.com", false},
		{"http://example.com:8080/path", false},
		{"ftp://example.com", true},
		{"https://user:pass@example.com", true},
		{"https://", true},
	}

	for _, testCase := range cases {
		_, err := guard.ValidateURL(testCase.url)
		if (err != nil) != testCase.wantErr {
			t.Errorf("ValidateURL(%q) error = %v, wantErr %t", testCase.url, err, testCase.wantErr)
		}
	}
}

func TestParseIPFamily(t *testing.T) {
	if got := ParseIPFamily("ipv4"); got != IPFamilyIPv4 {
		t.Fatalf("expected ipv4, got %s", got)
	}
	if got := ParseIPFamily("ipv6"); got != IPFamilyIPv6 {
		t.Fatalf("expected ipv6, got %s", got)
	}
	if got := ParseIPFamily(""); got != IPFamilyAuto {
		t.Fatalf("expected auto for empty, got %s", got)
	}
	if got := ParseIPFamily("bogus"); got != IPFamilyAuto {
		t.Fatalf("expected auto for unknown, got %s", got)
	}
}

func TestFilterFamily(t *testing.T) {
	v4 := net.ParseIP("1.1.1.1")
	v6 := net.ParseIP("2606:4700:4700::1111")
	all := []net.IP{v4, v6}

	both := filterFamily(all, IPFamilyAuto)
	if len(both) != 2 {
		t.Fatalf("expected both families for auto, got %d", len(both))
	}

	onlyV4 := filterFamily(all, IPFamilyIPv4)
	if len(onlyV4) != 1 || onlyV4[0].To4() == nil {
		t.Fatalf("expected only ipv4, got %v", onlyV4)
	}

	onlyV6 := filterFamily(all, IPFamilyIPv6)
	if len(onlyV6) != 1 || onlyV6[0].To4() != nil {
		t.Fatalf("expected only ipv6, got %v", onlyV6)
	}
}

func asBlocked(err error, target **BlockedTargetError) bool {
	blocked, ok := err.(*BlockedTargetError)
	if ok {
		*target = blocked
	}
	return ok
}
