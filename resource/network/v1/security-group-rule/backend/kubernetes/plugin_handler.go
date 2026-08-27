package kubernetes

import (
	"context"
	"errors"
	"fmt"

	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"

	frameworkbackend "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
)

// SecurityGroupRulePluginHandler drives the SecurityGroupRule reconciliation state machine.
type SecurityGroupRulePluginHandler struct {
	frameworkbackend.GenericPluginHandler[*securitygroupruledom.SecurityGroupRule]
	repo   persistence.Repo[*securitygroupruledom.SecurityGroupRule]
	plugin SecurityGroupRulePlugin
}

var _ backendport.PluginHandler[*securitygroupruledom.SecurityGroupRule] = (*SecurityGroupRulePluginHandler)(nil)

// NewSecurityGroupRulePluginHandler creates a new SecurityGroupRulePluginHandler.
func NewSecurityGroupRulePluginHandler(
	repo persistence.Repo[*securitygroupruledom.SecurityGroupRule],
	plugin SecurityGroupRulePlugin,
	maxConditions int,
) *SecurityGroupRulePluginHandler {
	handler := &SecurityGroupRulePluginHandler{
		repo:   repo,
		plugin: plugin,
	}
	handler.MaxConditions = maxConditions

	return handler
}

func (h *SecurityGroupRulePluginHandler) HandleReconcile(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule) error {
	// An active resource has no lifecycle transition left to make, so it takes the update
	// path instead of the create/delete state machine below. See commonbackend.HandleUpdate.
	if isSecurityGroupRuleActive(resource) {
		return commonbackend.HandleUpdate(ctx, resource, &resource.Status.Status, h.plugin.Update, h.repo, h.MaxConditions)
	}

	var delegate backendport.DelegatedFunc[*securitygroupruledom.SecurityGroupRule]

	switch {
	case isSecurityGroupRuleAccepted(resource):
		delegate = frameworkbackend.BypassDelegated[*securitygroupruledom.SecurityGroupRule]
	case isSecurityGroupRulePending(resource):
		delegate = frameworkbackend.BypassDelegated[*securitygroupruledom.SecurityGroupRule]
	case isSecurityGroupRuleCreating(resource):
		delegate = h.plugin.Create
	case wantSecurityGroupRuleDelete(resource):
		delegate = frameworkbackend.BypassDelegated[*securitygroupruledom.SecurityGroupRule]
	case isSecurityGroupRuleDeleting(resource):
		delegate = h.plugin.Delete
	case wantSecurityGroupRuleRetryCreate(resource):
		delegate = frameworkbackend.BypassDelegated[*securitygroupruledom.SecurityGroupRule]
	default:
		return nil // Nothing to do.
	}

	if err := delegate(ctx, resource); err != nil {
		var rq backendport.RequeueError
		if errors.As(err, &rq) {
			// Not a failure, and not ours to reinterpret: the plugin named its own cadence.
			return err
		}

		// The failure is recorded on the resource; retry it on the next pass, unless it is
		// already gone, in which case there is nothing left to reconcile.
		return commonbackend.RequeueAfterState(h.setResourceErrorState(ctx, resource, err))
	}

	switch {
	case isSecurityGroupRuleAccepted(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStatePending))
	case isSecurityGroupRulePending(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))
	case isSecurityGroupRuleCreating(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStateActive))
	case wantSecurityGroupRuleDelete(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting))
	case isSecurityGroupRuleDeleting(resource):
		return nil
	case wantSecurityGroupRuleRetryCreate(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))
	default:
		return fmt.Errorf("unreachable reconcile state for security group rule %q", resource.GetName())
	}
}

func (h *SecurityGroupRulePluginHandler) setResourceState(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule, state commondomain.ResourceState) error {
	if resource.Status == nil {
		resource.Status = &securitygroupruledom.SecurityGroupRuleStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromState(state))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, err := h.repo.UpdateStatus(ctx, resource)

	return err
}

func (h *SecurityGroupRulePluginHandler) setResourceErrorState(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule, err error) error {
	if resource.Status == nil {
		resource.Status = &securitygroupruledom.SecurityGroupRuleStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromError(err))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, updateErr := h.repo.UpdateStatus(ctx, resource)

	return updateErr
}

func isSecurityGroupRuleActive(resource *securitygroupruledom.SecurityGroupRule) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateActive
}

func isSecurityGroupRuleAccepted(resource *securitygroupruledom.SecurityGroupRule) bool {
	return resource.Status == nil
}

func isSecurityGroupRulePending(resource *securitygroupruledom.SecurityGroupRule) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == commondomain.ResourceStatePending)
}

func isSecurityGroupRuleCreating(resource *securitygroupruledom.SecurityGroupRule) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateCreating
}

func securityGroupRuleIsNotDeleting(resource *securitygroupruledom.SecurityGroupRule) bool {
	return resource.Status == nil ||
		resource.Status.State != commondomain.ResourceStateDeleting
}

func wantSecurityGroupRuleDelete(resource *securitygroupruledom.SecurityGroupRule) bool {
	return resource.DeletedAt != nil && securityGroupRuleIsNotDeleting(resource)
}

func isSecurityGroupRuleDeleting(resource *securitygroupruledom.SecurityGroupRule) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateDeleting
}

func wantSecurityGroupRuleRetryCreate(resource *securitygroupruledom.SecurityGroupRule) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[1].State == commondomain.ResourceStateCreating
}
