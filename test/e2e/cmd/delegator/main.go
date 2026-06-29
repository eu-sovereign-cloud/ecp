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
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	frameworkbuilder "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	rak8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment/backend/kubernetes"
	rolek8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role/backend/kubernetes"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	ssk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/storage-sku/backend/kubernetes"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"

	arubaconverter "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
	arubahandler "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/handler"
	arubarepository "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/repository"
	dummyplugin "github.com/eu-sovereign-cloud/ecp/csp/dummy/pkg/plugin"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(bsk8s.AddToScheme(scheme))
	utilruntime.Must(netk8s.AddToScheme(scheme))
	utilruntime.Must(nick8s.AddToScheme(scheme))
	utilruntime.Must(wsk8s.AddToScheme(scheme))
	utilruntime.Must(ssk8s.AddToScheme(scheme))
	utilruntime.Must(rolek8s.AddToScheme(scheme))
	utilruntime.Must(rak8s.AddToScheme(scheme))
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

	dynClient, err := dynamic.NewForConfig(mgr.GetConfig())
	if err != nil {
		logger.Error("unable to create dynamic client", "error", err)
		os.Exit(1)
	}

	// 3. Read PLUGIN environment variable
	pluginType := os.Getenv("PLUGIN")
	if pluginType == "" {
		pluginType = "default" // Default to "default" if not set
	}

	controllerOpts := []frameworkbuilder.Option{
		frameworkbuilder.WithLogger(logger.With("component", "controller-set")),
		frameworkbuilder.WithRequeueAfter(1 * time.Second),
	}

	controllerSet := frameworkbuilder.NewControllerSet()

	// 4. Load the appropriate plugin set based on the environment variable
	switch pluginType {
	case "default", "aruba":
		if err := loadArubaControllers(context.Background(), dynClient, mgr, logger, controllerSet, controllerOpts); err != nil {
			logger.Error("failed to load aruba controllers", "error", err)
			os.Exit(1)
		}
	case "dummy":
		loadDummyControllers(logger, dynClient, mgr, controllerSet, controllerOpts)
	default:
		fmt.Fprintf(os.Stderr, "Error: Invalid plugin type specified. Got '%s', expected 'aruba' or 'dummy'.\n", pluginType)
		os.Exit(1)
	}

	// 5. Setup controllers with manager
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

	// 6. Start manager
	logger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error("problem running manager", "error", err)
		os.Exit(1)
	}
}

func loadArubaControllers(ctx context.Context, dynClient dynamic.Interface, mgr ctrl.Manager, logger *slog.Logger, controllerSet *frameworkbuilder.ControllerSet, controllerOpts []frameworkbuilder.Option) error {
	logger.Info("Loading 'aruba' plugin set")

	// Instantiate seca-specific read-only repositories (for aruba BlockStorageHandler dependencies)
	secaWsRepo := k8sadapter.NewReaderAdapter(dynClient, wsk8s.WorkspaceGVR, logger, wsk8s.WorkspaceFromCR)
	secaSkuRepo := k8sadapter.NewReaderAdapter(dynClient, ssk8s.StorageSKUGVR, logger, ssk8s.StorageSKUFromCR)

	// Instantiate aruba-specific repositories
	wr := arubarepository.NewProjectRepository(ctx, mgr.GetClient(), mgr.GetCache())
	br := arubarepository.NewBlockStorageRepository(ctx, mgr.GetClient(), mgr.GetCache())

	// Instantiate aruba-specific converters
	wc := arubaconverter.NewWorkspaceProjectConverter()
	bc := arubaconverter.NewBlockStorageConverter()

	// Create aruba-specific handlers
	wsPlugin := arubahandler.NewWorkspaceHandler(wr, wc)
	bsPlugin := arubahandler.NewBlockStorageHandler(secaWsRepo, secaSkuRepo, br, wr, bc, wc)

	controllerSet.Add(bsk8s.NewController(mgr.GetClient(), dynClient, bsPlugin, controllerOpts...))
	controllerSet.Add(wsk8s.NewController(mgr.GetClient(), dynClient, wsPlugin, controllerOpts...))

	return nil
}

func loadDummyControllers(logger *slog.Logger, dynClient dynamic.Interface, mgr ctrl.Manager, controllerSet *frameworkbuilder.ControllerSet, controllerOpts []frameworkbuilder.Option) {
	logger.Info("Loading 'dummy' plugin set")

	bsPlugin := dummyplugin.NewBlockStorage(logger.With("plugin", "blockstorage"))
	wsPlugin := dummyplugin.NewWorkspace(logger.With("plugin", "workspace"))
	netPlugin := dummyplugin.NewNetwork(logger.With("plugin", "network"))
	nicPlugin := dummyplugin.NewNic(logger.With("plugin", "nic"))
	rolePlugin := dummyplugin.NewRole(logger.With("plugin", "role"))
	raPlugin := dummyplugin.NewRoleAssignment(logger.With("plugin", "roleassignment"))

	controllerSet.Add(bsk8s.NewController(mgr.GetClient(), dynClient, bsPlugin, controllerOpts...))
	controllerSet.Add(wsk8s.NewController(mgr.GetClient(), dynClient, wsPlugin, controllerOpts...))
	controllerSet.Add(netk8s.NewController(mgr.GetClient(), dynClient, netPlugin, controllerOpts...))
	controllerSet.Add(nick8s.NewController(mgr.GetClient(), dynClient, nicPlugin, controllerOpts...))
	controllerSet.Add(rolek8s.NewController(mgr.GetClient(), dynClient, rolePlugin, controllerOpts...))
	controllerSet.Add(rak8s.NewController(mgr.GetClient(), dynClient, raPlugin, controllerOpts...))
}
