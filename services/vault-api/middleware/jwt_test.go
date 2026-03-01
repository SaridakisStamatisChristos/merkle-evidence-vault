package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJWT_TestModeRoles(t *testing.T) {
	// enable explicit test-mode for this unit test
	t.Setenv("ENABLE_TEST_JWT", "true")
	// handler that inspects roles from context
	h := JWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roles := RolesFromContext(r.Context())
		if len(roles) == 0 {
			w.WriteHeader(204)
			return
		}
		// echo first role
		if _, err := w.Write([]byte(roles[0])); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))

	// auditor token
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer auditor-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
	if got := rr.Body.String(); got != "auditor" {
		t.Fatalf("expected auditor role, got %q", got)
	}

	// ingester token
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer ingester-token")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
	if got := rr.Body.String(); got != "ingester" {
		t.Fatalf("expected ingester role, got %q", got)
	}

	// missing auth -> 401
	req = httptest.NewRequest("GET", "/", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestJWT_TestModeDisabledInProductionEnv(t *testing.T) {
	t.Setenv("ENABLE_TEST_JWT", "true")
	t.Setenv("APP_ENV", "production")

	h := JWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer auditor-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestJWT_JWKSConfiguredButUnavailableFailsClosed(t *testing.T) {
	t.Setenv("ENABLE_TEST_JWT", "true")
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWKS_URL", "http://127.0.0.1:1/unreachable-jwks")
	t.Setenv("JWT_JWKS_MAX_ATTEMPTS", "1")
	t.Setenv("JWT_JWKS_RETRY_MS", "0")
	t.Setenv("JWT_REQUIRED_ISSUER", "test-issuer")
	t.Setenv("JWT_REQUIRED_AUDIENCE", "test-audience")

	h := JWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer auditor-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}
