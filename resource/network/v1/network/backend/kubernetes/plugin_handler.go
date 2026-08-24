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
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
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

func (h *NetworkPluginHandler) HandleReconcile(ctx context.Context, resource *netdom.Network) error {
	// An active resource has no lifecycle transition left to make, so it takes the update path
	// instead of the create/delete state machine below. See commonbackend.HandleUpdate.
	if isNetworkActive(resource) {
		return commonbackend.HandleUpdate(ctx, resource, &resource.Status.Status, h.plugin.Update, h.repo, h.MaxConditions)
	}

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

	case isNetworkAccepted(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStatePending))

	case isNetworkPending(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))

	case isNetworkCreating(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStateActive))

	case wantNetworkDelete(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting))

	case isNetworkDeleting(resource):
		// Nothing to do: the controller will remove the finalizers to end the deletion process.
		return nil

	case wantNetworkRetryCreate(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))

	default:
		log.Fatal("must never achieve that condition")
	}

	return nil
}

func (h *NetworkPluginHandler) setResourceState(ctx context.Context, resource *netdom.Network, state commondomain.ResourceState) error {
	if resource.Status == nil {
		resource.Status = &netdom.NetworkStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromState(state))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, err := h.repo.UpdateStatus(ctx, resource)

	return err
}

func (h *NetworkPluginHandler) setResourceErrorState(ctx context.Context, resource *netdom.Network, err error) error {
	if resource.Status == nil {
		resource.Status = &netdom.NetworkStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromError(err))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, updateErr := h.repo.UpdateStatus(ctx, resource)

	return updateErr
}

func isNetworkAccepted(resource *netdom.Network) bool {
	return resource.Status == nil
}

func isNetworkActive(resource *netdom.Network) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateActive
}

func isNetworkPending(resource *netdom.Network) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == commondomain.ResourceStatePending)
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
		resource.Status.Conditions[1].State == commondomain.ResourceStateCreating
}
