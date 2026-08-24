package kubernetes

import (
	"context"
	"errors"
	"log"

	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"

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

func (h *WorkspacePluginHandler) HandleReconcile(ctx context.Context, resource *wsdom.Workspace) error {
	// An active resource has no lifecycle transition left to make, so it takes the update
	// path instead of the create/delete state machine below. See commonbackend.HandleUpdate.
	if isWorkspaceActive(resource) {
		return commonbackend.HandleUpdate(ctx, resource, &resource.Status.Status, h.plugin.Update, h.repo, h.MaxConditions)
	}

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

	case isWorkspaceAccepted(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStatePending))

	case isWorkspacePending(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))

	case isWorkspaceCreating(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStateActive))

	case wantWorkspaceDelete(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting))

	case isWorkspaceDeleting(resource):
		// Nothing to do: the controller will remove the finalizers to end the deletion process.
		return nil

	case wantWorkspaceRetryCreate(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))

	default:
		log.Fatal("must never achieve that condition")
	}

	return nil
}

func (h *WorkspacePluginHandler) setResourceState(ctx context.Context, resource *wsdom.Workspace, state commondomain.ResourceState) error {
	if resource.Status == nil {
		resource.Status = &wsdom.WorkspaceStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromState(state))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, err := h.repo.UpdateStatus(ctx, resource)

	return err
}

func (h *WorkspacePluginHandler) setResourceErrorState(ctx context.Context, resource *wsdom.Workspace, err error) error {
	if resource.Status == nil {
		resource.Status = &wsdom.WorkspaceStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromError(err))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, updateErr := h.repo.UpdateStatus(ctx, resource)

	return updateErr
}

func isWorkspaceActive(resource *wsdom.Workspace) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateActive
}

func isWorkspaceAccepted(resource *wsdom.Workspace) bool {
	return resource.Status == nil
}

func isWorkspacePending(resource *wsdom.Workspace) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == commondomain.ResourceStatePending)
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
		resource.Status.Conditions[1].State == commondomain.ResourceStateCreating
}
