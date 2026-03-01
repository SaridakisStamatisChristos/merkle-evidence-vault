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

	vaultRequestsByOperationStatus sync.Map // map[string]*uint64, key=operation|statusClass
	vaultDurationSumByOperation    sync.Map // map[string]*uint64, microseconds
	vaultDurationCountByOperation  sync.Map // map[string]*uint64
	vaultDurationBucketsByOp       sync.Map // map[string][]*uint64
)

var durationBucketsSeconds = []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}

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

		op := classifyOperation(r.Method, r.URL.Path)
		statusClass := classifyStatusClass(sr.status)
		opStatusKey := op + "|" + statusClass
		opStatusPtr, _ := vaultRequestsByOperationStatus.LoadOrStore(opStatusKey, new(uint64))
		atomic.AddUint64(opStatusPtr.(*uint64), 1)

		sumPtr, _ := vaultDurationSumByOperation.LoadOrStore(op, new(uint64))
		atomic.AddUint64(sumPtr.(*uint64), durMicros)
		countPtr, _ := vaultDurationCountByOperation.LoadOrStore(op, new(uint64))
		atomic.AddUint64(countPtr.(*uint64), 1)

		bucketVal, _ := vaultDurationBucketsByOp.LoadOrStore(op, newBucketCounters())
		for idx, le := range durationBucketsSeconds {
			if float64(durMicros)/1_000_000.0 <= le {
				atomic.AddUint64(bucketVal.([]*uint64)[idx], 1)
			}
		}
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

		b.WriteString("# HELP vault_api_http_requests_operation_total HTTP requests partitioned by operation and status class.\n")
		b.WriteString("# TYPE vault_api_http_requests_operation_total counter\n")
		vaultRequestsByOperationStatus.Range(func(k, v interface{}) bool {
			parts := strings.SplitN(k.(string), "|", 2)
			count := atomic.LoadUint64(v.(*uint64))
			b.WriteString(fmt.Sprintf("vault_api_http_requests_operation_total{operation=\"%s\",status_class=\"%s\"} %d\n", parts[0], parts[1], count))
			return true
		})

		b.WriteString("# HELP vault_api_http_request_duration_seconds_bucket Cumulative request duration histogram buckets by operation.\n")
		b.WriteString("# TYPE vault_api_http_request_duration_seconds_bucket counter\n")
		vaultDurationBucketsByOp.Range(func(k, v interface{}) bool {
			op := k.(string)
			counters := v.([]*uint64)
			for i, le := range durationBucketsSeconds {
				cnt := atomic.LoadUint64(counters[i])
				b.WriteString(fmt.Sprintf("vault_api_http_request_duration_seconds_bucket{operation=\"%s\",le=\"%g\"} %d\n", op, le, cnt))
			}
			countPtr, _ := vaultDurationCountByOperation.Load(op)
			count := atomic.LoadUint64(countPtr.(*uint64))
			b.WriteString(fmt.Sprintf("vault_api_http_request_duration_seconds_bucket{operation=\"%s\",le=\"+Inf\"} %d\n", op, count))
			return true
		})

		b.WriteString("# HELP vault_api_http_request_duration_seconds_sum_by_operation Sum of request durations in seconds by operation.\n")
		b.WriteString("# TYPE vault_api_http_request_duration_seconds_sum_by_operation counter\n")
		vaultDurationSumByOperation.Range(func(k, v interface{}) bool {
			op := k.(string)
			sumSeconds := float64(atomic.LoadUint64(v.(*uint64))) / 1_000_000.0
			b.WriteString(fmt.Sprintf("vault_api_http_request_duration_seconds_sum_by_operation{operation=\"%s\"} %f\n", op, sumSeconds))
			return true
		})

		b.WriteString("# HELP vault_api_http_request_duration_seconds_count_by_operation Number of observed requests by operation.\n")
		b.WriteString("# TYPE vault_api_http_request_duration_seconds_count_by_operation counter\n")
		vaultDurationCountByOperation.Range(func(k, v interface{}) bool {
			op := k.(string)
			count := atomic.LoadUint64(v.(*uint64))
			b.WriteString(fmt.Sprintf("vault_api_http_request_duration_seconds_count_by_operation{operation=\"%s\"} %d\n", op, count))
			return true
		})

		_, _ = w.Write([]byte(b.String()))
	})
}

func classifyOperation(method string, path string) string {
	if method == http.MethodPost && path == "/api/v1/evidence" {
		return "ingest"
	}
	if method == http.MethodGet && strings.HasPrefix(path, "/api/v1/evidence/") && strings.HasSuffix(path, "/proof") {
		return "proof"
	}
	return "other"
}

func classifyStatusClass(status int) string {
	if status >= 500 {
		return "5xx"
	}
	if status >= 400 {
		return "4xx"
	}
	if status >= 300 {
		return "3xx"
	}
	return "2xx"
}

func newBucketCounters() []*uint64 {
	out := make([]*uint64, len(durationBucketsSeconds))
	for i := range out {
		out[i] = new(uint64)
	}
	return out
}
