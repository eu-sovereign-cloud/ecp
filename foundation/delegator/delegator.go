package delegator

import (
	"context"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	p "github.com/eu-sovereign-cloud/ecp/foundation/delegator/plugin"
	// Provider plugin packages imported for side-effect registration
	_ "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba"
	_ "github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos"
)

// Run starts the delegator manager.
func Run(ctx context.Context) error {
	logger := zap.New(zap.UseDevMode(true))
	ctrl.SetLogger(logger)

	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = AddToScheme(scheme)

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("get k8s config: %w", err)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: ":8081",
		LeaderElection:         false,
		LeaderElectionID:       "delegator.secapi.cloud",
	})
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	if err := SetupStorageController(mgr); err != nil {
		return fmt.Errorf("setup storage controller: %w", err)
	}

	// Initialize plugins
	for name, plug := range p.Registry {
		if err := plug.Init(ctx); err != nil {
			return fmt.Errorf("plugin %s init: %w", name, err)
		}
		if cplug, ok := plug.(p.UsesClient); ok {
			cplug.SetClient(mgr.GetClient())
		}
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		return err
	}
	if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		logger.Info("context canceled - shutting down delegator")
	}()

	return mgr.Start(ctx)
}

// main wrapper if using as standalone binary
func main() {
	if err := Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "delegator failed: %v\n", err)
		os.Exit(1)
	}
}
