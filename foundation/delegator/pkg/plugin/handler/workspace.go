package handler

import (
	"context"
	"log"
	"time"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	gateway_port "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	delegator_port "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

type WorkspacePluginHandler struct {
	GenericPluginHandler[*regional.WorkspaceDomain]
	repo   gateway_port.Repo[*regional.WorkspaceDomain]
	plugin plugin.Workspace
}

var _ delegator_port.PluginHandler[*regional.WorkspaceDomain] = (*WorkspacePluginHandler)(nil)

func NewWorkspacePluginHandler(
	repo gateway_port.Repo[*regional.WorkspaceDomain],
	plugin plugin.Workspace,
) *WorkspacePluginHandler {
	handler := &WorkspacePluginHandler{
		repo:   repo,
		plugin: plugin,
	}

	return handler
}

func (h *WorkspacePluginHandler) HandleReconcile(ctx context.Context, resource *regional.WorkspaceDomain) error {
	var delegate delegator_port.DelegatedFunc[*regional.WorkspaceDomain]

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
		return nil // Nothing to do.
	}

	if err := delegate(ctx, resource); err != nil {
		state := regional.ResourceStateError
		resource.Status.State = &state

		resource.Status.Conditions = append(resource.Status.Conditions, regional.StatusConditionDomain{
			LastTransitionAt: time.Now(),
			Message:          err.Error(),
			State:            state,
		})

		if _, err := h.repo.Update(ctx, resource); err != nil {
			return err
		}

		return nil
	}

	switch {
	case isWorkspacePending(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateCreating)

	case wantWorkspaceCreate(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateActive)

	case wantWorkspaceDelete(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateDeleting)

	case wantWorkspaceRetryCreate(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateCreating)

	default:
		log.Fatal("must never achieve that condition")
	}

	return nil
}

func (h *WorkspacePluginHandler) setResourceState(ctx context.Context, resource *regional.WorkspaceDomain, state regional.ResourceStateDomain) error {
	resource.Status.State = &state

	resource.Status.Conditions = append(resource.Status.Conditions, regional.StatusConditionDomain{
		LastTransitionAt: time.Now(),
		State:            state,
	})

	if _, err := h.repo.Update(ctx, resource); err != nil {
		return err
	}

	return nil
}

func isWorkspacePending(resource *regional.WorkspaceDomain) bool {
	return *(resource.Status.State) == regional.ResourceStatePending
}

func wantWorkspaceCreate(resource *regional.WorkspaceDomain) bool {
	return *(resource.Status.State) == regional.ResourceStateCreating
}

func wantWorkspaceDelete(resource *regional.WorkspaceDomain) bool {
	return *(resource.Status.State) == regional.ResourceStateDeleting
}

func wantWorkspaceRetryCreate(resource *regional.WorkspaceDomain) bool {
	return *(resource.Status.State) == regional.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == regional.ResourceStateCreating
}
