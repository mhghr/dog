import type { MonitorType } from "@/types/monitor";
import type { ParameterDefinition } from "@/types/health";

const pingParameters: ParameterDefinition[] = [
  {
    key: "reachability",
    name: "Reachability",
    description: "Whether the target responds to ICMP echo requests",
    data_type: "boolean",
    unit: "",
    direction: "BOOLEAN_FAILURE",
    default_profile: "Recommended",
  },
  {
    key: "avg_rtt",
    name: "Average RTT",
    description: "Average round-trip time across all sent packets",
    data_type: "float",
    unit: "ms",
    direction: "HIGHER_IS_WORSE",
    default_profile: "Recommended",
  },
  {
    key: "min_rtt",
    name: "Minimum RTT",
    description: "Shortest round-trip time among sent packets",
    data_type: "float",
    unit: "ms",
    direction: "HIGHER_IS_WORSE",
    default_profile: "Relaxed",
  },
  {
    key: "max_rtt",
    name: "Maximum RTT",
    description: "Longest round-trip time among sent packets",
    data_type: "float",
    unit: "ms",
    direction: "HIGHER_IS_WORSE",
    default_profile: "Recommended",
  },
  {
    key: "packet_loss",
    name: "Packet Loss",
    description: "Percentage of packets lost during the probe",
    data_type: "float",
    unit: "%",
    direction: "HIGHER_IS_WORSE",
    default_profile: "Recommended",
  },
  {
    key: "jitter",
    name: "Jitter",
    description: "Variation in round-trip time between consecutive packets",
    data_type: "float",
    unit: "ms",
    direction: "HIGHER_IS_WORSE",
    default_profile: "Relaxed",
  },
  {
    key: "ttl",
    name: "TTL",
    description: "Time-to-live value of the received ICMP reply",
    data_type: "integer",
    unit: "hops",
    direction: "LOWER_IS_WORSE",
    default_profile: "Relaxed",
  },
];

const httpParameters: ParameterDefinition[] = [
  {
    key: "reachability",
    name: "Reachability",
    description: "Whether the HTTP endpoint responded within the timeout",
    data_type: "boolean",
    unit: "",
    direction: "BOOLEAN_FAILURE",
    default_profile: "Recommended",
  },
  {
    key: "status_code",
    name: "Status Code",
    description: "HTTP response status code",
    data_type: "integer",
    unit: "",
    direction: "ENUM_STATE",
    default_profile: "Recommended",
  },
  {
    key: "total_duration",
    name: "Total Duration",
    description: "Total time from request start to response completion",
    data_type: "float",
    unit: "ms",
    direction: "HIGHER_IS_WORSE",
    default_profile: "Recommended",
  },
  {
    key: "ttfb",
    name: "Time to First Byte",
    description: "Time from request start until the first byte of response",
    data_type: "float",
    unit: "ms",
    direction: "HIGHER_IS_WORSE",
    default_profile: "Recommended",
  },
  {
    key: "dns_time",
    name: "DNS Resolution Time",
    description: "Time spent resolving the hostname",
    data_type: "float",
    unit: "ms",
    direction: "HIGHER_IS_WORSE",
    default_profile: "Relaxed",
  },
  {
    key: "connect_time",
    name: "TCP Connect Time",
    description: "Time to establish the TCP connection",
    data_type: "float",
    unit: "ms",
    direction: "HIGHER_IS_WORSE",
    default_profile: "Relaxed",
  },
  {
    key: "tls_time",
    name: "TLS Handshake Time",
    description: "Time spent on the TLS handshake",
    data_type: "float",
    unit: "ms",
    direction: "HIGHER_IS_WORSE",
    default_profile: "Relaxed",
  },
  {
    key: "content_assertion",
    name: "Content Assertion",
    description: "Whether the response body matches the expected content",
    data_type: "boolean",
    unit: "",
    direction: "BOOLEAN_FAILURE",
    default_profile: "Relaxed",
  },
];

const tcpParameters: ParameterDefinition[] = [
  {
    key: "reachability",
    name: "Reachability",
    description: "Whether the TCP port accepted a connection",
    data_type: "boolean",
    unit: "",
    direction: "BOOLEAN_FAILURE",
    default_profile: "Recommended",
  },
  {
    key: "connect_time",
    name: "Connect Time",
    description: "Time to establish the TCP connection",
    data_type: "float",
    unit: "ms",
    direction: "HIGHER_IS_WORSE",
    default_profile: "Recommended",
  },
];

const dnsParameters: ParameterDefinition[] = [
  {
    key: "reachability",
    name: "Reachability",
    description: "Whether the DNS server responded with a valid answer",
    data_type: "boolean",
    unit: "",
    direction: "BOOLEAN_FAILURE",
    default_profile: "Recommended",
  },
  {
    key: "total_duration",
    name: "Query Duration",
    description: "Total time for the DNS query to complete",
    data_type: "float",
    unit: "ms",
    direction: "HIGHER_IS_WORSE",
    default_profile: "Recommended",
  },
];

const tlsParameters: ParameterDefinition[] = [
  {
    key: "reachability",
    name: "Reachability",
    description: "Whether the TLS handshake completed successfully",
    data_type: "boolean",
    unit: "",
    direction: "BOOLEAN_FAILURE",
    default_profile: "Recommended",
  },
  {
    key: "days_until_expiry",
    name: "Days Until Expiry",
    description: "Number of days before the certificate expires",
    data_type: "integer",
    unit: "days",
    direction: "LOWER_IS_WORSE",
    default_profile: "Recommended",
  },
  {
    key: "connect_time",
    name: "Handshake Duration",
    description: "Total TLS handshake time",
    data_type: "float",
    unit: "ms",
    direction: "HIGHER_IS_WORSE",
    default_profile: "Relaxed",
  },
];

const domainExpirationParameters: ParameterDefinition[] = [
  {
    key: "reachability",
    name: "Reachability",
    description: "Whether the domain WHOIS lookup succeeded",
    data_type: "boolean",
    unit: "",
    direction: "BOOLEAN_FAILURE",
    default_profile: "Recommended",
  },
  {
    key: "days_until_expiry",
    name: "Days Until Expiry",
    description: "Days before the domain registration expires",
    data_type: "integer",
    unit: "days",
    direction: "LOWER_IS_WORSE",
    default_profile: "Recommended",
  },
];

const smtpParameters: ParameterDefinition[] = [
  {
    key: "reachability",
    name: "Reachability",
    description: "Whether the SMTP server accepted the connection",
    data_type: "boolean",
    unit: "",
    direction: "BOOLEAN_FAILURE",
    default_profile: "Recommended",
  },
  {
    key: "connect_time",
    name: "Connect Time",
    description: "Time to establish the connection and receive banner",
    data_type: "float",
    unit: "ms",
    direction: "HIGHER_IS_WORSE",
    default_profile: "Recommended",
  },
];

const ntpParameters: ParameterDefinition[] = [
  {
    key: "reachability",
    name: "Reachability",
    description: "Whether the NTP server responded with a valid time",
    data_type: "boolean",
    unit: "",
    direction: "BOOLEAN_FAILURE",
    default_profile: "Recommended",
  },
  {
    key: "offset",
    name: "Clock Offset",
    description: "Absolute difference between local clock and NTP time",
    data_type: "float",
    unit: "ms",
    direction: "HIGHER_IS_WORSE",
    default_profile: "Recommended",
  },
];

const PARAMETER_CATALOG: Record<MonitorType, ParameterDefinition[]> = {
  ping: pingParameters,
  http: httpParameters,
  tcp: tcpParameters,
  dns: dnsParameters,
  tls: tlsParameters,
  domain_expiration: domainExpirationParameters,
  smtp: smtpParameters,
  ntp: ntpParameters,
};

export function getParametersForType(
  monitorType: MonitorType,
): ParameterDefinition[] {
  return PARAMETER_CATALOG[monitorType] ?? [];
}

export const PROFILE_ORDER: Record<string, { warning: number; error: number }> =
  {
    Sensitive: { warning: 0.5, error: 0.8 },
    Recommended: { warning: 0.7, error: 1.0 },
    Relaxed: { warning: 1.0, error: 1.5 },
  };
