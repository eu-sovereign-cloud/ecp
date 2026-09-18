package builder

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
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
	Clientset kubernetes.Interface
	Logger    *slog.Logger
	// Regions is the region scope read from REGIONS: the only CRs this delegator reconciles.
	// Empty means every region. It never reaches a plugin — see ControllerSet.ScopeToRegions.
	Regions     []string
	Controllers *ControllerSet
}

// NewDelegator builds a Delegator against the cluster ctrl.GetConfig finds: KUBECONFIG, the
// in-cluster config, then ~/.kube/config. The controller-runtime logger is set before that config
// is loaded — anything controller-runtime logs earlier is discarded — so a pod that cannot find
// one says why instead of exiting silently. The manager's scheme holds the client-go types
// plus every registration in schemes — resource/scheme.AddToScheme for the SECA CRs, and the
// provider types the plugin writes, if any. opts reaches the manager unchanged except that its
// Scheme is always replaced and an empty HealthProbeBindAddress becomes ":8081", the port
// charts/delegator probes. The /healthz and /readyz checks are already added.
//
// REGIONS (comma-separated) restricts which CRs the delegator reconciles: with it set, a
// controller only sees a CR whose internal region label names one of those regions, which is
// what a delegator deployed for one region of a shared cluster wants. Unset — the default —
// reconciles every region. It is a watch filter and never a region a plugin is told about: the
// plugin reads the region off the CR it is handed.
func NewDelegator(opts ctrl.Options, schemes ...func(*runtime.Scheme) error) (*Delegator, error) {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}

	opts.Scheme = runtime.NewScheme()
	sb := runtime.NewSchemeBuilder(append(schemes, clientgoscheme.AddToScheme)...)
	if err := sb.AddToScheme(opts.Scheme); err != nil {
		return nil, fmt.Errorf("register scheme: %w", err)
	}
	if opts.HealthProbeBindAddress == "" {
		opts.HealthProbeBindAddress = ":8081"
	}

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

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	regions := regionsFromEnv()
	if len(regions) == 0 {
		logger.Info("REGIONS unset, reconciling every region")
	} else {
		logger.Info("scoped to regions", "regions", regions)
	}
	controllers := NewControllerSet()
	controllers.ScopeToRegions(regions)

	return &Delegator{
		Manager:     mgr,
		Dynamic:     dynClient,
		Clientset:   clientset,
		Logger:      logger,
		Regions:     regions,
		Controllers: controllers,
	}, nil
}

// regionsFromEnv parses REGIONS, the delegator's region scope: a comma-separated list, with
// blanks and duplicates dropped so a trailing comma or a repeated entry is not a startup
// failure. An unset or all-blank value yields no scope at all, i.e. every region.
func regionsFromEnv() []string {
	var regions []string
	for _, r := range strings.Split(os.Getenv("REGIONS"), ",") {
		if r = strings.TrimSpace(r); r != "" && !slices.Contains(regions, r) {
			regions = append(regions, r)
		}
	}
	return regions
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
