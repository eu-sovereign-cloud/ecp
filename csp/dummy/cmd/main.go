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
	instancek8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
	internetgatewayk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway/backend/kubernetes"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	publicipk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip/backend/kubernetes"
	routetablek8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
	securitygrouprulek8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
	securitygroupk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group/backend/kubernetes"
	subnetk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(bsk8s.AddToScheme(scheme))
	utilruntime.Must(imgk8s.AddToScheme(scheme))
	utilruntime.Must(netk8s.AddToScheme(scheme))
	utilruntime.Must(nick8s.AddToScheme(scheme))
	utilruntime.Must(publicipk8s.AddToScheme(scheme))
	utilruntime.Must(internetgatewayk8s.AddToScheme(scheme))
	utilruntime.Must(routetablek8s.AddToScheme(scheme))
	utilruntime.Must(subnetk8s.AddToScheme(scheme))
	utilruntime.Must(securitygroupk8s.AddToScheme(scheme))
	utilruntime.Must(securitygrouprulek8s.AddToScheme(scheme))
	utilruntime.Must(instancek8s.AddToScheme(scheme))
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
	imgPlugin := dummyplugin.NewImage(logger.With("plugin", "image"))
	wsPlugin := dummyplugin.NewWorkspace(logger.With("plugin", "workspace"))
	netPlugin := dummyplugin.NewNetwork(logger.With("plugin", "network"))
	nicPlugin := dummyplugin.NewNic(logger.With("plugin", "nic"))
	publicIpPlugin := dummyplugin.NewPublicIp(logger.With("plugin", "publicip"))
	internetGatewayPlugin := dummyplugin.NewInternetGateway(logger.With("plugin", "internetgateway"))
	routeTablePlugin := dummyplugin.NewRouteTable(logger.With("plugin", "routetable"))
	subnetPlugin := dummyplugin.NewSubnet(logger.With("plugin", "subnet"))
	securityGroupPlugin := dummyplugin.NewSecurityGroup(logger.With("plugin", "securitygroup"))
	securityGroupRulePlugin := dummyplugin.NewSecurityGroupRule(logger.With("plugin", "securitygrouprule"))
	instancePlugin := dummyplugin.NewInstance(logger.With("plugin", "instance"))

	controllerOpts := []frameworkbuilder.Option{
		frameworkbuilder.WithLogger(logger.With("component", "controller-set")),
		frameworkbuilder.WithRequeueAfter(1 * time.Second),
	}

	controllerSet := frameworkbuilder.NewControllerSet()
	controllerSet.Add(bsk8s.NewController(mgr.GetClient(), dynClient, bsPlugin, controllerOpts...))
	controllerSet.Add(imgk8s.NewController(mgr.GetClient(), dynClient, imgPlugin, controllerOpts...))
	controllerSet.Add(netk8s.NewController(mgr.GetClient(), dynClient, netPlugin, controllerOpts...))
	controllerSet.Add(nick8s.NewController(mgr.GetClient(), dynClient, nicPlugin, controllerOpts...))
	controllerSet.Add(publicipk8s.NewController(mgr.GetClient(), dynClient, publicIpPlugin, controllerOpts...))
	controllerSet.Add(internetgatewayk8s.NewController(mgr.GetClient(), dynClient, internetGatewayPlugin, controllerOpts...))
	controllerSet.Add(routetablek8s.NewController(mgr.GetClient(), dynClient, routeTablePlugin, controllerOpts...))
	controllerSet.Add(subnetk8s.NewController(mgr.GetClient(), dynClient, subnetPlugin, controllerOpts...))
	controllerSet.Add(securitygroupk8s.NewController(mgr.GetClient(), dynClient, securityGroupPlugin, controllerOpts...))
	controllerSet.Add(securitygrouprulek8s.NewController(mgr.GetClient(), dynClient, securityGroupRulePlugin, controllerOpts...))
	controllerSet.Add(instancek8s.NewController(mgr.GetClient(), dynClient, instancePlugin, controllerOpts...))
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
