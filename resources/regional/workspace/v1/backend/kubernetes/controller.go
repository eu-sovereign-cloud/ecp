package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	builder "github.com/eu-sovereign-cloud/ecp/framework/backend/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/controller"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1/domain"
)

// Controller drives workspace reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*wsdom.WorkspaceDomain]
}

// NewController wires together the workspace controller.
// ctrlClient is the controller-runtime client used for reconciliation.
// dynClient is the dynamic client used by the persistence repo adapter.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	plugin WorkspacePlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*wsdom.WorkspaceDomain](
		dynClient,
		WorkspaceGVR,
		options.Logger,
		MapWorkspaceDomainToCR,
		MapCRToWorkspaceDomain,
	)
	handler := NewWorkspacePluginHandler(repo, plugin, options.MaxConditions)
	return &Controller{
		GenericController: frameworkcontroller.NewGenericController[*wsdom.WorkspaceDomain](
			ctrlClient,
			MapCRToWorkspaceDomain,
			handler,
			&Workspace{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
}
