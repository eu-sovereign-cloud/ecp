package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	builder "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role-assignment/v1"
)

// Controller drives role assignment reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*radom.RoleAssignment]
}

// NewController wires together the role assignment controller.
// ctrlClient is the controller-runtime client used for reconciliation.
// dynClient is the dynamic client used by the persistence repo adapter.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	plugin RoleAssignmentPlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*radom.RoleAssignment](
		dynClient,
		RoleAssignmentGVR,
		options.Logger,
		RoleAssignmentToCR,
		RoleAssignmentFromCR,
	)
	handler := NewRoleAssignmentPluginHandler(repo, plugin, options.MaxConditions)
	return &Controller{
		GenericController: frameworkcontroller.NewGenericController[*radom.RoleAssignment](
			ctrlClient,
			RoleAssignmentFromCR,
			handler,
			&RoleAssignment{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
}
