package main

import (
	"context"
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

	arubaconverter "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
	arubahandler "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/handler"
	arubarepository "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/repository"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	frameworkbuilder "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	ssk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/storage-sku/backend/kubernetes"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(bsk8s.AddToScheme(scheme))
	utilruntime.Must(ssk8s.AddToScheme(scheme))
	utilruntime.Must(wsk8s.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
}

func main() {
	opts := zap.Options{Development: true}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

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

	controllerOpts := []frameworkbuilder.Option{
		frameworkbuilder.WithLogger(logger.With("component", "controller-set")),
		frameworkbuilder.WithRequeueAfter(1 * time.Second),
	}

	controllerSet := frameworkbuilder.NewControllerSet()
	loadControllers(context.Background(), dynClient, mgr, logger, controllerSet, controllerOpts)

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

	logger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error("problem running manager", "error", err)
		os.Exit(1)
	}
}

func loadControllers(ctx context.Context, dynClient dynamic.Interface, mgr ctrl.Manager, logger *slog.Logger, controllerSet *frameworkbuilder.ControllerSet, controllerOpts []frameworkbuilder.Option) {
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
}
