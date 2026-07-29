package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

// Controller drives InternetGateway reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*internetgatewaydom.InternetGateway]
}

// NewController wires together the InternetGateway controller.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	plugin InternetGatewayPlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*internetgatewaydom.InternetGateway](
		dynClient,
		InternetGatewayGVR,
		options.Logger,
		InternetGatewayToCR,
		InternetGatewayFromCR,
	)
	handler := NewInternetGatewayPluginHandler(repo, plugin, options.MaxConditions)
	return &Controller{
		GenericController: frameworkcontroller.NewGenericController[*internetgatewaydom.InternetGateway](
			ctrlClient,
			InternetGatewayFromCR,
			handler,
			&InternetGateway{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
}
