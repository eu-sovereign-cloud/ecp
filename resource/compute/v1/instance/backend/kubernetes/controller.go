package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// Controller drives Instance reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*instancedom.Instance]
}

// NewController wires together the Instance controller.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	plugin InstancePlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*instancedom.Instance](
		dynClient,
		InstanceGVR,
		options.Logger,
		InstanceToCR,
		InstanceFromCR,
	)
	handler := NewInstancePluginHandler(repo, plugin, options.MaxConditions)
	return &Controller{
		GenericController: frameworkcontroller.NewGenericController[*instancedom.Instance](
			ctrlClient,
			InstanceFromCR,
			handler,
			&Instance{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
}
