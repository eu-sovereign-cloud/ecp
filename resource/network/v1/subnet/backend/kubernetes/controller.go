package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

// Controller drives Subnet reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*subnetdom.Subnet]
}

// NewController wires together the Subnet controller.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	plugin SubnetPlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*subnetdom.Subnet](
		dynClient,
		SubnetGVR,
		options.Logger,
		Converter,
	)
	handler := NewSubnetPluginHandler(repo, plugin, options.MaxConditions)
	return &Controller{
		GenericController: frameworkcontroller.NewGenericController[*subnetdom.Subnet](
			ctrlClient,
			SubnetFromCR,
			handler,
			&Subnet{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
}
