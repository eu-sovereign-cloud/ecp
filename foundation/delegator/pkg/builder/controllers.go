package builder

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/controller"
	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/foundation/persistence/adapters/kubernetes"
	kubernetes2domain "github.com/eu-sovereign-cloud/ecp/foundation/persistence/adapters/kubernetes2domain"
	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/storage/block-storages/v1"
	imagev1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/storage/images/v1"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/workspace/v1"
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

func newImageController(
	client client.Client,
	dynamicClient dynamic.Interface,
	plugin plugin.Image,
	opts Options,
) controller.ImageController {
	repo := kubernetesadapter.NewRepoAdapter(
		dynamicClient,
		imagev1.ImageGVR,
		opts.Logger,
		kubernetes2domain.MapImageDomainToCR,
		kubernetes2domain.MapCRToImageDomain,
	)

	return controller.NewImageController(client, repo, plugin, opts.RequeueAfter, opts.Logger, opts.MaxConditions)
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
