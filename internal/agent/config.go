package agent

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type AgentConfig struct {
	ControlPlane    string `yaml:"control_plane"`
	Gateway         string `yaml:"agent_gateway"`
	StateDir        string `yaml:"state_dir"`
	EnrollmentToken string `yaml:"enrollment_token"`

	HealthAddress string `yaml:"health_address"`
	LogLevel      string `yaml:"log_level"`
	LogFormat     string `yaml:"log_format"`

	WorkerName        string `yaml:"worker_name"`
	WorkerConcurrency int    `yaml:"worker_concurrency"`

	SpoolDir string `yaml:"spool_dir"`

	ProbeLocationID string `yaml:"probe_location_id"`

	PingPrivileged   bool `yaml:"ping_privileged"`
	SSRFAllowPrivate bool `yaml:"ssrf_allow_private"`
}

func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		ControlPlane:      "http://localhost:5000",
		Gateway:           "localhost:8443",
		StateDir:          "/var/lib/probe-agent",
		EnrollmentToken:   "",
		HealthAddress:     ":8081",
		LogLevel:          "info",
		LogFormat:         "json",
		WorkerName:        defaultAgentName(),
		WorkerConcurrency: 8,
		SpoolDir:          "/var/lib/probe-agent/spool",
		PingPrivileged:    true,
		SSRFAllowPrivate:  false,
	}
}

func LoadAgentConfig(path string) (AgentConfig, error) {
	cfg := DefaultAgentConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg.applyEnvOverrides()
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config file %s: %w", path, err)
	}

	cfg.applyEnvOverrides()
	return cfg, nil
}

func (c *AgentConfig) applyEnvOverrides() {
	if v := os.Getenv("AGENT_CONTROL_PLANE"); v != "" {
		c.ControlPlane = v
	}
	if v := os.Getenv("AGENT_GATEWAY"); v != "" {
		c.Gateway = v
	}
	if v := os.Getenv("AGENT_STATE_DIR"); v != "" {
		c.StateDir = v
	}
	if v := os.Getenv("AGENT_ENROLLMENT_TOKEN"); v != "" {
		c.EnrollmentToken = v
	}
	if v := os.Getenv("AGENT_HEALTH_ADDRESS"); v != "" {
		c.HealthAddress = v
	}
	if v := os.Getenv("AGENT_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("AGENT_LOG_FORMAT"); v != "" {
		c.LogFormat = v
	}
	if v := os.Getenv("AGENT_NAME"); v != "" {
		c.WorkerName = v
	}
	if v := os.Getenv("AGENT_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.WorkerConcurrency = n
		}
	}
	if v := os.Getenv("AGENT_SPOOL_DIR"); v != "" {
		c.SpoolDir = v
	}
	if v := os.Getenv("PROBE_LOCATION_ID"); v != "" {
		c.ProbeLocationID = v
	}
}

func defaultAgentName() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		return "agent-unknown"
	}
	return "agent-" + strings.ReplaceAll(hostname, ".", "-")
}
