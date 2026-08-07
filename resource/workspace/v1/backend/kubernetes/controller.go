package kubernetes

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	builder "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	instancek8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
	internetgatewayk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway/backend/kubernetes"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	publicipk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip/backend/kubernetes"
	securitygrouprulek8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
	securitygroupk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group/backend/kubernetes"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// ChildResourceGVRs is the closed set of SECA types that live in the namespace a Workspace owns
// for its children. It gates the gateway's refusal to delete a non-empty Workspace (409) and the
// controller's re-check before it deletes that namespace, so a type missing from it makes the
// namespace look empty when it is not — and the namespace delete cascades.
//
// Subnet and RouteTable are absent on purpose: they are network-scoped and live in the Network's
// own child namespace. See network's ChildResourceGVRs.
var ChildResourceGVRs = []schema.GroupVersionResource{
	bsk8s.BlockStorageGVR,
	imgk8s.ImageGVR,
	netk8s.NetworkGVR,
	nick8s.NICGVR,
	publicipk8s.PublicIPGVR,
	internetgatewayk8s.InternetGatewayGVR,
	securitygroupk8s.SecurityGroupGVR,
	securitygrouprulek8s.SecurityGroupRuleGVR,
	instancek8s.InstanceGVR,
}

// Controller drives workspace reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*wsdom.Workspace]
}

// NewController wires together the workspace controller.
// ctrlClient is the controller-runtime client used for reconciliation.
// dynClient is the dynamic client used by the persistence repo adapter.
// clientset is the typed client used to delete the namespace the workspace owns for its children.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	clientset kubernetes.Interface,
	plugin WorkspacePlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*wsdom.Workspace](
		dynClient,
		WorkspaceGVR,
		options.Logger,
		WorkspaceToCR,
		WorkspaceFromCR,
	)
	handler := NewWorkspacePluginHandler(repo, plugin, options.MaxConditions)
	c := &Controller{
		GenericController: frameworkcontroller.NewGenericController[*wsdom.Workspace](
			ctrlClient,
			WorkspaceFromCR,
			handler,
			&Workspace{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
	c.WithCleanup(k8sadapter.NamespaceCleanup[*wsdom.Workspace](
		dynClient,
		clientset,
		options.Logger,
		k8sadapter.WorkspaceChildren,
		ChildResourceGVRs,
	))

	return c
}
