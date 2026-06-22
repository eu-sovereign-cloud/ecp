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
	frameworkbuilder "github.com/eu-sovereign-cloud/ecp/framework/backend/builder"
	netk8s "github.com/eu-sovereign-cloud/ecp/resources/regional/network/networks/v1/backend/kubernetes"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1/backend/kubernetes"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1/backend/kubernetes"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(bsk8s.AddToScheme(scheme))
	utilruntime.Must(netk8s.AddToScheme(scheme))
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

	bsPlugin := dummyplugin.NewBlockStorage(logger.With("plugin", "blockstorage"))
	wsPlugin := dummyplugin.NewWorkspace(logger.With("plugin", "workspace"))
	netPlugin := dummyplugin.NewNetwork(logger.With("plugin", "network"))

	controllerOpts := []frameworkbuilder.Option{
		frameworkbuilder.WithLogger(logger.With("component", "controller-set")),
		frameworkbuilder.WithRequeueAfter(1 * time.Second),
	}

	controllerSet := frameworkbuilder.NewControllerSet()
	controllerSet.Add(bsk8s.NewController(mgr.GetClient(), dynClient, bsPlugin, controllerOpts...))
	controllerSet.Add(netk8s.NewController(mgr.GetClient(), dynClient, netPlugin, controllerOpts...))
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
