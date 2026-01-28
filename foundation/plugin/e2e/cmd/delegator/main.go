package main

import (
	"context"
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

	// 3. Instantiate workspace and block storage repository
	wr := repository.NewProjectRepository(ctx, mgr.GetClient(), mgr.GetCache())
	br := repository.NewBlockStorageRepository(ctx, mgr.GetClient(), mgr.GetCache())

	// 4. Instantiate worksace  and block storage converter

	wc := converter.NewWorkspaceProjectConverter()
	bc := converter.NewBlockStorageConverter()
	// 5 . Create workspace and block storage handler
	wsPlugin := aruba.NewWorkspaceHandler(wr, wc)
	bsPlugin := aruba.NewBlockStorageHandler(br, bc)

	// 6. Create a plugin set
	pluginSet := builder.NewPluginSet(
		builder.WithBlockStorage(bsPlugin),
		builder.WithWorkspace(wsPlugin),
	)

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
