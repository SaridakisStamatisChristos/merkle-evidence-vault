package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsHandler(t *testing.T) {
	IncSignRequests()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "checkpoint_svc_sign_requests_total") {
		t.Fatalf("missing checkpoint_svc_sign_requests_total in output")
	}
}
