package pipeline

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const testAgentID = "ag_1"
const testAgentSecret = "s3cret"

func authMetadata(t *testing.T, agentID, secret, timestamp string) metadata.MD {
	t.Helper()
	if timestamp == "" {
		timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	return metadata.Pairs(
		"x-agent-id", agentID,
		"x-timestamp", timestamp,
		"x-signature", computeSignature(agentID, secret, timestamp),
	)
}

func authContext(t *testing.T, md metadata.MD) context.Context {
	t.Helper()
	return metadata.NewIncomingContext(context.Background(), md)
}

func TestAuthenticateSuccess(t *testing.T) {
	auth := NewHMACAuthenticator(func(ctx context.Context, agentID string) (string, error) {
		return testAgentSecret, nil
	})

	identity, err := auth.Authenticate(authContext(t, authMetadata(t, testAgentID, testAgentSecret, "")))
	if err != nil {
		t.Fatalf("expected authentication to succeed, got error: %v", err)
	}
	if identity.AgentID != testAgentID {
		t.Fatalf("expected AgentID %q, got %q", testAgentID, identity.AgentID)
	}
}

func TestAuthenticateMissingMetadata(t *testing.T) {
	auth := NewHMACAuthenticator(func(ctx context.Context, agentID string) (string, error) {
		return testAgentSecret, nil
	})

	_, err := auth.Authenticate(context.Background())
	if err == nil {
		t.Fatal("expected an error for missing metadata")
	}
	if err.Error() != "missing metadata" {
		t.Fatalf("expected %q, got %q", "missing metadata", err.Error())
	}
}

func TestAuthenticateMissingCredentials(t *testing.T) {
	auth := NewHMACAuthenticator(func(ctx context.Context, agentID string) (string, error) {
		return testAgentSecret, nil
	})

	md := metadata.Pairs("x-agent-id", testAgentID)
	_, err := auth.Authenticate(authContext(t, md))
	if err == nil {
		t.Fatal("expected an error for missing credentials")
	}
	if err.Error() != "missing agent credentials" {
		t.Fatalf("expected %q, got %q", "missing agent credentials", err.Error())
	}
}

func TestAuthenticateMissingTimestamp(t *testing.T) {
	auth := NewHMACAuthenticator(func(ctx context.Context, agentID string) (string, error) {
		return testAgentSecret, nil
	})

	md := metadata.Pairs(
		"x-agent-id", testAgentID,
		"x-signature", computeSignature(testAgentID, testAgentSecret, time.Now().UTC().Format(time.RFC3339)),
	)
	_, err := auth.Authenticate(authContext(t, md))
	if err == nil {
		t.Fatal("expected an error for missing timestamp")
	}
	if err.Error() != "missing timestamp" {
		t.Fatalf("expected %q, got %q", "missing timestamp", err.Error())
	}
}

func TestAuthenticateInvalidTimestamp(t *testing.T) {
	auth := NewHMACAuthenticator(func(ctx context.Context, agentID string) (string, error) {
		return testAgentSecret, nil
	})

	_, err := auth.Authenticate(authContext(t, authMetadata(t, testAgentID, testAgentSecret, "garbage")))
	if err == nil {
		t.Fatal("expected an error for invalid timestamp")
	}
	if err.Error() != "invalid timestamp format" {
		t.Fatalf("expected %q, got %q", "invalid timestamp format", err.Error())
	}
}

func TestAuthenticateStaleTimestamp(t *testing.T) {
	auth := NewHMACAuthenticator(func(ctx context.Context, agentID string) (string, error) {
		return testAgentSecret, nil
	})

	stale := time.Now().UTC().Add(-2 * time.Minute).Format(time.RFC3339)
	_, err := auth.Authenticate(authContext(t, authMetadata(t, testAgentID, testAgentSecret, stale)))
	if err == nil {
		t.Fatal("expected an error for stale timestamp")
	}
	if err.Error() != "timestamp outside acceptable window" {
		t.Fatalf("expected %q, got %q", "timestamp outside acceptable window", err.Error())
	}
}

func TestAuthenticateBadSignature(t *testing.T) {
	auth := NewHMACAuthenticator(func(ctx context.Context, agentID string) (string, error) {
		return testAgentSecret, nil
	})

	md := metadata.Pairs(
		"x-agent-id", testAgentID,
		"x-timestamp", time.Now().UTC().Format(time.RFC3339),
		"x-signature", "wrong-signature",
	)
	_, err := auth.Authenticate(authContext(t, md))
	if err == nil {
		t.Fatal("expected an error for bad signature")
	}
	if err.Error() != "invalid signature" {
		t.Fatalf("expected %q, got %q", "invalid signature", err.Error())
	}
}

func TestAuthenticateUnknownAgent(t *testing.T) {
	auth := NewHMACAuthenticator(func(ctx context.Context, agentID string) (string, error) {
		return "", errors.New("agent not found")
	})

	_, err := auth.Authenticate(authContext(t, authMetadata(t, testAgentID, testAgentSecret, "")))
	if err == nil {
		t.Fatal("expected an error for unknown agent")
	}
	if err.Error() != "agent not found" {
		t.Fatalf("expected %q, got %q", "agent not found", err.Error())
	}
}

func TestAuthenticateSetsHostname(t *testing.T) {
	auth := NewHMACAuthenticator(func(ctx context.Context, agentID string) (string, error) {
		return testAgentSecret, nil
	})
	auth.WithHostnameResolver(func(ctx context.Context, agentID string) (string, error) {
		return "web-01", nil
	})

	identity, err := auth.Authenticate(authContext(t, authMetadata(t, testAgentID, testAgentSecret, "")))
	if err != nil {
		t.Fatalf("expected authentication to succeed, got error: %v", err)
	}
	if identity.Hostname != "web-01" {
		t.Fatalf("expected hostname %q, got %q", "web-01", identity.Hostname)
	}
}

func TestAuthenticateReplayWithDifferentAgentID(t *testing.T) {
	auth := NewHMACAuthenticator(func(ctx context.Context, agentID string) (string, error) {
		return testAgentSecret, nil
	})

	timestamp := time.Now().UTC().Format(time.RFC3339)
	md := metadata.Pairs(
		"x-agent-id", "ag_2",
		"x-timestamp", timestamp,
		"x-signature", computeSignature("ag_1", testAgentSecret, timestamp),
	)
	_, err := auth.Authenticate(authContext(t, md))
	if err == nil {
		t.Fatal("expected an error for replayed signature with different agent ID")
	}
	if err.Error() != "invalid signature" {
		t.Fatalf("expected %q, got %q", "invalid signature", err.Error())
	}
}

func TestAuthenticateSetsSourceIP(t *testing.T) {
	auth := NewHMACAuthenticator(func(ctx context.Context, agentID string) (string, error) {
		return testAgentSecret, nil
	})

	ctx := peer.NewContext(authContext(t, authMetadata(t, testAgentID, testAgentSecret, "")), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("10.0.0.1"), Port: 54321},
	})

	identity, err := auth.Authenticate(ctx)
	if err != nil {
		t.Fatalf("expected authentication to succeed, got error: %v", err)
	}
	if identity.SourceIP != "10.0.0.1:54321" {
		t.Fatalf("expected SourceIP %q, got %q", "10.0.0.1:54321", identity.SourceIP)
	}
}
