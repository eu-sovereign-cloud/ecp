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
		log.Println("-->DETECT isWorkspaceAccepted", "resource", resource)
		delegate = BypassDelegated[*regional.WorkspaceDomain]
	case isWorkspacePending(resource):
		log.Println("-->DETECT isWorkspacePending", "resource", resource)
		delegate = BypassDelegated[*regional.WorkspaceDomain]

	case isWorkspaceCreating(resource):
		log.Println("-->DETECT isWorkspaceCreating", "resource", resource)
		delegate = h.plugin.Create

	case wantWorkspaceDelete(resource):
		log.Println("-->DETECT wantWorkspaceDelete (K8s deletion started)", "resource", resource)
		// Transition to deleting state before calling plugin.Delete
		delegate = BypassDelegated[*regional.WorkspaceDomain]

	case isWorkspaceDeleting(resource):
		log.Println("-->DETECT isWorkspaceDeleting", "resource", resource)
		delegate = h.plugin.Delete

	case wantWorkspaceRetryCreate(resource):
		log.Println("-->DETECT wantWorkspaceRetryCreate", "resource", resource)
		delegate = BypassDelegated[*regional.WorkspaceDomain]

	default:
		log.Println("-->DETECT default", "resource", resource)
		return false, nil // Nothing to do.
	}

	if err := delegate(ctx, resource); err != nil {
		if errors.Is(err, delegator.ErrStillProcessing) {
			log.Println("-->DETECT still processing", "resource", resource)
			return true, nil
		}
		log.Println("-->DETECT error from delegate", "error", err, "resource", resource)
		if requeue, err := h.setResourceErrorState(ctx, resource, err, false); err != nil {
			return requeue, err // TODO: better errors handling
		}

		return true, nil
	}

	switch {

	case isWorkspaceAccepted(resource):
		log.Println("-->REACT isWorskspaceAccepted", "resource", resource)
		return h.setResourceState(ctx, resource, regional.ResourceStatePending, false)

	case isWorkspacePending(resource):
		log.Println("-->REACT isWorkspacePending", "resource", resource)
		return h.setResourceState(ctx, resource, regional.ResourceStateCreating, true)

	case isWorkspaceCreating(resource):
		log.Println("-->REACT isWorkspaceCreating", "resource", resource)
		return h.setResourceState(ctx, resource, regional.ResourceStateActive, false)

	case wantWorkspaceDelete(resource):
		log.Println("-->REACT wantWorkspaceDelete (setting state to Deleting)", "resource", resource)
		return h.setResourceState(ctx, resource, regional.ResourceStateDeleting, true)

	case isWorkspaceDeleting(resource):
		log.Println("-->REACT isWorkspaceDeleting (cleanup done, controller will remove finalizer)")
		return false, nil // Let the controller handle finalizer removal and actual deletion from K8s

	case wantWorkspaceRetryCreate(resource):
		log.Println("-->REACT wantWorkspaceRetryCreate", "resource", resource)
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
	resource.Status.State = &state

	if resource.Status.Conditions == nil {
		resource.Status.Conditions = []regional.StatusConditionDomain{}
	}

	resource.Status.Conditions = append(resource.Status.Conditions, conditionFromState(state))

	if _, err := h.repo.Update(ctx, resource); err != nil {
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
	state := regional.ResourceStateError
	resource.Status.State = &state

	if resource.Status.Conditions == nil {
		resource.Status.Conditions = []regional.StatusConditionDomain{}
	}

	resource.Status.Conditions = append(resource.Status.Conditions, conditionFromError(err))

	if _, err := h.repo.Update(ctx, resource); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return false, nil
		}

		return requeue, err
	}

	return requeue, nil
}

func isWorkspaceAccepted(resource *regional.WorkspaceDomain) bool {
	return resource.Status == nil || resource.Status.State == nil
}

func isWorkspacePending(resource *regional.WorkspaceDomain) bool {
	return resource.Status != nil && resource.Status.State != nil && *(resource.Status.State) == regional.ResourceStatePending
}

func isWorkspaceCreating(resource *regional.WorkspaceDomain) bool {
	return resource.DeletedAt == nil && resource.Status != nil && resource.Status.State != nil && *(resource.Status.State) == regional.ResourceStateCreating
}

func wantWorkspaceDelete(resource *regional.WorkspaceDomain) bool {
	return resource.DeletedAt != nil && (resource.Status == nil || resource.Status.State == nil || *(resource.Status.State) != regional.ResourceStateDeleting)
}

func isWorkspaceDeleting(resource *regional.WorkspaceDomain) bool {
	return resource.DeletedAt != nil && resource.Status != nil && resource.Status.State != nil && *(resource.Status.State) == regional.ResourceStateDeleting
}

func wantWorkspaceRetryCreate(resource *regional.WorkspaceDomain) bool {
	return resource.DeletedAt == nil && resource.Status != nil && resource.Status.State != nil &&
		*(resource.Status.State) == regional.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == regional.ResourceStateCreating
}
