package builder

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	DefaultRequeueTime   = 5 * time.Minute
	DefaultMaxConditions = 5 // use 0 or a negative value to impose no limit
)

// Reconciler is any controller that can be registered with a controller-runtime Manager.
// Each resource slice's backend/kubernetes/controller.go returns a value implementing
// this interface from its NewController factory.
type Reconciler interface {
	SetupWithManager(mgr ctrl.Manager) error
}

// ControllerSet is a generic aggregator of Reconciler instances.
// CSP cmd/main.go builds one ControllerSet, adds each resource controller to it,
// then calls SetupWithManager once to bind every controller to the manager.
type ControllerSet struct {
	reconcilers []Reconciler
}

// NewControllerSet creates an empty ControllerSet.
func NewControllerSet() *ControllerSet {
	return &ControllerSet{}
}

// Add registers a Reconciler with this ControllerSet and returns the same set
// for chaining.
func (cs *ControllerSet) Add(r Reconciler) *ControllerSet {
	cs.reconcilers = append(cs.reconcilers, r)
	return cs
}

// SetupWithManager calls SetupWithManager on every registered Reconciler.
// It returns the first error encountered, if any.
func (cs *ControllerSet) SetupWithManager(mgr ctrl.Manager) error {
	if cs == nil {
		return errors.New("controller set cannot be nil")
	}

	if len(cs.reconcilers) == 0 {
		return fmt.Errorf("controller set has no reconcilers registered")
	}

	for _, r := range cs.reconcilers {
		if err := r.SetupWithManager(mgr); err != nil {
			return fmt.Errorf("failed to set up controller: %w", err)
		}
	}

	return nil
}

// Options contains optional configuration shared by all controllers in the set.
type Options struct {
	Logger        *slog.Logger
	RequeueAfter  time.Duration
	MaxConditions int
}

// Option is a function that applies a configuration change to an Options struct.
type Option func(*Options)

// WithLogger configures the logger. If nil, nothing is changed.
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		if logger == nil {
			return
		}
		o.Logger = logger
	}
}

// WithRequeueAfter configures the requeue duration. If zero, nothing is changed.
func WithRequeueAfter(requeueAfter time.Duration) Option {
	return func(o *Options) {
		if requeueAfter == 0 {
			return
		}
		o.RequeueAfter = requeueAfter
	}
}

// WithMaxConditions sets the maximum number of StatusConditions retained in the
// resource status. A value of 0 or negative means no limit (all conditions are
// kept). Pass this option explicitly to override DefaultMaxConditions.
func WithMaxConditions(maxConditions int) Option {
	return func(o *Options) {
		o.MaxConditions = maxConditions
	}
}

// ApplyOptions applies Option funcs to a default Options and returns the result.
func ApplyOptions(opts []Option) Options {
	o := Options{
		RequeueAfter:  DefaultRequeueTime,
		Logger:        slog.Default(),
		MaxConditions: DefaultMaxConditions,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return o
}
