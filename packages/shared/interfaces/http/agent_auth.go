package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"monitoring-platform/packages/shared/security"
)

// agentHMACAuthenticate verifies the x-agent-id / x-timestamp / x-signature
// headers against the stored agent secret. On success the agent ID is stored
// in the request context.
func (h *Handler) agentHMACAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agentID := r.Header.Get("x-agent-id")
		timestamp := r.Header.Get("x-timestamp")
		signature := r.Header.Get("x-signature")

		if agentID == "" || timestamp == "" || signature == "" {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing agent credentials", nil)
			return
		}

		ts, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "Invalid timestamp", nil)
			return
		}
		if time.Since(ts) > 30*time.Second || time.Since(ts) < -30*time.Second {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "Timestamp outside acceptable window", nil)
			return
		}

		agent, err := h.deps.MonitoringAgents.GetByAgentID(r.Context(), agentID)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "Agent not found", nil)
			return
		}

		secret, err := security.DecryptSecret(h.deps.Config.AgentSecretEncryptionKey, agent.SecretEncrypted)
		if err != nil {
			h.deps.Logger.Error("failed to decrypt agent secret", "agent_id", agentID, "error", err)
			writeError(w, r, http.StatusInternalServerError, "internal_error", "Internal error", nil)
			return
		}

		expected := agentHMACSignature(agentID, timestamp, secret)
		if !hmac.Equal([]byte(expected), []byte(signature)) {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "Invalid signature", nil)
			return
		}

		// The authenticated agent must match the {agentID} path parameter so a
		// valid credential for one agent cannot read or mutate another's data.
		if urlID := chi.URLParam(r, "agentID"); urlID != "" && urlID != agentID {
			writeError(w, r, http.StatusForbidden, "forbidden", "Agent credential does not match requested agent", nil)
			return
		}

		// Store agent identity in context for downstream handlers.
		ctx := context.WithValue(r.Context(), agentIDContextKey{}, agentID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// agentHMACSignature computes base64url(HMAC-SHA256(secret, "agent_id:timestamp")).
func agentHMACSignature(agentID, timestamp, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(agentID))
	mac.Write([]byte(":"))
	mac.Write([]byte(timestamp))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

type agentIDContextKey struct{}

// agentIDFromContext returns the authenticated agent ID from the request context.
func agentIDFromContext(r *http.Request) (string, bool) {
	id, ok := r.Context().Value(agentIDContextKey{}).(string)
	return id, ok
}
