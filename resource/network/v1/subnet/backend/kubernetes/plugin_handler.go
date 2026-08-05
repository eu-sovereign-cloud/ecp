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

func (h *SubnetPluginHandler) HandleReconcile(ctx context.Context, resource *subnetdom.Subnet) (bool, error) {
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
	case isSubnetAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending, false)
	case isSubnetPending(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	case isSubnetCreating(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive, false)
	case wantSubnetDelete(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting, true)
	case isSubnetDeleting(resource):
		return false, nil
	case wantSubnetRetryCreate(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *SubnetPluginHandler) setResourceState(ctx context.Context, resource *subnetdom.Subnet, state commondomain.ResourceState, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &subnetdom.SubnetStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromState(state))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	if _, err := h.repo.UpdateStatus(ctx, resource); err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			return false, nil
		}
		return requeue, err
	}

	return requeue, nil
}

func (h *SubnetPluginHandler) setResourceErrorState(ctx context.Context, resource *subnetdom.Subnet, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &subnetdom.SubnetStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromError(err))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	if _, updateErr := h.repo.UpdateStatus(ctx, resource); updateErr != nil {
		if errors.Is(updateErr, kernel.ErrNotFound) {
			return false, nil
		}
		return requeue, updateErr
	}

	return requeue, nil
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
