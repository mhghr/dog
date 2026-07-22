package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"monitoring-platform/internal/agent"
	"monitoring-platform/internal/agent/enrollment"
	"monitoring-platform/internal/agent/identity"
	"monitoring-platform/internal/logging"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: probe-agent [run|enroll|version|diagnose]\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		run()
	case "enroll":
		enrollCmd()
	case "version":
		fmt.Println("probe-agent version", version)
	case "diagnose":
		diagnoseCmd()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		fmt.Fprintf(os.Stderr, "usage: probe-agent [run|enroll|version|diagnose]\n")
		os.Exit(1)
	}
}

func loadConfig() agent.AgentConfig {
	configPath := os.Getenv("AGENT_CONFIG_PATH")
	if configPath == "" {
		for _, p := range []string{"/etc/probe-agent/config.yaml", "./config.yaml"} {
			if _, err := os.Stat(p); err == nil {
				configPath = p
				break
			}
		}
	}

	cfg, err := agent.LoadAgentConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	return cfg
}

func run() {
	cfg := loadConfig()
	logger := logging.New(cfg.LogLevel, cfg.LogFormat, "probe-agent")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if !identity.HasIdentity(cfg.StateDir) {
		logger.Info("no identity found, enrolling")

		result, err := enrollment.Enroll(ctx, cfg, version, logger)
		if err != nil {
			logger.Error("enrollment failed", "error", err)
			os.Exit(1)
		}

		logger.Info("enrollment successful", "agent_id", result.AgentID)

		if err := identity.SaveIdentity(cfg.StateDir, result.AgentID, result.Certificate, result.PrivateKey); err != nil {
			logger.Error("failed to save identity", "error", err)
			os.Exit(1)
		}

		logger.Info("identity saved, restart to connect to gateway")
		<-ctx.Done()
		return
	}

	agentID, certPEM, keyPEM, err := identity.LoadIdentity(cfg.StateDir)
	if err != nil {
		logger.Error("failed to load identity", "error", err)
		os.Exit(1)
	}

	logger.Info("identity loaded", "agent_id", agentID)
	_ = certPEM
	_ = keyPEM

	<-ctx.Done()
	logger.Info("probe-agent stopped")
}

func enrollCmd() {
	cfg := loadConfig()
	logger := logging.New(cfg.LogLevel, cfg.LogFormat, "probe-agent")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if identity.HasIdentity(cfg.StateDir) {
		logger.Error("identity already exists, remove it first to re-enroll")
		os.Exit(1)
	}

	result, err := enrollment.Enroll(ctx, cfg, version, logger)
	if err != nil {
		logger.Error("enrollment failed", "error", err)
		os.Exit(1)
	}

	logger.Info("enrollment completed", "agent_id", result.AgentID)

	if err := identity.SaveIdentity(cfg.StateDir, result.AgentID, result.Certificate, result.PrivateKey); err != nil {
		logger.Error("failed to save identity", "error", err)
		os.Exit(1)
	}

	logger.Info("identity saved successfully")
}

func diagnoseCmd() {
	cfg := loadConfig()
	logger := logging.New(cfg.LogLevel, cfg.LogFormat, "probe-agent")

	fmt.Println("=== probe-agent diagnostics ===")
	fmt.Println()

	fmt.Println("[config]")
	fmt.Printf("  Config file found: %t\n", cfg.EnrollmentToken != "" || cfg.ControlPlane != "")
	fmt.Printf("  Control plane: %s\n", cfg.ControlPlane)
	fmt.Printf("  Gateway: %s\n", cfg.Gateway)
	fmt.Printf("  State dir: %s\n", cfg.StateDir)
	fmt.Println()

	fmt.Println("[identity]")
	if identity.HasIdentity(cfg.StateDir) {
		agentID, _, _, err := identity.LoadIdentity(cfg.StateDir)
		if err != nil {
			fmt.Printf("  ERROR loading identity: %v\n", err)
		} else {
			fmt.Printf("  Identity found: agent_id=%s\n", agentID)
		}
	} else {
		fmt.Println("  No identity found (will enroll on first run)")
	}

	fmt.Println("[disk]")
	if info, err := os.Stat(cfg.StateDir); err == nil {
		fmt.Printf("  State directory: %s (exists=%t)\n", cfg.StateDir, info.IsDir())
	} else {
		fmt.Printf("  State directory: %s (does not exist yet)\n", cfg.StateDir)
	}

	fmt.Println()
	_ = logger
}
