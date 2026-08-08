package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"monitoring-platform/packages/shared/domain"
)

const tokenIssuerName = "monitoring-platform"

// TokenIssuer signs and verifies short-lived access tokens (HS256 JWT).
type TokenIssuer struct {
	secret    []byte
	accessTTL time.Duration
}

func NewTokenIssuer(secret string, accessTTL time.Duration) *TokenIssuer {
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}

	return &TokenIssuer{
		secret:    []byte(secret),
		accessTTL: accessTTL,
	}
}

func (i *TokenIssuer) AccessTTL() time.Duration {
	return i.accessTTL
}

func (i *TokenIssuer) IssueAccessToken(user domain.User, orgID string) (string, int, error) {
	now := time.Now()
	expiresAt := now.Add(i.accessTTL)

	claims := jwt.MapClaims{
		"iss":   tokenIssuerName,
		"sub":   user.ID,
		"typ":   "access",
		"name":  user.Name,
		"email": user.Email,
		"org":   orgID,
		"iat":   now.Unix(),
		"exp":   expiresAt.Unix(),
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
	if err != nil {
		return "", 0, fmt.Errorf("sign access token: %w", err)
	}

	return token, int(i.accessTTL.Seconds()), nil
}

// VerifyAccessToken validates signature, expiry, issuer, and token type,
// returning the authenticated user id.
func (i *TokenIssuer) VerifyAccessToken(raw string) (string, error) {
	parsed, err := jwt.Parse(
		raw,
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method %q", token.Header["alg"])
			}
			return i.secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(tokenIssuerName),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return "", err
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		return "", fmt.Errorf("invalid token claims")
	}

	if tokenType, _ := claims["typ"].(string); tokenType != "access" {
		return "", fmt.Errorf("unexpected token type")
	}

	subject, err := claims.GetSubject()
	if err != nil || subject == "" {
		return "", fmt.Errorf("token subject missing")
	}

	return subject, nil
}

type OrgClaims struct {
	OrgID string
}

func (i *TokenIssuer) ParseOrgClaims(rawToken string) (OrgClaims, error) {
	parsed, err := jwt.Parse(
		rawToken,
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method %q", token.Header["alg"])
			}
			return i.secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(tokenIssuerName),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return OrgClaims{}, err
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		return OrgClaims{}, fmt.Errorf("invalid token claims")
	}

	orgID, _ := claims["org"].(string)
	return OrgClaims{OrgID: orgID}, nil
}
