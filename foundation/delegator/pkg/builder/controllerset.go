package builder

import (
	"log"
	"log/slog"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage/block-storages/v1"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"
	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/controller"
)

const (
	DefaultRequeueTime = 5 * time.Minute
)

// ControllerSet is a collection of controllers for each of the plugins in the
// PluginSet.
type ControllerSet struct {
	BlockStorage *controller.BlockStorageController
	Workspace    *controller.WorkspaceController
}

// Options is a collection of options to configure the ControllerSet.
type Options struct {
	Client       client.Client
	Config       *rest.Config
	Logger       *slog.Logger
	Plugins      *PluginSet
	RequeueAfter time.Duration
}

// Option is a function that configures the Options.
type Option func(*Options)

// NewControllerSet creates a new ControllerSet with the provided options.
func NewControllerSet(opts ...Option) (*ControllerSet, error) {
	// 1. Initialize options with defaults
	o := &Options{
		RequeueAfter: DefaultRequeueTime,
		Logger:       slog.Default(),
	}

	// 2. Apply all the options to override the defaults
	for _, opt := range opts {
		opt(o)
	}

	// 3. Create the gateway repos for each controller
	dynamicClient, err := dynamic.NewForConfig(o.Config)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(o.Config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	bsRepo := kubernetesadapter.NewRepoAdapter(
		dynamicClient,
		blockstoragev1.BlockStorageGVR,
		o.Logger,
		kubernetesadapter.MapBlockStorageDomainToCR,
		kubernetesadapter.MapCRToBlockStorageDomain,
	)
	// todo - use namespace NewNamespaceManagingWriterAdapter to auto-cleanup namespace
	wsRepo := kubernetesadapter.NewNamespaceManagingRepoAdapter(
		dynamicClient,
		clientset,
		workspacev1.WorkspaceGVR,
		o.Logger,
		kubernetesadapter.MapWorkspaceDomainToCR,
		kubernetesadapter.MapCRToWorkspaceDomain,
	)

	// 4. Create the controllers
	bsController := controller.NewBlockStorageController(
		o.Client,
		bsRepo,
		o.Plugins.BlockStorage,
		o.RequeueAfter,
		o.Logger,
	)

	wsController := controller.NewWorkspaceController(
		o.Client,
		wsRepo,
		o.Plugins.Workspace,
		o.RequeueAfter,
		o.Logger,
	)

	return &ControllerSet{
		BlockStorage: bsController,
		Workspace:    wsController,
	}, nil
}

// SetupWithManager binds all the controllers to a Kubernetes controller manager.
func (c *ControllerSet) SetupWithManager(mgr ctrl.Manager) error {
	if err := (*controller.GenericController[*regional.BlockStorageDomain])(c.BlockStorage).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (*controller.GenericController[*regional.WorkspaceDomain])(c.Workspace).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}

// WithPlugins configures the plugins for the ControllerSet.
func WithPlugins(plugins *PluginSet) Option {
	return func(o *Options) {
		o.Plugins = plugins
	}
}

// WithClient configures the Kubernetes client for the ControllerSet.
func WithClient(client client.Client) Option {
	return func(o *Options) {
		o.Client = client
	}
}

// WithConfig configures the Kubernetes config for the ControllerSet.
func WithConfig(config *rest.Config) Option {
	return func(o *Options) {
		o.Config = config
	}
}

// WithLogger configures the logger for the ControllerSet.
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithRequeueAfter configures the requeue time for the ControllerSet.
func WithRequeueAfter(requeueAfter time.Duration) Option {
	return func(o *Options) {
		o.RequeueAfter = requeueAfter
	}
}
