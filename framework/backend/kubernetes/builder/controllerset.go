package builder

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
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

// RegionScoped is a Reconciler whose watch can be capped to a set of regions. The generic
// controller every resource slice builds on implements it, so every slice's controller does
// through embedding; a plugin's own Reconciler that does not is left watching every region.
type RegionScoped interface {
	ScopeToRegions(regions []string)
}

// ControllerSet is a generic aggregator of Reconciler instances.
// NewDelegator builds one for each CSP cmd/main.go to add its resource controllers to;
// Delegator.Run then calls SetupWithManager once to bind every controller to the manager.
type ControllerSet struct {
	reconcilers []Reconciler
	regions     []string
}

// NewControllerSet creates an empty ControllerSet restricted to the regions named: every
// RegionScoped controller it holds is capped to them as the set is bound to the manager
// (REGIONS, see NewDelegator). Naming none — the default, and the global deployment —
// watches every region.
func NewControllerSet(regions ...string) *ControllerSet {
	return &ControllerSet{regions: regions}
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
		return kernel.NewError(kernel.KindValidation, errors.New("controller set cannot be nil"))
	}

	if len(cs.reconcilers) == 0 {
		return kernel.NewError(kernel.KindValidation, errors.New("controller set has no reconcilers registered"))
	}

	for _, r := range cs.reconcilers {
		if scoped, ok := r.(RegionScoped); ok {
			scoped.ScopeToRegions(cs.regions)
		}
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
