package main

import (
	"log/slog"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/builder"

	dummyplugin "github.com/eu-sovereign-cloud/ecp/foundation/plugin/dummy/pkg/plugin"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(workspacev1.AddToScheme(scheme))
	utilruntime.Must(storage.AddToScheme(scheme))
}

func main() {
	// 1. Setup logger
	opts := zap.Options{
		Development: true,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 2. Create manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: ":8081",
		LeaderElection:         false,
	})
	if err != nil {
		logger.Error("unable to start manager", "error", err)
		os.Exit(1)
	}

	// 3. Instantiate dummy plugins
	bsPlugin := dummyplugin.NewBlockStorage(logger.With("plugin", "blockstorage"))
	wsPlugin := dummyplugin.NewWorkspace(logger.With("plugin", "workspace"))

	// 4. Create a plugin set
	pluginSet := builder.NewPluginSet(
		builder.WithBlockStorage(bsPlugin),
		builder.WithWorkspace(wsPlugin),
	)

	// 5. Create the controller set
	controllerSet, err := builder.NewControllerSet(
		builder.WithConfig(mgr.GetConfig()),
		builder.WithClient(mgr.GetClient()),
		builder.WithPlugins(pluginSet),
		builder.WithLogger(logger.With("component", "controller-set")),
		builder.WithRequeueAfter(1*time.Second), // TODO: parameter for that
	)
	if err != nil {
		logger.Error("unable to create controller set", "error", err)
		os.Exit(1)
	}

	// 6. Setup controllers with manager
	if err := controllerSet.SetupWithManager(mgr); err != nil {
		logger.Error("unable to setup controllers with manager", "error", err)
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error("unable to set up health check", "error", err)
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error("unable to set up ready check", "error", err)
		os.Exit(1)
	}

	// 7. Start manager
	logger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error("problem running manager", "error", err)
		os.Exit(1)
	}
}
