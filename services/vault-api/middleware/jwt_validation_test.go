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

	if !validateStandardClaims(base, "issuer-a", []string{"vault-api"}, now, 60*time.Second) {
		t.Fatalf("expected valid claims")
	}
	if validateStandardClaims(base, "wrong-issuer", []string{"vault-api"}, now, 60*time.Second) {
		t.Fatalf("expected issuer mismatch to fail")
	}
	if validateStandardClaims(base, "issuer-a", []string{"other-aud"}, now, 60*time.Second) {
		t.Fatalf("expected audience mismatch to fail")
	}

	expired := jwt.MapClaims{
		"sub": "user-1",
		"exp": float64(now.Add(-2 * time.Minute).Unix()),
	}
	if validateStandardClaims(expired, "", nil, now, 0) {
		t.Fatalf("expected expired token to fail")
	}
}
