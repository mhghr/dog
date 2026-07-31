package pipeline

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// AgentIdentity carries the authenticated agent's identity.
type AgentIdentity struct {
	AgentID  string
	TenantID string
	Hostname string
	SourceIP string
}

// Authenticator authenticates a request context and returns the agent identity.
// Implementations may swap to mTLS without changing callers.
type Authenticator interface {
	Authenticate(ctx context.Context) (*AgentIdentity, error)
}

// AuthError is returned when authentication fails.
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string { return e.Message }

// HMACAuthenticator verifies HMAC-SHA256 signed agent credentials from gRPC metadata.
type HMACAuthenticator struct {
	getSecret func(ctx context.Context, agentID string) (string, error)
	window    time.Duration
}

// NewHMACAuthenticator creates an authenticator that verifies signatures using
// the agent's raw secret (retrieved via getSecret, e.g. decrypted from storage).
func NewHMACAuthenticator(getSecret func(ctx context.Context, agentID string) (string, error)) *HMACAuthenticator {
	return &HMACAuthenticator{getSecret: getSecret, window: 30 * time.Second}
}

// Authenticate reads x-agent-id, x-timestamp, x-signature metadata and verifies
// the HMAC-SHA256 signature over "agent_id:timestamp".
func (a *HMACAuthenticator) Authenticate(ctx context.Context) (*AgentIdentity, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, &AuthError{"missing metadata"}
	}

	agentIDs := md.Get("x-agent-id")
	signatures := md.Get("x-signature")
	timestamps := md.Get("x-timestamp")

	if len(agentIDs) == 0 || len(signatures) == 0 {
		return nil, &AuthError{"missing agent credentials"}
	}

	agentID := agentIDs[0]
	signature := signatures[0]
	timestamp := ""
	if len(timestamps) > 0 {
		timestamp = timestamps[0]
	}

	// Replay protection: timestamp must be within window.
	if timestamp != "" {
		ts, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return nil, &AuthError{"invalid timestamp format"}
		}
		if absDuration(time.Since(ts)) > a.window {
			return nil, &AuthError{"timestamp outside acceptable window"}
		}
	}

	secret, err := a.getSecret(ctx, agentID)
	if err != nil {
		return nil, &AuthError{"agent not found"}
	}

	expected := computeSignature(agentID, secret, timestamp)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return nil, &AuthError{"invalid signature"}
	}

	identity := &AgentIdentity{AgentID: agentID}
	if p, ok := peer.FromContext(ctx); ok {
		identity.SourceIP = p.Addr.String()
	}

	return identity, nil
}

// computeSignature returns base64url(HMAC-SHA256(secret, "agent_id:timestamp")).
func computeSignature(agentID, secret, timestamp string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(agentID))
	mac.Write([]byte(":"))
	mac.Write([]byte(timestamp))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
