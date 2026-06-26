package main

import (
	"log/slog"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	dummyplugin "github.com/eu-sovereign-cloud/ecp/csp/dummy/pkg/plugin"
	frameworkbuilder "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	rak8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment/backend/kubernetes"
	rolek8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role/backend/kubernetes"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/network/v1/backend/kubernetes"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(rolek8s.AddToScheme(scheme))
	utilruntime.Must(bsk8s.AddToScheme(scheme))
	utilruntime.Must(imgk8s.AddToScheme(scheme))
	utilruntime.Must(netk8s.AddToScheme(scheme))
	utilruntime.Must(rak8s.AddToScheme(scheme))
	utilruntime.Must(wsk8s.AddToScheme(scheme))
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

	rolePlugin := dummyplugin.NewRole(logger.With("plugin", "role"))
	bsPlugin := dummyplugin.NewBlockStorage(logger.With("plugin", "blockstorage"))
	imgPlugin := dummyplugin.NewImage(logger.With("plugin", "image"))
	wsPlugin := dummyplugin.NewWorkspace(logger.With("plugin", "workspace"))
	netPlugin := dummyplugin.NewNetwork(logger.With("plugin", "network"))
	raPlugin := dummyplugin.NewRoleAssignment(logger.With("plugin", "roleassignment"))

	controllerOpts := []frameworkbuilder.Option{
		frameworkbuilder.WithLogger(logger.With("component", "controller-set")),
		frameworkbuilder.WithRequeueAfter(1 * time.Second),
	}

	controllerSet := frameworkbuilder.NewControllerSet()
	controllerSet.Add(rolek8s.NewController(mgr.GetClient(), dynClient, rolePlugin, controllerOpts...))
	controllerSet.Add(bsk8s.NewController(mgr.GetClient(), dynClient, bsPlugin, controllerOpts...))
	controllerSet.Add(imgk8s.NewController(mgr.GetClient(), dynClient, imgPlugin, controllerOpts...))
	controllerSet.Add(netk8s.NewController(mgr.GetClient(), dynClient, netPlugin, controllerOpts...))
	controllerSet.Add(rak8s.NewController(mgr.GetClient(), dynClient, raPlugin, controllerOpts...))
	controllerSet.Add(wsk8s.NewController(mgr.GetClient(), dynClient, wsPlugin, controllerOpts...))

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
