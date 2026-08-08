package auth

import (
	"net/http"
	"os"
	"strings"
)

func RequireAdmin(unauthorized http.HandlerFunc) func(http.Handler) http.Handler {
	admins := make(map[string]bool)
	if env := os.Getenv("PLATFORM_ADMINS"); env != "" {
		for _, id := range strings.Split(env, ",") {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				admins[trimmed] = true
			}
		}
	}

	devMode := strings.EqualFold(os.Getenv("APP_ENV"), "development")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				unauthorized(w, r)
				return
			}
			if devMode {
				next.ServeHTTP(w, r)
				return
			}
			if !admins[userID] {
				unauthorized(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
