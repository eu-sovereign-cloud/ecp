package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
)

// Controller drives SecurityGroup reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*securitygroupdom.SecurityGroup]
}

// NewController wires together the SecurityGroup controller.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	plugin SecurityGroupPlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*securitygroupdom.SecurityGroup](
		dynClient,
		SecurityGroupGVR,
		options.Logger,
		SecurityGroupToCR,
		SecurityGroupFromCR,
	)
	handler := NewSecurityGroupPluginHandler(repo, plugin, options.MaxConditions)
	return &Controller{
		GenericController: frameworkcontroller.NewGenericController[*securitygroupdom.SecurityGroup](
			ctrlClient,
			SecurityGroupFromCR,
			handler,
			&SecurityGroup{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
}
