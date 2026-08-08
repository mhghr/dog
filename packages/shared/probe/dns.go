package probe

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"

	"monitoring-platform/packages/shared/domain"
)

type DNSExecutor struct {
	deps Deps
}

func NewDNSExecutor(deps Deps) *DNSExecutor {
	return &DNSExecutor{deps: deps}
}

func (e *DNSExecutor) Type() domain.MonitorType {
	return domain.MonitorDNS
}

func (e *DNSExecutor) Execute(ctx context.Context, job domain.ProbeJob) domain.ProbeResult {
	result := newBaseResult(job)

	server := stringConfig(job.Config, "server", "1.1.1.1:53")
	if !strings.Contains(server, ":") || strings.HasSuffix(server, "]") {
		server = net.JoinHostPort(server, "53")
	}

	serverHost, _, err := net.SplitHostPort(server)
	if err != nil {
		return finishFailure(result, "invalid_target", fmt.Errorf("invalid DNS server %q", server))
	}
	if _, err := e.deps.Guard.ResolveAndValidate(ctx, serverHost); err != nil {
		return finishFailure(result, "blocked_target", err)
	}

	recordType := strings.ToUpper(stringConfig(job.Config, "record_type", "A"))
	queryType, ok := dns.StringToType[recordType]
	if !ok {
		return finishFailure(
			result,
			"invalid_record_type",
			fmt.Errorf("unsupported DNS record type: %s", recordType),
		)
	}

	message := new(dns.Msg)
	message.SetQuestion(dns.Fqdn(job.Target), queryType)
	message.RecursionDesired = true

	client := &dns.Client{Net: "udp"}

	response, duration, err := client.ExchangeContext(ctx, message, server)
	if err != nil {
		return finishFailure(result, "dns_query_failed", err)
	}

	if response.Rcode != dns.RcodeSuccess {
		return finishFailure(
			result,
			"dns_rcode_failed",
			fmt.Errorf("DNS returned RCODE %s", dns.RcodeToString[response.Rcode]),
		)
	}

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
		}
	}

	if len(values) == 0 {
		return finishFailure(
			result,
			"empty_dns_answer",
			fmt.Errorf("DNS response contains no matching answers"),
		)
	}

	result.Attributes["answers"] = values
	result.Attributes["rcode"] = dns.RcodeToString[response.Rcode]
	result.Attributes["server"] = server
	result.Metrics["dns_duration_ms"] = duration.Milliseconds()

	expectedValues := stringSliceConfig(job.Config, "expected_values", nil)
	if len(expectedValues) > 0 && !hasCommonValue(values, expectedValues) {
		return finishFailure(
			result,
			"dns_assertion_failed",
			fmt.Errorf("received values %v do not match expected values %v", values, expectedValues),
		)
	}

	return finishSuccess(result)
}

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
