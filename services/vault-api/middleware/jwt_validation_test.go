package middleware

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name    string
		auth    string
		wantOK  bool
		wantTok string
	}{
		{name: "valid bearer", auth: "Bearer abc", wantOK: true, wantTok: "abc"},
		{name: "case insensitive", auth: "bearer xyz", wantOK: true, wantTok: "xyz"},
		{name: "missing prefix", auth: "xyz", wantOK: false},
		{name: "empty token", auth: "Bearer   ", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractBearerToken(tt.auth)
			if ok != tt.wantOK {
				t.Fatalf("expected ok=%v got=%v", tt.wantOK, ok)
			}
			if got != tt.wantTok {
				t.Fatalf("expected token %q got %q", tt.wantTok, got)
			}
		})
	}
}

func TestValidateStandardClaims(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	base := jwt.MapClaims{
		"sub": "user-1",
		"iss": "issuer-a",
		"aud": []string{"vault-api"},
		"iat": float64(now.Add(-30 * time.Second).Unix()),
		"nbf": float64(now.Add(-30 * time.Second).Unix()),
		"exp": float64(now.Add(5 * time.Minute).Unix()),
	}

	if !validateStandardClaims(base, "issuer-a", []string{"vault-api"}, now, 60*time.Second, 0) {
		t.Fatalf("expected valid claims")
	}
	if validateStandardClaims(base, "wrong-issuer", []string{"vault-api"}, now, 60*time.Second, 0) {
		t.Fatalf("expected issuer mismatch to fail")
	}
	if validateStandardClaims(base, "issuer-a", []string{"other-aud"}, now, 60*time.Second, 0) {
		t.Fatalf("expected audience mismatch to fail")
	}

	expired := jwt.MapClaims{
		"sub": "user-1",
		"exp": float64(now.Add(-2 * time.Minute).Unix()),
	}
	if validateStandardClaims(expired, "", nil, now, 0, 0) {
		t.Fatalf("expected expired token to fail")
	}
}

func TestIsTestJWTAllowedEnv(t *testing.T) {
	tests := []struct {
		name   string
		appEnv string
		want   bool
	}{
		{name: "empty defaults to allowed", appEnv: "", want: true},
		{name: "development allowed", appEnv: "development", want: true},
		{name: "dev allowed", appEnv: "dev", want: true},
		{name: "ci allowed", appEnv: "ci", want: true},
		{name: "production disallowed", appEnv: "production", want: false},
		{name: "staging disallowed", appEnv: "staging", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTestJWTAllowedEnv(tt.appEnv); got != tt.want {
				t.Fatalf("expected %v got %v", tt.want, got)
			}
		})
	}
}

func TestValidateStandardClaims_RequiresSubject(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	claims := jwt.MapClaims{
		"iss": "issuer-a",
		"aud": []string{"vault-api"},
		"iat": float64(now.Add(-30 * time.Second).Unix()),
		"nbf": float64(now.Add(-30 * time.Second).Unix()),
		"exp": float64(now.Add(5 * time.Minute).Unix()),
	}
	if validateStandardClaims(claims, "issuer-a", []string{"vault-api"}, now, 60*time.Second, 0) {
		t.Fatalf("expected missing subject to fail")
	}
}

func TestValidateStandardClaims_MaxTokenTTL(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	claims := jwt.MapClaims{
		"sub": "user-1",
		"iss": "issuer-a",
		"aud": []string{"vault-api"},
		"iat": float64(now.Add(-30 * time.Second).Unix()),
		"nbf": float64(now.Add(-30 * time.Second).Unix()),
		"exp": float64(now.Add(2 * time.Hour).Unix()),
	}
	if validateStandardClaims(claims, "issuer-a", []string{"vault-api"}, now, 60*time.Second, 30*time.Minute) {
		t.Fatalf("expected token ttl above max to fail")
	}
	if !validateStandardClaims(claims, "issuer-a", []string{"vault-api"}, now, 60*time.Second, 3*time.Hour) {
		t.Fatalf("expected token ttl under max to pass")
	}
}

func TestValidateStandardClaims_RejectsExpBeforeIAT(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	claims := jwt.MapClaims{
		"sub": "user-1",
		"iss": "issuer-a",
		"aud": []string{"vault-api"},
		"iat": float64(now.Add(10 * time.Minute).Unix()),
		"nbf": float64(now.Add(-30 * time.Second).Unix()),
		"exp": float64(now.Add(5 * time.Minute).Unix()),
	}
	if validateStandardClaims(claims, "issuer-a", []string{"vault-api"}, now, 60*time.Second, 30*time.Minute) {
		t.Fatalf("expected exp before iat to fail")
	}
}

func TestResolveAuthPolicy(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T)
		appEnv   string
		jwksURL  string
		wantMode string
	}{
		{
			name: "explicit jwks rbac",
			setup: func(t *testing.T) {
				t.Setenv("AUTH_POLICY", "jwks_rbac")
			},
			wantMode: authPolicyJWKSRBAC,
		},
		{
			name: "legacy roles maps to jwks rbac",
			setup: func(t *testing.T) {
				t.Setenv("JWT_REQUIRE_ROLES", "true")
			},
			wantMode: authPolicyJWKSRBAC,
		},
		{
			name: "jwks url maps to strict",
			setup: func(t *testing.T) {
				t.Setenv("JWKS_URL", "http://example/jwks.json")
			},
			jwksURL:  "http://example/jwks.json",
			wantMode: authPolicyJWKSStrict,
		},
		{
			name: "dev env defaults to dev policy",
			setup: func(t *testing.T) {
			},
			appEnv:   "development",
			wantMode: authPolicyDev,
		},
		{
			name: "prod env defaults to strict",
			setup: func(t *testing.T) {
			},
			appEnv:   "production",
			wantMode: authPolicyJWKSStrict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)
			got := resolveAuthPolicy(tt.appEnv, tt.jwksURL)
			if got.Mode != tt.wantMode {
				t.Fatalf("expected mode %q got %q", tt.wantMode, got.Mode)
			}
		})
	}
}

func TestHasMinimumRBACRoles(t *testing.T) {
	if hasMinimumRBACRoles([]string{"viewer"}) {
		t.Fatalf("expected viewer-only roles to fail")
	}
	if !hasMinimumRBACRoles([]string{"auditor"}) {
		t.Fatalf("expected auditor role to pass")
	}
	if !hasMinimumRBACRoles([]string{"ingester"}) {
		t.Fatalf("expected ingester role to pass")
	}
}
