package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	builder "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	bsdom "github.com/eu-sovereign-cloud/ecp/resources/storage/block-storages/v1"
)

// Controller drives block-storage reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*bsdom.BlockStorage]
}

// NewController wires together the block-storage controller.
// ctrlClient is the controller-runtime client used for reconciliation.
// dynClient is the dynamic client used by the persistence repo adapter.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	plugin BlockStoragePlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*bsdom.BlockStorage](
		dynClient,
		BlockStorageGVR,
		options.Logger,
		MapBlockStorageDomainToCR,
		MapCRToBlockStorageDomain,
	)
	handler := NewBlockStoragePluginHandler(repo, plugin, options.MaxConditions)
	return &Controller{
		GenericController: frameworkcontroller.NewGenericController[*bsdom.BlockStorage](
			ctrlClient,
			MapCRToBlockStorageDomain,
			handler,
			&BlockStorage{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
}
