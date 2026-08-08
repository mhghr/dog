package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userIDContextKey contextKey = "auth.user_id"

const (
	AccessTokenCookie  = "mp_at"
	RefreshTokenCookie = "mp_rt"
)

// UserIDFromContext returns the authenticated user id set by RequireAuth.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok && userID != ""
}

// RequireAuth accepts either an Authorization bearer header (mobile/API
// clients) or the HttpOnly access-token cookie (web console).
func RequireAuth(issuer *TokenIssuer, unauthorized http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken := ""

			if header := r.Header.Get("Authorization"); strings.HasPrefix(header, "Bearer ") {
				rawToken = strings.TrimPrefix(header, "Bearer ")
			} else if cookie, err := r.Cookie(AccessTokenCookie); err == nil {
				rawToken = cookie.Value
			}

			if rawToken == "" {
				unauthorized(w, r)
				return
			}

			userID, err := issuer.VerifyAccessToken(rawToken)
			if err != nil {
				unauthorized(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
