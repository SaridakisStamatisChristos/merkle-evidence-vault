package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

var (
	signRequestsTotal   uint64
	signFailuresTotal   uint64
	lastSignSuccessUnix int64
)

func IncSignRequests() {
	atomic.AddUint64(&signRequestsTotal, 1)
}

func IncSignFailures() {
	atomic.AddUint64(&signFailuresTotal, 1)
}

func RecordSignSuccess() {
	atomic.StoreInt64(&lastSignSuccessUnix, time.Now().Unix())
}

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		requests := atomic.LoadUint64(&signRequestsTotal)
		failures := atomic.LoadUint64(&signFailuresTotal)
		lastSuccess := atomic.LoadInt64(&lastSignSuccessUnix)
		_, _ = w.Write([]byte(fmt.Sprintf(`# HELP checkpoint_svc_sign_requests_total Total number of /sign requests.
# TYPE checkpoint_svc_sign_requests_total counter
checkpoint_svc_sign_requests_total %d
# HELP checkpoint_svc_sign_failures_total Total number of failed /sign requests.
# TYPE checkpoint_svc_sign_failures_total counter
checkpoint_svc_sign_failures_total %d
# HELP checkpoint_svc_last_sign_success_unixtime Unix timestamp of the latest successful sign operation.
# TYPE checkpoint_svc_last_sign_success_unixtime gauge
checkpoint_svc_last_sign_success_unixtime %d
`, requests, failures, lastSuccess)))
	})
}
