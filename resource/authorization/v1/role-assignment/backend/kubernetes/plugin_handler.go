package kubernetes

import (
	"context"
	"errors"
	"log"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"

	frameworkbackend "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
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

func (h *RoleAssignmentPluginHandler) HandleReconcile(ctx context.Context, resource *radom.RoleAssignment) error {
	// An active resource has no lifecycle transition left to make, so it takes the update
	// path instead of the create/delete state machine below. See commonbackend.HandleUpdate.
	if isRoleAssignmentActive(resource) {
		return commonbackend.HandleUpdate(ctx, resource, &resource.Status.Status, h.plugin.Update, h.repo, h.MaxConditions)
	}

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

	case isRoleAssignmentAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending)

	case isRoleAssignmentPending(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))

	case isRoleAssignmentCreating(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive)

	case wantRoleAssignmentDelete(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting))

	case isRoleAssignmentDeleting(resource):
		// Nothing to do: the controller will remove the finalizers to end the deletion process.
		return nil

	case wantRoleAssignmentRetryCreate(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))

	default:
		log.Fatal("must never achieve that condition")
	}

	return nil
}

func (h *RoleAssignmentPluginHandler) setResourceState(ctx context.Context, resource *radom.RoleAssignment, state commondomain.ResourceState) error {
	if resource.Status == nil {
		resource.Status = &radom.RoleAssignmentStatus{}
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

func (h *RoleAssignmentPluginHandler) setResourceErrorState(ctx context.Context, resource *radom.RoleAssignment, err error) error {
	if resource.Status == nil {
		resource.Status = &radom.RoleAssignmentStatus{}
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

func isRoleAssignmentActive(resource *radom.RoleAssignment) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateActive
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
		resource.Status.Conditions[1].State == commondomain.ResourceStateCreating
}
