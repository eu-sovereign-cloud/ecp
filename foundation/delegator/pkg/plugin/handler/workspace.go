package handler

import (
	"context"
	"errors"
	"log"

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
	case isWorkspacePending(resource):
		delegate = BypassDelegated[*regional.WorkspaceDomain]

	case wantWorkspaceCreate(resource):
		delegate = h.plugin.Create

	case wantWorkspaceDelete(resource):
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
		if err := h.setResourceErrorState(ctx, resource, err); err != nil {
			return false, err // TODO: better errors handling
		}

		return true, nil
	}

	switch {
	case isWorkspacePending(resource):
		return true, h.setResourceState(ctx, resource, regional.ResourceStateCreating)

	case wantWorkspaceCreate(resource):
		return false, h.setResourceState(ctx, resource, regional.ResourceStateActive)

	case wantWorkspaceDelete(resource):
		return false, h.repo.Delete(ctx, resource)

	case wantWorkspaceRetryCreate(resource):
		return true, h.setResourceState(ctx, resource, regional.ResourceStateCreating)

	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *WorkspacePluginHandler) setResourceState(ctx context.Context, resource *regional.WorkspaceDomain, state regional.ResourceStateDomain) error {
	if resource.Status == nil {
		resource.Status = &regional.WorkspaceStatusDomain{}
	}
	resource.Status.State = &state

	if resource.Status.Conditions == nil {
		resource.Status.Conditions = []regional.StatusConditionDomain{}
	}

	resource.Status.Conditions = append(resource.Status.Conditions, conditionFromState(state))

	if _, err := h.repo.Update(ctx, resource); err != nil {
		return err
	}

	return nil
}

func (h *WorkspacePluginHandler) setResourceErrorState(ctx context.Context, resource *regional.WorkspaceDomain, err error) error {
	if resource.Status == nil {
		resource.Status = &regional.WorkspaceStatusDomain{}
	}
	state := regional.ResourceStateError
	resource.Status.State = &state

	if resource.Status.Conditions == nil {
		resource.Status.Conditions = []regional.StatusConditionDomain{}
	}

	resource.Status.Conditions = append(resource.Status.Conditions, conditionFromError(err))

	if _, err := h.repo.Update(ctx, resource); err != nil {
		return err
	}

	return nil
}

func isWorkspacePending(resource *regional.WorkspaceDomain) bool {
	return resource.Status == nil || resource.Status.State == nil || *(resource.Status.State) == regional.ResourceStatePending
}

func wantWorkspaceCreate(resource *regional.WorkspaceDomain) bool {
	return resource.Status != nil && *(resource.Status.State) == regional.ResourceStateCreating
}

func wantWorkspaceDelete(resource *regional.WorkspaceDomain) bool {
	return resource.DeletedAt == nil && resource.Status != nil && resource.Status.State != nil && *(resource.Status.State) == regional.ResourceStateDeleting
}

func wantWorkspaceRetryCreate(resource *regional.WorkspaceDomain) bool {
	return resource.Status != nil && *(resource.Status.State) == regional.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == regional.ResourceStateCreating
}
