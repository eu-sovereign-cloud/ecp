package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/builder"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/converter"
	aruba "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/handler"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/repository"
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
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
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

	ctx := context.TODO()

	// 3. Read PLUGIN environment variable
	pluginType := os.Getenv("PLUGIN")
	if pluginType == "" {
		pluginType = "default" // Default to "default" if not set
	}

	// 4. Load the appropriate plugin set based on the environment variable
	var pluginSet *builder.PluginSet

	switch pluginType {
	case "default", "aruba":
		pluginSet, err = loadArubaPluginSet(ctx, mgr, logger)
	case "dummy":
		pluginSet, err = loadDummyPluginSet(logger)
	default:
		// Use fmt.Fprintf for fatal errors before logger is fully propagated
		fmt.Fprintf(os.Stderr, "Error: Invalid plugin type specified. Got '%s', expected 'aruba' or 'dummy'.\n", pluginType)
		os.Exit(1)
	}

	if err != nil {
		logger.Error("failed to load plugin set", "error", err)
		os.Exit(1)
	}

	// 7. Create the controller set
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

	// 8. Setup controllers with manager
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

	// 9. Start manager
	logger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error("problem running manager", "error", err)
		os.Exit(1)
	}
}

func loadArubaPluginSet(ctx context.Context, mgr ctrl.Manager, logger *slog.Logger) (*builder.PluginSet, error) {
	logger.Info("Loading 'aruba' plugin set")
	// Instantiate aruba-specific repositories
	wr := repository.NewProjectRepository(ctx, mgr.GetClient(), mgr.GetCache())
	br := repository.NewBlockStorageRepository(ctx, mgr.GetClient(), mgr.GetCache())

	// Instantiate aruba-specific converters
	wc := converter.NewWorkspaceProjectConverter()
	bc := converter.NewBlockStorageConverter()

	// Create aruba-specific handlers
	wsPlugin := aruba.NewWorkspaceHandler(wr, wc)
	bsPlugin := aruba.NewBlockStorageHandler(br, bc)

	// Create and return the plugin set
	return builder.NewPluginSet(
		builder.WithBlockStorage(bsPlugin),
		builder.WithWorkspace(wsPlugin),
	), nil
}

func loadDummyPluginSet(logger *slog.Logger) (*builder.PluginSet, error) {
	logger.Info("Loading 'dummy' plugin set")
	bsPlugin := dummyplugin.NewBlockStorage(logger.With("plugin", "blockstorage"))
	wsPlugin := dummyplugin.NewWorkspace(logger.With("plugin", "workspace"))

	return builder.NewPluginSet(
		builder.WithBlockStorage(bsPlugin),
		builder.WithWorkspace(wsPlugin),
	), nil
}
