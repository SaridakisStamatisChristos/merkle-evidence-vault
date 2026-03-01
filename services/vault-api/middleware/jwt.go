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

	authPolicyDev        = "dev"
	authPolicyJWKSStrict = "jwks_strict"
	authPolicyJWKSRBAC   = "jwks_rbac"
)

type resolvedAuthPolicy struct {
	Mode                 string
	Source               string
	EnableTestJWT        bool
	JWKSRequired         bool
	EnforceRequiredClaim bool
	RequireKid           bool
	RequireRoles         bool
}

// JWT is a middleware that validates a Bearer token.
func JWT(next http.Handler) http.Handler {
	var jwks *keyfunc.JWKS
	jwksURL := strings.TrimSpace(os.Getenv("JWKS_URL"))
	appEnv := strings.ToLower(strings.TrimSpace(firstNonEmptyEnv("APP_ENV", "ENVIRONMENT", "DEPLOY_ENV")))
	policy := resolveAuthPolicy(appEnv, jwksURL)

	requiredIssuer := strings.TrimSpace(os.Getenv("JWT_REQUIRED_ISSUER"))
	requiredAudience := parseCSVEnv("JWT_REQUIRED_AUDIENCE")
	allowedAlgs := parseCSVEnvDefault("JWT_ALLOWED_ALGS", []string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512", "EdDSA"})
	clockSkew := parseIntEnvDefault("JWT_CLOCK_SKEW_SECONDS", 60)
	jwksMaxAttempts := parseIntEnvDefault("JWT_JWKS_MAX_ATTEMPTS", 12)
	jwksRetryMs := parseIntEnvDefault("JWT_JWKS_RETRY_MS", 2000)
	maxTokenTTLSeconds := parseIntEnvDefault("JWT_MAX_TOKEN_TTL_SECONDS", 0)

	strictConfigValid := true
	if policy.JWKSRequired && jwksURL == "" {
		strictConfigValid = false
		log.Error().Msg("invalid JWT configuration for JWKS policy: JWKS_URL must be set")
	}
	if policy.EnforceRequiredClaim && (requiredIssuer == "" || len(requiredAudience) == 0) {
		strictConfigValid = false
		log.Error().
			Bool("missing_required_issuer", requiredIssuer == "").
			Bool("missing_required_audience", len(requiredAudience) == 0).
			Str("auth_policy_mode", policy.Mode).
			Msgf("invalid JWT configuration for %s policy: JWT_REQUIRED_ISSUER and JWT_REQUIRED_AUDIENCE must both be set", policy.Mode)
	}

	log.Info().
		Str("resolved_auth_policy", policy.Mode).
		Str("auth_policy_source", policy.Source).
		Bool("enable_test_jwt", policy.EnableTestJWT).
		Bool("jwks_required", policy.JWKSRequired).
		Bool("require_kid", policy.RequireKid).
		Bool("require_roles", policy.RequireRoles).
		Bool("enforce_required_claims", policy.EnforceRequiredClaim).
		Bool("strict_config_valid", strictConfigValid).
		Str("jwks_url", jwksURL).
		Str("app_env", appEnv).
		Str("required_issuer", requiredIssuer).
		Strs("required_audience", requiredAudience).
		Strs("allowed_algs", allowedAlgs).
		Int64("clock_skew_seconds", clockSkew).
		Int64("jwks_max_attempts", jwksMaxAttempts).
		Int64("jwks_retry_ms", jwksRetryMs).
		Int64("max_token_ttl_seconds", maxTokenTTLSeconds).
		Msg("JWT middleware configuration")

	if policy.JWKSRequired && jwksURL != "" {
		var err error
		for i := int64(0); i < jwksMaxAttempts; i++ {
			jwks, err = keyfunc.Get(jwksURL, keyfunc.Options{RefreshInterval: time.Hour})
			if err == nil && jwks != nil {
				log.Info().Str("jwks_url", jwksURL).Msg("JWKS loaded successfully")
				break
			}
			log.Warn().Err(err).Int64("attempt", i+1).Int64("max_attempts", jwksMaxAttempts).Str("jwks_url", jwksURL).Msg("failed to load JWKS, will retry")
			time.Sleep(time.Duration(jwksRetryMs) * time.Millisecond)
		}
		if jwks == nil {
			log.Error().Str("jwks_url", jwksURL).Msg("giving up loading JWKS after retries; JWT auth is fail-closed until JWKS is available")
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr, ok := extractBearerToken(r.Header.Get("Authorization"))
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if policy.JWKSRequired && (!strictConfigValid || (jwksURL != "" && jwks == nil)) {
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
			if policy.RequireKid {
				kid, _ := token.Header["kid"].(string)
				if strings.TrimSpace(kid) == "" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
			}
			if !validateStandardClaims(claims, requiredIssuer, requiredAudience, time.Now(), time.Duration(clockSkew)*time.Second, time.Duration(maxTokenTTLSeconds)*time.Second) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			roles := parseRoles(claims)
			if policy.RequireRoles && !hasMinimumRBACRoles(roles) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			sub, _ := claims["sub"].(string)
			ctx := context.WithValue(r.Context(), ctxKeyRoles, roles)
			ctx = context.WithValue(ctx, ctxKeySub, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if !policy.EnableTestJWT {
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

func resolveAuthPolicy(appEnv, jwksURL string) resolvedAuthPolicy {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("AUTH_POLICY")))
	source := "explicit_auth_policy"
	if mode == "" {
		requireRolesLegacy := parseBoolEnvDefault("JWT_REQUIRE_ROLES", false)
		requireKidLegacy := parseBoolEnvDefault("JWT_REQUIRE_KID", false)
		enforceRequiredClaimsLegacy := parseBoolEnvDefault("JWT_ENFORCE_REQUIRED_CLAIMS", false)
		enableTestLegacy := parseBoolEnvDefault("ENABLE_TEST_JWT", false)
		switch {
		case requireRolesLegacy:
			mode = authPolicyJWKSRBAC
			source = "legacy_jwt_require_roles"
		case strings.TrimSpace(jwksURL) != "" || requireKidLegacy || enforceRequiredClaimsLegacy:
			mode = authPolicyJWKSStrict
			source = "legacy_jwks_settings"
		case enableTestLegacy:
			mode = authPolicyDev
			source = "legacy_enable_test_jwt"
		default:
			mode = authPolicyJWKSStrict
			source = "default_strict"
		}
	}

	switch mode {
	case authPolicyDev:
		if !isTestJWTAllowedEnv(appEnv) {
			log.Error().Str("app_env", appEnv).Msg("AUTH_POLICY=dev is only allowed in local/dev/test environments; falling back to jwks_strict")
			return resolvedAuthPolicy{Mode: authPolicyJWKSStrict, Source: source + "_downgraded_non_dev", JWKSRequired: strings.TrimSpace(jwksURL) != "", EnforceRequiredClaim: true, RequireKid: true}
		}
		return resolvedAuthPolicy{Mode: authPolicyDev, Source: source, EnableTestJWT: true}
	case authPolicyJWKSRBAC:
		return resolvedAuthPolicy{Mode: authPolicyJWKSRBAC, Source: source, JWKSRequired: true, EnforceRequiredClaim: true, RequireKid: true, RequireRoles: true}
	case authPolicyJWKSStrict:
		return resolvedAuthPolicy{Mode: authPolicyJWKSStrict, Source: source, JWKSRequired: strings.TrimSpace(jwksURL) != "", EnforceRequiredClaim: true, RequireKid: true}
	default:
		log.Error().Str("auth_policy", mode).Msg("invalid AUTH_POLICY value; defaulting to jwks_strict")
		return resolvedAuthPolicy{Mode: authPolicyJWKSStrict, Source: "invalid_auth_policy_default", JWKSRequired: strings.TrimSpace(jwksURL) != "", EnforceRequiredClaim: true, RequireKid: true}
	}
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

func hasMinimumRBACRoles(roles []string) bool {
	for _, role := range roles {
		if role == "auditor" || role == "ingester" {
			return true
		}
	}
	return false
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

func validateStandardClaims(claims jwt.MapClaims, issuer string, audiences []string, now time.Time, skew time.Duration, maxTokenTTL time.Duration) bool {
	sub, ok := claims["sub"].(string)
	if !ok || strings.TrimSpace(sub) == "" {
		return false
	}
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
	if maxTokenTTL > 0 {
		issuedAt, hasIAT := claims["iat"].(float64)
		expiresAt, hasEXP := claims["exp"].(float64)
		if !hasIAT || !hasEXP {
			return false
		}
		if expiresAt < issuedAt {
			return false
		}
		if time.Unix(int64(expiresAt), 0).Sub(time.Unix(int64(issuedAt), 0)) > maxTokenTTL {
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

func parseBoolEnvDefault(name string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
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
