package kubernetes

import (
	"context"
	"errors"
	"log"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
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

func (h *SecurityGroupRulePluginHandler) HandleReconcile(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule) (bool, error) {
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
		return false, nil // Nothing to do.
	}

	if err := delegate(ctx, resource); err != nil {
		if errors.Is(err, backendport.ErrStillProcessing) {
			return true, nil
		}
		if requeue, err := h.setResourceErrorState(ctx, resource, err, false); err != nil {
			return requeue, err
		}
		return true, nil
	}

	switch {
	case isSecurityGroupRuleAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending, false)
	case isSecurityGroupRulePending(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	case isSecurityGroupRuleCreating(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive, false)
	case wantSecurityGroupRuleDelete(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting, true)
	case isSecurityGroupRuleDeleting(resource):
		return false, nil
	case wantSecurityGroupRuleRetryCreate(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *SecurityGroupRulePluginHandler) setResourceState(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule, state commondomain.ResourceState, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &securitygroupruledom.SecurityGroupRuleStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromState(state))
	for h.MaxConditions > 0 && len(resource.Status.Conditions) > h.MaxConditions {
		resource.Status.PopCondition()
	}

	if _, err := h.repo.UpdateStatus(ctx, resource); err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			return false, nil
		}
		return requeue, err
	}

	return requeue, nil
}

func (h *SecurityGroupRulePluginHandler) setResourceErrorState(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &securitygroupruledom.SecurityGroupRuleStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromError(err))
	for h.MaxConditions > 0 && len(resource.Status.Conditions) > h.MaxConditions {
		resource.Status.PopCondition()
	}

	if _, updateErr := h.repo.UpdateStatus(ctx, resource); updateErr != nil {
		if errors.Is(updateErr, kernel.ErrNotFound) {
			return false, nil
		}
		return requeue, updateErr
	}

	return requeue, nil
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
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == commondomain.ResourceStateCreating
}
