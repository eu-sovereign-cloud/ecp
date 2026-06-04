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

type NetworkPluginHandler struct {
	GenericPluginHandler[*regional.NetworkDomain]
	repo   gateway.Repo[*regional.NetworkDomain]
	plugin plugin.Network
}

var _ delegator.PluginHandler[*regional.NetworkDomain] = (*NetworkPluginHandler)(nil)

func NewNetworkPluginHandler(
	repo gateway.Repo[*regional.NetworkDomain],
	plugin plugin.Network,
) *NetworkPluginHandler {
	handler := &NetworkPluginHandler{
		repo:   repo,
		plugin: plugin,
	}

	return handler
}

func (h *NetworkPluginHandler) HandleReconcile(ctx context.Context, resource *regional.NetworkDomain) (bool, error) {
	var delegate delegator.DelegatedFunc[*regional.NetworkDomain]

	switch {

	case isNetworkAccepted(resource):
		delegate = BypassDelegated[*regional.NetworkDomain]

	case isNetworkPending(resource):
		delegate = BypassDelegated[*regional.NetworkDomain]

	case isNetworkCreating(resource):
		delegate = h.plugin.Create

	case wantNetworkDelete(resource):
		delegate = BypassDelegated[*regional.NetworkDomain]

	case isNetworkDeleting(resource):
		delegate = h.plugin.Delete

	case wantNetworkRetryCreate(resource):
		delegate = BypassDelegated[*regional.NetworkDomain]

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

	case isNetworkAccepted(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStatePending, false)

	case isNetworkPending(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateCreating, true)

	case isNetworkCreating(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateActive, false)

	case wantNetworkDelete(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateDeleting, true)

	case isNetworkDeleting(resource):
		// Nothing to do: the delegator controller will remove the finalizers
		// in order to end the deletion process.
		return false, nil

	case wantNetworkRetryCreate(resource):
		return h.setResourceState(ctx, resource, regional.ResourceStateCreating, true)

	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *NetworkPluginHandler) setResourceState(ctx context.Context, resource *regional.NetworkDomain, state regional.ResourceStateDomain, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &regional.NetworkStatusDomain{}
	}

	resource.Status.PushCondition(conditionFromState(state))
	for h.MaxConditions > 0 && len(resource.Status.Conditions) > h.MaxConditions {
		resource.Status.PopCondition()
	}

	if _, err := h.repo.UpdateStatus(ctx, resource); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return false, nil
		}

		return requeue, err
	}

	return requeue, nil
}

func (h *NetworkPluginHandler) setResourceErrorState(ctx context.Context, resource *regional.NetworkDomain, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &regional.NetworkStatusDomain{}
	}

	resource.Status.PushCondition(conditionFromError(err))
	for h.MaxConditions > 0 && len(resource.Status.Conditions) > h.MaxConditions {
		resource.Status.PopCondition()
	}

	if _, err := h.repo.UpdateStatus(ctx, resource); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return false, nil
		}

		return requeue, err
	}

	return requeue, nil
}

func isNetworkAccepted(resource *regional.NetworkDomain) bool {
	return resource.Status == nil
}

func isNetworkPending(resource *regional.NetworkDomain) bool {
	return resource.Status != nil &&
		resource.Status.State == regional.ResourceStatePending
}

func isNetworkCreating(resource *regional.NetworkDomain) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == regional.ResourceStateCreating
}

func networkIsNotDeleting(resource *regional.NetworkDomain) bool {
	return resource.Status == nil ||
		resource.Status.State != regional.ResourceStateDeleting
}

func wantNetworkDelete(resource *regional.NetworkDomain) bool {
	return resource.DeletedAt != nil && networkIsNotDeleting(resource)
}

func isNetworkDeleting(resource *regional.NetworkDomain) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == regional.ResourceStateDeleting
}

func wantNetworkRetryCreate(resource *regional.NetworkDomain) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == regional.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == regional.ResourceStateCreating
}
