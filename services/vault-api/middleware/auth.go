package middleware

import "net/http"

// Auth is kept for backwards compatibility and delegates to JWT middleware.
func Auth(next http.Handler) http.Handler {
	return JWT(next)
}
