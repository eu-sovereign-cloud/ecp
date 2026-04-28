package builder

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/controller"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

const (
	DefaultRequeueTime   = 5 * time.Minute
	DefaultMaxConditions = 5 // use 0 or a negative value to impose no limit
)

// ControllerSet is a collection of controllers for each of the plugins in the
// PluginSet.
type ControllerSet struct {
	blockStorage *controller.BlockStorageController
	workspace    *controller.WorkspaceController
}

// Options contains optional configuration for the ControllerSet.
type Options struct {
	Logger        *slog.Logger
	RequeueAfter  time.Duration
	MaxConditions int
}

// Option is a function that configures the Options.
type Option func(*Options)

// NewControllerSet creates a new ControllerSet with the provided mandatory
// parameters and optional configuration.
func NewControllerSet(config *rest.Config, k8sClient client.Client, plugins PluginSet, opts ...Option) (*ControllerSet, error) {
	// 1. Validate required parameters
	if config == nil {
		return nil, errors.New("kubernetes rest config is required")
	}
	if k8sClient == nil {
		return nil, errors.New("kubernetes client is required")
	}
	if err := plugins.Validate(); err != nil {
		return nil, err
	}

	// 2. Initialize options with defaults
	o := Options{
		RequeueAfter:  DefaultRequeueTime,
		Logger:        slog.Default(),
		MaxConditions: DefaultMaxConditions,
	}

	// 3. Apply all the options to override the defaults
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}

	// 4. Create shared clients
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	// 5. Create the controllers
	bs := newBlockStorageController(k8sClient, dynamicClient, plugins.BlockStorage, o)
	ws := newWorkspaceController(k8sClient, dynamicClient, clientset, plugins.Workspace, o)

	return &ControllerSet{
		blockStorage: &bs,
		workspace:    &ws,
	}, nil
}

// Validate checks that the resource controllers are set.
func (c *ControllerSet) validate() error {
	if c == nil {
		return errors.New("controller set cannot be nil")
	}

	if c.blockStorage == nil {
		return errors.New("blockStorage is required")
	}

	if c.workspace == nil {
		return errors.New("workspace is required")
	}

	return nil
}

// SetupWithManager binds all the controllers to a Kubernetes controller manager.
func (c *ControllerSet) SetupWithManager(mgr ctrl.Manager) error {
	if c == nil {
		return errors.New("controller set cannot be nil")
	}

	if err := c.validate(); err != nil {
		return fmt.Errorf("failed to validate controller set: %w", err)
	}

	if err := (*controller.GenericController[*regional.BlockStorageDomain])(c.blockStorage).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (*controller.GenericController[*regional.WorkspaceDomain])(c.workspace).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}

// WithLogger configures the logger for the ControllerSet. If nil logger is passed, nothing will be changed.
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		if logger == nil {
			return
		}
		o.Logger = logger
	}
}

// WithRequeueAfter configures the requeue time for the ControllerSet. If requeueAfter is 0, nothing will be changed.
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
