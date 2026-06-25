package kubernetes

import (
	"context"
	"errors"
	"log"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"

	frameworkbackend "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role-assignment/v1"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// RoleAssignmentPluginHandler drives the role assignment reconciliation state machine.
type RoleAssignmentPluginHandler struct {
	frameworkbackend.GenericPluginHandler[*radom.RoleAssignment]
	repo   persistence.Repo[*radom.RoleAssignment]
	plugin RoleAssignmentPlugin
}

var _ backendport.PluginHandler[*radom.RoleAssignment] = (*RoleAssignmentPluginHandler)(nil)

// NewRoleAssignmentPluginHandler creates a new RoleAssignmentPluginHandler.
func NewRoleAssignmentPluginHandler(
	repo persistence.Repo[*radom.RoleAssignment],
	plugin RoleAssignmentPlugin,
	maxConditions int,
) *RoleAssignmentPluginHandler {
	handler := &RoleAssignmentPluginHandler{
		repo:   repo,
		plugin: plugin,
	}
	handler.MaxConditions = maxConditions

	return handler
}

func (h *RoleAssignmentPluginHandler) HandleReconcile(ctx context.Context, resource *radom.RoleAssignment) (bool, error) {
	var delegate backendport.DelegatedFunc[*radom.RoleAssignment]

	switch {

	case isRoleAssignmentAccepted(resource):
		delegate = frameworkbackend.BypassDelegated[*radom.RoleAssignment]

	case isRoleAssignmentPending(resource):
		delegate = frameworkbackend.BypassDelegated[*radom.RoleAssignment]

	case isRoleAssignmentCreating(resource):
		delegate = h.plugin.Create

	case wantRoleAssignmentDelete(resource):
		delegate = frameworkbackend.BypassDelegated[*radom.RoleAssignment]

	case isRoleAssignmentDeleting(resource):
		delegate = h.plugin.Delete

	case wantRoleAssignmentRetryCreate(resource):
		delegate = frameworkbackend.BypassDelegated[*radom.RoleAssignment]

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

	case isRoleAssignmentAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending, false)

	case isRoleAssignmentPending(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)

	case isRoleAssignmentCreating(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive, false)

	case wantRoleAssignmentDelete(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting, true)

	case isRoleAssignmentDeleting(resource):
		// Nothing to do: the controller will remove the finalizers to end the deletion process.
		return false, nil

	case wantRoleAssignmentRetryCreate(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)

	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *RoleAssignmentPluginHandler) setResourceState(ctx context.Context, resource *radom.RoleAssignment, state commondomain.ResourceState, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &radom.RoleAssignmentStatus{}
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

func (h *RoleAssignmentPluginHandler) setResourceErrorState(ctx context.Context, resource *radom.RoleAssignment, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &radom.RoleAssignmentStatus{}
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

func isRoleAssignmentAccepted(resource *radom.RoleAssignment) bool {
	return resource.Status == nil
}

func isRoleAssignmentPending(resource *radom.RoleAssignment) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == commondomain.ResourceStatePending)
}

func isRoleAssignmentCreating(resource *radom.RoleAssignment) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateCreating
}

func roleAssignmentIsNotDeleting(resource *radom.RoleAssignment) bool {
	return resource.Status == nil ||
		resource.Status.State != commondomain.ResourceStateDeleting
}

func wantRoleAssignmentDelete(resource *radom.RoleAssignment) bool {
	return resource.DeletedAt != nil && roleAssignmentIsNotDeleting(resource)
}

func isRoleAssignmentDeleting(resource *radom.RoleAssignment) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateDeleting
}

func wantRoleAssignmentRetryCreate(resource *radom.RoleAssignment) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == commondomain.ResourceStateCreating
}
