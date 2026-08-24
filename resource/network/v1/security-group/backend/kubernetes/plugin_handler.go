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

func (h *SecurityGroupPluginHandler) HandleReconcile(ctx context.Context, resource *securitygroupdom.SecurityGroup) error {
	// An active resource has no lifecycle transition left to make, so it takes the update
	// path instead of the create/delete state machine below. See commonbackend.HandleUpdate.
	if isSecurityGroupActive(resource) {
		return commonbackend.HandleUpdate(ctx, resource, &resource.Status.Status, h.plugin.Update, h.repo, h.MaxConditions)
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
		return nil // Nothing to do.
	}

	if err := delegate(ctx, resource); err != nil {
		var rq backendport.RequeueError
		if errors.As(err, &rq) {
			// Not a failure, and not ours to reinterpret: the plugin named its own cadence.
			return err
		}

		if stateErr := h.setResourceErrorState(ctx, resource, err); stateErr != nil {
			return stateErr
		}

		// The failure is recorded on the resource; retry it on the next pass.
		return backendport.StillProcessing
	}

	switch {
	case isSecurityGroupAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending)
	case isSecurityGroupPending(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))
	case isSecurityGroupCreating(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive)
	case wantSecurityGroupDelete(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting))
	case isSecurityGroupDeleting(resource):
		return nil
	case wantSecurityGroupRetryCreate(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))
	default:
		log.Fatal("must never achieve that condition")
	}

	return nil
}

func (h *SecurityGroupPluginHandler) setResourceState(ctx context.Context, resource *securitygroupdom.SecurityGroup, state commondomain.ResourceState) error {
	if resource.Status == nil {
		resource.Status = &securitygroupdom.SecurityGroupStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromState(state))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	if _, err := h.repo.UpdateStatus(ctx, resource); err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			// The resource is gone; there is nothing left to reconcile.
			return nil
		}
		return err
	}

	return nil
}

func (h *SecurityGroupPluginHandler) setResourceErrorState(ctx context.Context, resource *securitygroupdom.SecurityGroup, err error) error {
	if resource.Status == nil {
		resource.Status = &securitygroupdom.SecurityGroupStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromError(err))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	if _, updateErr := h.repo.UpdateStatus(ctx, resource); updateErr != nil {
		if errors.Is(updateErr, kernel.ErrNotFound) {
			// The resource is gone; there is nothing left to reconcile.
			return nil
		}
		return updateErr
	}

	return nil
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
