package kubernetes

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	builder "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	routetablek8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
	subnetk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
)

// ChildResourceGVRs is the closed set of SECA types that live in the namespace a Network owns for
// its children. It gates the gateway's refusal to delete a non-empty Network (409) and the
// controller's re-check before it deletes that namespace, so a type missing from it makes the
// namespace look empty when it is not — and the namespace delete cascades.
var ChildResourceGVRs = []schema.GroupVersionResource{
	routetablek8s.RouteTableGVR,
	subnetk8s.SubnetGVR,
}

// Controller drives network reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*netdom.Network]
}

// NewController wires together the network controller.
// ctrlClient is the controller-runtime client used for reconciliation.
// dynClient is the dynamic client used by the persistence repo adapter.
// clientset is the typed client used to delete the namespace the network owns for its children.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	clientset kubernetes.Interface,
	plugin NetworkPlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*netdom.Network](
		dynClient,
		NetworkGVR,
		options.Logger,
		NetworkToCR,
		NetworkFromCR,
	)
	handler := NewNetworkPluginHandler(repo, plugin, options.MaxConditions)
	c := &Controller{
		GenericController: frameworkcontroller.NewGenericController[*netdom.Network](
			ctrlClient,
			NetworkFromCR,
			handler,
			&Network{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
	c.WithCleanup(k8sadapter.NamespaceCleanup[*netdom.Network](
		dynClient,
		clientset,
		options.Logger,
		k8sadapter.NetworkChildren,
		ChildResourceGVRs,
	))

	return c
}
