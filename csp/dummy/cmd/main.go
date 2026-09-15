package main

import (
	"log/slog"
	"os"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"

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
	resourcescheme "github.com/eu-sovereign-cloud/ecp/resource/scheme"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

func main() {
	d, err := frameworkbuilder.NewDelegator(ctrl.GetConfigOrDie(), ctrl.Options{}, resourcescheme.AddToScheme)
	if err != nil {
		slog.Error("unable to bootstrap delegator", "error", err)
		os.Exit(1)
	}
	logger, mgr := d.Logger, d.Manager

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

	d.Controllers.
		Add(bsk8s.NewController(mgr.GetClient(), d.Dynamic, bsPlugin, controllerOpts...)).
		Add(imgk8s.NewController(mgr.GetClient(), d.Dynamic, imgPlugin, controllerOpts...)).
		Add(netk8s.NewController(mgr.GetClient(), d.Dynamic, d.Clientset, netPlugin, controllerOpts...)).
		Add(nick8s.NewController(mgr.GetClient(), d.Dynamic, nicPlugin, controllerOpts...)).
		Add(publicipk8s.NewController(mgr.GetClient(), d.Dynamic, publicIpPlugin, controllerOpts...)).
		Add(internetgatewayk8s.NewController(mgr.GetClient(), d.Dynamic, internetGatewayPlugin, controllerOpts...)).
		Add(routetablek8s.NewController(mgr.GetClient(), d.Dynamic, routeTablePlugin, controllerOpts...)).
		Add(subnetk8s.NewController(mgr.GetClient(), d.Dynamic, subnetPlugin, controllerOpts...)).
		Add(securitygroupk8s.NewController(mgr.GetClient(), d.Dynamic, securityGroupPlugin, controllerOpts...)).
		Add(securitygrouprulek8s.NewController(mgr.GetClient(), d.Dynamic, securityGroupRulePlugin, controllerOpts...)).
		Add(instancek8s.NewController(mgr.GetClient(), d.Dynamic, instancePlugin, controllerOpts...)).
		Add(wsk8s.NewController(mgr.GetClient(), d.Dynamic, d.Clientset, wsPlugin, controllerOpts...))

	if err := d.Run(ctrl.SetupSignalHandler()); err != nil {
		logger.Error("delegator stopped", "error", err)
		os.Exit(1)
	}
}
