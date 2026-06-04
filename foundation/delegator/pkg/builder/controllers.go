package builder

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/controller"
	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/converters/kubernetes2domain"
	networksv1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/network/networks/v1"
	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/storage/block-storages/v1"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/workspace/v1"
	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/foundation/persistence/adapters/kubernetes"
)

func newBlockStorageController(
	client client.Client,
	dynamicClient dynamic.Interface,
	plugin plugin.BlockStorage,
	opts Options,
) controller.BlockStorageController {
	repo := kubernetesadapter.NewRepoAdapter(
		dynamicClient,
		blockstoragev1.BlockStorageGVR,
		opts.Logger,
		kubernetes2domain.MapBlockStorageDomainToCR,
		kubernetes2domain.MapCRToBlockStorageDomain,
	)

	return controller.NewBlockStorageController(client, repo, plugin, opts.RequeueAfter, opts.Logger, opts.MaxConditions)
}

func newWorkspaceController(
	client client.Client,
	dynamicClient dynamic.Interface,
	clientset kubernetes.Interface,
	plugin plugin.Workspace,
	opts Options,
) controller.WorkspaceController {
	repo := kubernetesadapter.NewNamespaceManagingRepoAdapter(
		dynamicClient,
		clientset,
		workspacev1.WorkspaceGVR,
		opts.Logger,
		kubernetes2domain.MapWorkspaceDomainToCR,
		kubernetes2domain.MapCRToWorkspaceDomain,
	)

	return controller.NewWorkspaceController(client, repo, plugin, opts.RequeueAfter, opts.Logger, opts.MaxConditions)
}

func newNetworkController(
	client client.Client,
	dynamicClient dynamic.Interface,
	plugin plugin.Network,
	opts Options,
) controller.NetworkController {
	repo := kubernetesadapter.NewRepoAdapter(
		dynamicClient,
		networksv1.NetworkGVR,
		opts.Logger,
		kubernetesadapter.MapNetworkDomainToCR,
		kubernetesadapter.MapCRToNetworkDomain,
	)

	return controller.NewNetworkController(client, repo, plugin, opts.RequeueAfter, opts.Logger, opts.MaxConditions)
}
