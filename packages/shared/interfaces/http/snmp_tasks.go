package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"monitoring-platform/packages/shared/config"
	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/ingestion/messagebus"
	"monitoring-platform/packages/shared/repository"
	"monitoring-platform/packages/shared/security"
	snmplib "monitoring-platform/packages/shared/snmp"
)

// SNMPTaskRunner submits an on-demand SNMP operation to the collector. The
// NATS implementation dispatches the task to the SNMP worker (production);
// the inline implementation runs the identical collector code inside the API
// when no bus is available (local development).
type SNMPTaskRunner interface {
	Submit(ctx context.Context, task *domain.SNMPTask) error
	// Start begins any background consumption the runner needs (NATS results).
	// Inline runners return nil.
	Start(ctx context.Context) error
}

// NATSSNMPTaskRunner publishes tasks to the SNMP worker via NATS JetStream and
// consumes the results back.
type NATSSNMPTaskRunner struct {
	bus    *messagebus.NATSBus
	repo   repository.SNMPRepository
	logger *slog.Logger
}

// NewNATSSNMPTaskRunner builds a NATS-backed runner.
func NewNATSSNMPTaskRunner(bus *messagebus.NATSBus, repo repository.SNMPRepository, logger *slog.Logger) *NATSSNMPTaskRunner {
	return &NATSSNMPTaskRunner{bus: bus, repo: repo, logger: logger}
}

const (
	snmpTaskSubject    = "snmp.tasks"
	snmpTaskResultSubject = "snmp.tasks.result"
)

// Submit publishes the task to the worker and subscribes (once) for the result.
func (r *NATSSNMPTaskRunner) Submit(ctx context.Context, task *domain.SNMPTask) error {
	task.ReplySubject = snmpTaskResultSubject + "." + task.TaskID
	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}
	if err := r.bus.Publish(ctx, messagebus.PublishOptions{
		Subject: snmpTaskSubject + "." + task.TaskID,
		Data:    payload,
		Headers: map[string]string{"task_id": task.TaskID, "kind": string(task.Kind)},
	}); err != nil {
		return err
	}
	return nil
}

// Start subscribes to the result subject and persists results.
func (r *NATSSNMPTaskRunner) Start(ctx context.Context) error {
	return r.bus.Subscribe(ctx, messagebus.SubscribeOptions{
		Subject:    snmpTaskResultSubject + ".>",
		Queue:      "snmp-task-results",
		Durable:    "snmp-task-result-store",
		DeliverNew: true,
	}, func(ctx context.Context, msg messagebus.Message) error {
		var result domain.SNMPTaskResult
		if err := json.Unmarshal(msg.Data, &result); err != nil {
			r.logger.Error("snmp task result unmarshal failed", "error", err)
			return nil
		}
		taskID := msg.Headers["task_id"]
		if taskID == "" {
			return nil
		}
		raw, _ := json.Marshal(result)
		status := domain.SNMPTaskFailed
		if result.OK {
			status = domain.SNMPTaskSuccess
		}
		if err := r.repo.FinishTask(ctx, taskID, status, raw, result.Detail); err != nil {
			r.logger.Error("snmp task result persist failed", "error", err, "task_id", taskID)
			return err
		}
		r.logger.Info("snmp task completed", "task_id", taskID, "kind", result.Kind, "ok", result.OK)
		return nil
	})
}

// InlineSNMPTaskRunner runs the task synchronously in a background goroutine.
type InlineSNMPTaskRunner struct {
	repo   repository.SNMPRepository
	key    string
	logger *slog.Logger
}

// NewInlineSNMPTaskRunner builds an inline runner (local development).
func NewInlineSNMPTaskRunner(repo repository.SNMPRepository, key string, logger *slog.Logger) *InlineSNMPTaskRunner {
	return &InlineSNMPTaskRunner{repo: repo, key: key, logger: logger}
}

// Submit runs the task in a goroutine and persists the outcome.
func (r *InlineSNMPTaskRunner) Submit(ctx context.Context, task *domain.SNMPTask) error {
	go func() {
		runCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		if err := r.repo.SetTaskRunning(runCtx, task.TaskID); err != nil {
			r.logger.Error("snmp task running mark failed", "error", err, "task_id", task.TaskID)
		}
		result, err := snmplib.ExecuteTask(runCtx, task.Kind, task.Config, r.key, snmplib.DefaultRegistry())
		raw, _ := json.Marshal(result)
		status := domain.SNMPTaskSuccess
		if err != nil || !result.OK {
			status = domain.SNMPTaskFailed
		}
		if err := r.repo.FinishTask(runCtx, task.TaskID, status, raw, result.Detail); err != nil {
			r.logger.Error("snmp task persist failed", "error", err, "task_id", task.TaskID)
		}
	}()
	return nil
}

// Start is a no-op for the inline runner.
func (r *InlineSNMPTaskRunner) Start(ctx context.Context) error { return nil }

// selectSNMPTaskRunner picks the runner based on configuration: NATS when the
// pipeline is NATS or a dedicated SNMP task URL is provided, inline otherwise.
func selectSNMPTaskRunner(cfg *config.Config, repo repository.SNMPRepository, logger *slog.Logger) (SNMPTaskRunner, bool) {
	natsURL := cfg.NATSURL
	if env := os.Getenv("SNMP_TASK_NATS_URL"); env != "" {
		natsURL = env
	}
	if cfg.TelemetryPipeline.Mode == "nats" || os.Getenv("SNMP_TASK_NATS_URL") != "" {
		bus, err := messagebus.NewNATSBus(messagebus.NATSConfig{URL: natsURL, Reconnect: true, MaxReconn: 10}, logger)
		if err != nil {
			logger.Warn("snmp nats runner unavailable, falling back to inline", "error", err)
		} else {
			runner := NewNATSSNMPTaskRunner(bus, repo, logger)
			return runner, true
		}
	}
	return NewInlineSNMPTaskRunner(repo, cfg.AgentSecretEncryptionKey, logger), false
}

// SelectSNMPTaskRunner is the exported variant used by the API binary.
func SelectSNMPTaskRunner(cfg *config.Config, repo repository.SNMPRepository, logger *slog.Logger) (SNMPTaskRunner, bool) {
	return selectSNMPTaskRunner(cfg, repo, logger)
}

// ── Handlers ───────────────────────────────────────────────────────────────

// snmpTestConnection submits an async test task and returns its id. The task
// runs on the SNMP collector (worker in NATS mode, inline fallback otherwise)
// and performs a real SNMP GET with the stored (decrypted) credentials.
func (h *Handler) snmpTestConnection(w http.ResponseWriter, r *http.Request) {
	h.submitSnmpTask(w, r, domain.SNMPTaskTest)
}

// snmpDiscover submits an async discovery task.
func (h *Handler) snmpDiscover(w http.ResponseWriter, r *http.Request) {
	h.submitSnmpTask(w, r, domain.SNMPTaskDiscovery)
}

func (h *Handler) submitSnmpTask(w http.ResponseWriter, r *http.Request, kind domain.SNMPTaskKind) {
	monitor, ok := h.loadSnmpMonitor(w, r, chi.URLParam(r, "monitorID"))
	if !ok {
		return
	}
	// The monitor configuration already carries encrypted secrets; the task
	// executes on the collector which decrypts them in-memory.
	task := &domain.SNMPTask{
		WorkspaceID: monitorWorkspace(h, r.Context(), monitor.ResourceID),
		ResourceID:  monitor.ResourceID,
		MonitorID:   monitor.ID,
		Kind:        kind,
		Config:      monitor.Configuration,
		CreatedAt:   time.Now().UTC(),
	}
	if err := h.deps.SNMP.CreateTask(r.Context(), task); err != nil {
		writeDomainError(w, r, err)
		return
	}
	if err := h.deps.SNMPRunner.Submit(r.Context(), task); err != nil {
		writeError(w, r, http.StatusInternalServerError, "task_submit_failed", "could not submit SNMP task", nil)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"task_id": task.TaskID, "kind": kind})
}

func monitorWorkspace(h *Handler, ctx context.Context, resourceID string) string {
	if resourceID == "" {
		return ""
	}
	res, err := h.deps.ResourceRepo.GetByID(ctx, resourceID)
	if err != nil || res.WorkspaceID == nil {
		return ""
	}
	return *res.WorkspaceID
}

// snmpGetTask returns the current task status/result. Tenant isolation is
// enforced through the task's resource.
func (h *Handler) snmpGetTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if _, err := uuid.Parse(taskID); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "task id must be a valid UUID", nil)
		return
	}
	task, err := h.deps.SNMP.GetTask(r.Context(), taskID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	if !h.resourceBelongsToOrg(w, r, task.ResourceID) {
		return
	}

	response := map[string]any{
		"task_id":     task.TaskID,
		"kind":        task.Kind,
		"status":      task.Status,
		"created_at":  task.CreatedAt,
		"finished_at": task.FinishedAt,
	}
	if len(task.Result) > 0 {
		response["result"] = json.RawMessage(task.Result)
	}
	if task.Error != "" {
		response["error"] = task.Error
	}
	writeJSON(w, http.StatusOK, response)
}

// snmpSourceIPs returns the public SNMP collector source IPs that customers
// must allow on their firewall (UDP/161). The values are configuration-driven
// (SNMP_SOURCE_IPS env), never hard-coded in the frontend.
func (h *Handler) snmpSourceIPs(w http.ResponseWriter, r *http.Request) {
	ips := h.deps.Config.SNMPCollectorSourceIPs()
	writeJSON(w, http.StatusOK, map[string]any{
		"ips":  ips,
		"port": 161,
		"note": "Allow UDP/161 from these source IPs only",
	})
}

// snmpDiagnostics exposes non-sensitive collector diagnostics for a monitor:
// last state, error, partial failures, response time, device identity and
// collector version. Secrets are never included.
func (h *Handler) snmpDiagnostics(w http.ResponseWriter, r *http.Request) {
	monitor, ok := h.loadSnmpMonitor(w, r, chi.URLParam(r, "monitorID"))
	if !ok {
		return
	}

	results, _, err := h.deps.MonitorRepo.ListResults(r.Context(), monitor.ID, 1, 0)
	if err != nil || len(results) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"collector_version": snmplib.CollectorVersion(), "monitor_id": monitor.ID})
		return
	}
	latest := results[0]

	diag := map[string]any{
		"collector_version": snmplib.CollectorVersion(),
		"monitor_id":        monitor.ID,
		"last_status":       latest.Status,
		"success":           latest.Success,
		"error_code":        latest.ErrorCode,
		"duration_millis":   latest.DurationMillis,
		"last_check_at":     latest.StartedAt,
		"state":             latest.Attributes["snmp.state"],
		"snmp_version":      latest.Attributes["snmp.version"],
		"partial_failures":  latest.Attributes["snmp.partial_failures"],
	}
	if dev, ok := latest.Attributes["snmp.device"].(string); ok {
		var parsed any
		if json.Unmarshal([]byte(dev), &parsed) == nil {
			diag["device"] = parsed
		}
	}
	writeJSON(w, http.StatusOK, diag)
}

// ensureSNMPCredential persists a credential row (best-effort, workspace
// required) and records the reference in the monitor configuration. Called
// before secrets are encrypted so plaintext is available. The reference lets
// the UI show "Configured" without ever re-sending the secret value.
func (h *Handler) ensureSNMPCredential(ctx context.Context, resourceID string, config map[string]any) {
	workspaceID := monitorWorkspace(h, ctx, resourceID)
	if workspaceID == "" || h.deps.SNMP == nil {
		return
	}

	version := stringConfigFrom(config, "version", "2c")
	cred := &domain.SNMPCredential{
		WorkspaceID:              workspaceID,
		Name:                     "snmp-" + resourceID[:8],
		Version:                  version,
		Community:                stringConfigFrom(config, "community", ""),
		Username:                 stringConfigFrom(config, "username", ""),
		AuthenticationProtocol:   stringConfigFrom(config, "authentication_protocol", ""),
		AuthenticationPassphrase: stringConfigFrom(config, "authentication_secret", ""),
		PrivacyProtocol:          stringConfigFrom(config, "privacy_protocol", ""),
		PrivacyPassphrase:        stringConfigFrom(config, "privacy_secret", ""),
		SecurityLevel:            domain.SNMPSecurityLevel(stringConfigFrom(config, "security_level", "noAuthNoPriv")),
		ContextName:              stringConfigFrom(config, "context_name", ""),
	}

	// Encrypt the passphrases for the credentials table.
	if key := h.deps.Config.AgentSecretEncryptionKey; key != "" {
		if enc, err := security.EncryptSecret(key, cred.Community); err == nil {
			cred.Community = enc
		}
		if cred.AuthenticationPassphrase != "" {
			if enc, err := security.EncryptSecret(key, cred.AuthenticationPassphrase); err == nil {
				cred.AuthenticationPassphrase = enc
			}
		}
		if cred.PrivacyPassphrase != "" {
			if enc, err := security.EncryptSecret(key, cred.PrivacyPassphrase); err == nil {
				cred.PrivacyPassphrase = enc
			}
		}
	}

	if err := h.deps.SNMP.CreateCredential(ctx, cred); err != nil {
		h.deps.Logger.Debug("snmp credential persist skipped", "error", err)
		return
	}
	config["credential_reference"] = cred.ID
}

func stringConfigFrom(config map[string]any, key, fallback string) string {
	if value, ok := config[key].(string); ok && value != "" {
		return value
	}
	return fallback
}

func intConfigFrom(config map[string]any, key string, fallback int) int {
	switch value := config[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

// validateSnmpTarget enforces SSRF-like isolation: the SNMP target must be the
// owning resource's own address (the tenant's public network device). A user
// cannot point the collector at arbitrary internal IPs through the monitor
// configuration because the host is bound to the resource target.
func (h *Handler) validateSnmpTarget(ctx context.Context, resourceID string, config map[string]any) error {
	res, err := h.deps.ResourceRepo.GetByID(ctx, resourceID)
	if err != nil {
		return fmt.Errorf("load resource target: %w", err)
	}
	host := strings.TrimSpace(stringConfigFrom(config, "host", ""))
	if host != "" {
		if !strings.EqualFold(host, strings.TrimSpace(res.Target)) {
			return fmt.Errorf("SNMP target must match the resource address")
		}
	}
	port := intConfigFrom(config, "port", 161)
	if port < 1 || port > 65535 {
		return fmt.Errorf("SNMP port must be between 1 and 65535")
	}
	version := stringConfigFrom(config, "version", "2c")
	if version != "1" && version != "2c" && version != "3" {
		return fmt.Errorf("unsupported SNMP version")
	}
	return nil
}
