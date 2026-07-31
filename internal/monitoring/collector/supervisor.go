package collector

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// Supervisor manages the embedded OTel Collector subprocess lifecycle.
type Supervisor struct {
	configDir  string
	binaryPath string
	configPath string
	logger     *slog.Logger
	restartCh  chan struct{}

	mu     sync.Mutex
	cmd    *exec.Cmd
	cancel context.CancelFunc
	done   chan struct{}
}

// NewSupervisor creates a collector supervisor rooted at configDir.
func NewSupervisor(configDir, binaryPath string, logger *slog.Logger) *Supervisor {
	return &Supervisor{
		configDir:  configDir,
		binaryPath: binaryPath,
		configPath: filepath.Join(configDir, "otel-config.yaml"),
		logger:     logger,
		restartCh:  make(chan struct{}, 1),
	}
}

// Start launches the collector process if it is not already running.
func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cmd != nil && s.cmd.Process != nil {
		return fmt.Errorf("collector already running")
	}

	if err := os.MkdirAll(s.configDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	done := make(chan struct{})

	cmd := exec.CommandContext(runCtx, s.binaryPath, "--config", s.configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start collector: %w", err)
	}

	s.cmd = cmd
	s.done = done
	s.logger.Info("collector started", "binary", s.binaryPath, "config", s.configPath)

	go func() {
		err := cmd.Wait()
		close(done)
		s.mu.Lock()
		s.cmd = nil
		s.done = nil
		s.mu.Unlock()

		if err != nil && runCtx.Err() == nil {
			s.logger.Error("collector exited unexpectedly", "error", err)
			select {
			case s.restartCh <- struct{}{}:
			default:
			}
		}
	}()

	return nil
}

// Reload writes the config file and signals the collector to reload (SIGHUP),
// restarting the process if signaling is not possible.
func (s *Supervisor) Reload(configYAML string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.configDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(s.configPath, []byte(configYAML), 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	if s.cmd != nil && s.cmd.Process != nil {
		if err := s.cmd.Process.Signal(syscall.SIGHUP); err != nil {
			s.logger.Warn("failed to signal reload, restarting", "error", err)
			return s.restartLocked()
		}
		s.logger.Info("collector config reloaded")
	}

	return nil
}

// Stop gracefully stops the collector, waiting up to 10 seconds.
func (s *Supervisor) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Signal(syscall.SIGTERM)
		s.waitProcessLocked()
	}
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

// WatchRestarts automatically restarts the collector after an unexpected exit.
func (s *Supervisor) WatchRestarts(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.restartCh:
			s.logger.Info("restarting collector after unexpected exit")
			time.Sleep(2 * time.Second)

			s.mu.Lock()
			err := s.restartLocked()
			s.mu.Unlock()
			if err != nil {
				s.logger.Error("failed to restart collector", "error", err)
			}
		}
	}
}

// restartLocked restarts the collector. Caller must hold s.mu.
func (s *Supervisor) restartLocked() error {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		s.waitProcessLocked()
	}
	if s.cancel != nil {
		s.cancel()
	}

	runCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	done := make(chan struct{})

	cmd := exec.CommandContext(runCtx, s.binaryPath, "--config", s.configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("restart collector: %w", err)
	}
	s.cmd = cmd
	s.done = done
	s.logger.Info("collector restarted")

	go func() {
		err := cmd.Wait()
		close(done)
		s.mu.Lock()
		s.cmd = nil
		s.done = nil
		s.mu.Unlock()

		if err != nil && runCtx.Err() == nil {
			s.logger.Error("collector exited unexpectedly", "error", err)
			select {
			case s.restartCh <- struct{}{}:
			default:
			}
		}
	}()

	return nil
}

// waitProcessLocked waits for the running process to exit, killing it after a
// 10s timeout. Caller must hold s.mu. The process exit is observed through the
// done channel closed by the Start/restart goroutine, so exec.Cmd.Wait is never
// called concurrently.
func (s *Supervisor) waitProcessLocked() {
	if s.done == nil {
		return
	}
	select {
	case <-s.done:
	case <-time.After(10 * time.Second):
		if s.cmd != nil && s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
		<-s.done
	}
}
