package kubernetes

import (
	"context"
	"errors"
	"log"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"

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
func (h *RolePluginHandler) HandleReconcile(ctx context.Context, resource *roledom.Role) (bool, error) {
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

	case isRoleAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending, false)

	case isRolePending(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)

	case isRoleCreating(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive, false)

	case wantRoleDelete(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting, true)

	case isRoleDeleting(resource):
		// Nothing to do: the controller will remove the finalizers to end the deletion process.
		return false, nil

	case wantRoleRetryCreate(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)

	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *RolePluginHandler) setResourceState(ctx context.Context, resource *roledom.Role, state commondomain.ResourceState, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &roledom.RoleStatus{}
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

func (h *RolePluginHandler) setResourceErrorState(ctx context.Context, resource *roledom.Role, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &roledom.RoleStatus{}
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
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == commondomain.ResourceStateCreating
}
