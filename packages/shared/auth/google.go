package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	googleTokenURL = "https://oauth2.googleapis.com/token"
	jwksCacheTTL   = 6 * time.Hour
)

// Package-level vars (not consts) so tests can point them at local fixtures.
var (
	googleJWKSURL      = "https://www.googleapis.com/oauth2/v3/certs"
	googleTokenInfoURL = "https://oauth2.googleapis.com/tokeninfo"
	googleUserInfoURL  = "https://openidconnect.googleapis.com/v1/userinfo"
)

type GoogleClaims struct {
	Sub           string
	Email         string
	EmailVerified bool
	Name          string
	Picture       string
}

// GoogleVerifier exchanges authorization codes and verifies Google ID tokens
// against Google's JWKS (cached, refreshed on unknown key ids).
type GoogleVerifier struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client

	mu          sync.Mutex
	keys        map[string]*rsa.PublicKey
	keysFetched time.Time
}

func NewGoogleVerifier(clientID, clientSecret string) *GoogleVerifier {
	return &GoogleVerifier{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		keys:         map[string]*rsa.PublicKey{},
	}
}

func (g *GoogleVerifier) Enabled() bool {
	return g.clientID != "" && g.clientSecret != ""
}

// ExchangeCode swaps an authorization code for the ID token and the access
// token (used to enrich the profile from the userinfo endpoint).
func (g *GoogleVerifier) ExchangeCode(ctx context.Context, code, redirectURI string) (idToken, accessToken string, err error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {g.clientID},
		"client_secret": {g.clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, googleTokenURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := g.httpClient.Do(request)
	if err != nil {
		return "", "", fmt.Errorf("google token exchange: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", "", err
	}

	if response.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("google token exchange failed (%s): %s", response.Status, truncate(string(body), 300))
	}

	var payload struct {
		IDToken     string `json:"id_token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", "", fmt.Errorf("decode google token response: %w", err)
	}
	if payload.IDToken == "" {
		return "", "", fmt.Errorf("google token response contains no id_token")
	}

	return payload.IDToken, payload.AccessToken, nil
}

// FetchUserInfo reads the OIDC userinfo endpoint. Some Google ID tokens omit
// profile claims (name/picture); userinfo is the authoritative fallback.
func (g *GoogleVerifier) FetchUserInfo(ctx context.Context, accessToken string) (GoogleClaims, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return GoogleClaims{}, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)

	response, err := g.httpClient.Do(request)
	if err != nil {
		return GoogleClaims{}, fmt.Errorf("google userinfo: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return GoogleClaims{}, err
	}
	if response.StatusCode != http.StatusOK {
		return GoogleClaims{}, fmt.Errorf("google userinfo failed (%s): %s", response.Status, truncate(string(body), 300))
	}

	var payload struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return GoogleClaims{}, fmt.Errorf("decode google userinfo response: %w", err)
	}

	return GoogleClaims{
		Sub:           payload.Sub,
		Email:         payload.Email,
		EmailVerified: payload.EmailVerified,
		Name:          payload.Name,
		Picture:       payload.Picture,
	}, nil
}

// VerifyIDToken validates the Google ID token. Primary path: local signature
// verification against Google's JWKS (RS256, cached keys). Fallback path: some
// networks cannot reach the JWKS endpoint (it can be geo-blocked with 403
// while the OAuth token endpoint still works); in that case the token is
// validated by Google's tokeninfo endpoint, which checks the signature
// server-side, and audience/issuer/expiry are verified here.
func (g *GoogleVerifier) VerifyIDToken(ctx context.Context, rawToken string) (GoogleClaims, error) {
	claims, jwksErr := g.verifyWithJWKS(ctx, rawToken)
	if jwksErr == nil {
		return claims, nil
	}

	claims, tokenInfoErr := g.verifyWithTokenInfo(ctx, rawToken)
	if tokenInfoErr != nil {
		return GoogleClaims{}, errors.Join(jwksErr, tokenInfoErr)
	}

	return claims, nil
}

func (g *GoogleVerifier) verifyWithJWKS(ctx context.Context, rawToken string) (GoogleClaims, error) {
	parsed, err := jwt.Parse(
		rawToken,
		func(token *jwt.Token) (any, error) {
			kid, _ := token.Header["kid"].(string)
			if kid == "" {
				return nil, fmt.Errorf("id token missing kid header")
			}
			return g.publicKey(ctx, kid)
		},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithAudience(g.clientID),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return GoogleClaims{}, fmt.Errorf("verify google id token: %w", err)
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		return GoogleClaims{}, fmt.Errorf("invalid google id token claims")
	}

	issuer, _ := claims["iss"].(string)
	if issuer != "https://accounts.google.com" && issuer != "accounts.google.com" {
		return GoogleClaims{}, fmt.Errorf("unexpected id token issuer %q", issuer)
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return GoogleClaims{}, fmt.Errorf("id token subject missing")
	}

	email, _ := claims["email"].(string)
	emailVerified, _ := claims["email_verified"].(bool)
	name, _ := claims["name"].(string)
	picture, _ := claims["picture"].(string)

	return GoogleClaims{
		Sub:           sub,
		Email:         email,
		EmailVerified: emailVerified,
		Name:          name,
		Picture:       picture,
	}, nil
}

func (g *GoogleVerifier) verifyWithTokenInfo(ctx context.Context, rawToken string) (GoogleClaims, error) {
	form := url.Values{"id_token": {rawToken}}

	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, googleTokenInfoURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return GoogleClaims{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := g.httpClient.Do(request)
	if err != nil {
		return GoogleClaims{}, fmt.Errorf("google tokeninfo: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return GoogleClaims{}, err
	}

	if response.StatusCode != http.StatusOK {
		return GoogleClaims{}, fmt.Errorf("google tokeninfo failed (%s): %s", response.Status, truncate(string(body), 300))
	}

	var payload struct {
		Aud           string `json:"aud"`
		Iss           string `json:"iss"`
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Exp           string `json:"exp"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return GoogleClaims{}, fmt.Errorf("decode google tokeninfo response: %w", err)
	}

	if payload.Aud != g.clientID {
		return GoogleClaims{}, fmt.Errorf("tokeninfo audience mismatch")
	}
	if payload.Iss != "https://accounts.google.com" && payload.Iss != "accounts.google.com" {
		return GoogleClaims{}, fmt.Errorf("unexpected tokeninfo issuer %q", payload.Iss)
	}
	if payload.Sub == "" {
		return GoogleClaims{}, fmt.Errorf("tokeninfo subject missing")
	}
	expiresAt, err := strconv.ParseInt(payload.Exp, 10, 64)
	if err != nil || time.Now().Unix() >= expiresAt {
		return GoogleClaims{}, fmt.Errorf("tokeninfo token expired")
	}

	return GoogleClaims{
		Sub:           payload.Sub,
		Email:         payload.Email,
		EmailVerified: payload.EmailVerified == "true",
		Name:          payload.Name,
		Picture:       payload.Picture,
	}, nil
}

func (g *GoogleVerifier) publicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if key, ok := g.keys[kid]; ok && time.Since(g.keysFetched) < jwksCacheTTL {
		return key, nil
	}

	if err := g.fetchKeysLocked(ctx); err != nil {
		return nil, err
	}

	key, ok := g.keys[kid]
	if !ok {
		return nil, fmt.Errorf("google jwks does not contain key %q", kid)
	}

	return key, nil
}

func (g *GoogleVerifier) fetchKeysLocked(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, googleJWKSURL, nil)
	if err != nil {
		return err
	}

	response, err := g.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("fetch google jwks: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("google jwks returned %s", response.Status)
	}

	var payload struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return fmt.Errorf("decode google jwks: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(payload.Keys))
	for _, key := range payload.Keys {
		if key.Kty != "RSA" || key.Kid == "" {
			continue
		}

		modulus, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			continue
		}
		exponentBytes, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil {
			continue
		}

		exponent := 0
		for _, b := range exponentBytes {
			exponent = exponent<<8 | int(b)
		}
		if exponent == 0 {
			continue
		}

		keys[key.Kid] = &rsa.PublicKey{
			N: new(big.Int).SetBytes(modulus),
			E: exponent,
		}
	}

	if len(keys) == 0 {
		return fmt.Errorf("google jwks contained no usable RSA keys")
	}

	g.keys = keys
	g.keysFetched = time.Now()
	return nil
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
