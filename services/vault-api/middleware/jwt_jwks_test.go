package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestJWT_JWKSFailureModes(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	jwksBody, err := json.Marshal(map[string]interface{}{
		"keys": []map[string]string{{
			"kty": "RSA",
			"kid": "k1",
			"alg": "RS256",
			"use": "sig",
			"n":   b64u(priv.N.Bytes()),
			"e":   b64u(big.NewInt(int64(priv.PublicKey.E)).Bytes()),
		}},
	})
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}

	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, writeErr := w.Write(jwksBody); writeErr != nil {
			t.Errorf("write jwks response: %v", writeErr)
		}
	}))
	defer jwksSrv.Close()

	t.Setenv("JWKS_URL", jwksSrv.URL)
	t.Setenv("ENABLE_TEST_JWT", "false")
	t.Setenv("JWT_ALLOWED_ALGS", "RS256")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if SubjectFromContext(r.Context()) == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	validToken := mintRS256Token(t, priv, "k1", "issuer-ok", []string{"vault-api"}, []string{"auditor"}, "subject-1")
	hsToken := mintHS256Token(t, "issuer-ok", []string{"vault-api"}, "subject-hs")

	tests := []struct {
		name     string
		auth     string
		issuer   string
		audience string
		wantCode int
	}{
		{name: "valid token", auth: "Bearer " + validToken, issuer: "issuer-ok", audience: "vault-api", wantCode: http.StatusOK},
		{name: "invalid auth scheme", auth: "Token " + validToken, issuer: "issuer-ok", audience: "vault-api", wantCode: http.StatusUnauthorized},
		{name: "issuer mismatch", auth: "Bearer " + validToken, issuer: "issuer-other", audience: "vault-api", wantCode: http.StatusUnauthorized},
		{name: "audience mismatch", auth: "Bearer " + validToken, issuer: "issuer-ok", audience: "other-api", wantCode: http.StatusUnauthorized},
		{name: "disallowed algorithm", auth: "Bearer " + hsToken, issuer: "issuer-ok", audience: "vault-api", wantCode: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_REQUIRED_ISSUER", tt.issuer)
			t.Setenv("JWT_REQUIRED_AUDIENCE", tt.audience)

			h := JWT(next)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tt.auth)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != tt.wantCode {
				t.Fatalf("expected %d got %d", tt.wantCode, rr.Code)
			}
		})
	}
}

func mintRS256Token(t *testing.T, priv *rsa.PrivateKey, kid, iss string, aud, roles []string, sub string) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   iss,
		"aud":   aud,
		"sub":   sub,
		"roles": roles,
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"exp":   now.Add(5 * time.Minute).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("sign rs256: %v", err)
	}
	return s
}

func mintHS256Token(t *testing.T, iss string, aud []string, sub string) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": iss,
		"aud": aud,
		"sub": sub,
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign hs256: %v", err)
	}
	return s
}

func mintRS256TokenWithoutKid(t *testing.T, priv *rsa.PrivateKey, iss string, aud, roles []string, sub string) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   iss,
		"aud":   aud,
		"sub":   sub,
		"roles": roles,
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"exp":   now.Add(5 * time.Minute).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	s, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("sign rs256 without kid: %v", err)
	}
	return s
}
func b64u(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func TestJWT_JWKSModeRequiresIssuerAndAudienceConfig(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	jwksBody, err := json.Marshal(map[string]interface{}{
		"keys": []map[string]string{{
			"kty": "RSA",
			"kid": "k1",
			"alg": "RS256",
			"use": "sig",
			"n":   b64u(priv.N.Bytes()),
			"e":   b64u(big.NewInt(int64(priv.PublicKey.E)).Bytes()),
		}},
	})
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}

	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, writeErr := w.Write(jwksBody); writeErr != nil {
			t.Errorf("write jwks response: %v", writeErr)
		}
	}))
	defer jwksSrv.Close()

	t.Setenv("JWKS_URL", jwksSrv.URL)
	t.Setenv("ENABLE_TEST_JWT", "true")
	t.Setenv("JWT_ALLOWED_ALGS", "RS256")
	t.Setenv("JWT_ENFORCE_REQUIRED_CLAIMS", "true")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	validToken := mintRS256Token(t, priv, "k1", "issuer-ok", []string{"vault-api"}, []string{"auditor"}, "subject-1")

	tests := []struct {
		name     string
		issuer   string
		audience string
	}{
		{name: "missing issuer", issuer: "", audience: "vault-api"},
		{name: "missing audience", issuer: "issuer-ok", audience: ""},
		{name: "missing both", issuer: "", audience: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_REQUIRED_ISSUER", tt.issuer)
			t.Setenv("JWT_REQUIRED_AUDIENCE", tt.audience)

			h := JWT(next)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+validToken)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected %d got %d", http.StatusUnauthorized, rr.Code)
			}
		})
	}
}

func TestJWT_JWKSModeMissingIssuerAudienceRejectedInStrictPolicy(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	jwksBody, err := json.Marshal(map[string]interface{}{
		"keys": []map[string]string{{
			"kty": "RSA",
			"kid": "k1",
			"alg": "RS256",
			"use": "sig",
			"n":   b64u(priv.N.Bytes()),
			"e":   b64u(big.NewInt(int64(priv.PublicKey.E)).Bytes()),
		}},
	})
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}

	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, writeErr := w.Write(jwksBody); writeErr != nil {
			t.Errorf("write jwks response: %v", writeErr)
		}
	}))
	defer jwksSrv.Close()

	t.Setenv("JWKS_URL", jwksSrv.URL)
	t.Setenv("ENABLE_TEST_JWT", "true")
	t.Setenv("JWT_ALLOWED_ALGS", "RS256")
	t.Setenv("JWT_REQUIRED_ISSUER", "")
	t.Setenv("JWT_REQUIRED_AUDIENCE", "")
	t.Setenv("JWT_ENFORCE_REQUIRED_CLAIMS", "false")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	validToken := mintRS256Token(t, priv, "k1", "issuer-ok", []string{"vault-api"}, []string{"auditor"}, "subject-1")

	h := JWT(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestJWT_JWKSRoleRequirement(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	jwksBody, err := json.Marshal(map[string]interface{}{
		"keys": []map[string]string{{
			"kty": "RSA",
			"kid": "k1",
			"alg": "RS256",
			"use": "sig",
			"n":   b64u(priv.N.Bytes()),
			"e":   b64u(big.NewInt(int64(priv.PublicKey.E)).Bytes()),
		}},
	})
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}

	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, writeErr := w.Write(jwksBody); writeErr != nil {
			t.Errorf("write jwks response: %v", writeErr)
		}
	}))
	defer jwksSrv.Close()

	t.Setenv("JWKS_URL", jwksSrv.URL)
	t.Setenv("JWT_ALLOWED_ALGS", "RS256")
	t.Setenv("JWT_REQUIRED_ISSUER", "issuer-ok")
	t.Setenv("JWT_REQUIRED_AUDIENCE", "vault-api")
	t.Setenv("JWT_REQUIRE_ROLES", "true")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	withRoles := mintRS256Token(t, priv, "k1", "issuer-ok", []string{"vault-api"}, []string{"auditor"}, "subject-1")
	withoutRoles := mintRS256Token(t, priv, "k1", "issuer-ok", []string{"vault-api"}, nil, "subject-2")

	tests := []struct {
		name     string
		token    string
		wantCode int
	}{
		{name: "token with roles", token: withRoles, wantCode: http.StatusOK},
		{name: "token without roles", token: withoutRoles, wantCode: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := JWT(next)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != tt.wantCode {
				t.Fatalf("expected %d got %d", tt.wantCode, rr.Code)
			}
		})
	}
}

func TestJWT_JWKSRequireKid(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	jwksBody, err := json.Marshal(map[string]interface{}{
		"keys": []map[string]string{{
			"kty": "RSA",
			"kid": "k1",
			"alg": "RS256",
			"use": "sig",
			"n":   b64u(priv.N.Bytes()),
			"e":   b64u(big.NewInt(int64(priv.PublicKey.E)).Bytes()),
		}},
	})
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}

	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, writeErr := w.Write(jwksBody); writeErr != nil {
			t.Errorf("write jwks response: %v", writeErr)
			return
		}
	}))
	defer jwksSrv.Close()

	t.Setenv("JWKS_URL", jwksSrv.URL)
	t.Setenv("JWT_ALLOWED_ALGS", "RS256")
	t.Setenv("JWT_REQUIRED_ISSUER", "issuer-ok")
	t.Setenv("JWT_REQUIRED_AUDIENCE", "vault-api")
	t.Setenv("JWT_REQUIRE_KID", "true")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	withKid := mintRS256Token(t, priv, "k1", "issuer-ok", []string{"vault-api"}, []string{"auditor"}, "subject-1")
	withoutKid := mintRS256TokenWithoutKid(t, priv, "issuer-ok", []string{"vault-api"}, []string{"auditor"}, "subject-2")

	tests := []struct {
		name     string
		token    string
		wantCode int
	}{
		{name: "token with kid", token: withKid, wantCode: http.StatusOK},
		{name: "token without kid", token: withoutKid, wantCode: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := JWT(next)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != tt.wantCode {
				t.Fatalf("expected %d got %d", tt.wantCode, rr.Code)
			}
		})
	}
}
