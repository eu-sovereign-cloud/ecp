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

func (h *NicPluginHandler) HandleReconcile(ctx context.Context, resource *nicdom.Nic) (bool, error) {
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
	case isNicAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending, false)
	case isNicPending(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	case isNicCreating(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive, false)
	case wantNicDelete(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting, true)
	case isNicDeleting(resource):
		return false, nil
	case wantNicRetryCreate(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *NicPluginHandler) setResourceState(ctx context.Context, resource *nicdom.Nic, state commondomain.ResourceState, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &nicdom.NicStatus{}
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

func (h *NicPluginHandler) setResourceErrorState(ctx context.Context, resource *nicdom.Nic, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &nicdom.NicStatus{}
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
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == commondomain.ResourceStateCreating
}
