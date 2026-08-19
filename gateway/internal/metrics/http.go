package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "ecp_gateway_http_request_duration_seconds",
	Help:    "End-to-end latency of an inbound HTTP request.",
	Buckets: httpBuckets,
}, []string{"method", "status", "route", "provider"})

// HTTPMiddleware records ecp_gateway_http_request_duration_seconds for every
// request except /metrics (scrapes must not observe themselves). Wire it as the
// outermost handler around the shared mux so it runs with auth enabled or disabled.
//
// After the inner handler returns, route is taken from r.Pattern when the
// ServeMux matched a template; otherwise the path is normalized to a
// low-cardinality template (see routeFromRequest).
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		route, provider := routeFromRequest(r)
		httpRequestDuration.WithLabelValues(
			normalizeMethod(r.Method),
			strconv.Itoa(rec.status),
			route,
			provider,
		).Observe(time.Since(start).Seconds())
	})
}

// statusRecorder captures the HTTP status code written by the handler.
// Default is 200, matching net/http when Write is used without WriteHeader.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.wroteHeader = true
		// status already defaults to 200
	}
	return r.ResponseWriter.Write(b)
}

func normalizeMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions:
		return method
	default:
		return "OTHER"
	}
}
