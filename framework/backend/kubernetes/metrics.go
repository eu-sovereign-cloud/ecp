package kubernetes

import (
	"context"
	"errors"
	"sync"
	"time"

	kerrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
)

// Upstream operation labels for Observe (closed set).
const (
	OpGet          = "get"
	OpList         = "list"
	OpCreate       = "create"
	OpUpdate       = "update"
	OpUpdateStatus = "update_status"
	OpDelete       = "delete"
)

// Upstream result labels for Observe (closed set).
const (
	ResultOK            = "ok"
	ResultNotFound      = "not_found"
	ResultAlreadyExists = "already_exists"
	ResultConflict      = "conflict"
	ResultForbidden     = "forbidden"
	ResultTimeout       = "timeout"
	ResultError         = "error"
)

// Observer records one upstream Kubernetes adapter operation.
// Implementations must be safe for concurrent use.
type Observer interface {
	Observe(resource, group, operation, result string, d time.Duration)
}

type noopObserver struct{}

func (noopObserver) Observe(string, string, string, string, time.Duration) {}

// upstream holds the process-wide observer. RWMutex keeps different concrete
// Observer types swappable (atomic.Value requires a single concrete type).
var (
	upstreamMu sync.RWMutex
	upstream   Observer = noopObserver{}
)

// SetUpstreamObserver installs the observer used by resource adapters.
// Pass nil to restore the no-op observer. Safe for concurrent use; intended
// to be called once at process start (e.g. gateway main).
func SetUpstreamObserver(o Observer) {
	upstreamMu.Lock()
	defer upstreamMu.Unlock()
	if o == nil {
		upstream = noopObserver{}
		return
	}
	upstream = o
}

func currentUpstream() Observer {
	upstreamMu.RLock()
	defer upstreamMu.RUnlock()
	return upstream
}

// namespaceGVR labels core Namespace API calls made by the adapters.
var namespaceGVR = schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}

// observeUpstream records one adapter-level Kubernetes operation.
func observeUpstream(gvr schema.GroupVersionResource, operation string, start time.Time, err error) {
	group := gvr.Group
	if group == "" {
		group = "core"
	}
	currentUpstream().Observe(gvr.Resource, group, operation, classifyUpstreamResult(err), time.Since(start))
}

// classifyUpstreamResult maps an error to a closed result label set.
func classifyUpstreamResult(err error) string {
	if err == nil {
		return ResultOK
	}

	switch {
	case kerrs.IsNotFound(err):
		return ResultNotFound
	case kerrs.IsAlreadyExists(err):
		return ResultAlreadyExists
	case kerrs.IsConflict(err):
		return ResultConflict
	case kerrs.IsForbidden(err), kerrs.IsUnauthorized(err):
		return ResultForbidden
	case kerrs.IsTimeout(err), kerrs.IsServerTimeout(err):
		return ResultTimeout
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return ResultTimeout
	}

	if de := kernel.AsError(err); de != nil {
		switch de.Kind {
		case kernel.KindNotFound:
			return ResultNotFound
		case kernel.KindAlreadyExists:
			return ResultAlreadyExists
		case kernel.KindConflict, kernel.KindPreconditionFailed:
			return ResultConflict
		case kernel.KindForbidden, kernel.KindUnauthorized:
			return ResultForbidden
		case kernel.KindUnavailable:
			// May be timeout-like or general upstream failure.
			if errors.Is(de, context.DeadlineExceeded) {
				return ResultTimeout
			}
			return ResultError
		default:
			return ResultError
		}
	}

	return ResultError
}
