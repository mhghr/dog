package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"monitoring-platform/internal/domain"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	issuer := NewTokenIssuer("test-secret", 15*time.Minute)

	user := domain.User{ID: "11111111-1111-1111-1111-111111111111", Name: "Test", Email: "t@example.com"}

	token, expiresIn, err := issuer.IssueAccessToken(user, "")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if expiresIn != int((15 * time.Minute).Seconds()) {
		t.Fatalf("unexpected expiresIn %d", expiresIn)
	}

	userID, err := issuer.VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if userID != user.ID {
		t.Fatalf("expected %s, got %s", user.ID, userID)
	}
}

func TestAccessTokenRejectsWrongSecret(t *testing.T) {
	issuer := NewTokenIssuer("secret-a", time.Minute)
	other := NewTokenIssuer("secret-b", time.Minute)

	token, _, err := issuer.IssueAccessToken(domain.User{ID: "user-1"}, "")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	if _, err := other.VerifyAccessToken(token); err == nil {
		t.Fatal("expected verification failure with wrong secret")
	}
}

func TestAccessTokenRejectsExpired(t *testing.T) {
	issuer := NewTokenIssuer("secret", time.Minute)

	claims := jwt.MapClaims{
		"iss": tokenIssuerName,
		"sub": "user-1",
		"typ": "access",
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
		"exp": time.Now().Add(-time.Hour).Unix(),
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := issuer.VerifyAccessToken(token); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestAccessTokenRejectsWrongType(t *testing.T) {
	issuer := NewTokenIssuer("secret", time.Minute)

	claims := jwt.MapClaims{
		"iss": tokenIssuerName,
		"sub": "user-1",
		"typ": "refresh",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := issuer.VerifyAccessToken(token); err == nil {
		t.Fatal("expected wrong-type token to be rejected")
	}
}
