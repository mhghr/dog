package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"monitoring-platform/packages/shared/agents/agent"
	"monitoring-platform/packages/shared/agents/agent/enrollment"
	"monitoring-platform/packages/shared/agents/agent/identity"
	"monitoring-platform/packages/shared/logging"
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

	agentID, _, _, err := identity.LoadIdentity(cfg.StateDir)
	if err != nil {
		logger.Info("no identity found, enrolling")
		result, enrollErr := enrollment.Enroll(ctx, cfg, version, logger)
		if enrollErr != nil {
			logger.Error("enrollment failed", "error", enrollErr)
			os.Exit(1)
		}

		logger.Info("enrollment successful", "agent_id", result.AgentID)

		if saveErr := identity.SaveIdentity(cfg.StateDir, result.AgentID, result.Certificate, result.PrivateKey); saveErr != nil {
			logger.Error("failed to save identity", "error", saveErr)
			os.Exit(1)
		}

		if result.Certificate == "" {
			logger.Info("identity saved, waiting for admin approval")
			logger.Info("re-run 'probe-agent run' after approval to connect to gateway")
			<-ctx.Done()
			return
		}

		logger.Info("certificate received, connecting to gateway")
		agentID = result.AgentID
	} else {
		logger.Info("identity loaded, connecting to gateway", "agent_id", agentID)
	}

	if err := identity.ClearEnrollmentToken(cfg); err != nil {
		logger.Warn("failed to clear enrollment token from config", "error", err)
	}

	grpcAgent, err := agent.NewGRPCAgent(cfg, agentID, logger)
	if err != nil {
		logger.Error("failed to create gRPC agent", "error", err)
		os.Exit(1)
	}
	defer grpcAgent.Close()

	logger.Info("probe-agent running, connected to gateway")
	logger.Info("gateway address", "address", cfg.Gateway)

	if err := grpcAgent.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Error("agent stopped with error", "error", err)
		os.Exit(1)
	}

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

	if err := identity.ClearEnrollmentToken(cfg); err != nil {
		logger.Warn("failed to clear enrollment token from config", "error", err)
	}

	logger.Info("identity saved successfully")
	logger.Info("wait for admin approval, then run 'probe-agent run' to connect to gateway")
}

func diagnoseCmd() {
	cfg := loadConfig()
	logger := logging.New(cfg.LogLevel, cfg.LogFormat, "probe-agent")

	fmt.Println("=== probe-agent diagnostics ===")
	fmt.Println()

	fmt.Println("[config]")
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

	fmt.Println()

	_ = logger
}
