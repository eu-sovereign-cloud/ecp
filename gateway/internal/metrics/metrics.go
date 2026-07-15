// Package metrics provides Prometheus instrumentation for the ECP gateway.
//
// Three histograms are registered on the default registry so that the standard
// go_* and process_* collectors are also exported — useful for comparing memory
// and CPU overhead between the direct and cached RBAC checker implementations.
//
//   - ecp_gateway_auth_middleware_duration_seconds{provider} — end-to-end latency
//     of the full authenticated request (authn + authz + provider handler). Metric (a).
//   - ecp_gateway_authz_check_duration_seconds{impl} — latency of a single
//     Checker.Authorize call. Metric (b).
//   - ecp_gateway_rbac_fetch_duration_seconds{impl} — latency of the RBAC data
//     fetch (Kubernetes List for the direct checker; informer cache read for cached).
//     Metric (c).
//
// Buckets span ≈50µs–3s (18 exponential steps, factor 2) to resolve both the
// sub-millisecond cached path and the multi-millisecond direct path in detail.
package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// buckets covers ~50µs to ~3.3s in 18 exponential steps (factor 2).
var buckets = prometheus.ExponentialBuckets(50e-6, 2, 18)

var (
	authMiddlewareDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ecp_gateway_auth_middleware_duration_seconds",
		Help:    "End-to-end latency of an authenticated request (authn + authz + handler).",
		Buckets: buckets,
	}, []string{"provider"})

	authzCheckDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ecp_gateway_authz_check_duration_seconds",
		Help:    "Latency of a single Checker.Authorize call.",
		Buckets: buckets,
	}, []string{"impl"})

	rbacFetchDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ecp_gateway_rbac_fetch_duration_seconds",
		Help:    "Latency of the RBAC data fetch inside the checker (List or cache read).",
		Buckets: buckets,
	}, []string{"impl"})
)

// Handler returns the standard Prometheus metrics HTTP handler.
// Mount it outside the provider HandlerWithOptions so it requires no bearer token.
func Handler() http.Handler {
	return promhttp.Handler()
}

// Middleware returns an outer HTTP middleware that records the end-to-end latency
// of an authenticated request for the named provider. Wire it as the outermost
// wrapper in auth.ProviderMWs so it times authn + authz + the provider handler.
func Middleware(provider string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			authMiddlewareDuration.WithLabelValues(provider).Observe(time.Since(start).Seconds())
		})
	}
}

// ObserveAuthzCheck records the duration of a single Checker.Authorize call.
// Call with the impl label ("direct" or "cached") and the elapsed duration.
func ObserveAuthzCheck(impl string, d time.Duration) {
	authzCheckDuration.WithLabelValues(impl).Observe(d.Seconds())
}

// ObserveRBACFetch records the duration of the RBAC data fetch inside a checker.
// Call with the impl label ("direct" or "cached") and the elapsed duration.
func ObserveRBACFetch(impl string, d time.Duration) {
	rbacFetchDuration.WithLabelValues(impl).Observe(d.Seconds())
}
