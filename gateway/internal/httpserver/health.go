package httpserver

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

// DefaultReadyCheckTimeout bounds dependency checks on /readyz so a hung
// apiserver cannot stall kubelet readiness probes indefinitely.
const DefaultReadyCheckTimeout = 2 * time.Second

// Readiness is a process-level ready gate. It starts false so probes fail
// until wiring is complete, and should be cleared when shutdown begins so
// traffic is removed from the Service before the drain window ends.
type Readiness struct {
	ready atomic.Bool
}

// NewReadiness returns a gate that is not ready.
func NewReadiness() *Readiness {
	return &Readiness{}
}

// Set marks the process ready (true) or not ready (false).
func (r *Readiness) Set(ready bool) {
	if r == nil {
		return
	}
	r.ready.Store(ready)
}

// Ready reports whether the process gate is open.
func (r *Readiness) Ready() bool {
	return r != nil && r.ready.Load()
}

// CheckFunc is a dependency probe used by /readyz. Return nil when healthy.
type CheckFunc func(ctx context.Context) error

// RegisterProbes mounts unauthenticated liveness and readiness endpoints:
//
//   - GET /healthz — always 200 while the process can serve HTTP (liveness)
//   - GET /readyz  — 200 only when the process gate is ready and every check
//     succeeds (readiness); otherwise 503
func RegisterProbes(mux *http.ServeMux, gate *Readiness, checks ...CheckFunc) {
	if mux == nil {
		return
	}
	mux.Handle("GET /healthz", LiveHandler())
	mux.Handle("GET /readyz", ReadyHandler(gate, checks...))
}

// LiveHandler returns a liveness handler that always succeeds.
func LiveHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		// The status is already on the wire and the only reader is a kubelet that has hung up.
		// Nothing here can act on the write failing, and these handlers take no logger precisely
		// so a probe stays free of dependencies.
		_, _ = w.Write([]byte("ok\n"))
	})
}

// ReadyHandler returns a readiness handler gated by process ready state and
// optional dependency checks (e.g. apiserver discovery).
func ReadyHandler(gate *Readiness, checks ...CheckFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !gate.Ready() {
			http.Error(w, "not ready\n", http.StatusServiceUnavailable)
			return
		}
		if len(checks) > 0 {
			ctx, cancel := context.WithTimeout(r.Context(), DefaultReadyCheckTimeout)
			defer cancel()
			for _, check := range checks {
				if check == nil {
					continue
				}
				if err := check(ctx); err != nil {
					http.Error(w, "not ready\n", http.StatusServiceUnavailable)
					return
				}
			}
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		// The status is already on the wire and the only reader is a kubelet that has hung up.
		// Nothing here can act on the write failing, and these handlers take no logger precisely
		// so a probe stays free of dependencies.
		_, _ = w.Write([]byte("ok\n"))
	})
}
