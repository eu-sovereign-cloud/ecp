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

func (h *RouteTablePluginHandler) HandleReconcile(ctx context.Context, resource *routetabledom.RouteTable) (bool, error) {
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
	case isRouteTableAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending, false)
	case isRouteTablePending(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	case isRouteTableCreating(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive, false)
	case wantRouteTableDelete(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting, true)
	case isRouteTableDeleting(resource):
		return false, nil
	case wantRouteTableRetryCreate(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *RouteTablePluginHandler) setResourceState(ctx context.Context, resource *routetabledom.RouteTable, state commondomain.ResourceState, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &routetabledom.RouteTableStatus{}
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

func (h *RouteTablePluginHandler) setResourceErrorState(ctx context.Context, resource *routetabledom.RouteTable, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &routetabledom.RouteTableStatus{}
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
