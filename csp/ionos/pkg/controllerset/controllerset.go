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
	ctrl "sigs.k8s.io/controller-runtime"

	blockstoragectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/block_storage"
	networkctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/network"
	workspacectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/workspace"
	"github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/service"
	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/adapter/crossplane"
	frameworkbuilder "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

// Add wires the IONOS (Crossplane-backed) block-storage, network and workspace
// controllers into cs. The plugin handlers write IONOS Crossplane managed
// resources through mgr's client, so the caller must have registered the IONOS
// provider types in mgr's scheme.
func Add(cs *frameworkbuilder.ControllerSet, mgr ctrl.Manager, dynClient dynamic.Interface, logger *slog.Logger, opts ...frameworkbuilder.Option) {
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

	cs.Add(bsk8s.NewController(mgr.GetClient(), dynClient, bsPlugin, opts...))
	cs.Add(netk8s.NewController(mgr.GetClient(), dynClient, netPlugin, opts...))
	cs.Add(wsk8s.NewController(mgr.GetClient(), dynClient, wsPlugin, opts...))
}
