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
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/network/v1"
)

// NetworkPluginHandler drives the network reconciliation state machine.
type NetworkPluginHandler struct {
	frameworkbackend.GenericPluginHandler[*netdom.Network]
	repo   persistence.Repo[*netdom.Network]
	plugin NetworkPlugin
}

var _ backendport.PluginHandler[*netdom.Network] = (*NetworkPluginHandler)(nil)

// NewNetworkPluginHandler creates a new NetworkPluginHandler.
func NewNetworkPluginHandler(
	repo persistence.Repo[*netdom.Network],
	plugin NetworkPlugin,
	maxConditions int,
) *NetworkPluginHandler {
	handler := &NetworkPluginHandler{
		repo:   repo,
		plugin: plugin,
	}
	handler.MaxConditions = maxConditions

	return handler
}

func (h *NetworkPluginHandler) HandleReconcile(ctx context.Context, resource *netdom.Network) (bool, error) {
	var delegate backendport.DelegatedFunc[*netdom.Network]

	switch {

	case isNetworkAccepted(resource):
		delegate = frameworkbackend.BypassDelegated[*netdom.Network]

	case isNetworkPending(resource):
		delegate = frameworkbackend.BypassDelegated[*netdom.Network]

	case isNetworkCreating(resource):
		delegate = h.plugin.Create

	case wantNetworkDelete(resource):
		delegate = frameworkbackend.BypassDelegated[*netdom.Network]

	case isNetworkDeleting(resource):
		delegate = h.plugin.Delete

	case wantNetworkRetryCreate(resource):
		delegate = frameworkbackend.BypassDelegated[*netdom.Network]

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

	case isNetworkAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending, false)

	case isNetworkPending(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)

	case isNetworkCreating(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive, false)

	case wantNetworkDelete(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting, true)

	case isNetworkDeleting(resource):
		// Nothing to do: the controller will remove the finalizers to end the deletion process.
		return false, nil

	case wantNetworkRetryCreate(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)

	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *NetworkPluginHandler) setResourceState(ctx context.Context, resource *netdom.Network, state commondomain.ResourceState, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &netdom.NetworkStatus{}
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

func (h *NetworkPluginHandler) setResourceErrorState(ctx context.Context, resource *netdom.Network, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &netdom.NetworkStatus{}
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

func isNetworkAccepted(resource *netdom.Network) bool {
	return resource.Status == nil
}

func isNetworkPending(resource *netdom.Network) bool {
	return resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStatePending
}

func isNetworkCreating(resource *netdom.Network) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateCreating
}

func networkIsNotDeleting(resource *netdom.Network) bool {
	return resource.Status == nil ||
		resource.Status.State != commondomain.ResourceStateDeleting
}

func wantNetworkDelete(resource *netdom.Network) bool {
	return resource.DeletedAt != nil && networkIsNotDeleting(resource)
}

func isNetworkDeleting(resource *netdom.Network) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateDeleting
}

func wantNetworkRetryCreate(resource *netdom.Network) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == commondomain.ResourceStateCreating
}
