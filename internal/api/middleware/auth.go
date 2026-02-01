package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/cloo-solutions/neotexai/internal/api"
)

type contextKey string

const OrgIDKey contextKey = "org_id"

type AuthValidator interface {
	ValidateAPIKey(ctx context.Context, token string) (string, error)
}

func APIKeyAuth(validator AuthValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				api.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				api.Error(w, http.StatusUnauthorized, "invalid authorization format")
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")

			orgID, err := validator.ValidateAPIKey(r.Context(), token)
			if err != nil {
				api.Error(w, http.StatusUnauthorized, "invalid api key")
				return
			}

			r.Header.Set("X-Org-ID", orgID)
			ctx := context.WithValue(r.Context(), OrgIDKey, orgID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetOrgID(ctx context.Context) string {
	orgID, _ := ctx.Value(OrgIDKey).(string)
	return orgID
}
