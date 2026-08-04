package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
)

var upstreamKubeDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "ecp_gateway_upstream_kube_request_duration_seconds",
	Help:    "Latency of one gateway→apiserver operation via the resource adapter.",
	Buckets: httpBuckets,
}, []string{"resource", "group", "operation", "result"})

// promUpstreamObserver records adapter operations into Prometheus.
type promUpstreamObserver struct{}

// Observe implements k8sadapter.Observer.
func (promUpstreamObserver) Observe(resource, group, operation, result string, d time.Duration) {
	upstreamKubeDuration.WithLabelValues(resource, group, operation, result).Observe(d.Seconds())
}

// RegisterUpstreamObserver installs the Prometheus observer on the framework
// Kubernetes adapters. Call once at process start in both global and regional gateways.
func RegisterUpstreamObserver() {
	k8sadapter.SetUpstreamObserver(promUpstreamObserver{})
}
