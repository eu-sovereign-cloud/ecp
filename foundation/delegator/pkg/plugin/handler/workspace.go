package handler

import (
	"context"
	"errors"
	"log"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	gateway "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	delegator "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

type WorkspacePluginHandler struct {
	GenericPluginHandler[*regional.WorkspaceDomain]
	repo   gateway.Repo[*regional.WorkspaceDomain]
	plugin plugin.Workspace
}

var _ delegator.PluginHandler[*regional.WorkspaceDomain] = (*WorkspacePluginHandler)(nil)

func NewWorkspacePluginHandler(
	repo gateway.Repo[*regional.WorkspaceDomain],
	plugin plugin.Workspace,
) *WorkspacePluginHandler {
	handler := &WorkspacePluginHandler{
		repo:   repo,
		plugin: plugin,
	}

	return handler
}

func (h *WorkspacePluginHandler) HandleReconcile(ctx context.Context, resource *regional.WorkspaceDomain) (bool, error) {
	var delegate delegator.DelegatedFunc[*regional.WorkspaceDomain]

	switch {

	case isWorkspaceAccepted(resource):
		delegate = BypassDelegated[*regional.WorkspaceDomain]

	case isWorkspacePending(resource):
		delegate = BypassDelegated[*regional.WorkspaceDomain]

	case isWorkspaceCreating(resource):
		delegate = h.plugin.Create

	case wantWorkspaceDelete(resource):
		delegate = BypassDelegated[*regional.WorkspaceDomain]

	case isWorkspaceDeleting(resource):
		delegate = h.plugin.Delete

	case wantWorkspaceRetryCreate(resource):
		delegate = BypassDelegated[*regional.WorkspaceDomain]

	default:
		return false, nil // Nothing to do.
	}

	if err := delegate(ctx, resource); err != nil {
		if errors.Is(err, delegator.ErrStillProcessing) {
			return true, nil
		}

		if requeue, err := h.setResourceErrorState(ctx, resource, err, false); err != nil {
			return requeue, err // TODO: better errors handling
		}

		return true, nil
	}

	switch {

	case isWorkspaceAccepted(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStatePending, false)

	case isWorkspacePending(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateCreating, true)

	case isWorkspaceCreating(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateActive, false)

	case wantWorkspaceDelete(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateDeleting, true)

	case isWorkspaceDeleting(resource):
		// Nothing to do: the delegator controller will remove the finalizers
		// in order to end the deletion process.
		return false, nil

	case wantWorkspaceRetryCreate(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateCreating, true)

	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *WorkspacePluginHandler) setResourceState(ctx context.Context, resource *regional.WorkspaceDomain, state regional.ResourceStateDomain, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &regional.WorkspaceStatusDomain{}
	}
	resource.Status.State = state

	if resource.Status.Conditions == nil {
		resource.Status.Conditions = []regional.StatusConditionDomain{}
	}

	resource.Status.Conditions = append(resource.Status.Conditions, conditionFromState(state))

	if _, err := h.repo.UpdateStatus(ctx, resource); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return false, nil
		}

		return requeue, err
	}

	return requeue, nil
}

func (h *WorkspacePluginHandler) setResourceErrorState(ctx context.Context, resource *regional.WorkspaceDomain, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &regional.WorkspaceStatusDomain{}
	}

	resource.Status.State = regional.ResourceStateError

	if resource.Status.Conditions == nil {
		resource.Status.Conditions = []regional.StatusConditionDomain{}
	}

	resource.Status.Conditions = append(resource.Status.Conditions, conditionFromError(err))

	if _, err := h.repo.UpdateStatus(ctx, resource); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return false, nil
		}

		return requeue, err
	}

	return requeue, nil
}

func isWorkspaceAccepted(resource *regional.WorkspaceDomain) bool {
	return resource.Status == nil
}

func isWorkspacePending(resource *regional.WorkspaceDomain) bool {
	return resource.Status != nil &&
		resource.Status.State == regional.ResourceStatePending
}

func isWorkspaceCreating(resource *regional.WorkspaceDomain) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == regional.ResourceStateCreating
}

func workspaceIsNotDeleting(resource *regional.WorkspaceDomain) bool {
	return resource.Status == nil ||
		resource.Status.State != regional.ResourceStateDeleting
}

func wantWorkspaceDelete(resource *regional.WorkspaceDomain) bool {
	return resource.DeletedAt != nil && workspaceIsNotDeleting(resource)
}

func isWorkspaceDeleting(resource *regional.WorkspaceDomain) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == regional.ResourceStateDeleting
}

func wantWorkspaceRetryCreate(resource *regional.WorkspaceDomain) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == regional.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == regional.ResourceStateCreating
}
