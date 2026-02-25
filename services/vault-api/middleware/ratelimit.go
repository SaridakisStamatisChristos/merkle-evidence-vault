package middleware

import (
	"net/http"
)

// RateLimit is a stub for Redis-backed sliding window.
func RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: implement redis-based sliding window per-subject
		next.ServeHTTP(w, r)
	})
}
