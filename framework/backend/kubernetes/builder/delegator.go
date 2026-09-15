package builder

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Delegator is the process a CSP plugin runs: one manager, the clients its controllers are
// built from, and the set they are added to. Each csp/<plugin>/cmd/main.go builds one with
// NewDelegator, adds its controllers to Controllers, then calls Run.
type Delegator struct {
	Manager ctrl.Manager
	// Dynamic backs the repo adapters every slice's NewController assembles.
	Dynamic dynamic.Interface
	// Clientset is the typed client for the Namespace API: the Workspace and Network
	// controllers tear down the namespace they own for their children.
	Clientset   kubernetes.Interface
	Logger      *slog.Logger
	Controllers *ControllerSet
}

// NewDelegator builds a Delegator against cfg. The manager's scheme holds the client-go types
// plus every registration in schemes — resource/scheme.AddToScheme for the SECA CRs, and the
// provider types the plugin writes, if any. opts reaches the manager unchanged except that its
// Scheme is always replaced and an empty HealthProbeBindAddress becomes ":8081", the port
// charts/delegator probes. The /healthz and /readyz checks are already added.
func NewDelegator(cfg *rest.Config, opts ctrl.Options, schemes ...func(*runtime.Scheme) error) (*Delegator, error) {
	opts.Scheme = runtime.NewScheme()
	sb := runtime.NewSchemeBuilder(append(schemes, clientgoscheme.AddToScheme)...)
	if err := sb.AddToScheme(opts.Scheme); err != nil {
		return nil, fmt.Errorf("register scheme: %w", err)
	}
	if opts.HealthProbeBindAddress == "" {
		opts.HealthProbeBindAddress = ":8081"
	}

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	mgr, err := ctrl.NewManager(cfg, opts)
	if err != nil {
		return nil, fmt.Errorf("create manager: %w", err)
	}
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("add health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("add ready check: %w", err)
	}

	dynClient, err := dynamic.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("create dynamic client: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}

	return &Delegator{
		Manager:     mgr,
		Dynamic:     dynClient,
		Clientset:   clientset,
		Logger:      slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		Controllers: NewControllerSet(),
	}, nil
}

// Run binds every controller in d.Controllers to the manager and serves them until ctx is done.
// Pass ctrl.SetupSignalHandler() to stop on SIGTERM.
func (d *Delegator) Run(ctx context.Context) error {
	if err := d.Controllers.SetupWithManager(d.Manager); err != nil {
		return fmt.Errorf("set up controllers: %w", err)
	}
	d.Logger.Info("starting manager")
	if err := d.Manager.Start(ctx); err != nil {
		return fmt.Errorf("run manager: %w", err)
	}
	return nil
}
