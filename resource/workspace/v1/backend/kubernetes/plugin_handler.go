package kubernetes

import (
	"context"
	"errors"
	"log"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"

	frameworkbackend "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// WorkspacePluginHandler drives the workspace reconciliation state machine.
type WorkspacePluginHandler struct {
	frameworkbackend.GenericPluginHandler[*wsdom.Workspace]
	repo   persistence.Repo[*wsdom.Workspace]
	plugin WorkspacePlugin
}

var _ backendport.PluginHandler[*wsdom.Workspace] = (*WorkspacePluginHandler)(nil)

// NewWorkspacePluginHandler creates a new WorkspacePluginHandler.
func NewWorkspacePluginHandler(
	repo persistence.Repo[*wsdom.Workspace],
	plugin WorkspacePlugin,
	maxConditions int,
) *WorkspacePluginHandler {
	handler := &WorkspacePluginHandler{
		repo:   repo,
		plugin: plugin,
	}
	handler.MaxConditions = maxConditions

	return handler
}

func (h *WorkspacePluginHandler) HandleReconcile(ctx context.Context, resource *wsdom.Workspace) (bool, error) {
	var delegate backendport.DelegatedFunc[*wsdom.Workspace]

	switch {

	case isWorkspaceAccepted(resource):
		delegate = frameworkbackend.BypassDelegated[*wsdom.Workspace]

	case isWorkspacePending(resource):
		delegate = frameworkbackend.BypassDelegated[*wsdom.Workspace]

	case isWorkspaceCreating(resource):
		delegate = h.plugin.Create

	case wantWorkspaceDelete(resource):
		delegate = frameworkbackend.BypassDelegated[*wsdom.Workspace]

	case isWorkspaceDeleting(resource):
		delegate = h.plugin.Delete

	case wantWorkspaceRetryCreate(resource):
		delegate = frameworkbackend.BypassDelegated[*wsdom.Workspace]

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

	case isWorkspaceAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending, false)

	case isWorkspacePending(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)

	case isWorkspaceCreating(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive, false)

	case wantWorkspaceDelete(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting, true)

	case isWorkspaceDeleting(resource):
		// Nothing to do: the controller will remove the finalizers to end the deletion process.
		return false, nil

	case wantWorkspaceRetryCreate(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)

	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *WorkspacePluginHandler) setResourceState(ctx context.Context, resource *wsdom.Workspace, state commondomain.ResourceState, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &wsdom.WorkspaceStatus{}
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

func (h *WorkspacePluginHandler) setResourceErrorState(ctx context.Context, resource *wsdom.Workspace, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &wsdom.WorkspaceStatus{}
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

func isWorkspaceAccepted(resource *wsdom.Workspace) bool {
	return resource.Status == nil
}

func isWorkspacePending(resource *wsdom.Workspace) bool {
	return resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStatePending
}

func isWorkspaceCreating(resource *wsdom.Workspace) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateCreating
}

func workspaceIsNotDeleting(resource *wsdom.Workspace) bool {
	return resource.Status == nil ||
		resource.Status.State != commondomain.ResourceStateDeleting
}

func wantWorkspaceDelete(resource *wsdom.Workspace) bool {
	return resource.DeletedAt != nil && workspaceIsNotDeleting(resource)
}

func isWorkspaceDeleting(resource *wsdom.Workspace) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateDeleting
}

func wantWorkspaceRetryCreate(resource *wsdom.Workspace) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == commondomain.ResourceStateCreating
}
