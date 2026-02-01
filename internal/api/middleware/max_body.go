package middleware

import (
	"net/http"

	"github.com/cloo-solutions/neotexai/internal/api"
)

// MaxBodyBytes limits request body size.
func MaxBodyBytes(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limit <= 0 || r.Body == nil {
				next.ServeHTTP(w, r)
				return
			}

			if r.ContentLength > limit && r.ContentLength != -1 {
				api.Error(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
