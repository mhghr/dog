package api

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"monitoring-platform/internal/agents"
	"monitoring-platform/internal/auth"
)

func (h *Handler) createEnrollmentToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LocationCode string `json:"location_code"`
		TTLMinutes   int    `json:"ttl_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_body", "Invalid request body", nil)
		return
	}
	if req.LocationCode == "" {
		writeError(w, r, http.StatusBadRequest, "missing_location", "location_code is required", nil)
		return
	}
	if req.TTLMinutes <= 0 {
		req.TTLMinutes = 60
	}
	if req.TTLMinutes > 1440 {
		writeError(w, r, http.StatusBadRequest, "ttl_too_large", "ttl_minutes must be at most 1440", nil)
		return
	}

	loc, err := h.deps.Locations.GetByCode(r.Context(), req.LocationCode)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	rawToken := uuid.NewString() + "-" + uuid.NewString()

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required", nil)
		return
	}

	locID, err := uuid.Parse(loc.ID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Invalid location ID", nil)
		return
	}

	token, err := h.deps.AgentRepo.CreateEnrollmentToken(r.Context(), agents.CreateTokenParams{
		Token:      rawToken,
		LocationID: locID,
		ExpiresAt:  time.Now().Add(time.Duration(req.TTLMinutes) * time.Minute),
		CreatedBy:  uuid.MustParse(userID),
	})
	if err != nil {
		h.deps.Logger.Error("create enrollment token failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to create token", nil)
		return
	}

	_ = token

	writeJSON(w, http.StatusCreated, map[string]any{
		"token":       rawToken,
		"location_id": loc.ID,
		"expires_at":  time.Now().Add(time.Duration(req.TTLMinutes) * time.Minute).Format(time.RFC3339),
	})
}

func (h *Handler) listAgents(w http.ResponseWriter, r *http.Request) {
	statusStr := r.URL.Query().Get("status")
	var status *agents.AgentStatus
	if statusStr != "" {
		s := agents.AgentStatus(statusStr)
		status = &s
	}

	result, err := h.deps.AgentRepo.ListAgents(r.Context(), agents.ListAgentsParams{
		Status: status,
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		h.deps.Logger.Error("list agents failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to list agents", nil)
		return
	}

	if result == nil {
		result = []agents.ProbeAgent{}
	}

	type agentJSON struct {
		ID             string   `json:"id"`
		LocationID     string   `json:"location_id"`
		Name           string   `json:"name"`
		Hostname       string   `json:"hostname"`
		Version        string   `json:"version"`
		OS             string   `json:"operating_system"`
		Arch           string   `json:"architecture"`
		PublicIP       string   `json:"public_ip"`
		Capabilities   []string `json:"capabilities"`
		MaxConcurrency int32    `json:"max_concurrency"`
		Status         string   `json:"status"`
		LastSeenAt     *string  `json:"last_seen_at,omitempty"`
		CreatedAt      string   `json:"created_at"`
	}

	items := make([]agentJSON, 0, len(result))
	for _, a := range result {
		aj := agentJSON{
			ID:             a.ID.String(),
			LocationID:     a.LocationID.String(),
			Name:           a.Name,
			Hostname:       a.Hostname,
			Version:        a.Version,
			OS:             a.OperatingSystem,
			Arch:           a.Architecture,
			PublicIP:       a.PublicIP,
			Capabilities:   a.Capabilities,
			MaxConcurrency: a.MaxConcurrency,
			Status:         string(a.Status),
			CreatedAt:      a.CreatedAt.Format(time.RFC3339),
		}
		if a.LastSeenAt != nil {
			s := a.LastSeenAt.Format(time.RFC3339)
			aj.LastSeenAt = &s
		}
		items = append(items, aj)
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) getAgent(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")
	id, err := uuid.Parse(agentID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "Invalid agent ID", nil)
		return
	}

	agent, err := h.deps.AgentRepo.GetAgent(r.Context(), id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":               agent.ID.String(),
		"location_id":      agent.LocationID.String(),
		"name":             agent.Name,
		"hostname":         agent.Hostname,
		"version":          agent.Version,
		"operating_system": agent.OperatingSystem,
		"architecture":     agent.Architecture,
		"public_ip":        agent.PublicIP,
		"capabilities":     agent.Capabilities,
		"max_concurrency":  agent.MaxConcurrency,
		"status":           string(agent.Status),
		"last_seen_at":     agent.LastSeenAt,
		"created_at":       agent.CreatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) approveAgent(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")
	id, err := uuid.Parse(agentID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "Invalid agent ID", nil)
		return
	}

	agent, err := h.deps.AgentRepo.GetAgent(r.Context(), id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	if !agents.CanTransition(agent.Status, agents.AgentApproved) {
		writeError(w, r, http.StatusConflict, "invalid_transition",
			"Cannot transition from "+string(agent.Status)+" to "+string(agents.AgentApproved), nil)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required", nil)
		return
	}

	actorID := uuid.MustParse(userID)
	now := time.Now()

	opts := agents.StatusUpdateOpts{
		ApprovedBy: &actorID,
		ApprovedAt: &now,
	}

	if err := h.deps.AgentRepo.UpdateAgentStatus(r.Context(), id, agents.AgentApproved, opts); err != nil {
		h.deps.Logger.Error("update agent status failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to update agent status", nil)
		return
	}

	var certPEM, serial string
	if h.deps.CA != nil && agent.PublicKey != "" {
		block, _ := pem.Decode([]byte(agent.PublicKey))
		if block != nil {
			pub, pubErr := x509.ParsePKIXPublicKey(block.Bytes)
			if pubErr == nil {
				certPEM, serial, pubErr = h.deps.CA.IssueAgentCertificate(id.String(), agent.Hostname, pub)
				if pubErr == nil {
					if setErr := h.deps.AgentRepo.SetAgentCertificate(r.Context(), id, serial); setErr != nil {
						h.deps.Logger.Warn("set agent certificate failed", "error", setErr)
					}
					if setErr := h.deps.AgentRepo.SetAgentGatewayCert(r.Context(), id, certPEM); setErr != nil {
						h.deps.Logger.Warn("set agent gateway cert failed", "error", setErr)
					}
				} else {
					h.deps.Logger.Warn("issue certificate failed", "error", pubErr)
				}
			}
		}
	}

	prevState, _ := json.Marshal(map[string]string{"status": string(agent.Status)})
	nextState, _ := json.Marshal(map[string]string{"status": string(agents.AgentApproved)})
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)

	auditErr := h.deps.AgentRepo.AuditLog(r.Context(), agents.AuditEntry{
		AgentID:       id,
		ActorUserID:   &actorID,
		Action:        "approve",
		PreviousState: prevState,
		NextState:     nextState,
		RemoteIP:      remoteIP,
	})
	if auditErr != nil {
		h.deps.Logger.Warn("audit log failed", "error", auditErr)
	}

	resp := map[string]any{
		"id":     id.String(),
		"status": string(agents.AgentApproved),
	}
	if certPEM != "" {
		resp["certificate"] = certPEM
		resp["serial_number"] = serial
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) rejectAgent(w http.ResponseWriter, r *http.Request) {
	h.updateAgentStatus(w, r, agents.AgentRejected, "reject")
}

func (h *Handler) disableAgent(w http.ResponseWriter, r *http.Request) {
	h.updateAgentStatus(w, r, agents.AgentDisabled, "disable")
}

func (h *Handler) enableAgent(w http.ResponseWriter, r *http.Request) {
	h.updateAgentStatus(w, r, agents.AgentApproved, "enable")
}

func (h *Handler) revokeAgent(w http.ResponseWriter, r *http.Request) {
	h.updateAgentStatus(w, r, agents.AgentRevoked, "revoke")
}

func (h *Handler) drainAgent(w http.ResponseWriter, r *http.Request) {
	h.updateAgentStatus(w, r, agents.AgentDraining, "drain")
}

func (h *Handler) updateAgentStatus(w http.ResponseWriter, r *http.Request, status agents.AgentStatus, action string) {
	agentID := chi.URLParam(r, "agentID")
	id, err := uuid.Parse(agentID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "Invalid agent ID", nil)
		return
	}

	agent, err := h.deps.AgentRepo.GetAgent(r.Context(), id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	if !agents.CanTransition(agent.Status, status) {
		writeError(w, r, http.StatusConflict, "invalid_transition",
			"Cannot transition from "+string(agent.Status)+" to "+string(status), nil)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	var actorID uuid.UUID
	var actorIDPtr *uuid.UUID
	if ok {
		actorID = uuid.MustParse(userID)
		actorIDPtr = &actorID
	}

	opts := agents.StatusUpdateOpts{}
	if status == agents.AgentApproved {
		now := time.Now()
		opts.ApprovedBy = actorIDPtr
		opts.ApprovedAt = &now
	}

	if err := h.deps.AgentRepo.UpdateAgentStatus(r.Context(), id, status, opts); err != nil {
		h.deps.Logger.Error("update agent status failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to update agent status", nil)
		return
	}

	prevState, _ := json.Marshal(map[string]string{"status": string(agent.Status)})
	nextState, _ := json.Marshal(map[string]string{"status": string(status)})
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)

	auditErr := h.deps.AgentRepo.AuditLog(r.Context(), agents.AuditEntry{
		AgentID:       id,
		ActorUserID:   actorIDPtr,
		Action:        action,
		PreviousState: prevState,
		NextState:     nextState,
		RemoteIP:      remoteIP,
	})
	if auditErr != nil {
		h.deps.Logger.Warn("audit log failed", "error", auditErr)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":     id.String(),
		"status": string(status),
	})
}

func (h *Handler) agentStatus(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")
	id, err := uuid.Parse(agentID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "Invalid agent ID", nil)
		return
	}

	secret := r.URL.Query().Get("secret")
	if secret == "" {
		writeError(w, r, http.StatusUnauthorized, "missing_secret", "Secret query parameter is required", nil)
		return
	}

	agent, err := h.deps.AgentRepo.GetAgentByIDAndSecret(r.Context(), id, secret)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	resp := map[string]any{
		"id":          agent.ID.String(),
		"status":      string(agent.Status),
		"location_id": agent.LocationID.String(),
	}
	if agent.CertificateSerial != "" {
		resp["certificate_serial"] = agent.CertificateSerial
	}
	if agent.GatewayCert != "" {
		resp["certificate"] = agent.GatewayCert
	}
	if agent.ApprovedAt != nil {
		resp["approved_at"] = agent.ApprovedAt.Format(time.RFC3339)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) agentEnroll(w http.ResponseWriter, r *http.Request) {
	var req agents.EnrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_body", "Invalid request body", nil)
		return
	}

	if req.EnrollmentToken == "" {
		writeError(w, r, http.StatusBadRequest, "missing_token", "enrollment_token is required", nil)
		return
	}
	if req.Hostname == "" || req.MachineFingerprint == "" || req.PublicKey == "" {
		writeError(w, r, http.StatusBadRequest, "missing_fields", "hostname, machine_fingerprint, and public_key are required", nil)
		return
	}

	agentSecret := uuid.NewString()

	agent, err := h.deps.AgentRepo.CreateAgentWithToken(r.Context(), req.EnrollmentToken, agents.CreateAgentParams{
		Name:               req.Hostname,
		Hostname:           req.Hostname,
		MachineFingerprint: req.MachineFingerprint,
		PublicKey:          req.PublicKey,
		Version:            req.Version,
		OperatingSystem:    req.OperatingSystem,
		Architecture:       req.Architecture,
		PrivateIPs:         req.PrivateIPs,
		Capabilities:       req.Capabilities,
		MaxConcurrency:     req.MaxConcurrency,
		AgentSecret:        agentSecret,
	})
	if err != nil {
		if err == agents.ErrInvalidToken {
			writeError(w, r, http.StatusUnauthorized, "invalid_token", "Enrollment token is invalid, expired, or already used", nil)
			return
		}
		h.deps.Logger.Error("create agent with token failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to register agent", nil)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"agent_id":     agent.ID.String(),
		"status":       string(agent.Status),
		"message":      "Agent registered successfully. Waiting for admin approval.",
		"agent_secret": agent.AgentSecret,
	})
}
