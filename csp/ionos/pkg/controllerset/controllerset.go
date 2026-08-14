// Package controllerset exposes the IONOS plugin's controller wiring for the
// standalone IONOS delegator binary (csp/ionos/cmd), published as the
// delegator-ionos image.
//
// The concrete plugin/service/controller types live in csp/ionos/internal and are
// therefore not importable from other modules; this package is the public seam.
package controllerset

import (
	"log/slog"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

	blockstoragectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/block_storage"
	instancectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/instance"
	networkctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/network"
	nicctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/nic"
	publicipctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/public_ip"
	routetablectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/route_table"
	subnetctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/subnet"
	workspacectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/workspace"
	"github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/service"
	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/adapter/crossplane"
	frameworkbuilder "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	instancek8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	publicipk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip/backend/kubernetes"
	routetablek8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
	subnetk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

// Add wires the IONOS (Crossplane-backed) block-storage, network and workspace
// controllers into cs. The plugin handlers write IONOS Crossplane managed
// resources through mgr's client, so the caller must have registered the IONOS
// provider types in mgr's scheme.
func Add(cs *frameworkbuilder.ControllerSet, mgr ctrl.Manager, dynClient dynamic.Interface, clientset kubernetes.Interface, logger *slog.Logger, opts ...frameworkbuilder.Option) {
	wsAdapter := crossplane.NewWorkspaceStore(mgr.GetClient(), logger.With("adapter", "workspace"))
	bsAdapter := crossplane.NewBlockStorageStore(mgr.GetClient(), logger.With("adapter", "block-storage"))
	netAdapter := crossplane.NewNetworkStore(mgr.GetClient(), logger.With("adapter", "network"))
	publicIPAdapter := crossplane.NewPublicIPStore(mgr.GetClient(), logger.With("adapter", "public-ip"))
	nicAdapter := crossplane.NewNicStore(mgr.GetClient(), logger.With("adapter", "nic"))
	subnetAdapter := crossplane.NewSubnetStore(mgr.GetClient(), logger.With("adapter", "subnet"))
	routeTableAdapter := crossplane.NewRouteTableStore(mgr.GetClient(), logger.With("adapter", "route-table"))
	instanceAdapter := crossplane.NewInstanceStore(mgr.GetClient(), logger.With("adapter", "instance"))

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
	publicIPPlugin := &service.PublicIP{
		Creator: &publicipctrl.CreatePublicIP{Store: publicIPAdapter},
		Deleter: &publicipctrl.DeletePublicIP{Store: publicIPAdapter},
	}
	nicPlugin := &service.Nic{
		Creator: &nicctrl.CreateNic{Store: nicAdapter},
		Deleter: &nicctrl.DeleteNic{Store: nicAdapter},
	}
	subnetPlugin := &service.Subnet{
		Creator: &subnetctrl.CreateSubnet{Store: subnetAdapter},
		Deleter: &subnetctrl.DeleteSubnet{Store: subnetAdapter},
	}
	routeTablePlugin := &service.RouteTable{
		Creator: &routetablectrl.CreateRouteTable{Store: routeTableAdapter},
		Deleter: &routetablectrl.DeleteRouteTable{Store: routeTableAdapter},
	}
	instancePlugin := &service.Instance{
		Creator:    &instancectrl.CreateInstance{Store: instanceAdapter},
		Deleter:    &instancectrl.DeleteInstance{Store: instanceAdapter},
		PowerOner:  &instancectrl.PowerOnInstance{Store: instanceAdapter},
		PowerOffer: &instancectrl.PowerOffInstance{Store: instanceAdapter},
	}

	cs.Add(bsk8s.NewController(mgr.GetClient(), dynClient, bsPlugin, opts...))
	cs.Add(netk8s.NewController(mgr.GetClient(), dynClient, clientset, netPlugin, opts...))
	cs.Add(wsk8s.NewController(mgr.GetClient(), dynClient, clientset, wsPlugin, opts...))
	cs.Add(publicipk8s.NewController(mgr.GetClient(), dynClient, publicIPPlugin, opts...))
	cs.Add(nick8s.NewController(mgr.GetClient(), dynClient, nicPlugin, opts...))
	cs.Add(subnetk8s.NewController(mgr.GetClient(), dynClient, subnetPlugin, opts...))
	cs.Add(routetablek8s.NewController(mgr.GetClient(), dynClient, routeTablePlugin, opts...))
	cs.Add(instancek8s.NewController(mgr.GetClient(), dynClient, instancePlugin, opts...))
}
