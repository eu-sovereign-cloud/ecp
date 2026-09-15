package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"

	arubaconverter "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
	arubahandler "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/handler"
	arubarepository "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/repository"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	frameworkbuilder "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	instancek8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
	computeskuk8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku/backend/kubernetes"
	igwk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway/backend/kubernetes"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	pipk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip/backend/kubernetes"
	routetablek8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
	sgrk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
	sgk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group/backend/kubernetes"
	subnetk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
	resourcescheme "github.com/eu-sovereign-cloud/ecp/resource/scheme"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
	ssk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/storage-sku/backend/kubernetes"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

func main() {
	d, err := frameworkbuilder.NewDelegator(ctrl.Options{}, resourcescheme.AddToScheme, v1alpha1.AddToScheme)
	if err != nil {
		slog.Error("unable to bootstrap delegator", "error", err)
		os.Exit(1)
	}

	controllerOpts := []frameworkbuilder.Option{
		frameworkbuilder.WithLogger(d.Logger.With("component", "controller-set")),
		frameworkbuilder.WithRequeueAfter(1 * time.Second),
	}
	loadControllers(context.Background(), d, controllerOpts)

	if err := d.Run(ctrl.SetupSignalHandler()); err != nil {
		d.Logger.Error("delegator stopped", "error", err)
		os.Exit(1)
	}
}

func loadControllers(ctx context.Context, d *frameworkbuilder.Delegator, controllerOpts []frameworkbuilder.Option) {
	dynClient, clientset, mgr, logger, controllerSet := d.Dynamic, d.Clientset, d.Manager, d.Logger, d.Controllers
	logger.Info("Loading 'aruba' plugin set")

	// Instantiate seca-specific read-only repositories. The handlers below read these SECA
	// resources to gate on them (workspace) or to resolve a dependency graph (an instance's NICs,
	// their security groups, and the standalone rules those groups reference).
	secaWsRepo := k8sadapter.NewReaderAdapter(dynClient, wsk8s.WorkspaceGVR, logger, wsk8s.WorkspaceFromCR)
	secaSkuRepo := k8sadapter.NewReaderAdapter(dynClient, ssk8s.StorageSKUGVR, logger, ssk8s.StorageSKUFromCR)
	secaIgwRepo := k8sadapter.NewReaderAdapter(dynClient, igwk8s.InternetGatewayGVR, logger, igwk8s.InternetGatewayFromCR)
	secaNicRepo := k8sadapter.NewReaderAdapter(dynClient, nick8s.NICGVR, logger, nick8s.NicFromCR)
	secaSgRepo := k8sadapter.NewReaderAdapter(dynClient, sgk8s.SecurityGroupGVR, logger, sgk8s.SecurityGroupFromCR)
	secaSgrRepo := k8sadapter.NewReaderAdapter(dynClient, sgrk8s.SecurityGroupRuleGVR, logger, sgrk8s.SecurityGroupRuleFromCR)
	secaComputeSkuRepo := k8sadapter.NewReaderAdapter(dynClient, computeskuk8s.InstanceSKUGVR, logger, computeskuk8s.InstanceSKUFromCR)

	// Instantiate aruba-specific repositories (the arubacloud.com CRs the plugin writes).
	wr := arubarepository.NewProjectRepository(ctx, mgr.GetClient(), mgr.GetCache())
	br := arubarepository.NewBlockStorageRepository(ctx, mgr.GetClient(), mgr.GetCache())
	vpcRepo := arubarepository.NewVPCRepository(ctx, mgr.GetClient(), mgr.GetCache())
	subnetRepo := arubarepository.NewSubnetRepository(ctx, mgr.GetClient(), mgr.GetCache())
	eipRepo := arubarepository.NewElasticIPRepository(ctx, mgr.GetClient(), mgr.GetCache())
	sgRepo := arubarepository.NewSecurityGroupRepository(ctx, mgr.GetClient(), mgr.GetCache())
	srRepo := arubarepository.NewSecurityRuleRepository(ctx, mgr.GetClient(), mgr.GetCache())
	keyPairRepo := arubarepository.NewKeyPairRepository(ctx, mgr.GetClient(), mgr.GetCache())
	cloudServerRepo := arubarepository.NewCloudServerRepository(ctx, mgr.GetClient(), mgr.GetCache())

	// Instantiate aruba-specific converters
	wc := arubaconverter.NewWorkspaceProjectConverter()
	bc := arubaconverter.NewBlockStorageConverter()
	netConv := arubaconverter.NewNetworkVPCConverter()
	subnetConv := arubaconverter.NewSubnetConverter()
	pipConv := arubaconverter.NewPublicIpElasticIpConverter()

	// Create aruba-specific handlers
	wsPlugin := arubahandler.NewWorkspaceHandler(wr, wc)
	bsPlugin := arubahandler.NewBlockStorageHandler(secaWsRepo, secaSkuRepo, br, wr, bc, wc)
	netPlugin := arubahandler.NewNetworkHandler(secaWsRepo, secaIgwRepo, vpcRepo, wr, netConv, wc)
	subnetPlugin := arubahandler.NewSubnetHandler(secaWsRepo, subnetRepo, vpcRepo, wr, subnetConv, wc)
	pipPlugin := arubahandler.NewPublicIpHandler(secaWsRepo, eipRepo, wr, pipConv, wc)
	instancePlugin := arubahandler.NewComputeInstanceHandler(secaWsRepo, secaNicRepo, secaSgRepo, secaSgrRepo,
		secaComputeSkuRepo, wr, subnetRepo, keyPairRepo, sgRepo, srRepo, br, eipRepo, cloudServerRepo)

	// Route table, internet gateway, nic, security group and security group rule have no Aruba CR
	// of their own: they are accepted and go active immediately (the real Aruba security groups are
	// materialised by the instance handler, per VPC, at attach time). See csp/aruba/README.md.
	igwPlugin := arubahandler.NewInternetGatewayHandler()
	rtPlugin := arubahandler.NewRouteTableHandler()
	sgPlugin := arubahandler.NewSecurityGroupHandler(sgRepo, srRepo)
	sgrPlugin := arubahandler.NewSecurityGroupRuleHandler()
	nicPlugin := arubahandler.NewNicHandler()
	imgPlugin := arubahandler.NewImageHandler()

	controllerSet.Add(bsk8s.NewController(mgr.GetClient(), dynClient, bsPlugin, controllerOpts...))
	controllerSet.Add(wsk8s.NewController(mgr.GetClient(), dynClient, clientset, wsPlugin, controllerOpts...))
	controllerSet.Add(netk8s.NewController(mgr.GetClient(), dynClient, clientset, netPlugin, controllerOpts...))
	controllerSet.Add(subnetk8s.NewController(mgr.GetClient(), dynClient, subnetPlugin, controllerOpts...))
	controllerSet.Add(pipk8s.NewController(mgr.GetClient(), dynClient, pipPlugin, controllerOpts...))
	controllerSet.Add(igwk8s.NewController(mgr.GetClient(), dynClient, igwPlugin, controllerOpts...))
	controllerSet.Add(routetablek8s.NewController(mgr.GetClient(), dynClient, rtPlugin, controllerOpts...))
	controllerSet.Add(sgk8s.NewController(mgr.GetClient(), dynClient, sgPlugin, controllerOpts...))
	controllerSet.Add(sgrk8s.NewController(mgr.GetClient(), dynClient, sgrPlugin, controllerOpts...))
	controllerSet.Add(nick8s.NewController(mgr.GetClient(), dynClient, nicPlugin, controllerOpts...))
	controllerSet.Add(imgk8s.NewController(mgr.GetClient(), dynClient, imgPlugin, controllerOpts...))
	controllerSet.Add(instancek8s.NewController(mgr.GetClient(), dynClient, instancePlugin, controllerOpts...))
}
