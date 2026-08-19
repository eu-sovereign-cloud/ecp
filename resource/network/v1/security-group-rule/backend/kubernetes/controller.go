package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	frameworkcontroller "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/controller"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

// Controller drives SecurityGroupRule reconciliation using the GenericController.
type Controller struct {
	frameworkcontroller.GenericController[*securitygroupruledom.SecurityGroupRule]
}

// NewController wires together the SecurityGroupRule controller.
func NewController(
	ctrlClient client.Client,
	dynClient dynamic.Interface,
	plugin SecurityGroupRulePlugin,
	opts ...builder.Option,
) *Controller {
	options := builder.ApplyOptions(opts)
	repo := k8sadapter.NewRepoAdapter[*securitygroupruledom.SecurityGroupRule](
		dynClient,
		SecurityGroupRuleGVR,
		options.Logger,
		Converter,
	)
	handler := NewSecurityGroupRulePluginHandler(repo, plugin, options.MaxConditions)
	return &Controller{
		GenericController: frameworkcontroller.NewGenericController[*securitygroupruledom.SecurityGroupRule](
			ctrlClient,
			SecurityGroupRuleFromCR,
			handler,
			&SecurityGroupRule{},
			options.RequeueAfter,
			options.Logger,
			options.MaxConditions,
		),
	}
}
