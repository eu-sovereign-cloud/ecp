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
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
)

// SecurityGroupPluginHandler drives the SecurityGroup reconciliation state machine.
type SecurityGroupPluginHandler struct {
	frameworkbackend.GenericPluginHandler[*securitygroupdom.SecurityGroup]
	repo   persistence.Repo[*securitygroupdom.SecurityGroup]
	plugin SecurityGroupPlugin
}

var _ backendport.PluginHandler[*securitygroupdom.SecurityGroup] = (*SecurityGroupPluginHandler)(nil)

// NewSecurityGroupPluginHandler creates a new SecurityGroupPluginHandler.
func NewSecurityGroupPluginHandler(
	repo persistence.Repo[*securitygroupdom.SecurityGroup],
	plugin SecurityGroupPlugin,
	maxConditions int,
) *SecurityGroupPluginHandler {
	handler := &SecurityGroupPluginHandler{
		repo:   repo,
		plugin: plugin,
	}
	handler.MaxConditions = maxConditions

	return handler
}

func (h *SecurityGroupPluginHandler) HandleReconcile(ctx context.Context, resource *securitygroupdom.SecurityGroup) (bool, error) {
	// An active resource has no lifecycle transition left to make, so it takes the update
	// path instead of the create/delete state machine below. See commonbackend.HandleUpdate.
	if isSecurityGroupActive(resource) {
		return commonbackend.HandleUpdate(ctx, resource, &resource.Status.Status, h.plugin.Update, h.persistStatus, h.MaxConditions)
	}

	var delegate backendport.DelegatedFunc[*securitygroupdom.SecurityGroup]

	switch {
	case isSecurityGroupAccepted(resource):
		delegate = frameworkbackend.BypassDelegated[*securitygroupdom.SecurityGroup]
	case isSecurityGroupPending(resource):
		delegate = frameworkbackend.BypassDelegated[*securitygroupdom.SecurityGroup]
	case isSecurityGroupCreating(resource):
		delegate = h.plugin.Create
	case wantSecurityGroupDelete(resource):
		delegate = frameworkbackend.BypassDelegated[*securitygroupdom.SecurityGroup]
	case isSecurityGroupDeleting(resource):
		delegate = h.plugin.Delete
	case wantSecurityGroupRetryCreate(resource):
		delegate = frameworkbackend.BypassDelegated[*securitygroupdom.SecurityGroup]
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
	case isSecurityGroupAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending, false)
	case isSecurityGroupPending(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	case isSecurityGroupCreating(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive, false)
	case wantSecurityGroupDelete(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting, true)
	case isSecurityGroupDeleting(resource):
		return false, nil
	case wantSecurityGroupRetryCreate(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *SecurityGroupPluginHandler) setResourceState(ctx context.Context, resource *securitygroupdom.SecurityGroup, state commondomain.ResourceState, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &securitygroupdom.SecurityGroupStatus{}
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

func (h *SecurityGroupPluginHandler) setResourceErrorState(ctx context.Context, resource *securitygroupdom.SecurityGroup, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &securitygroupdom.SecurityGroupStatus{}
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

// persistStatus writes the resource's status subresource. It is handed to the shared
// update helper, which owns the decision of when a write is warranted.
func (h *SecurityGroupPluginHandler) persistStatus(ctx context.Context, resource *securitygroupdom.SecurityGroup) error {
	_, err := h.repo.UpdateStatus(ctx, resource)

	return err
}

func isSecurityGroupActive(resource *securitygroupdom.SecurityGroup) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateActive
}

func isSecurityGroupAccepted(resource *securitygroupdom.SecurityGroup) bool {
	return resource.Status == nil
}

func isSecurityGroupPending(resource *securitygroupdom.SecurityGroup) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == commondomain.ResourceStatePending)
}

func isSecurityGroupCreating(resource *securitygroupdom.SecurityGroup) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateCreating
}

func securityGroupIsNotDeleting(resource *securitygroupdom.SecurityGroup) bool {
	return resource.Status == nil ||
		resource.Status.State != commondomain.ResourceStateDeleting
}

func wantSecurityGroupDelete(resource *securitygroupdom.SecurityGroup) bool {
	return resource.DeletedAt != nil && securityGroupIsNotDeleting(resource)
}

func isSecurityGroupDeleting(resource *securitygroupdom.SecurityGroup) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateDeleting
}

func wantSecurityGroupRetryCreate(resource *securitygroupdom.SecurityGroup) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[1].State == commondomain.ResourceStateCreating
}
