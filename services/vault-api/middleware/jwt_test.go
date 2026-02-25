package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestJWT_TestModeRoles(t *testing.T) {
	// enable explicit test-mode for this unit test
	os.Setenv("ENABLE_TEST_JWT", "true")
	defer os.Unsetenv("ENABLE_TEST_JWT")
	// handler that inspects roles from context
	h := JWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roles := RolesFromContext(r.Context())
		if len(roles) == 0 {
			w.WriteHeader(204)
			return
		}
		// echo first role
		w.Write([]byte(roles[0]))
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
