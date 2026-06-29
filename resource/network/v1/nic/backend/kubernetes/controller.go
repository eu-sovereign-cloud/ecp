package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

// Controller drives NIC reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*nicdom.Nic]
}

// NewController wires together the NIC controller.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	plugin NicPlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*nicdom.Nic](
		dynClient,
		NICGVR,
		options.Logger,
		NicToCR,
		NicFromCR,
	)
	handler := NewNicPluginHandler(repo, plugin, options.MaxConditions)
	return &Controller{
		GenericController: frameworkcontroller.NewGenericController[*nicdom.Nic](
			ctrlClient,
			NicFromCR,
			handler,
			&NIC{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
}
