package probe

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/security"
)

// DNSExecutor performs a single DNS query and validates the answer. It
// supports the common record types (A, AAAA, CNAME, MX, TXT, NS) through a
// small per-type table so adding a new record type is a one-line entry, never
// a rewrite. The query may run against the platform's system resolver or
// against an explicitly configured resolver whose address is validated by the
// security Guard before any packet is sent.
type DNSExecutor struct {
	deps Deps
}

func NewDNSExecutor(deps Deps) *DNSExecutor {
	return &DNSExecutor{deps: deps}
}

func (e *DNSExecutor) Type() domain.MonitorType {
	return domain.MonitorDNS
}

// maxDNSUDPPayload caps the EDNS0 advertised UDP payload so a single check
// cannot be abused for amplification or unbounded response sizes. 1232 bytes
// is the modern DNS flag-day recommendation and fits a normal answer.
const maxDNSUDPPayload = 1232

// maxDNSReceiveBytes bounds the receive buffer for a single exchange.
const maxDNSReceiveBytes = 4096

// supportedDNSRecordType restricts the executor to the record types the
// answer extractors and the system resolver path understand. The table is
// intentionally explicit: unknown types fail fast with a config error instead
// of silently querying something the validator cannot interpret.
func supportedDNSRecordType(recordType string) bool {
	switch recordType {
	case "A", "AAAA", "CNAME", "MX", "TXT", "NS":
		return true
	default:
		return false
	}
}

// resolverConfig returns the explicitly configured DNS server, or "" when the
// platform's system resolver should be used. Accepts the legacy config keys
// ("server", "nameserver") beside the current "resolver".
func resolverConfig(config map[string]any) string {
	if resolver := stringConfig(config, "resolver", ""); resolver != "" {
		return resolver
	}
	if server := stringConfig(config, "server", ""); server != "" {
		return server
	}
	return stringConfig(config, "nameserver", "")
}

// expectedAnswers collects the configured expected answer values from the
// supported config shapes (expected_values / expected_answers arrays and the
// singular expected_value string).
func expectedAnswers(config map[string]any) []string {
	if values := stringSliceConfig(config, "expected_values", nil); len(values) > 0 {
		return values
	}
	if values := stringSliceConfig(config, "expected_answers", nil); len(values) > 0 {
		return values
	}
	if value := stringConfig(config, "expected_value", ""); value != "" {
		return []string{value}
	}
	return nil
}

func (e *DNSExecutor) Execute(ctx context.Context, job domain.ProbeJob) domain.ProbeResult {
	result := newBaseResult(job)

	queryName := strings.TrimSuffix(strings.TrimSpace(job.Target), ".")
	if queryName == "" {
		result.Metrics["reachability"] = 0.0
		result.Attributes["error_type"] = "invalid_target"
		return finishFailure(result, "invalid_target", fmt.Errorf("DNS query name is required"))
	}

	recordType := strings.ToUpper(stringConfig(job.Config, "record_type", "A"))
	queryType, known := dns.StringToType[recordType]
	if !known || !supportedDNSRecordType(recordType) {
		result.Metrics["reachability"] = 0.0
		result.Attributes["error_type"] = "invalid_record_type"
		return finishFailure(
			result,
			"invalid_record_type",
			fmt.Errorf("unsupported DNS record type: %s", recordType),
		)
	}

	family := security.ParseIPFamily(stringConfig(job.Config, "ip_version", string(security.IPFamilyAuto)))

	result.Attributes["query_name"] = queryName
	result.Attributes["record_type"] = recordType
	result.Attributes["ip_version"] = string(family)

	if timeout := intConfig(job.Config, "timeout_ms", 0); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
		defer cancel()
	}

	queryStartedAt := time.Now()

	resolver := resolverConfig(job.Config)
	var answers []string
	var rcode string
	var resolverUsed string
	var minTTL int64
	var err error

	if resolver == "" {
		// System resolver path: the platform owns the resolver configuration,
		// so no per-target SSRF check applies here.
		answers, rcode, err = querySystemResolver(ctx, queryName, recordType)
		resolverUsed = "system"
	} else {
		resolverAddress, validateErr := e.validateResolver(ctx, resolver, family)
		if validateErr != nil {
			result.Attributes["resolver"] = resolver
			return finishDNSFailure(result, classifyResolverValidationError(validateErr), validateErr)
		}
		answers, rcode, minTTL, err = queryDirectResolver(ctx, resolverAddress, queryName, queryType)
		resolverUsed = resolverAddress
	}
	result.Attributes["resolver"] = resolverUsed

	responseTime := time.Since(queryStartedAt)

	if err != nil {
		return finishDNSFailure(result, classifyDNSQueryError(err), err)
	}

	switch rcode {
	case "NOERROR":
	case "NXDOMAIN":
		result.Attributes["rcode"] = rcode
		return finishDNSFailure(result, "nxdomain", fmt.Errorf("DNS returned NXDOMAIN for %q", queryName))
	case "SERVFAIL":
		result.Attributes["rcode"] = rcode
		return finishDNSFailure(result, "server_failure", fmt.Errorf("DNS resolver returned SERVFAIL"))
	case "REFUSED":
		result.Attributes["rcode"] = rcode
		return finishDNSFailure(result, "refused", fmt.Errorf("DNS resolver refused the query"))
	default:
		result.Attributes["rcode"] = rcode
		return finishDNSFailure(result, "dns_rcode_failed", fmt.Errorf("DNS returned RCODE %s", rcode))
	}

	result.Attributes["resolver"] = resolverUsed
	result.Attributes["rcode"] = rcode
	result.Attributes["answer_count"] = len(answers)
	result.Attributes["answers"] = answers
	result.Metrics["reachability"] = 1.0
	result.Metrics["response_time_ms"] = float64(responseTime.Milliseconds())
	result.Metrics["answer_count"] = float64(len(answers))
	if minTTL > 0 {
		result.Metrics["ttl_seconds"] = float64(minTTL)
	}

	expected := expectedAnswers(job.Config)
	if len(expected) > 0 {
		if !hasCommonValue(answers, expected) {
			if len(answers) == 0 {
				result.Metrics["expected_record_match"] = 0.0
				return finishDNSFailure(
					result,
					"expected_record_not_found",
					fmt.Errorf("no answers matched the expected values %v", expected),
				)
			}
			result.Metrics["expected_record_match"] = 0.0
			return finishDNSFailure(
				result,
				"answer_mismatch",
				fmt.Errorf("received answers %v do not match the expected values %v", answers, expected),
			)
		}
		result.Metrics["expected_record_match"] = 1.0
	} else if len(answers) == 0 {
		return finishDNSFailure(
			result,
			"no_answer",
			fmt.Errorf("DNS responded with NOERROR but no %s records", recordType),
		)
	}

	return finishSuccess(result)
}

// validateResolver resolves the configured DNS server and validates every
// candidate address through the security Guard. The returned address is
// pinned to a validated IP so a rebinding resolver cannot swap to an internal
// address between validation and the actual query.
func (e *DNSExecutor) validateResolver(ctx context.Context, resolver string, family security.IPFamily) (string, error) {
	host, port, err := net.SplitHostPort(resolver)
	if err != nil {
		host = resolver
		port = "53"
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return "", fmt.Errorf("invalid DNS resolver port %q", port)
	}

	ips, err := e.deps.Guard.ResolveAndValidate(ctx, host)
	if err != nil {
		return "", fmt.Errorf("DNS resolver %q is not allowed: %w", resolver, err)
	}

	pinned := pickFamilyIP(ips, family)
	if pinned == nil {
		return "", fmt.Errorf("DNS resolver %q has no address matching the configured IP family", resolver)
	}

	return net.JoinHostPort(pinned.String(), port), nil
}

// pickFamilyIP selects the first address matching the requested family,
// preferring any available address when the family is auto.
func pickFamilyIP(ips []net.IP, family security.IPFamily) net.IP {
	if family != security.IPFamilyIPv4 && family != security.IPFamilyIPv6 {
		return ips[0]
	}
	for _, ip := range ips {
		isIPv4 := ip.To4() != nil
		if (family == security.IPFamilyIPv4 && isIPv4) || (family == security.IPFamilyIPv6 && !isIPv4) {
			return ip
		}
	}
	return nil
}

// queryDirectResolver performs the DNS query over UDP with a fallback to TCP
// when the answer is truncated. The resolver address is already validated and
// pinned. It returns the extracted answer values, the RCODE, and the minimum
// TTL across answers.
func queryDirectResolver(ctx context.Context, resolverAddress, queryName string, queryType uint16) ([]string, string, int64, error) {
	message := new(dns.Msg)
	message.SetQuestion(dns.Fqdn(queryName), queryType)
	message.RecursionDesired = true
	// Bounded EDNS0 payload: keeps responses small (amplification guard) and
	// makes truncation-over-TCP the well-defined escalation path.
	message.SetEdns0(maxDNSUDPPayload, false)

	client := &dns.Client{
		Net:     "udp",
		Timeout: 5 * time.Second,
		UDPSize: maxDNSReceiveBytes,
	}

	response, _, err := client.ExchangeContext(ctx, message, resolverAddress)
	if err != nil {
		return nil, "", 0, err
	}

	if response.Truncated {
		tcpClient := &dns.Client{
			Net:     "tcp",
			Timeout: 5 * time.Second,
			UDPSize: maxDNSReceiveBytes,
		}
		response, _, err = tcpClient.ExchangeContext(ctx, message, resolverAddress)
		if err != nil {
			return nil, "", 0, err
		}
	}

	rcode := dns.RcodeToString[response.Rcode]
	if response.Rcode != dns.RcodeSuccess {
		return nil, rcode, 0, nil
	}

	answers := extractDNSAnswers(response)
	minTTL := int64(0)
	for _, rr := range response.Answer {
		if ttl := int64(rr.Header().Ttl); minTTL == 0 || ttl < minTTL {
			minTTL = ttl
		}
	}

	return answers, rcode, minTTL, nil
}

// extractDNSAnswers flattens a successful DNS answer into the record values
// the validator understands. Adding a record type only requires a case here.
func extractDNSAnswers(response *dns.Msg) []string {
	values := make([]string, 0, len(response.Answer))
	for _, answer := range response.Answer {
		switch record := answer.(type) {
		case *dns.A:
			values = append(values, record.A.String())
		case *dns.AAAA:
			values = append(values, record.AAAA.String())
		case *dns.CNAME:
			values = append(values, record.Target)
		case *dns.MX:
			values = append(values, record.Mx)
		case *dns.TXT:
			values = append(values, strings.Join(record.Txt, ""))
		case *dns.NS:
			values = append(values, record.Ns)
		case *dns.PTR:
			values = append(values, record.Ptr)
		}
	}
	return values
}

// querySystemResolver resolves the record type through the platform resolver.
func querySystemResolver(ctx context.Context, queryName, recordType string) ([]string, string, error) {
	switch recordType {
	case "A", "AAAA":
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", queryName)
		if err != nil {
			return nil, "", err
		}
		values := make([]string, 0, len(ips))
		for _, ip := range ips {
			if recordType == "A" && ip.To4() == nil {
				continue
			}
			if recordType == "AAAA" && ip.To4() != nil {
				continue
			}
			values = append(values, ip.String())
		}
		return values, "NOERROR", nil
	case "CNAME":
		cname, err := net.DefaultResolver.LookupCNAME(ctx, queryName)
		if err != nil {
			return nil, "", err
		}
		if cname == "" || strings.EqualFold(strings.TrimSuffix(cname, "."), queryName) {
			return nil, "", nil
		}
		return []string{cname}, "NOERROR", nil
	case "MX":
		records, err := net.DefaultResolver.LookupMX(ctx, queryName)
		if err != nil {
			return nil, "", err
		}
		values := make([]string, 0, len(records))
		for _, record := range records {
			values = append(values, record.Host)
		}
		return values, "NOERROR", nil
	case "TXT":
		records, err := net.DefaultResolver.LookupTXT(ctx, queryName)
		if err != nil {
			return nil, "", err
		}
		return records, "NOERROR", nil
	case "NS":
		records, err := net.DefaultResolver.LookupNS(ctx, queryName)
		if err != nil {
			return nil, "", err
		}
		values := make([]string, 0, len(records))
		for _, record := range records {
			values = append(values, record.Host)
		}
		return values, "NOERROR", nil
	default:
		return nil, "", fmt.Errorf("unsupported DNS record type: %s", recordType)
	}
}

// classifyResolverValidationError separates a blocked resolver (policy) from
// a resolver that simply cannot be resolved.
func classifyResolverValidationError(err error) string {
	if isBlockedError(err) {
		return "blocked_target"
	}
	return "resolver_unreachable"
}

// classifyDNSQueryError maps exchange failures and system-resolver errors to
// the deterministic DNS error taxonomy.
func classifyDNSQueryError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		switch {
		case dnsErr.IsNotFound:
			return "nxdomain"
		case dnsErr.IsTimeout:
			return "timeout"
		default:
			return "resolver_unreachable"
		}
	}

	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "bad rdata"), strings.Contains(lower, "bad dns"),
		strings.Contains(lower, "overflow"), strings.Contains(lower, "unpack"),
		strings.Contains(lower, "short read"), strings.Contains(lower, "bad"):
		return "malformed_response"
	case strings.Contains(lower, "i/o timeout"), strings.Contains(lower, "timed out"):
		return "timeout"
	case strings.Contains(lower, "connection refused"):
		return "resolver_unreachable"
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return "resolver_unreachable"
	}

	return "resolver_unreachable"
}

// finishDNSFailure marks a DNS failure with reachability=0 and the classified
// error type.
func finishDNSFailure(result domain.ProbeResult, code string, err error) domain.ProbeResult {
	result.Metrics["reachability"] = 0.0
	result.Attributes["error_type"] = code
	return finishFailure(result, code, err)
}

// hasCommonValue reports whether any actual answer matches any expected value.
// Values are normalized case/dot-insensitively and IPs are compared
// canonically so "2001:0db8::1" and "2001:db8::1" match.
func hasCommonValue(actual, expected []string) bool {
	for _, actualValue := range actual {
		for _, expectedValue := range expected {
			if normalizeDNSValue(actualValue) == normalizeDNSValue(expectedValue) {
				return true
			}

			actualIP := net.ParseIP(actualValue)
			expectedIP := net.ParseIP(expectedValue)
			if actualIP != nil && expectedIP != nil && actualIP.Equal(expectedIP) {
				return true
			}
		}
	}
	return false
}

func normalizeDNSValue(value string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(value), "."))
}
