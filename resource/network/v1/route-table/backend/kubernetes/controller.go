package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

// Controller drives RouteTable reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*routetabledom.RouteTable]
}

// NewController wires together the RouteTable controller.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	plugin RouteTablePlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*routetabledom.RouteTable](
		dynClient,
		RouteTableGVR,
		options.Logger,
		Converter,
	)
	handler := NewRouteTablePluginHandler(repo, plugin, options.MaxConditions)
	return &Controller{
		GenericController: frameworkcontroller.NewGenericController[*routetabledom.RouteTable](
			ctrlClient,
			RouteTableFromCR,
			handler,
			&RouteTable{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
}
