package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	builder "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
)

// Controller drives role reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*roledom.Role]
}

// NewController wires together the role controller.
// ctrlClient is the controller-runtime client used for reconciliation.
// dynClient is the dynamic client used by the persistence repo adapter.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	plugin RolePlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*roledom.Role](
		dynClient,
		RoleGVR,
		options.Logger,
		RoleToCR,
		RoleFromCR,
	)
	handler := NewRolePluginHandler(repo, plugin, options.MaxConditions)
	return &Controller{
		GenericController: frameworkcontroller.NewGenericController[*roledom.Role](
			ctrlClient,
			RoleFromCR,
			handler,
			&Role{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
}
