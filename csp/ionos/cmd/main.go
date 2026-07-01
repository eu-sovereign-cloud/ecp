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
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	ionosapis "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"

	blockstoragectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/block_storage"
	networkctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/network"
	nicctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/nic"
	workspacectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/workspace"
	"github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/service"
	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/adapter/crossplane"
	frameworkbuilder "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(bsk8s.AddToScheme(scheme))
	utilruntime.Must(netk8s.AddToScheme(scheme))
	utilruntime.Must(nick8s.AddToScheme(scheme))
	utilruntime.Must(wsk8s.AddToScheme(scheme))
	utilruntime.Must(ionosapis.AddToScheme(scheme))
}

func main() {
	opts := zap.Options{Development: true}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			SecureServing: false,
			BindAddress:   ":8083",
		},
		HealthProbeBindAddress: ":8082",
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

	wsAdapter := crossplane.NewWorkspaceStore(mgr.GetClient(), logger.With("adapter", "workspace"))
	bsAdapter := crossplane.NewBlockStorageStore(mgr.GetClient(), logger.With("adapter", "block-storage"))
	netAdapter := crossplane.NewNetworkStore(mgr.GetClient(), logger.With("adapter", "network"))
	nicAdapter := crossplane.NewNicStore(mgr.GetClient(), logger.With("adapter", "nic"))

	wsPlugin := &service.Workspace{
		Creator: &workspacectrl.CreateWorkspace{Store: wsAdapter},
		Deleter: &workspacectrl.DeleteWorkspace{Store: wsAdapter},
	}
	bsPlugin := &service.BlockStorage{
		Creator:       &blockstoragectrl.CreateBlockStorage{Store: bsAdapter},
		Deleter:       &blockstoragectrl.DeleteBlockStorage{Store: bsAdapter},
		SizeIncreaser: &blockstoragectrl.IncreaseSizeBlockStorage{Store: bsAdapter},
	}
	netPlugin := &service.Network{
		Creator: &networkctrl.CreateNetwork{Store: netAdapter},
		Deleter: &networkctrl.DeleteNetwork{Store: netAdapter},
	}
	nicPlugin := &service.Nic{
		Creator: &nicctrl.CreateNic{Store: nicAdapter},
		Deleter: &nicctrl.DeleteNic{Store: nicAdapter},
	}

	controllerOpts := []frameworkbuilder.Option{
		frameworkbuilder.WithLogger(logger.With("component", "controller-set")),
		frameworkbuilder.WithRequeueAfter(1 * time.Second),
		frameworkbuilder.WithMaxConditions(5),
	}

	controllerSet := frameworkbuilder.NewControllerSet()
	controllerSet.Add(bsk8s.NewController(mgr.GetClient(), dynClient, bsPlugin, controllerOpts...))
	controllerSet.Add(netk8s.NewController(mgr.GetClient(), dynClient, netPlugin, controllerOpts...))
	controllerSet.Add(nick8s.NewController(mgr.GetClient(), dynClient, nicPlugin, controllerOpts...))
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
