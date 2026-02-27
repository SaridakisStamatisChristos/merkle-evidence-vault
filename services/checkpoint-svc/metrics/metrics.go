package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

var (
	signRequestsTotal uint64
	signFailuresTotal uint64
)

func IncSignRequests() {
	atomic.AddUint64(&signRequestsTotal, 1)
}

func IncSignFailures() {
	atomic.AddUint64(&signFailuresTotal, 1)
}

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		requests := atomic.LoadUint64(&signRequestsTotal)
		failures := atomic.LoadUint64(&signFailuresTotal)
		_, _ = w.Write([]byte(fmt.Sprintf(`# HELP checkpoint_svc_sign_requests_total Total number of /sign requests.
# TYPE checkpoint_svc_sign_requests_total counter
checkpoint_svc_sign_requests_total %d
# HELP checkpoint_svc_sign_failures_total Total number of failed /sign requests.
# TYPE checkpoint_svc_sign_failures_total counter
checkpoint_svc_sign_failures_total %d
`, requests, failures)))
	})
}
