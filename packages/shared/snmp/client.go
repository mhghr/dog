package snmp

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/security"
)

// AuthProtoName is a supported SNMPv3 authentication protocol name.
type AuthProtoName string

const (
	AuthMD5    AuthProtoName = "MD5"
	AuthSHA    AuthProtoName = "SHA"
	AuthSHA256 AuthProtoName = "SHA-256"
)

// PrivProtoName is a supported SNMPv3 privacy protocol name.
type PrivProtoName string

const (
	PrivDES    PrivProtoName = "DES"
	PrivAES    PrivProtoName = "AES"
	PrivAES256 PrivProtoName = "AES-256"
)

// authProtocol maps a protocol name to gosnmp's authentication protocol.
func authProtocol(name string) (gosnmp.SnmpV3AuthProtocol, bool) {
	switch AuthProtoName(strings.ToUpper(strings.TrimSpace(name))) {
	case AuthMD5:
		return gosnmp.MD5, true
	case AuthSHA:
		return gosnmp.SHA, true
	case AuthSHA256:
		return gosnmp.SHA256, true
	default:
		return gosnmp.NoAuth, false
	}
}

// privProtocol maps a protocol name to gosnmp's privacy protocol.
func privProtocol(name string) (gosnmp.SnmpV3PrivProtocol, bool) {
	switch PrivProtoName(strings.ToUpper(strings.TrimSpace(name))) {
	case PrivDES:
		return gosnmp.DES, true
	case PrivAES:
		return gosnmp.AES, true
	case PrivAES256:
		return gosnmp.AES256, true
	default:
		return gosnmp.NoPriv, false
	}
}

// ConnectParams carries the resolved, decrypted connection parameters.
type ConnectParams struct {
	Host           string
	Port           int
	Version        domain.SNMPVersion
	Transport      string
	Timeout        time.Duration
	Retries        int
	MaxRepetitions int
	Community      string
	Username       string
	SecurityLevel  domain.SNMPSecurityLevel
	AuthProto      string
	AuthSecret     string
	PrivProto      string
	PrivSecret     string
	ContextName    string
}

// DecryptSecretValues decrypts the AES-256-GCM encrypted secret fields of a
// device config. It never returns plaintext in error messages and requires a
// non-empty master key.
func DecryptSecretValues(key string, cfg *domain.SNMPDeviceConfig) error {
	if key == "" {
		return fmt.Errorf("snmp secret decryption key is not configured")
	}
	community, err := security.DecryptSecret(key, cfg.Community)
	if err != nil {
		return fmt.Errorf("snmp community could not be decrypted")
	}
	cfg.Community = community
	if cfg.AuthenticationSecret != "" {
		auth, err := security.DecryptSecret(key, cfg.AuthenticationSecret)
		if err != nil {
			return fmt.Errorf("snmp authentication secret could not be decrypted")
		}
		cfg.AuthenticationSecret = auth
	}
	if cfg.PrivacySecret != "" {
		priv, err := security.DecryptSecret(key, cfg.PrivacySecret)
		if err != nil {
			return fmt.Errorf("snmp privacy secret could not be decrypted")
		}
		cfg.PrivacySecret = priv
	}
	return nil
}

// BuildParams validates a decrypted device config and produces concrete
// connection parameters.
func BuildParams(cfg *domain.SNMPDeviceConfig) (ConnectParams, error) {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		return ConnectParams{}, fmt.Errorf("snmp host is required")
	}
	port := cfg.Port
	if port <= 0 || port > 65535 {
		port = 161
	}
	version := cfg.Version
	if version == "" {
		version = domain.SNMPv2c
	}
	if version != domain.SNMPv1 && version != domain.SNMPv2c && version != domain.SNMPv3 {
		return ConnectParams{}, fmt.Errorf("unsupported snmp version %q", version)
	}
	transport := cfg.Transport
	if transport != "udp" && transport != "tcp" {
		transport = "udp"
	}

	params := ConnectParams{
		Host:           host,
		Port:           port,
		Version:        version,
		Transport:      transport,
		Timeout:        resolveTimeout(cfg.TimeoutSeconds),
		Retries:        normalizeRetries(cfg.Retries),
		MaxRepetitions: normalizeMaxRepetitions(cfg.MaxRepetitions),
		Community:      cfg.Community,
		Username:       cfg.Username,
		SecurityLevel:  cfg.SecurityLevel,
		AuthProto:      cfg.AuthenticationProto,
		AuthSecret:     cfg.AuthenticationSecret,
		PrivProto:      cfg.PrivacyProto,
		PrivSecret:     cfg.PrivacySecret,
		ContextName:    cfg.ContextName,
	}

	if version == domain.SNMPv3 {
		if err := validateV3Params(&params); err != nil {
			return ConnectParams{}, err
		}
	}

	return params, nil
}

func resolveTimeout(timeoutSeconds int) time.Duration {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		return 3 * time.Second
	}
	return timeout
}

func normalizeRetries(retries int) int {
	if retries < 0 {
		return 0
	}
	return retries
}

func normalizeMaxRepetitions(maxReps int) int {
	if maxReps <= 0 {
		return 10
	}
	return maxReps
}

// validateV3Params enforces the SNMPv3 username, auth and privacy requirements
// implied by the requested security level.
func validateV3Params(params *ConnectParams) error {
	if params.Username == "" {
		return fmt.Errorf("snmpv3 username is required")
	}
	level := params.SecurityLevel
	if level == "" {
		level = domain.SNMPNoAuthNoPriv
	}
	if level == domain.SNMPAuthNoPriv || level == domain.SNMPAuthPriv {
		if _, ok := authProtocol(params.AuthProto); !ok {
			return fmt.Errorf("unsupported snmpv3 authentication protocol %q", params.AuthProto)
		}
		if params.AuthSecret == "" {
			return fmt.Errorf("snmpv3 authentication secret is required for %s", level)
		}
	}
	if level == domain.SNMPAuthPriv {
		if _, ok := privProtocol(params.PrivProto); !ok {
			return fmt.Errorf("unsupported snmpv3 privacy protocol %q", params.PrivProto)
		}
		if params.PrivSecret == "" {
			return fmt.Errorf("snmpv3 privacy secret is required for authPriv")
		}
	}
	return nil
}

// NewClient builds a gosnmp.GoSNMP client from concrete parameters. It does
// not connect; connection happens lazily on the first request. The returned
// client is safe to reuse for the duration of one poll cycle.
func NewClient(params ConnectParams) (*gosnmp.GoSNMP, error) {
	var version gosnmp.SnmpVersion
	switch params.Version {
	case domain.SNMPv1:
		version = gosnmp.Version1
	case domain.SNMPv3:
		version = gosnmp.Version3
	default:
		version = gosnmp.Version2c
	}

	client := &gosnmp.GoSNMP{
		Target:         params.Host,
		Port:           uint16(params.Port),
		Transport:      params.Transport,
		Community:      params.Community,
		Version:        version,
		Timeout:        params.Timeout,
		Retries:        params.Retries,
		MaxOids:        params.MaxRepetitions,
		ContextName:    params.ContextName,
		MaxRepetitions: uint32(params.MaxRepetitions),
	}

	if version == gosnmp.Version3 {
		applyV3Security(client, params)
	}

	return client, nil
}

// applyV3Security configures the USM security parameters for an SNMPv3 client
// according to the requested security level.
func applyV3Security(client *gosnmp.GoSNMP, params ConnectParams) {
	sec := &gosnmp.UsmSecurityParameters{}
	sec.UserName = params.Username
	sec.AuthenticationProtocol = gosnmp.NoAuth
	sec.PrivacyProtocol = gosnmp.NoPriv

	var flags gosnmp.SnmpV3MsgFlags = gosnmp.NoAuthNoPriv
	switch params.SecurityLevel {
	case domain.SNMPAuthNoPriv:
		flags = gosnmp.AuthNoPriv
		if proto, ok := authProtocol(params.AuthProto); ok {
			sec.AuthenticationProtocol = proto
		}
		sec.AuthenticationPassphrase = params.AuthSecret
	case domain.SNMPAuthPriv:
		flags = gosnmp.AuthPriv
		if proto, ok := authProtocol(params.AuthProto); ok {
			sec.AuthenticationProtocol = proto
		}
		if proto, ok := privProtocol(params.PrivProto); ok {
			sec.PrivacyProtocol = proto
		}
		sec.AuthenticationPassphrase = params.AuthSecret
		sec.PrivacyPassphrase = params.PrivSecret
	}
	client.SecurityModel = gosnmp.UserSecurityModel
	client.MsgFlags = flags
	client.SecurityParameters = sec
}

// DialErrorState maps a connection-level error to the failure taxonomy.
// Sensitive details are never echoed back.
func DialErrorState(err error) domain.SNMPFailureState {
	if err == nil {
		return domain.SNMPStateSuccess
	}
	msg := strings.ToLower(err.Error())

	if errors.Is(err, net.ErrClosed) {
		return domain.SNMPStateDeviceUnreachable
	}
	switch {
	case strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "no such host"),
		strings.Contains(msg, "network is unreachable"),
		strings.Contains(msg, "host is down"),
		strings.Contains(msg, "no route to host"),
		strings.Contains(msg, "i/o timeout"):
		return domain.SNMPStateDeviceUnreachable
	case strings.Contains(msg, "timeout"),
		strings.Contains(msg, "timed out"):
		return domain.SNMPStateTimeout
	default:
		return domain.SNMPStateDeviceUnreachable
	}
}

// ResponseErrorState maps a gosnmp request-level error to the taxonomy.
func ResponseErrorState(err error) domain.SNMPFailureState {
	if err == nil {
		return domain.SNMPStateSuccess
	}
	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "authentication failure"),
		strings.Contains(msg, "incorrect password"),
		strings.Contains(msg, "community"),
		strings.Contains(msg, "auth failure"),
		strings.Contains(msg, "usm"):
		return domain.SNMPStateAuthentication
	case strings.Contains(msg, "no access"),
		strings.Contains(msg, "authorization"),
		strings.Contains(msg, "not accessible"),
		strings.Contains(msg, "forbidden"):
		return domain.SNMPStateAuthorization
	case strings.Contains(msg, "request timeout"),
		strings.Contains(msg, "timed out"),
		strings.Contains(msg, "no response"):
		return domain.SNMPStateTimeout
	case strings.Contains(msg, "invalid oid"),
		strings.Contains(msg, "nosuchname"),
		strings.Contains(msg, "no such name"):
		return domain.SNMPStateInvalidOID
	case strings.Contains(msg, "unsupported"),
		strings.Contains(msg, "nosuchobject"),
		strings.Contains(msg, "no such object"):
		return domain.SNMPStateUnsupported
	case strings.Contains(msg, "unknown snmp packet"),
		strings.Contains(msg, "malformed"):
		return domain.SNMPStateInternalError
	default:
		return domain.SNMPStateInternalError
	}
}
