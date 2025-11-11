package main

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	p "github.com/eu-sovereign-cloud/ecp/foundation/delegator/plugin"
	// Provider plugin packages imported for side-effect registration
	_ "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba"
	_ "github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos"
)

// Run starts the delegator manager.
func Run(ctx context.Context, kubeconfig string) error {
	logger := zap.New(zap.UseDevMode(true))
	ctrl.SetLogger(logger)
	var cfg *rest.Config
	var err error
	if kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("build kubeconfig: %w", err)
		}
	} else {
		cfg = ctrl.GetConfigOrDie()
	}
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(AddToScheme(scheme))

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
		if pluginWithClient, ok := plug.(p.UsesClient); ok {
			pluginWithClient.SetClient(mgr.GetClient())
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
