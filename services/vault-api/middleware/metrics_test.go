package middleware

import (
	"net/http"
	"net/http/httptest"
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
	if body := mrr.Body.String(); body == "" {
		t.Fatalf("expected non-empty metrics body")
	}
}
