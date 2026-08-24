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
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
)

// SubnetPluginHandler drives the Subnet reconciliation state machine.
type SubnetPluginHandler struct {
	frameworkbackend.GenericPluginHandler[*subnetdom.Subnet]
	repo   persistence.Repo[*subnetdom.Subnet]
	plugin SubnetPlugin
}

var _ backendport.PluginHandler[*subnetdom.Subnet] = (*SubnetPluginHandler)(nil)

// NewSubnetPluginHandler creates a new SubnetPluginHandler.
func NewSubnetPluginHandler(
	repo persistence.Repo[*subnetdom.Subnet],
	plugin SubnetPlugin,
	maxConditions int,
) *SubnetPluginHandler {
	handler := &SubnetPluginHandler{
		repo:   repo,
		plugin: plugin,
	}
	handler.MaxConditions = maxConditions

	return handler
}

func (h *SubnetPluginHandler) HandleReconcile(ctx context.Context, resource *subnetdom.Subnet) error {
	// An active resource has no lifecycle transition left to make, so it takes the update
	// path instead of the create/delete state machine below. See commonbackend.HandleUpdate.
	if isSubnetActive(resource) {
		return commonbackend.HandleUpdate(ctx, resource, &resource.Status.Status, h.plugin.Update, h.repo, h.MaxConditions)
	}

	var delegate backendport.DelegatedFunc[*subnetdom.Subnet]

	switch {
	case isSubnetAccepted(resource):
		delegate = frameworkbackend.BypassDelegated[*subnetdom.Subnet]
	case isSubnetPending(resource):
		delegate = frameworkbackend.BypassDelegated[*subnetdom.Subnet]
	case isSubnetCreating(resource):
		delegate = h.plugin.Create
	case wantSubnetDelete(resource):
		delegate = frameworkbackend.BypassDelegated[*subnetdom.Subnet]
	case isSubnetDeleting(resource):
		delegate = h.plugin.Delete
	case wantSubnetRetryCreate(resource):
		delegate = frameworkbackend.BypassDelegated[*subnetdom.Subnet]
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
	case isSubnetAccepted(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStatePending))
	case isSubnetPending(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))
	case isSubnetCreating(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStateActive))
	case wantSubnetDelete(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting))
	case isSubnetDeleting(resource):
		return nil
	case wantSubnetRetryCreate(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))
	default:
		log.Fatal("must never achieve that condition")
	}

	return nil
}

func (h *SubnetPluginHandler) setResourceState(ctx context.Context, resource *subnetdom.Subnet, state commondomain.ResourceState) error {
	if resource.Status == nil {
		resource.Status = &subnetdom.SubnetStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromState(state))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, err := h.repo.UpdateStatus(ctx, resource)

	return err
}

func (h *SubnetPluginHandler) setResourceErrorState(ctx context.Context, resource *subnetdom.Subnet, err error) error {
	if resource.Status == nil {
		resource.Status = &subnetdom.SubnetStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromError(err))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, updateErr := h.repo.UpdateStatus(ctx, resource)

	return updateErr
}

func isSubnetActive(resource *subnetdom.Subnet) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateActive
}

func isSubnetAccepted(resource *subnetdom.Subnet) bool {
	return resource.Status == nil
}

func isSubnetPending(resource *subnetdom.Subnet) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == commondomain.ResourceStatePending)
}

func isSubnetCreating(resource *subnetdom.Subnet) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateCreating
}

func subnetIsNotDeleting(resource *subnetdom.Subnet) bool {
	return resource.Status == nil ||
		resource.Status.State != commondomain.ResourceStateDeleting
}

func wantSubnetDelete(resource *subnetdom.Subnet) bool {
	return resource.DeletedAt != nil && subnetIsNotDeleting(resource)
}

func isSubnetDeleting(resource *subnetdom.Subnet) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateDeleting
}

func wantSubnetRetryCreate(resource *subnetdom.Subnet) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[1].State == commondomain.ResourceStateCreating
}
