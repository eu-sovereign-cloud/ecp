package kubernetes

import (
	"context"
	"errors"
	"log"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"

	frameworkbackend "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
)

// RouteTablePluginHandler drives the RouteTable reconciliation state machine.
type RouteTablePluginHandler struct {
	frameworkbackend.GenericPluginHandler[*routetabledom.RouteTable]
	repo   persistence.Repo[*routetabledom.RouteTable]
	plugin RouteTablePlugin
}

var _ backendport.PluginHandler[*routetabledom.RouteTable] = (*RouteTablePluginHandler)(nil)

// NewRouteTablePluginHandler creates a new RouteTablePluginHandler.
func NewRouteTablePluginHandler(
	repo persistence.Repo[*routetabledom.RouteTable],
	plugin RouteTablePlugin,
	maxConditions int,
) *RouteTablePluginHandler {
	handler := &RouteTablePluginHandler{
		repo:   repo,
		plugin: plugin,
	}
	handler.MaxConditions = maxConditions

	return handler
}

func (h *RouteTablePluginHandler) HandleReconcile(ctx context.Context, resource *routetabledom.RouteTable) error {
	// An active resource has no lifecycle transition left to make, so it takes the update
	// path instead of the create/delete state machine below. See commonbackend.HandleUpdate.
	if isRouteTableActive(resource) {
		return commonbackend.HandleUpdate(ctx, resource, &resource.Status.Status, h.plugin.Update, h.repo, h.MaxConditions)
	}

	var delegate backendport.DelegatedFunc[*routetabledom.RouteTable]

	switch {
	case isRouteTableAccepted(resource):
		delegate = frameworkbackend.BypassDelegated[*routetabledom.RouteTable]
	case isRouteTablePending(resource):
		delegate = frameworkbackend.BypassDelegated[*routetabledom.RouteTable]
	case isRouteTableCreating(resource):
		delegate = h.plugin.Create
	case wantRouteTableDelete(resource):
		delegate = frameworkbackend.BypassDelegated[*routetabledom.RouteTable]
	case isRouteTableDeleting(resource):
		delegate = h.plugin.Delete
	case wantRouteTableRetryCreate(resource):
		delegate = frameworkbackend.BypassDelegated[*routetabledom.RouteTable]
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
	case isRouteTableAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending)
	case isRouteTablePending(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))
	case isRouteTableCreating(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive)
	case wantRouteTableDelete(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting))
	case isRouteTableDeleting(resource):
		return nil
	case wantRouteTableRetryCreate(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))
	default:
		log.Fatal("must never achieve that condition")
	}

	return nil
}

func (h *RouteTablePluginHandler) setResourceState(ctx context.Context, resource *routetabledom.RouteTable, state commondomain.ResourceState) error {
	if resource.Status == nil {
		resource.Status = &routetabledom.RouteTableStatus{}
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

func (h *RouteTablePluginHandler) setResourceErrorState(ctx context.Context, resource *routetabledom.RouteTable, err error) error {
	if resource.Status == nil {
		resource.Status = &routetabledom.RouteTableStatus{}
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

func isRouteTableActive(resource *routetabledom.RouteTable) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateActive
}

func isRouteTableAccepted(resource *routetabledom.RouteTable) bool {
	return resource.Status == nil
}

func isRouteTablePending(resource *routetabledom.RouteTable) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == commondomain.ResourceStatePending)
}

func isRouteTableCreating(resource *routetabledom.RouteTable) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateCreating
}

func routeTableIsNotDeleting(resource *routetabledom.RouteTable) bool {
	return resource.Status == nil ||
		resource.Status.State != commondomain.ResourceStateDeleting
}

func wantRouteTableDelete(resource *routetabledom.RouteTable) bool {
	return resource.DeletedAt != nil && routeTableIsNotDeleting(resource)
}

func isRouteTableDeleting(resource *routetabledom.RouteTable) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateDeleting
}

func wantRouteTableRetryCreate(resource *routetabledom.RouteTable) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[1].State == commondomain.ResourceStateCreating
}
