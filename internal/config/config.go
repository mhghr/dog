package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv        string
	HTTPAddress   string
	HealthAddress string

	DatabaseURL string

	RedisAddress  string
	RedisPassword string
	RedisDB       int

	VictoriaURL string

	WorkerToken       string
	WorkerName        string
	WorkerConcurrency int
	APIBaseURL        string

	SchedulerBatchSize int
	SchedulerInterval  time.Duration

	ProbeLocationID   string
	ProbeLocationCode string

	QueueLocationPrefix string
	QueueStream         string
	QueueGroup          string
	QueueDeadLetter     string
	QueueMaxLen         int64

	CORSAllowedOrigins []string

	LogLevel  string
	LogFormat string

	SSRFAllowPrivate bool
	PingPrivileged   bool

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURI  string

	AuthJWTSecret   string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	CookieDomain    string
	CookieSecure    bool
	OTPDevMode      bool
	SMSProvider     string

	GatewayAddress       string
	GatewayHealthAddress string
	GatewayTLSCertFile   string
	GatewayTLSKeyFile    string
	GatewayCACertFile    string

	AgentControlPlane    string
	AgentGateway         string
	AgentStateDir        string
	AgentEnrollmentToken string
	AgentSpoolDir        string

	AgentSecretEncryptionKey string

	OTELIngestAddress string
	NATSURL           string
}

func Load() *Config {
	loadDotEnv(".env")

	return &Config{
		AppEnv:        getString("APP_ENV", "development"),
		HTTPAddress:   getString("HTTP_ADDRESS", ":5000"),
		HealthAddress: getString("HEALTH_ADDRESS", ":8081"),

		DatabaseURL: getString("DATABASE_URL", ""),

		RedisAddress:  getString("REDIS_ADDRESS", "localhost:6380"),
		RedisPassword: getString("REDIS_PASSWORD", ""),
		RedisDB:       getInt("REDIS_DATABASE", 0),

		VictoriaURL: getString("VICTORIA_URL", "http://localhost:8428"),

		WorkerToken:       getString("WORKER_TOKEN", ""),
		WorkerName:        getString("WORKER_NAME", defaultWorkerName()),
		WorkerConcurrency: getInt("WORKER_CONCURRENCY", 8),
		APIBaseURL:        getString("API_BASE_URL", "http://localhost:5000"),

		SchedulerBatchSize: getInt("SCHEDULER_BATCH_SIZE", 100),
		SchedulerInterval:  getDuration("SCHEDULER_INTERVAL", time.Second),

		ProbeLocationID:   getString("PROBE_LOCATION_ID", ""),
		ProbeLocationCode: getString("PROBE_LOCATION_CODE", "local-dev"),

		QueueLocationPrefix: getString("QUEUE_LOCATION_PREFIX", "probe-jobs"),

		QueueStream:     getString("QUEUE_STREAM", "probe_jobs"),
		QueueGroup:      getString("QUEUE_GROUP", "probe_workers"),
		QueueDeadLetter: getString("QUEUE_DEAD_LETTER", "probe_results_dead_letter"),
		QueueMaxLen:     int64(getInt("QUEUE_MAX_LEN", 100000)),

		CORSAllowedOrigins: getSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:2000", "http://localhost:3000"}),

		LogLevel:  getString("LOG_LEVEL", "info"),
		LogFormat: getString("LOG_FORMAT", "json"),

		SSRFAllowPrivate: getBool("SSRF_ALLOW_PRIVATE", false),
		PingPrivileged:   getBool("PING_PRIVILEGED", true),

		GoogleClientID:     getString("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getString("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURI:  getString("GOOGLE_REDIRECT_URI", "http://localhost:2000/api/auth/callback/google"),

		AuthJWTSecret:   getString("AUTH_JWT_SECRET", "dev-insecure-jwt-secret-change-me"),
		AccessTokenTTL:  getDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: getDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		CookieDomain:    getString("AUTH_COOKIE_DOMAIN", ""),
		CookieSecure:    getBool("AUTH_COOKIE_SECURE", false),
		OTPDevMode:      getBool("OTP_DEV_MODE", strings.EqualFold(getString("APP_ENV", "development"), "development")),
		SMSProvider:     getString("SMS_PROVIDER", "log"),

		GatewayAddress:       getString("GATEWAY_ADDRESS", ":8443"),
		GatewayHealthAddress: getString("GATEWAY_HEALTH_ADDRESS", ":8081"),
		GatewayTLSCertFile:   getString("GATEWAY_TLS_CERT", "/etc/agent-gateway/server.crt"),
		GatewayTLSKeyFile:    getString("GATEWAY_TLS_KEY", "/etc/agent-gateway/server.key"),
		GatewayCACertFile:    getString("GATEWAY_CA_CERT", "/etc/agent-gateway/ca.crt"),

		AgentControlPlane:    getString("AGENT_CONTROL_PLANE", "http://localhost:5000"),
		AgentGateway:         getString("AGENT_GATEWAY", "localhost:8443"),
		AgentStateDir:        getString("AGENT_STATE_DIR", "/var/lib/probe-agent"),
		AgentEnrollmentToken: getString("AGENT_ENROLLMENT_TOKEN", ""),
		AgentSpoolDir:        getString("AGENT_SPOOL_DIR", "/var/lib/probe-agent/spool"),

		AgentSecretEncryptionKey: getString("AGENT_SECRET_ENCRYPTION_KEY", ""),

		OTELIngestAddress: getString("OTEL_INGEST_ADDRESS", ":4318"),
		NATSURL:           getString("NATS_URL", "nats://localhost:4222"),
	}
}

func (c *Config) Require(keys ...string) error {
	missing := make([]string, 0)

	for _, key := range keys {
		switch key {
		case "DATABASE_URL":
			if c.DatabaseURL == "" {
				missing = append(missing, key)
			}
		case "REDIS_ADDRESS":
			if c.RedisAddress == "" {
				missing = append(missing, key)
			}
		case "WORKER_TOKEN":
			if c.WorkerToken == "" {
				missing = append(missing, key)
			}
		case "API_BASE_URL":
			if c.APIBaseURL == "" {
				missing = append(missing, key)
			}
		case "VICTORIA_URL":
			if c.VictoriaURL == "" {
				missing = append(missing, key)
			}
		case "GATEWAY_ADDRESS":
			if c.GatewayAddress == "" {
				missing = append(missing, key)
			}
		case "AGENT_CONTROL_PLANE":
			if c.AgentControlPlane == "" {
				missing = append(missing, key)
			}
		case "AGENT_GATEWAY":
			if c.AgentGateway == "" {
				missing = append(missing, key)
			}
		case "OTEL_INGEST_ADDRESS":
			if c.OTELIngestAddress == "" {
				missing = append(missing, key)
			}
		case "NATS_URL":
			if c.NATSURL == "" {
				missing = append(missing, key)
			}
		case "AGENT_SECRET_ENCRYPTION_KEY":
			if c.AgentSecretEncryptionKey == "" {
				missing = append(missing, key)
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

func (c *Config) IsDevelopment() bool {
	return strings.EqualFold(c.AppEnv, "development")
}

func defaultWorkerName() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		return "worker-unknown"
	}

	return "worker-" + hostname
}

// loadDotEnv loads KEY=VALUE pairs from a local .env file for developer
// convenience. Existing environment variables always win.
func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)

		if key == "" {
			continue
		}

		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}

func getString(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	return strings.TrimSpace(value)
}

func getInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}

	return parsed
}

func getBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}

	return parsed
}

func getDuration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func getSlice(key string, fallback []string) []string {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return fallback
	}

	return result
}
