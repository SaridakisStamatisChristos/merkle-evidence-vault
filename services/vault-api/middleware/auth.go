package middleware

import (
	"net/http"
)

// Auth middleware placeholder - validates JWT and sets context values.
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: verify JWT via JWKS, extract roles
		token := r.Header.Get("Authorization")
		if token == "" {
			w.WriteHeader(401)
			return
		}
		next.ServeHTTP(w, r)
	})
}
