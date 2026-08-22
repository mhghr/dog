package health

func ptr(v float64) *float64 { return &v }

var AllParameters = map[string][]ParameterDefinition{
	"ping":               PingParameters,
	"http":               HTTPParameters,
	"tcp":                TCPParameters,
	"dns":                DNSParameters,
	"tls":                SSLParameters,
	"domain_expiration":  DomainExpirationParameters,
	"smtp":               SMTPParameters,
	"ntp":                NTPParameters,
	"snmp":               SNMPParameters,
}

var PingParameters = []ParameterDefinition{
	{
		Key: "ping.reachability", Name: "Reachability",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
	{
		Key: "ping.rtt.avg_ms", Name: "Average RTT",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
		DefaultWarning: ptr(150.0), DefaultError: ptr(300.0), Recovery: ptr(120.0),
	},
	{
		Key: "ping.rtt.min_ms", Name: "Minimum RTT",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
	},
	{
		Key: "ping.rtt.max_ms", Name: "Maximum RTT",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
	},
	{
		Key: "ping.packet_loss_percent", Name: "Packet Loss",
		DataType: "PERCENTAGE", Direction: "HIGHER_IS_WORSE", Unit: "%",
		DefaultWarning: ptr(5.0), DefaultError: ptr(20.0), Recovery: ptr(3.0),
	},
	{
		Key: "ping.jitter_ms", Name: "Jitter",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
		DefaultWarning: ptr(30.0), DefaultError: ptr(80.0), Recovery: ptr(20.0),
	},
}

var HTTPParameters = []ParameterDefinition{
	{
		Key: "http.reachability", Name: "Reachability",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
	{
		Key: "http.status_code", Name: "Status Code",
		DataType: "ENUM", Direction: "ENUM_STATE", Unit: "",
	},
	{
		Key: "http.total_duration_ms", Name: "Total Duration",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
		DefaultWarning: ptr(2000.0), DefaultError: ptr(5000.0), Recovery: ptr(1500.0),
	},
	{
		Key: "http.ttfb_ms", Name: "Time to First Byte",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
		DefaultWarning: ptr(1000.0), DefaultError: ptr(3000.0), Recovery: ptr(800.0),
	},
	{
		Key: "http.dns_duration_ms", Name: "DNS Duration",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
		DefaultWarning: ptr(500.0), DefaultError: ptr(2000.0), Recovery: ptr(400.0),
	},
	{
		Key: "http.connect_duration_ms", Name: "Connect Duration",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
		DefaultWarning: ptr(500.0), DefaultError: ptr(2000.0), Recovery: ptr(400.0),
	},
	{
		Key: "http.tls_duration_ms", Name: "TLS Duration",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
		DefaultWarning: ptr(500.0), DefaultError: ptr(2000.0), Recovery: ptr(400.0),
	},
	{
		Key: "http.content_assertion", Name: "Content Assertion",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
}

var TCPParameters = []ParameterDefinition{
	{
		Key: "tcp.reachability", Name: "Reachability",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
	{
		Key: "tcp.connect_time_ms", Name: "Connect Time",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
		DefaultWarning: ptr(500.0), DefaultError: ptr(2000.0), Recovery: ptr(400.0),
	},
}

var DNSParameters = []ParameterDefinition{
	{
		Key: "dns.reachability", Name: "Reachability",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
	{
		Key: "dns.response_time_ms", Name: "Response Time",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
		DefaultWarning: ptr(500.0), DefaultError: ptr(2000.0), Recovery: ptr(400.0),
	},
	{
		Key: "dns.answer_count", Name: "Answer Count",
		DataType: "NUMBER", Direction: "NONE", Unit: "",
	},
	{
		Key: "dns.expected_record_match", Name: "Expected Record Match",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
}

var SSLParameters = []ParameterDefinition{
	{
		Key: "ssl.reachability", Name: "Reachability",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
	{
		Key: "ssl.handshake_time_ms", Name: "Handshake Time",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
		DefaultWarning: ptr(500.0), DefaultError: ptr(2000.0), Recovery: ptr(400.0),
	},
	{
		Key: "ssl.certificate_expiry_days", Name: "Certificate Expiry",
		DataType: "NUMBER", Direction: "LOWER_IS_WORSE", Unit: "days",
		DefaultWarning: ptr(30.0), DefaultError: ptr(14.0), Recovery: ptr(60.0),
	},
	{
		Key: "ssl.certificate_valid", Name: "Certificate Valid",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
	{
		Key: "ssl.hostname_match", Name: "Hostname Match",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
	{
		Key: "ssl.chain_valid", Name: "Chain Valid",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
}

var DomainExpirationParameters = []ParameterDefinition{
	{
		Key: "domain_expiration.days_remaining", Name: "Days Remaining",
		DataType: "NUMBER", Direction: "LOWER_IS_WORSE", Unit: "days",
		DefaultWarning: ptr(60.0), DefaultError: ptr(30.0), Recovery: ptr(90.0),
	},
	{
		Key: "domain_expiration.registrar_match", Name: "Registrar Match",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
	{
		Key: "domain_expiration.nameserver_match", Name: "Nameserver Match",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
}

var SMTPParameters = []ParameterDefinition{
	{
		Key: "smtp.reachability", Name: "Reachability",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
	{
		Key: "smtp.banner_match", Name: "Banner Match",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
	{
		Key: "smtp.starttls_available", Name: "STARTTLS Available",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
	{
		Key: "smtp.handshake_duration_ms", Name: "Handshake Duration",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
		DefaultWarning: ptr(2000.0), DefaultError: ptr(5000.0), Recovery: ptr(1500.0),
	},
}

var NTPParameters = []ParameterDefinition{
	{
		Key: "ntp.reachability", Name: "Reachability",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
	{
		Key: "ntp.offset_ms", Name: "Offset",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
		DefaultWarning: ptr(100.0), DefaultError: ptr(500.0), Recovery: ptr(50.0),
	},
	{
		Key: "ntp.round_trip_ms", Name: "Round Trip",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
		DefaultWarning: ptr(500.0), DefaultError: ptr(2000.0), Recovery: ptr(400.0),
	},
	{
		Key: "ntp.jitter_ms", Name: "Jitter",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "ms",
		DefaultWarning: ptr(50.0), DefaultError: ptr(200.0), Recovery: ptr(30.0),
	},
	{
		Key: "ntp.stratum", Name: "Stratum",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "",
		DefaultWarning: ptr(4.0), DefaultError: ptr(5.0), Recovery: ptr(3.0),
	},
}

// SNMPParameters is the health catalog for network-device (SNMP) monitors.
// Per-interface signals are aggregated to the device level: the device is
// critical when a monitored interface is operationally down or utilization
// crosses its threshold. Per-interface detail stays available as raw metrics.
var SNMPParameters = []ParameterDefinition{
	{
		Key: "snmp.reachability", Name: "Reachability",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
	{
		Key: "snmp.device_health", Name: "Device Health",
		DataType: "BOOLEAN", Direction: "BOOLEAN_FAILURE", Unit: "",
	},
	{
		Key: "snmp.cpu_percent", Name: "CPU Utilization",
		DataType: "PERCENTAGE", Direction: "HIGHER_IS_WORSE", Unit: "%",
		DefaultWarning: ptr(80.0), DefaultError: ptr(95.0), Recovery: ptr(70.0),
	},
	{
		Key: "snmp.memory_percent", Name: "Memory Utilization",
		DataType: "PERCENTAGE", Direction: "HIGHER_IS_WORSE", Unit: "%",
		DefaultWarning: ptr(80.0), DefaultError: ptr(95.0), Recovery: ptr(70.0),
	},
	{
		Key: "snmp.temperature_celsius", Name: "Temperature",
		DataType: "NUMBER", Direction: "HIGHER_IS_WORSE", Unit: "°C",
		DefaultWarning: ptr(60.0), DefaultError: ptr(75.0), Recovery: ptr(55.0),
	},
	{
		Key: "snmp.uptime_seconds", Name: "Uptime",
		DataType: "NUMBER", Direction: "NONE", Unit: "s",
	},
	{
		Key: "snmp.interface_oper_status", Name: "Interface Down",
		DataType: "ENUM", Direction: "ENUM_STATE", Unit: "",
	},
	{
		Key: "snmp.interface_utilization_percent", Name: "Interface Utilization",
		DataType: "PERCENTAGE", Direction: "HIGHER_IS_WORSE", Unit: "%",
		DefaultWarning: ptr(80.0), DefaultError: ptr(95.0), Recovery: ptr(70.0),
	},
}
