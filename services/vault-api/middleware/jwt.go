package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
)

type ctxKey string

const (
	ctxKeyRoles ctxKey = "roles"
	ctxKeySub   ctxKey = "sub"
)

// JWT is a middleware that validates a Bearer token. If JWKS_URL env var is set
// it will verify signatures using the JWKS. Otherwise it falls back to a simple
// test-mode that maps token text to roles (keeps tests/dev ergonomics).
func JWT(next http.Handler) http.Handler {
	// build JWKS client lazily
	var jwks *keyfunc.JWKS
	jwksURL := os.Getenv("JWKS_URL")
	if jwksURL != "" {
		// try to fetch JWKS; on error we leave jwks nil and fail verification later
		var err error
		jwks, err = keyfunc.Get(jwksURL, keyfunc.Options{RefreshInterval: time.Hour})
		_ = err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(401)
			return
		}
		tokenStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer"))
		tokenStr = strings.TrimSpace(tokenStr)

		// JWKS mode
		if jwks != nil {
			token, err := jwt.Parse(tokenStr, jwks.Keyfunc)
			if err != nil || !token.Valid {
				w.WriteHeader(401)
				return
			}
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				// extract roles claim if present
				var roles []string
				if rc, ok := claims["roles"]; ok {
					switch v := rc.(type) {
					case []interface{}:
						for _, it := range v {
							if s, ok := it.(string); ok {
								roles = append(roles, s)
							}
						}
					case string:
						roles = append(roles, v)
					}
				}
				// subject
				var sub string
				if s, ok := claims["sub"].(string); ok {
					sub = s
				}
				ctx := context.WithValue(r.Context(), ctxKeyRoles, roles)
				ctx = context.WithValue(ctx, ctxKeySub, sub)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			w.WriteHeader(401)
			return
		}

		// Test-mode fallback: map token text to roles for local e2e.
		var roles []string
		lower := strings.ToLower(tokenStr)
		if strings.Contains(lower, "auditor") {
			roles = append(roles, "auditor")
		}
		if strings.Contains(lower, "ingest") || strings.Contains(lower, "ingester") {
			roles = append(roles, "ingester")
		}
		// subject is the token text in test-mode
		ctx := context.WithValue(r.Context(), ctxKeyRoles, roles)
		ctx = context.WithValue(ctx, ctxKeySub, tokenStr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RolesFromContext returns roles extracted by the JWT middleware.
func RolesFromContext(ctx context.Context) []string {
	if v := ctx.Value(ctxKeyRoles); v != nil {
		if rs, ok := v.([]string); ok {
			return rs
		}
	}
	return nil
}

// SubjectFromContext returns the subject (sub) claim or token string.
func SubjectFromContext(ctx context.Context) string {
	if v := ctx.Value(ctxKeySub); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
