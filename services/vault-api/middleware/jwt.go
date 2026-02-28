package middleware

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog/log"
)

type ctxKey string

const (
	ctxKeyRoles ctxKey = "roles"
	ctxKeySub   ctxKey = "sub"
)

// JWT is a middleware that validates a Bearer token. If JWKS_URL env var is set
// it verifies signatures using the JWKS endpoint. If JWKS is not configured,
// test-mode token mapping can be enabled explicitly via ENABLE_TEST_JWT=true.
func JWT(next http.Handler) http.Handler {
	var jwks *keyfunc.JWKS
	jwksURL := os.Getenv("JWKS_URL")
	enableTest := os.Getenv("ENABLE_TEST_JWT") == "true"
	appEnv := strings.ToLower(strings.TrimSpace(firstNonEmptyEnv("APP_ENV", "ENVIRONMENT", "DEPLOY_ENV")))
	if enableTest && !isTestJWTAllowedEnv(appEnv) {
		log.Error().Str("app_env", appEnv).Msg("ENABLE_TEST_JWT requested in non-development environment; disabling test JWT mode")
		enableTest = false
	}
	requiredIssuer := strings.TrimSpace(os.Getenv("JWT_REQUIRED_ISSUER"))
	requiredAudience := parseCSVEnv("JWT_REQUIRED_AUDIENCE")
	allowedAlgs := parseCSVEnvDefault("JWT_ALLOWED_ALGS", []string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512", "EdDSA"})
	clockSkew := parseIntEnvDefault("JWT_CLOCK_SKEW_SECONDS", 60)

	log.Info().
		Str("jwks_url", jwksURL).
		Bool("enable_test_jwt", enableTest).
		Str("app_env", appEnv).
		Str("required_issuer", requiredIssuer).
		Strs("required_audience", requiredAudience).
		Strs("allowed_algs", allowedAlgs).
		Int64("clock_skew_seconds", clockSkew).
		Msg("JWT middleware configuration")

	if jwksURL != "" {
		var err error
		maxAttempts := 12
		for i := 0; i < maxAttempts; i++ {
			jwks, err = keyfunc.Get(jwksURL, keyfunc.Options{RefreshInterval: time.Hour})
			if err == nil && jwks != nil {
				log.Info().Str("jwks_url", jwksURL).Msg("JWKS loaded successfully")
				break
			}
			log.Warn().Err(err).Int("attempt", i+1).Int("max_attempts", maxAttempts).Str("jwks_url", jwksURL).Msg("failed to load JWKS, will retry")
			time.Sleep(2 * time.Second)
		}
		if jwks == nil {
			log.Error().Str("jwks_url", jwksURL).Msg("giving up loading JWKS after retries; falling back to test-mode only if enabled")
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr, ok := extractBearerToken(r.Header.Get("Authorization"))
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if jwks != nil {
			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, jwks.Keyfunc, jwt.WithValidMethods(allowedAlgs))
			if err != nil || !token.Valid {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if !validateStandardClaims(claims, requiredIssuer, requiredAudience, time.Now(), time.Duration(clockSkew)*time.Second) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			roles := parseRoles(claims)
			sub, _ := claims["sub"].(string)
			ctx := context.WithValue(r.Context(), ctxKeyRoles, roles)
			ctx = context.WithValue(ctx, ctxKeySub, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if !enableTest {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var roles []string
		lower := strings.ToLower(tokenStr)
		if strings.Contains(lower, "auditor") {
			roles = append(roles, "auditor")
		}
		if strings.Contains(lower, "ingest") || strings.Contains(lower, "ingester") {
			roles = append(roles, "ingester")
		}
		ctx := context.WithValue(r.Context(), ctxKeyRoles, roles)
		ctx = context.WithValue(ctx, ctxKeySub, tokenStr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func firstNonEmptyEnv(names ...string) string {
	for _, name := range names {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	return ""
}

func isTestJWTAllowedEnv(appEnv string) bool {
	switch appEnv {
	case "", "dev", "development", "local", "test", "ci":
		return true
	default:
		return false
	}
}

func parseRoles(claims jwt.MapClaims) []string {
	var roles []string
	rc, ok := claims["roles"]
	if !ok {
		return roles
	}
	switch v := rc.(type) {
	case []interface{}:
		for _, it := range v {
			if s, ok := it.(string); ok {
				roles = append(roles, s)
			}
		}
	case []string:
		roles = append(roles, v...)
	case string:
		roles = append(roles, v)
	}
	return roles
}

func extractBearerToken(auth string) (string, bool) {
	auth = strings.TrimSpace(auth)
	if auth == "" {
		return "", false
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	tok := strings.TrimSpace(parts[1])
	if tok == "" {
		return "", false
	}
	return tok, true
}

func validateStandardClaims(claims jwt.MapClaims, issuer string, audiences []string, now time.Time, skew time.Duration) bool {
	if !claims.VerifyExpiresAt(now.Add(-skew).Unix(), true) {
		return false
	}
	if !claims.VerifyIssuedAt(now.Add(skew).Unix(), false) {
		return false
	}
	if !claims.VerifyNotBefore(now.Add(skew).Unix(), false) {
		return false
	}
	if issuer != "" && !claims.VerifyIssuer(issuer, true) {
		return false
	}
	if len(audiences) > 0 {
		ok := false
		for _, aud := range audiences {
			if claims.VerifyAudience(aud, true) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func parseCSVEnv(name string) []string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func parseCSVEnvDefault(name string, defaults []string) []string {
	if vals := parseCSVEnv(name); len(vals) > 0 {
		return vals
	}
	return defaults
}

func parseIntEnvDefault(name string, fallback int64) int64 {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	if n < 0 {
		return fallback
	}
	return n
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
