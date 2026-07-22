package domain

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type MonitorInput struct {
	Name            string         `json:"name"`
	Type            string         `json:"type"`
	Target          string         `json:"target"`
	IntervalSeconds *int           `json:"interval_seconds"`
	TimeoutMillis   *int           `json:"timeout_millis"`
	Retries         *int           `json:"retries"`
	Enabled         *bool          `json:"enabled"`
	Config          map[string]any `json:"config"`
}

type FieldErrors map[string][]string

func (f FieldErrors) add(field, message string) {
	f[field] = append(f[field], message)
}

var allowedConfigKeys = map[MonitorType]map[string]struct{}{
	MonitorHTTP: keySet(
		"method", "headers", "body", "expected_status_codes", "body_contains",
		"follow_redirects", "max_redirects", "verify_tls",
	),
	MonitorTCP:  keySet("port"),
	MonitorDNS:  keySet("server", "record_type", "expected_values"),
	MonitorPing: keySet("packet_count", "packet_interval_millis", "privileged", "warning_latency_millis", "critical_latency_millis", "warning_packet_loss_percent", "critical_packet_loss_percent", "warning_jitter_millis", "critical_jitter_millis"),
	MonitorTLS: keySet(
		"server_name", "port", "verify_chain", "verify_hostname", "minimum_tls_version",
		"warning_days", "critical_days", "expected_issuer_contains", "expected_fingerprint_sha256",
	),
	MonitorDomainExpiration: keySet(
		"warning_days", "critical_days", "check_nameservers",
		"expected_registrar_contains", "expected_nameservers",
	),
	MonitorSMTP: keySet(
		"port", "mode", "ehlo_domain", "expected_banner_contains",
		"require_starttls", "verify_tls", "expected_capabilities",
	),
	MonitorNTP: keySet(
		"port", "version", "max_offset_millis", "max_round_trip_millis",
		"allowed_stratum_min", "allowed_stratum_max", "warning_offset_millis", "warning_round_trip_millis",
	),
}

var sharedThresholdKeys = keySet("warning_duration_millis", "critical_duration_millis")

var validHTTPMethods = map[string]struct{}{
	"GET": {}, "POST": {}, "PUT": {}, "PATCH": {}, "DELETE": {}, "HEAD": {}, "OPTIONS": {},
}

var validDNSRecordTypes = map[string]struct{}{
	"A": {}, "AAAA": {}, "CNAME": {}, "MX": {}, "TXT": {}, "NS": {},
}

var validSMTPModes = map[string]struct{}{
	"plain": {}, "starttls": {}, "implicit_tls": {},
}

// ValidateMonitorInput normalizes and validates a create/update payload.
// It returns a Monitor template (without identity fields) or field errors.
func ValidateMonitorInput(input MonitorInput) (Monitor, FieldErrors) {
	fieldErrors := FieldErrors{}

	name := strings.TrimSpace(input.Name)
	if len(name) < 2 {
		fieldErrors.add("name", "name must contain at least 2 characters")
	}
	if len(name) > 200 {
		fieldErrors.add("name", "name must contain at most 200 characters")
	}

	monitorType, ok := ParseMonitorType(strings.TrimSpace(input.Type))
	if !ok {
		fieldErrors.add("type", fmt.Sprintf("type must be one of %s", joinTypes()))
		return Monitor{}, fieldErrors
	}

	target := strings.TrimSpace(input.Target)
	if target == "" {
		fieldErrors.add("target", "target is required")
	}

	interval := DefaultIntervalSeconds[monitorType]
	if input.IntervalSeconds != nil {
		interval = *input.IntervalSeconds
	}
	if minInterval := MinIntervalSeconds[monitorType]; interval < minInterval {
		fieldErrors.add("interval_seconds", fmt.Sprintf("interval_seconds must be at least %d for type %s", minInterval, monitorType))
	}
	if interval > 86400*7 {
		fieldErrors.add("interval_seconds", "interval_seconds must be at most 604800")
	}

	timeout := 5000
	if input.TimeoutMillis != nil {
		timeout = *input.TimeoutMillis
	}
	if timeout < 100 || timeout > 60000 {
		fieldErrors.add("timeout_millis", "timeout_millis must be between 100 and 60000")
	}

	retries := 1
	if input.Retries != nil {
		retries = *input.Retries
	}
	if retries < 0 || retries > 5 {
		fieldErrors.add("retries", "retries must be between 0 and 5")
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	monitorConfig := input.Config
	if monitorConfig == nil {
		monitorConfig = map[string]any{}
	}

	if allowed, exists := allowedConfigKeys[monitorType]; exists {
		for key := range monitorConfig {
			_, shared := sharedThresholdKeys[key]
			if _, known := allowed[key]; !known && !shared {
				fieldErrors.add("config."+key, "unknown configuration key for this monitor type")
			}
		}
	}
	validateIncreasingThresholds(monitorConfig, "warning_duration_millis", "critical_duration_millis", 1, 60000, fieldErrors)
	validateIncreasingThresholds(monitorConfig, "warning_packet_loss_percent", "critical_packet_loss_percent", 0, 100, fieldErrors)
	validateIncreasingThresholds(monitorConfig, "warning_jitter_millis", "critical_jitter_millis", 0, 60000, fieldErrors)

	if target != "" {
		validateTargetAndConfig(monitorType, target, monitorConfig, fieldErrors)
	}

	if len(fieldErrors) > 0 {
		return Monitor{}, fieldErrors
	}

	return Monitor{
		Name:            name,
		Type:            monitorType,
		Target:          target,
		IntervalSeconds: interval,
		TimeoutMillis:   timeout,
		Retries:         retries,
		Enabled:         enabled,
		Config:          monitorConfig,
		LastStatus:      StatusUnknown,
		NextRunAt:       time.Now().UTC(),
	}, nil
}

func validateIncreasingThresholds(cfg map[string]any, warningKey, criticalKey string, min, max int, fieldErrors FieldErrors) {
	wRaw, hasWarning := cfg[warningKey]
	cRaw, hasCritical := cfg[criticalKey]
	warning, warningOK := toInt(wRaw)
	critical, criticalOK := toInt(cRaw)
	if hasWarning && (!warningOK || warning < min || warning > max) {
		fieldErrors.add("config."+warningKey, fmt.Sprintf("%s must be between %d and %d", warningKey, min, max))
	}
	if hasCritical && (!criticalOK || critical < min || critical > max) {
		fieldErrors.add("config."+criticalKey, fmt.Sprintf("%s must be between %d and %d", criticalKey, min, max))
	}
	if hasWarning && warningOK && hasCritical && criticalOK && critical <= warning {
		fieldErrors.add("config."+criticalKey, criticalKey+" must be greater than "+warningKey)
	}
}

func validateTargetAndConfig(monitorType MonitorType, target string, cfg map[string]any, fieldErrors FieldErrors) {
	switch monitorType {
	case MonitorHTTP:
		validateHTTP(target, cfg, fieldErrors)
	case MonitorTCP:
		validateTCP(target, cfg, fieldErrors)
	case MonitorDNS:
		validateDNS(target, cfg, fieldErrors)
	case MonitorPing:
		validatePing(target, cfg, fieldErrors)
	case MonitorTLS:
		validateTLS(target, cfg, fieldErrors)
	case MonitorDomainExpiration:
		validateDomainExpiration(target, cfg, fieldErrors)
	case MonitorSMTP:
		validateSMTP(target, cfg, fieldErrors)
	case MonitorNTP:
		validateNTP(target, cfg, fieldErrors)
	}
}

func validateHTTP(target string, cfg map[string]any, fieldErrors FieldErrors) {
	parsed, err := url.Parse(target)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		fieldErrors.add("target", "HTTP target must be a valid URL starting with http:// or https://")
		return
	}
	if parsed.User != nil {
		fieldErrors.add("target", "URLs with embedded credentials are not allowed")
	}

	if method, ok := configString(cfg, "method"); ok {
		if _, valid := validHTTPMethods[strings.ToUpper(method)]; !valid {
			fieldErrors.add("config.method", "method must be a valid HTTP method")
		}
	}

	if codes, exists := cfg["expected_status_codes"]; exists {
		values, ok := codes.([]any)
		if !ok || len(values) == 0 {
			fieldErrors.add("config.expected_status_codes", "expected_status_codes must be a non-empty array of status codes")
		} else {
			for _, raw := range values {
				code, ok := toInt(raw)
				if !ok || code < 100 || code > 599 {
					fieldErrors.add("config.expected_status_codes", "status codes must be integers between 100 and 599")
					break
				}
			}
		}
	}

	if maxRedirects, exists := cfg["max_redirects"]; exists {
		value, ok := toInt(maxRedirects)
		if !ok || value < 0 || value > 20 {
			fieldErrors.add("config.max_redirects", "max_redirects must be between 0 and 20")
		}
	}

	if headers, exists := cfg["headers"]; exists {
		if _, ok := headers.(map[string]any); !ok {
			fieldErrors.add("config.headers", "headers must be an object of header name/value pairs")
		}
	}
}

func validateTCP(target string, cfg map[string]any, fieldErrors FieldErrors) {
	host := target
	if strings.Contains(target, "/") {
		fieldErrors.add("target", "TCP target must be a hostname or IP address, optionally with :port")
		return
	}

	if h, p, err := net.SplitHostPort(target); err == nil {
		host = h
		if !validPortString(p) {
			fieldErrors.add("target", "TCP target port must be between 1 and 65535")
		}
	} else if portRaw, exists := cfg["port"]; exists {
		if !validPortValue(portRaw) {
			fieldErrors.add("config.port", "port must be between 1 and 65535")
		}
	} else {
		fieldErrors.add("config.port", "TCP monitors require a port in the target (host:port) or config.port")
	}

	if !validHost(host) {
		fieldErrors.add("target", "TCP target must be a valid hostname or IP address")
	}
}

func validateDNS(target string, cfg map[string]any, fieldErrors FieldErrors) {
	if !validHost(target) {
		fieldErrors.add("target", "DNS target must be a valid domain name")
	}

	if recordType, ok := configString(cfg, "record_type"); ok {
		if _, valid := validDNSRecordTypes[strings.ToUpper(recordType)]; !valid {
			fieldErrors.add("config.record_type", "record_type must be one of A, AAAA, CNAME, MX, TXT, NS")
		}
	}

	if server, ok := configString(cfg, "server"); ok {
		host := server
		if h, p, err := net.SplitHostPort(server); err == nil {
			host = h
			if !validPortString(p) {
				fieldErrors.add("config.server", "DNS server port must be between 1 and 65535")
			}
		}
		if !validHost(host) {
			fieldErrors.add("config.server", "DNS server must be a valid host or host:port")
		}
	}
}

func validatePing(target string, cfg map[string]any, fieldErrors FieldErrors) {
	if !validHost(target) {
		fieldErrors.add("target", "Ping target must be a valid hostname or IP address")
	}

	warningRaw, hasWarning := cfg["warning_latency_millis"]
	warning, warningOK := toInt(warningRaw)
	criticalRaw, hasCritical := cfg["critical_latency_millis"]
	critical, criticalOK := toInt(criticalRaw)
	if hasWarning && !warningOK {
		fieldErrors.add("config.warning_latency_millis", "warning_latency_millis must be an integer")
	}
	if hasCritical && !criticalOK {
		fieldErrors.add("config.critical_latency_millis", "critical_latency_millis must be an integer")
	}
	if hasWarning && warningOK && (warning < 1 || warning > 60000) {
		fieldErrors.add("config.warning_latency_millis", "warning_latency_millis must be between 1 and 60000")
	}
	if hasCritical && criticalOK && (critical < 1 || critical > 60000) {
		fieldErrors.add("config.critical_latency_millis", "critical_latency_millis must be between 1 and 60000")
	}
	if hasWarning && warningOK && hasCritical && criticalOK && critical <= warning {
		fieldErrors.add("config.critical_latency_millis", "critical_latency_millis must be greater than warning_latency_millis")
	}

	if packetCount, exists := cfg["packet_count"]; exists {
		value, ok := toInt(packetCount)
		if !ok || value < 1 || value > 20 {
			fieldErrors.add("config.packet_count", "packet_count must be between 1 and 20")
		}
	}

	if packetInterval, exists := cfg["packet_interval_millis"]; exists {
		value, ok := toInt(packetInterval)
		if !ok || value < 10 || value > 10000 {
			fieldErrors.add("config.packet_interval_millis", "packet_interval_millis must be between 10 and 10000")
		}
	}
}

func validateTLS(target string, cfg map[string]any, fieldErrors FieldErrors) {
	if !validHost(target) {
		fieldErrors.add("target", "TLS target must be a valid hostname or IP address")
	}

	if portRaw, exists := cfg["port"]; exists && !validPortValue(portRaw) {
		fieldErrors.add("config.port", "port must be between 1 and 65535")
	}

	if version, ok := configString(cfg, "minimum_tls_version"); ok {
		if version != "1.2" && version != "1.3" {
			fieldErrors.add("config.minimum_tls_version", "minimum_tls_version must be 1.2 or 1.3")
		}
	}

	validateThresholdPair(cfg, fieldErrors, 1, 3650)
}

func validateDomainExpiration(target string, cfg map[string]any, fieldErrors FieldErrors) {
	if !validHost(target) || net.ParseIP(target) != nil || !strings.Contains(target, ".") {
		fieldErrors.add("target", "target must be a registrable domain name")
	}

	validateThresholdPair(cfg, fieldErrors, 1, 3650)

	if nameservers, exists := cfg["expected_nameservers"]; exists {
		if _, ok := nameservers.([]any); !ok {
			fieldErrors.add("config.expected_nameservers", "expected_nameservers must be an array of hostnames")
		}
	}
}

func validateSMTP(target string, cfg map[string]any, fieldErrors FieldErrors) {
	if !validHost(target) {
		fieldErrors.add("target", "SMTP target must be a valid hostname or IP address")
	}

	if portRaw, exists := cfg["port"]; exists && !validPortValue(portRaw) {
		fieldErrors.add("config.port", "port must be between 1 and 65535")
	}

	if mode, ok := configString(cfg, "mode"); ok {
		if _, valid := validSMTPModes[strings.ToLower(mode)]; !valid {
			fieldErrors.add("config.mode", "mode must be one of plain, starttls, implicit_tls")
		}
	}

	if ehloDomain, ok := configString(cfg, "ehlo_domain"); ok && !validHost(ehloDomain) {
		fieldErrors.add("config.ehlo_domain", "ehlo_domain must be a valid hostname")
	}
}

func validateNTP(target string, cfg map[string]any, fieldErrors FieldErrors) {
	if !validHost(target) {
		fieldErrors.add("target", "NTP target must be a valid hostname or IP address")
	}

	if portRaw, exists := cfg["port"]; exists && !validPortValue(portRaw) {
		fieldErrors.add("config.port", "port must be between 1 and 65535")
	}

	if version, exists := cfg["version"]; exists {
		value, ok := toInt(version)
		if !ok || (value != 3 && value != 4) {
			fieldErrors.add("config.version", "version must be 3 or 4")
		}
	}

	for _, key := range []string{"max_offset_millis", "max_round_trip_millis"} {
		if raw, exists := cfg[key]; exists {
			value, ok := toInt(raw)
			if !ok || value < 1 || value > 600000 {
				fieldErrors.add("config."+key, key+" must be between 1 and 600000")
			}
		}
	}

	minStratum, hasMin := 1, false
	maxStratum, hasMax := 15, false
	if raw, exists := cfg["allowed_stratum_min"]; exists {
		value, ok := toInt(raw)
		if !ok || value < 1 || value > 16 {
			fieldErrors.add("config.allowed_stratum_min", "allowed_stratum_min must be between 1 and 16")
		} else {
			minStratum, hasMin = value, true
		}
	}
	if raw, exists := cfg["allowed_stratum_max"]; exists {
		value, ok := toInt(raw)
		if !ok || value < 1 || value > 16 {
			fieldErrors.add("config.allowed_stratum_max", "allowed_stratum_max must be between 1 and 16")
		} else {
			maxStratum, hasMax = value, true
		}
	}
	if hasMin && hasMax && minStratum > maxStratum {
		fieldErrors.add("config.allowed_stratum_min", "allowed_stratum_min must not exceed allowed_stratum_max")
	}
}

func validateThresholdPair(cfg map[string]any, fieldErrors FieldErrors, min, max int) {
	warning, hasWarning := 0, false
	critical, hasCritical := 0, false

	if raw, exists := cfg["warning_days"]; exists {
		value, ok := toInt(raw)
		if !ok || value < min || value > max {
			fieldErrors.add("config.warning_days", fmt.Sprintf("warning_days must be between %d and %d", min, max))
		} else {
			warning, hasWarning = value, true
		}
	}

	if raw, exists := cfg["critical_days"]; exists {
		value, ok := toInt(raw)
		if !ok || value < min || value > max {
			fieldErrors.add("config.critical_days", fmt.Sprintf("critical_days must be between %d and %d", min, max))
		} else {
			critical, hasCritical = value, true
		}
	}

	if hasWarning && hasCritical && critical > warning {
		fieldErrors.add("config.critical_days", "critical_days must be less than or equal to warning_days")
	}
}

func configString(cfg map[string]any, key string) (string, bool) {
	raw, exists := cfg[key]
	if !exists {
		return "", false
	}

	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", false
	}

	return strings.TrimSpace(value), true
}

func toInt(value any) (int, bool) {
	switch number := value.(type) {
	case int:
		return number, true
	case int64:
		return int(number), true
	case float64:
		if number != float64(int(number)) {
			return 0, false
		}
		return int(number), true
	default:
		return 0, false
	}
}

func validPortString(raw string) bool {
	port, err := strconv.Atoi(raw)
	return err == nil && port >= 1 && port <= 65535
}

func validPortValue(raw any) bool {
	switch value := raw.(type) {
	case string:
		return validPortString(value)
	default:
		port, ok := toInt(raw)
		return ok && port >= 1 && port <= 65535
	}
}

func validHost(host string) bool {
	host = strings.TrimSuffix(strings.TrimSpace(host), ".")
	if host == "" || len(host) > 253 {
		return false
	}

	if ip := net.ParseIP(host); ip != nil {
		return true
	}

	labels := strings.Split(host, ".")
	for _, label := range labels {
		if label == "" || len(label) > 63 {
			return false
		}
		for index, char := range label {
			isAlnum := (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')
			isDash := char == '-' && index != 0 && index != len(label)-1
			isUnicode := char > 127
			if !isAlnum && !isDash && !isUnicode {
				return false
			}
		}
	}

	return true
}

func keySet(keys ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		set[key] = struct{}{}
	}
	return set
}

func joinTypes() string {
	names := make([]string, 0, len(AllMonitorTypes))
	for _, monitorType := range AllMonitorTypes {
		names = append(names, string(monitorType))
	}
	return strings.Join(names, ", ")
}
