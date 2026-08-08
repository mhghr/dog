package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func newTokenInfoServer(t *testing.T, payload map[string]string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil || r.PostFormValue("id_token") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(payload)
	}))
}

func TestVerifyIDTokenFallsBackToTokenInfoWhenJWKSBlocked(t *testing.T) {
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer jwks.Close()

	tokenInfo := newTokenInfoServer(t, map[string]string{
		"aud":            "client-id",
		"iss":            "https://accounts.google.com",
		"sub":            "user-123",
		"email":          "user@example.com",
		"email_verified": "true",
		"name":           "User",
		"picture":        "https://example.com/p.png",
		"exp":            strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10),
	})
	defer tokenInfo.Close()

	restoreJWKS, restoreTokenInfo := googleJWKSURL, googleTokenInfoURL
	googleJWKSURL, googleTokenInfoURL = jwks.URL, tokenInfo.URL
	defer func() {
		googleJWKSURL, googleTokenInfoURL = restoreJWKS, restoreTokenInfo
	}()

	verifier := NewGoogleVerifier("client-id", "secret")

	claims, err := verifier.VerifyIDToken(context.Background(), "opaque-token")
	if err != nil {
		t.Fatalf("expected tokeninfo fallback to succeed, got error: %v", err)
	}
	if claims.Sub != "user-123" {
		t.Fatalf("expected sub user-123, got %q", claims.Sub)
	}
	if !claims.EmailVerified {
		t.Fatal("expected email_verified to be true")
	}
	if claims.Email != "user@example.com" {
		t.Fatalf("unexpected email %q", claims.Email)
	}
}

func TestVerifyIDTokenTokenInfoRejectsAudienceMismatch(t *testing.T) {
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer jwks.Close()

	tokenInfo := newTokenInfoServer(t, map[string]string{
		"aud": "another-client",
		"iss": "https://accounts.google.com",
		"sub": "user-123",
		"exp": strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10),
	})
	defer tokenInfo.Close()

	restoreJWKS, restoreTokenInfo := googleJWKSURL, googleTokenInfoURL
	googleJWKSURL, googleTokenInfoURL = jwks.URL, tokenInfo.URL
	defer func() {
		googleJWKSURL, googleTokenInfoURL = restoreJWKS, restoreTokenInfo
	}()

	verifier := NewGoogleVerifier("client-id", "secret")

	if _, err := verifier.VerifyIDToken(context.Background(), "opaque-token"); err == nil {
		t.Fatal("expected audience mismatch error, got nil")
	}
}
