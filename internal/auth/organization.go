package auth

import (
	"context"
	"net/http"

	"monitoring-platform/internal/domain"
)

func OrgScoped(issuer *TokenIssuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := ""
			if cookie, err := r.Cookie(AccessTokenCookie); err == nil {
				raw = cookie.Value
			}
			if raw == "" {
				http.Error(w, `{"error":{"code":"unauthorized"}}`, http.StatusUnauthorized)
				return
			}

			claims, err := issuer.ParseOrgClaims(raw)
			if err != nil || claims.OrgID == "" {
				http.Error(w, `{"error":{"code":"forbidden","message":"no organization in session"}}`, http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), domain.OrgIDContextKey, claims.OrgID)
			if claims.ProjectID != "" {
				ctx = context.WithValue(ctx, domain.ProjectIDContextKey, claims.ProjectID)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
