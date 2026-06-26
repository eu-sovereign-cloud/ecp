package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	builder "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
)

// Controller drives image reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*imgdom.Image]
}

// NewController wires together the image controller.
// ctrlClient is the controller-runtime client used for reconciliation.
// dynClient is the dynamic client used by the persistence repo adapter.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	plugin ImagePlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*imgdom.Image](
		dynClient,
		ImageGVR,
		options.Logger,
		ImageToCR,
		ImageFromCR,
	)
	handler := NewImagePluginHandler(repo, plugin, options.MaxConditions)
	return &Controller{
		GenericController: frameworkcontroller.NewGenericController[*imgdom.Image](
			ctrlClient,
			ImageFromCR,
			handler,
			&Image{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
}
