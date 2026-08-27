package kubernetes

import (
	"context"
	"errors"
	"fmt"

	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"

	frameworkbackend "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// RolePluginHandler drives the role reconciliation state machine.
type RolePluginHandler struct {
	frameworkbackend.GenericPluginHandler[*roledom.Role]
	repo   persistence.Repo[*roledom.Role]
	plugin RolePlugin
}

var _ backendport.PluginHandler[*roledom.Role] = (*RolePluginHandler)(nil)

// NewRolePluginHandler creates a new RolePluginHandler.
func NewRolePluginHandler(
	repo persistence.Repo[*roledom.Role],
	plugin RolePlugin,
	maxConditions int,
) *RolePluginHandler {
	handler := &RolePluginHandler{
		repo:   repo,
		plugin: plugin,
	}
	handler.MaxConditions = maxConditions

	return handler
}

// HandleReconcile implements the role lifecycle state machine.
func (h *RolePluginHandler) HandleReconcile(ctx context.Context, resource *roledom.Role) error {
	// An active resource has no lifecycle transition left to make, so it takes the update
	// path instead of the create/delete state machine below. See commonbackend.HandleUpdate.
	if isRoleActive(resource) {
		return commonbackend.HandleUpdate(ctx, resource, &resource.Status.Status, h.plugin.Update, h.repo, h.MaxConditions)
	}

	var delegate backendport.DelegatedFunc[*roledom.Role]

	switch {

	case isRoleAccepted(resource):
		delegate = frameworkbackend.BypassDelegated[*roledom.Role]

	case isRolePending(resource):
		delegate = frameworkbackend.BypassDelegated[*roledom.Role]

	case isRoleCreating(resource):
		delegate = h.plugin.Create

	case wantRoleDelete(resource):
		delegate = frameworkbackend.BypassDelegated[*roledom.Role]

	case isRoleDeleting(resource):
		delegate = h.plugin.Delete

	case wantRoleRetryCreate(resource):
		delegate = frameworkbackend.BypassDelegated[*roledom.Role]

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

	case isRoleAccepted(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStatePending))

	case isRolePending(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))

	case isRoleCreating(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStateActive))

	case wantRoleDelete(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting))

	case isRoleDeleting(resource):
		// Nothing to do: the controller will remove the finalizers to end the deletion process.
		return nil

	case wantRoleRetryCreate(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))

	default:
		return fmt.Errorf("unreachable reconcile state for role %q", resource.GetName())
	}
}

func (h *RolePluginHandler) setResourceState(ctx context.Context, resource *roledom.Role, state commondomain.ResourceState) error {
	if resource.Status == nil {
		resource.Status = &roledom.RoleStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromState(state))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, err := h.repo.UpdateStatus(ctx, resource)

	return err
}

func (h *RolePluginHandler) setResourceErrorState(ctx context.Context, resource *roledom.Role, err error) error {
	if resource.Status == nil {
		resource.Status = &roledom.RoleStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromError(err))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, updateErr := h.repo.UpdateStatus(ctx, resource)

	return updateErr
}

func isRoleActive(resource *roledom.Role) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateActive
}

func isRoleAccepted(resource *roledom.Role) bool {
	return resource.Status == nil
}

func isRolePending(resource *roledom.Role) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == commondomain.ResourceStatePending)
}

func isRoleCreating(resource *roledom.Role) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateCreating
}

func roleIsNotDeleting(resource *roledom.Role) bool {
	return resource.Status == nil ||
		resource.Status.State != commondomain.ResourceStateDeleting
}

func wantRoleDelete(resource *roledom.Role) bool {
	return resource.DeletedAt != nil && roleIsNotDeleting(resource)
}

func isRoleDeleting(resource *roledom.Role) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateDeleting
}

func wantRoleRetryCreate(resource *roledom.Role) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[1].State == commondomain.ResourceStateCreating
}
