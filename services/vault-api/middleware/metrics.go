package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	vaultRequestsTotal         uint64
	vaultRequestDurationMicros uint64
	vaultRequestsByStatus      sync.Map // map[string]*uint64
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(sr, r)

		atomic.AddUint64(&vaultRequestsTotal, 1)
		durMicros := uint64(time.Since(start).Microseconds())
		atomic.AddUint64(&vaultRequestDurationMicros, durMicros)

		status := strconv.Itoa(sr.status)
		ptr, _ := vaultRequestsByStatus.LoadOrStore(status, new(uint64))
		atomic.AddUint64(ptr.(*uint64), 1)
	})
}

func MetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		total := atomic.LoadUint64(&vaultRequestsTotal)
		durationSum := float64(atomic.LoadUint64(&vaultRequestDurationMicros)) / 1_000_000.0
		avg := 0.0
		if total > 0 {
			avg = durationSum / float64(total)
		}

		var b strings.Builder
		b.WriteString("# HELP vault_api_http_requests_total Total number of HTTP requests handled by vault-api.\n")
		b.WriteString("# TYPE vault_api_http_requests_total counter\n")
		b.WriteString(fmt.Sprintf("vault_api_http_requests_total %d\n", total))
		b.WriteString("# HELP vault_api_http_request_duration_seconds_sum Sum of request durations in seconds.\n")
		b.WriteString("# TYPE vault_api_http_request_duration_seconds_sum counter\n")
		b.WriteString(fmt.Sprintf("vault_api_http_request_duration_seconds_sum %f\n", durationSum))
		b.WriteString("# HELP vault_api_http_request_duration_seconds_avg Average request duration in seconds.\n")
		b.WriteString("# TYPE vault_api_http_request_duration_seconds_avg gauge\n")
		b.WriteString(fmt.Sprintf("vault_api_http_request_duration_seconds_avg %f\n", avg))
		b.WriteString("# HELP vault_api_http_requests_by_status HTTP requests by status code.\n")
		b.WriteString("# TYPE vault_api_http_requests_by_status counter\n")
		vaultRequestsByStatus.Range(func(k, v interface{}) bool {
			status := k.(string)
			count := atomic.LoadUint64(v.(*uint64))
			b.WriteString(fmt.Sprintf("vault_api_http_requests_by_status{status=\"%s\"} %d\n", status, count))
			return true
		})

		_, _ = w.Write([]byte(b.String()))
	})
}
