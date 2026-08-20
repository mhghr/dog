// Package security implements SSRF protection for agentless probes.
//
// Every outbound probe connection must go through Guard so that targets
// resolving to private, loopback, link-local, or otherwise reserved address
// space are rejected both at validation time and at dial time (post-DNS,
// post-redirect), closing TOCTOU gaps.
package security

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

var blockedCIDRs = mustParseCIDRs(
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"::/128",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
	"ff00::/8",
)

type BlockedTargetError struct {
	Host   string
	IP     net.IP
	Reason string
}

func (e *BlockedTargetError) Error() string {
	if e.IP != nil {
		return fmt.Sprintf("target %q (%s) is blocked: %s", e.Host, e.IP, e.Reason)
	}
	return fmt.Sprintf("target %q is blocked: %s", e.Host, e.Reason)
}

type Guard struct {
	allowPrivate bool
	resolver     *net.Resolver
	family       IPFamily
}

// IPFamily restricts which address families a probe may dial. Auto allows
// both IPv4 and IPv6; IPv4/IPv6 force a single family for enterprise
// infrastructure monitoring where the target must be reached over a
// specific network stack.
type IPFamily string

const (
	IPFamilyAuto IPFamily = "auto"
	IPFamilyIPv4 IPFamily = "ipv4"
	IPFamilyIPv6 IPFamily = "ipv6"
)

// ParseIPFamily normalizes a user-supplied family string. Unknown values
// fall back to Auto so a bad config never hard-fails a probe.
func ParseIPFamily(value string) IPFamily {
	switch IPFamily(value) {
	case IPFamilyIPv4, IPFamilyIPv6:
		return IPFamily(value)
	default:
		return IPFamilyAuto
	}
}

func NewGuard(allowPrivate bool) *Guard {
	return &Guard{
		allowPrivate: allowPrivate,
		resolver:     net.DefaultResolver,
		family:       IPFamilyAuto,
	}
}

// WithIPFamily returns a copy of the guard restricted to one address family.
// The shared resolver/allowPrivate settings are preserved.
func (g *Guard) WithIPFamily(family IPFamily) *Guard {
	clone := *g
	clone.family = family
	return &clone
}

// DialContext dials only addresses that pass validation and the configured
// family filter. The connection is always established against a
// pre-validated IP, never a re-resolved name.
func (g *Guard) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", address, err)
	}

	ips, err := g.ResolveAndValidate(ctx, host)
	if err != nil {
		return nil, err
	}

	ips = filterFamily(ips, g.family)
	if len(ips) == 0 {
		return nil, &BlockedTargetError{Host: host, Reason: "no address matches the configured IP family"}
	}

	dialer := &net.Dialer{Timeout: 30 * time.Second}

	var lastErr error
	for _, ip := range ips {
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		lastErr = dialErr

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no reachable address for %q", host)
	}

	return nil, lastErr
}

// filterFamily keeps only the addresses the configured family allows.
func filterFamily(ips []net.IP, family IPFamily) []net.IP {
	if family != IPFamilyIPv4 && family != IPFamilyIPv6 {
		return ips
	}

	filtered := make([]net.IP, 0, len(ips))
	for _, ip := range ips {
		isIPv4 := ip.To4() != nil
		if (family == IPFamilyIPv4 && isIPv4) || (family == IPFamilyIPv6 && !isIPv4) {
			filtered = append(filtered, ip)
		}
	}
	return filtered
}
func (g *Guard) AllowPrivate() bool {
	return g.allowPrivate
}

// ValidatePublicIP rejects IPs inside reserved or private address space.
func ValidatePublicIP(ip net.IP) error {
	if ip == nil {
		return fmt.Errorf("invalid IP address")
	}

	for _, cidr := range blockedCIDRs {
		if cidr.Contains(ip) {
			return fmt.Errorf("target IP %s is not publicly routable", ip.String())
		}
	}

	return nil
}

func (g *Guard) ValidateIP(host string, ip net.IP) error {
	if g.allowPrivate {
		return nil
	}

	if err := ValidatePublicIP(ip); err != nil {
		return &BlockedTargetError{Host: host, IP: ip, Reason: "address is private or reserved"}
	}

	return nil
}

// ResolveAndValidate resolves a hostname and validates every returned IP.
func (g *Guard) ResolveAndValidate(ctx context.Context, host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		if err := g.ValidateIP(host, ip); err != nil {
			return nil, err
		}
		return []net.IP{ip}, nil
	}

	addresses, err := g.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("dns resolution failed for %q: %w", host, err)
	}

	ips := make([]net.IP, 0, len(addresses))
	for _, address := range addresses {
		if err := g.ValidateIP(host, address.IP); err != nil {
			return nil, err
		}
		ips = append(ips, address.IP)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("dns resolution returned no addresses for %q", host)
	}

	return ips, nil
}

// WrapTransport forces an HTTP transport through the guarded dialer.
func (g *Guard) WrapTransport(transport *http.Transport) *http.Transport {
	transport.DialContext = g.DialContext
	transport.Proxy = nil
	return transport
}

// ValidateURL enforces scheme, credential, and port restrictions for probe URLs.
func (g *Guard) ValidateURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("URL scheme %q is not allowed", parsed.Scheme)
	}

	if parsed.User != nil {
		return nil, &BlockedTargetError{Host: parsed.Host, Reason: "URLs with embedded credentials are not allowed"}
	}

	if parsed.Hostname() == "" {
		return nil, fmt.Errorf("URL host is required")
	}

	if portRaw := parsed.Port(); portRaw != "" {
		port, err := net.LookupPort("tcp", portRaw)
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("URL port %q is not allowed", portRaw)
		}
	}

	return parsed, nil
}

func mustParseCIDRs(values ...string) []*net.IPNet {
	parsed := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		_, cidr, err := net.ParseCIDR(value)
		if err != nil {
			panic(fmt.Sprintf("invalid CIDR %q: %v", value, err))
		}
		parsed = append(parsed, cidr)
	}
	return parsed
}
