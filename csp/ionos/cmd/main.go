package main

import (
	"log/slog"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	ionosapis "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"

	blockstoragectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/block_storage"
	networkctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/network"
	workspacectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/workspace"
	"github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/service"
	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/adapter/crossplane"
	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/builder"
	networkpersistence "github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/network"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/storage"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/workspace/v1"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(workspacev1.AddToScheme(scheme))
	utilruntime.Must(storage.AddToScheme(scheme))
	utilruntime.Must(networkpersistence.AddToScheme(scheme))
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

	wsAdapter := crossplane.NewWorkspaceStore(mgr.GetClient(), logger.With("adapter", "workspace"))
	bsAdapter := crossplane.NewBlockStorageStore(mgr.GetClient(), logger.With("adapter", "block-storage"))
	netAdapter := crossplane.NewNetworkStore(mgr.GetClient(), logger.With("adapter", "network"))

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

	pluginSet := builder.PluginSet{
		Workspace:    wsPlugin,
		BlockStorage: bsPlugin,
		Network:      netPlugin,
	}

	controllerSet, err := builder.NewControllerSet(
		mgr.GetConfig(),
		mgr.GetClient(),
		pluginSet,
		builder.WithLogger(logger.With("component", "controller-set")),
		builder.WithRequeueAfter(1*time.Second),
		builder.WithMaxConditions(5),
	)
	if err != nil {
		logger.Error("unable to create controller set", "error", err)
		os.Exit(1)
	}

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
