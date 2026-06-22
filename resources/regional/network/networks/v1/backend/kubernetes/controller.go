package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	builder "github.com/eu-sovereign-cloud/ecp/framework/backend/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/controller"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes"
	netdom "github.com/eu-sovereign-cloud/ecp/resources/regional/network/networks/v1"
)

// Controller drives network reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*netdom.NetworkDomain]
}

// NewController wires together the network controller.
// ctrlClient is the controller-runtime client used for reconciliation.
// dynClient is the dynamic client used by the persistence repo adapter.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	plugin NetworkPlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*netdom.NetworkDomain](
		dynClient,
		NetworkGVR,
		options.Logger,
		MapNetworkDomainToCR,
		MapCRToNetworkDomain,
	)
	handler := NewNetworkPluginHandler(repo, plugin, options.MaxConditions)
	return &Controller{
		GenericController: frameworkcontroller.NewGenericController[*netdom.NetworkDomain](
			ctrlClient,
			MapCRToNetworkDomain,
			handler,
			&Network{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
}
