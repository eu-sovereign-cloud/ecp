package plugin

import (
	"sync"
	"time"
)

// asyncTracker simulates an asynchronous backend without blocking the caller.
//
// The dummy plugin has no real provider to poll, so it fakes the latency of a
// remote operation: the first time an operation is observed it records a
// completion deadline, and the operation is only reported as complete once that
// deadline has elapsed. Because it never sleeps, the reconciliation worker is
// free to process other resources between passes (it is expected to requeue and
// call back, exactly like a real provider that returns "still processing").
type asyncTracker struct {
	mu        sync.Mutex
	deadlines map[string]time.Time
}

func newAsyncTracker() *asyncTracker {
	return &asyncTracker{deadlines: make(map[string]time.Time)}
}

// identifiable is the subset of a domain resource needed to build a stable key
// that is unique across tenants and workspaces.
type identifiable interface {
	GetTenant() string
	GetWorkspace() string
	GetName() string
}

// resourceKey builds a tracking key that uniquely identifies a resource.
func resourceKey(r identifiable) string {
	return r.GetTenant() + "/" + r.GetWorkspace() + "/" + r.GetName()
}

// done reports whether the simulated operation identified by key has completed.
//
// The first call schedules completion delay from now and returns false. Later
// calls keep returning false until the deadline elapses, then return true and
// forget the key so a subsequent operation on the same resource starts fresh.
func (t *asyncTracker) done(key string, delay time.Duration) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	deadline, ok := t.deadlines[key]
	if !ok {
		t.deadlines[key] = time.Now().Add(delay)
		return false
	}

	if time.Now().Before(deadline) {
		return false
	}

	delete(t.deadlines, key)

	return true
}
