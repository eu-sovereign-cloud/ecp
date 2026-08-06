package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	builder "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// Controller drives workspace reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*wsdom.Workspace]
}

// NewController wires together the workspace controller.
// ctrlClient is the controller-runtime client used for reconciliation.
// dynClient is the dynamic client used by the persistence repo adapter.
// clientset is the typed client used to delete the namespace the workspace owns for its children.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	clientset kubernetes.Interface,
	plugin WorkspacePlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*wsdom.Workspace](
		dynClient,
		WorkspaceGVR,
		options.Logger,
		WorkspaceToCR,
		WorkspaceFromCR,
	)
	handler := NewWorkspacePluginHandler(repo, plugin, options.MaxConditions)
	c := &Controller{
		GenericController: frameworkcontroller.NewGenericController[*wsdom.Workspace](
			ctrlClient,
			WorkspaceFromCR,
			handler,
			&Workspace{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
	c.WithEnsure(k8sadapter.NamespaceEnsure[*wsdom.Workspace](
		clientset,
		options.Logger,
		k8sadapter.WorkspaceChildren,
	))
	c.WithCleanup(k8sadapter.NamespaceCleanup[*wsdom.Workspace](
		dynClient,
		clientset,
		options.Logger,
		k8sadapter.WorkspaceChildren,
		ChildResourceGVRs,
	))

	return c
}
