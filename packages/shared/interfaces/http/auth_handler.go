package api

import (
	"errors"
	"net/http"
	"time"

	"monitoring-platform/packages/shared/auth"
	"monitoring-platform/packages/shared/domain"
)

type authUserResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	AvatarURL string `json:"avatar_url"`
}

func toAuthUserResponse(user domain.User) authUserResponse {
	return authUserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Phone:     user.Phone,
		AvatarURL: user.AvatarURL,
	}
}

type tokenResponse struct {
	User                  authUserResponse `json:"user"`
	AccessToken           string           `json:"access_token"`
	AccessTokenExpiresIn  int              `json:"access_token_expires_in"`
	RefreshToken          string           `json:"refresh_token"`
	RefreshTokenExpiresIn int              `json:"refresh_token_expires_in"`
}

// setAuthCookies stores tokens as HttpOnly cookies for the web console.
// Mobile clients ignore cookies and use the JSON body instead.
func (h *Handler) setAuthCookies(w http.ResponseWriter, pair auth.TokenPair) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.AccessTokenCookie,
		Value:    pair.AccessToken,
		Path:     "/",
		Domain:   h.deps.Config.CookieDomain,
		MaxAge:   pair.AccessExpiresIn,
		HttpOnly: true,
		Secure:   h.deps.Config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     auth.RefreshTokenCookie,
		Value:    pair.RefreshToken,
		Path:     "/",
		Domain:   h.deps.Config.CookieDomain,
		MaxAge:   pair.RefreshExpiresIn,
		HttpOnly: true,
		Secure:   h.deps.Config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearAuthCookies(w http.ResponseWriter) {
	for _, name := range []string{auth.AccessTokenCookie, auth.RefreshTokenCookie} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Domain:   h.deps.Config.CookieDomain,
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   h.deps.Config.CookieSecure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func (h *Handler) writeTokenResponse(w http.ResponseWriter, user domain.User, pair auth.TokenPair) {
	h.setAuthCookies(w, pair)
	writeJSON(w, http.StatusOK, tokenResponse{
		User:                  toAuthUserResponse(user),
		AccessToken:           pair.AccessToken,
		AccessTokenExpiresIn:  pair.AccessExpiresIn,
		RefreshToken:          pair.RefreshToken,
		RefreshTokenExpiresIn: pair.RefreshExpiresIn,
	})
}

func (h *Handler) handleAuthError(w http.ResponseWriter, r *http.Request, err error) {
	var rateLimited *auth.RateLimitError

	switch {
	case errors.As(err, &rateLimited):
		w.Header().Set("Retry-After", rateLimited.RetryAfter.Truncate(time.Second).String())
		writeError(w, r, http.StatusTooManyRequests, "rate_limited", "Too many requests, slow down", nil)
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeError(w, r, http.StatusUnauthorized, "invalid_credentials", "Authentication failed", nil)
	case errors.Is(err, auth.ErrInvalidRedirectURI):
		writeError(w, r, http.StatusBadRequest, "invalid_redirect_uri", "The redirect URI is not allowed", nil)
	case errors.Is(err, auth.ErrGoogleDisabled):
		writeError(w, r, http.StatusServiceUnavailable, "google_disabled", "Google login is not configured", nil)
	default:
		h.deps.Logger.Error("auth request failed", "error", err)
		writeDomainError(w, r, err)
	}
}

// POST /api/v1/auth/google/exchange — web console code flow.
func (h *Handler) googleExchange(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Code        string `json:"code"`
		RedirectURI string `json:"redirect_uri"`
	}
	if err := decodeJSON(r, &payload); err != nil || payload.Code == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "code and redirect_uri are required", nil)
		return
	}

	user, pair, err := h.deps.Auth.LoginWithGoogleCode(r.Context(), payload.Code, payload.RedirectURI)
	if err != nil {
		h.handleAuthError(w, r, err)
		return
	}

	h.writeTokenResponse(w, user, pair)
}

// POST /api/v1/auth/google/mobile — native apps post a Google ID token.
func (h *Handler) googleMobile(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		IDToken string `json:"id_token"`
	}
	if err := decodeJSON(r, &payload); err != nil || payload.IDToken == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "id_token is required", nil)
		return
	}

	user, pair, err := h.deps.Auth.LoginWithGoogleIDToken(r.Context(), payload.IDToken)
	if err != nil {
		h.handleAuthError(w, r, err)
		return
	}

	h.writeTokenResponse(w, user, pair)
}

// POST /api/v1/auth/otp/request
func (h *Handler) otpRequest(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Phone string `json:"phone"`
	}
	if err := decodeJSON(r, &payload); err != nil || payload.Phone == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "phone is required", nil)
		return
	}

	devCode, retryAfter, err := h.deps.Auth.RequestOTP(r.Context(), payload.Phone)
	if err != nil {
		h.handleAuthError(w, r, err)
		return
	}

	response := map[string]any{
		"status":      "sent",
		"retry_after": retryAfter,
	}
	if devCode != "" {
		response["dev_code"] = devCode
	}

	writeJSON(w, http.StatusOK, response)
}

// POST /api/v1/auth/otp/verify
func (h *Handler) otpVerify(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := decodeJSON(r, &payload); err != nil || payload.Phone == "" || payload.Code == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "phone and code are required", nil)
		return
	}

	user, pair, err := h.deps.Auth.VerifyOTP(r.Context(), payload.Phone, payload.Code)
	if err != nil {
		h.handleAuthError(w, r, err)
		return
	}

	h.writeTokenResponse(w, user, pair)
}

// POST /api/v1/auth/refresh — rotates the refresh token. The token comes
// from the HttpOnly cookie (web) or the JSON body (mobile).
func (h *Handler) authRefresh(w http.ResponseWriter, r *http.Request) {
	refreshToken := ""

	var payload struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := decodeJSON(r, &payload); err == nil && payload.RefreshToken != "" {
		refreshToken = payload.RefreshToken
	}
	if refreshToken == "" {
		if cookie, err := r.Cookie(auth.RefreshTokenCookie); err == nil {
			refreshToken = cookie.Value
		}
	}

	user, pair, err := h.deps.Auth.Refresh(r.Context(), refreshToken)
	if err != nil {
		h.clearAuthCookies(w)
		h.handleAuthError(w, r, err)
		return
	}

	h.writeTokenResponse(w, user, pair)
}

// POST /api/v1/auth/logout
func (h *Handler) authLogout(w http.ResponseWriter, r *http.Request) {
	refreshToken := ""

	var payload struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := decodeJSON(r, &payload); err == nil && payload.RefreshToken != "" {
		refreshToken = payload.RefreshToken
	}
	if refreshToken == "" {
		if cookie, err := r.Cookie(auth.RefreshTokenCookie); err == nil {
			refreshToken = cookie.Value
		}
	}

	if err := h.deps.Auth.Logout(r.Context(), refreshToken); err != nil {
		h.deps.Logger.Warn("logout failed", "error", err)
	}

	h.clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/auth/me
func (h *Handler) authMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required", nil)
		return
	}

	user, err := h.deps.Auth.GetUser(r.Context(), userID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": toAuthUserResponse(user)})
}
