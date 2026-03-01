package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsMiddlewareRecordsAndServesMetrics(t *testing.T) {
	h := Metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evidence", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected %d got %d", http.StatusCreated, rr.Code)
	}

	mrr := httptest.NewRecorder()
	mreq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	MetricsHandler().ServeHTTP(mrr, mreq)
	if mrr.Code != http.StatusOK {
		t.Fatalf("expected metrics status 200 got %d", mrr.Code)
	}
	body := mrr.Body.String()
	if body == "" {
		t.Fatalf("expected non-empty metrics body")
	}
	if !strings.Contains(body, "vault_api_http_request_duration_seconds_bucket{operation=\"ingest\",le=\"0.01\"}") {
		t.Fatalf("expected ingest histogram bucket metric in output")
	}
	if !strings.Contains(body, "vault_api_http_requests_operation_total{operation=\"ingest\",status_class=\"2xx\"}") {
		t.Fatalf("expected operation/status counter metric in output")
	}
}
