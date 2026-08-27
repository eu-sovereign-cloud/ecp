package kubernetes

import (
	"context"
	"errors"
	"fmt"

	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"

	frameworkbackend "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

// NicPluginHandler drives the NIC reconciliation state machine.
type NicPluginHandler struct {
	frameworkbackend.GenericPluginHandler[*nicdom.Nic]
	repo   persistence.Repo[*nicdom.Nic]
	plugin NicPlugin
}

var _ backendport.PluginHandler[*nicdom.Nic] = (*NicPluginHandler)(nil)

// NewNicPluginHandler creates a new NicPluginHandler.
func NewNicPluginHandler(
	repo persistence.Repo[*nicdom.Nic],
	plugin NicPlugin,
	maxConditions int,
) *NicPluginHandler {
	handler := &NicPluginHandler{
		repo:   repo,
		plugin: plugin,
	}
	handler.MaxConditions = maxConditions

	return handler
}

func (h *NicPluginHandler) HandleReconcile(ctx context.Context, resource *nicdom.Nic) error {
	// An active resource has no lifecycle transition left to make, so it takes the update
	// path instead of the create/delete state machine below. See commonbackend.HandleUpdate.
	if isNicActive(resource) {
		return commonbackend.HandleUpdate(ctx, resource, &resource.Status.Status, h.plugin.Update, h.repo, h.MaxConditions)
	}

	var delegate backendport.DelegatedFunc[*nicdom.Nic]

	switch {
	case isNicAccepted(resource):
		delegate = frameworkbackend.BypassDelegated[*nicdom.Nic]
	case isNicPending(resource):
		delegate = frameworkbackend.BypassDelegated[*nicdom.Nic]
	case isNicCreating(resource):
		delegate = h.plugin.Create
	case wantNicDelete(resource):
		delegate = frameworkbackend.BypassDelegated[*nicdom.Nic]
	case isNicDeleting(resource):
		delegate = h.plugin.Delete
	case wantNicRetryCreate(resource):
		delegate = frameworkbackend.BypassDelegated[*nicdom.Nic]
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
	case isNicAccepted(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStatePending))
	case isNicPending(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))
	case isNicCreating(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStateActive))
	case wantNicDelete(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting))
	case isNicDeleting(resource):
		return nil
	case wantNicRetryCreate(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))
	default:
		return fmt.Errorf("unreachable reconcile state for nic %q", resource.GetName())
	}
}

func (h *NicPluginHandler) setResourceState(ctx context.Context, resource *nicdom.Nic, state commondomain.ResourceState) error {
	if resource.Status == nil {
		resource.Status = &nicdom.NicStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromState(state))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, err := h.repo.UpdateStatus(ctx, resource)

	return err
}

func (h *NicPluginHandler) setResourceErrorState(ctx context.Context, resource *nicdom.Nic, err error) error {
	if resource.Status == nil {
		resource.Status = &nicdom.NicStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromError(err))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, updateErr := h.repo.UpdateStatus(ctx, resource)

	return updateErr
}

func isNicActive(resource *nicdom.Nic) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateActive
}

func isNicAccepted(resource *nicdom.Nic) bool {
	return resource.Status == nil
}

func isNicPending(resource *nicdom.Nic) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == commondomain.ResourceStatePending)
}

func isNicCreating(resource *nicdom.Nic) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateCreating
}

func nicIsNotDeleting(resource *nicdom.Nic) bool {
	return resource.Status == nil ||
		resource.Status.State != commondomain.ResourceStateDeleting
}

func wantNicDelete(resource *nicdom.Nic) bool {
	return resource.DeletedAt != nil && nicIsNotDeleting(resource)
}

func isNicDeleting(resource *nicdom.Nic) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateDeleting
}

func wantNicRetryCreate(resource *nicdom.Nic) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[1].State == commondomain.ResourceStateCreating
}
