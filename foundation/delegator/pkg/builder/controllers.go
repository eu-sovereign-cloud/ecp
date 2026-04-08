package builder

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/controller"
	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/regional/storage/block-storages/v1"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/regional/workspace/v1"
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
		kubernetesadapter.MapBlockStorageDomainToCR,
		kubernetesadapter.MapCRToBlockStorageDomain,
	)

	return controller.NewBlockStorageController(client, repo, plugin, opts.RequeueAfter, opts.Logger)
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
		kubernetesadapter.MapWorkspaceDomainToCR,
		kubernetesadapter.MapCRToWorkspaceDomain,
	)

	return controller.NewWorkspaceController(client, repo, plugin, opts.RequeueAfter, opts.Logger)
}
