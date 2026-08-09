package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/security"
)

type bootstrapRequest struct {
	BootstrapToken string   `json:"bootstrap_token"`
	Hostname       string   `json:"hostname"`
	OS             string   `json:"os"`
	Architecture   string   `json:"architecture"`
	AgentVersion   string   `json:"agent_version"`
	Capabilities   []string `json:"capabilities"`
}

type completeRegistrationRequest struct {
	AgentID     string            `json:"agent_id"`
	Signature   string            `json:"signature"`
	SecretProof string            `json:"secret_proof"`
	PrivateIPs  []string          `json:"private_ips"`
	Labels      map[string]string `json:"labels"`
}

func (h *Handler) bootstrapAgent(w http.ResponseWriter, r *http.Request) {
	var req bootstrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_body", "Invalid request body", nil)
		return
	}
	if req.BootstrapToken == "" || req.Hostname == "" {
		writeError(w, r, http.StatusBadRequest, "missing_fields", "bootstrap_token and hostname are required", nil)
		return
	}

	token, err := h.deps.BootstrapTokens.GetByTokenHash(r.Context(), sha256Hex(req.BootstrapToken))
	if err != nil {
		h.deps.Logger.Warn("bootstrap with unknown token", "error", err)
		writeError(w, r, http.StatusUnauthorized, "invalid_bootstrap_token", "invalid bootstrap token", nil)
		return
	}
	if !token.IsValid() {
		h.deps.Logger.Warn("bootstrap with invalid token", "token_id", token.ID, "expired", token.IsExpired(), "used", token.IsUsed(), "revoked", token.IsRevoked())
		writeError(w, r, http.StatusUnauthorized, "invalid_bootstrap_token", "bootstrap token is expired or already used", nil)
		return
	}

	// Atomically reserve the token before creating the agent so two
	// concurrent bootstraps with the same token cannot both succeed.
	if err := h.deps.BootstrapTokens.MarkUsedIfValid(r.Context(), token.ID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			h.deps.Logger.Warn("bootstrap token already consumed", "token_id", token.ID)
			writeError(w, r, http.StatusUnauthorized, "invalid_bootstrap_token", "bootstrap token is expired or already used", nil)
			return
		}
		h.deps.Logger.Error("reserve bootstrap token failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to register agent", nil)
		return
	}

	agentID := generateAgentID()
	agentSecret := generateAgentSecret()

	secretHash, err := bcrypt.GenerateFromPassword([]byte(agentSecret), bcrypt.DefaultCost)
	if err != nil {
		h.deps.Logger.Error("hash agent secret failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to register agent", nil)
		return
	}

	secretEncrypted, err := security.EncryptSecret(h.deps.Config.AgentSecretEncryptionKey, agentSecret)
	if err != nil {
		h.deps.Logger.Error("encrypt agent secret failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to register agent", nil)
		return
	}

	now := time.Now()
	agent := &domain.MonitoringAgent{
		ID:              uuid.NewString(),
		TenantID:        token.TenantID,
		ExternalID:      agentID,
		Hostname:        req.Hostname,
		OS:              req.OS,
		Arch:            req.Architecture,
		Version:         req.AgentVersion,
		AgentID:         agentID,
		SecretHash:      string(secretHash),
		SecretEncrypted: secretEncrypted,
		Status:          domain.AgentStatusActive,
		RegisteredAt:    now,
		UpdatedAt:       now,
		Labels: map[string]string{
			"hostname": req.Hostname,
			"os":       req.OS,
			"arch":     req.Architecture,
		},
		Capabilities:     req.Capabilities,
		PrivateIPs:       []string{},
		BootstrapTokenID: &token.ID,
	}

	if err := h.deps.MonitoringAgents.Create(r.Context(), agent); err != nil {
		h.deps.Logger.Error("create monitoring agent failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to register agent", nil)
		return
	}

	h.deps.Logger.Info("monitoring agent registered",
		"agent_id", agentID,
		"tenant_id", token.TenantID,
		"hostname", req.Hostname,
	)

	writeJSON(w, http.StatusCreated, map[string]any{
		"agent_id":            agentID,
		"agent_secret":        agentSecret,
		"config_url":          "/api/monitoring/agents/" + agentID + "/config",
		"heartbeat_url":       "/api/monitoring/agents/" + agentID + "/heartbeat",
		"config_poll_seconds": 60,
	})
}

func (h *Handler) completeAgentRegistration(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")

	var req completeRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_body", "Invalid request body", nil)
		return
	}

	agent, err := h.deps.MonitoringAgents.GetByAgentID(r.Context(), agentID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	if req.SecretProof == "" {
		writeError(w, r, http.StatusBadRequest, "missing_secret_proof", "secret_proof is required", nil)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(agent.SecretHash), []byte(req.SecretProof)); err != nil {
		writeError(w, r, http.StatusUnauthorized, "invalid_secret", "invalid agent secret", nil)
		return
	}

	if req.PrivateIPs != nil {
		agent.PrivateIPs = req.PrivateIPs
	}
	if req.Labels != nil {
		if agent.Labels == nil {
			agent.Labels = map[string]string{}
		}
		for k, v := range req.Labels {
			agent.Labels[k] = v
		}
	}

	now := time.Now()
	agent.LastSeenAt = &now
	agent.Status = domain.AgentStatusActive

	if err := h.deps.MonitoringAgents.Update(r.Context(), &agent); err != nil {
		h.deps.Logger.Error("update monitoring agent failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to complete registration", nil)
		return
	}

	if err := h.deps.AgentConfigs.Create(r.Context(), defaultAgentConfig(agent)); err != nil {
		h.deps.Logger.Error("create default agent config failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to complete registration", nil)
		return
	}

	h.deps.Logger.Info("monitoring agent registration complete", "agent_id", agentID)

	writeJSON(w, http.StatusOK, map[string]any{
		"approved": true,
		"status":   "approved",
		"message":  "agent registration complete",
	})
}

func defaultAgentConfig(agent domain.MonitoringAgent) *domain.AgentConfig {
	return &domain.AgentConfig{
		ID:                        uuid.NewString(),
		AgentID:                   agent.AgentID,
		TenantID:                  agent.TenantID,
		Version:                   1,
		CollectionIntervalSeconds: 60,
		BatchSize:                 500,
		ExportIntervalSeconds:     60,
		EnabledReceivers:          []string{"cpu", "memory", "filesystem", "disk", "network", "load"},
		MaxMetricsPerBatch:        2000,
		MaxLabelCount:             40,
		MaxLabelLength:            256,
		FeatureFlags:              map[string]bool{},
		Compress:                  true,
		RetryInitialIntervalMs:    1000,
		RetryMaxIntervalMs:        60000,
		RetryMaxElapsedMs:         300000,
		LogLevel:                  "info",
		IsActive:                  true,
	}
}

func generateAgentID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return "ag_" + base64.RawURLEncoding.EncodeToString(b)
}

func generateAgentSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
