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
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// InstancePluginHandler drives the Instance reconciliation state machine.
type InstancePluginHandler struct {
	frameworkbackend.GenericPluginHandler[*instancedom.Instance]
	repo   persistence.Repo[*instancedom.Instance]
	plugin InstancePlugin
}

var _ backendport.PluginHandler[*instancedom.Instance] = (*InstancePluginHandler)(nil)

// NewInstancePluginHandler creates a new InstancePluginHandler.
func NewInstancePluginHandler(
	repo persistence.Repo[*instancedom.Instance],
	plugin InstancePlugin,
	maxConditions int,
) *InstancePluginHandler {
	handler := &InstancePluginHandler{
		repo:   repo,
		plugin: plugin,
	}
	handler.MaxConditions = maxConditions

	return handler
}

func (h *InstancePluginHandler) HandleReconcile(ctx context.Context, resource *instancedom.Instance) (bool, error) {
	var delegate backendport.DelegatedFunc[*instancedom.Instance]

	switch {
	case isInstanceAccepted(resource):
		delegate = frameworkbackend.BypassDelegated[*instancedom.Instance]
	case isInstancePending(resource):
		delegate = frameworkbackend.BypassDelegated[*instancedom.Instance]
	case isInstanceCreating(resource):
		delegate = h.plugin.Create
	case wantInstanceDelete(resource):
		delegate = frameworkbackend.BypassDelegated[*instancedom.Instance]
	case isInstanceDeleting(resource):
		delegate = h.plugin.Delete
	case wantInstanceRetryCreate(resource):
		delegate = frameworkbackend.BypassDelegated[*instancedom.Instance]
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
	case isInstanceAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending, false)
	case isInstancePending(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	case isInstanceCreating(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive, false)
	case wantInstanceDelete(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting, true)
	case isInstanceDeleting(resource):
		return false, nil
	case wantInstanceRetryCreate(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *InstancePluginHandler) setResourceState(ctx context.Context, resource *instancedom.Instance, state commondomain.ResourceState, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &instancedom.InstanceStatus{}
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

func (h *InstancePluginHandler) setResourceErrorState(ctx context.Context, resource *instancedom.Instance, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &instancedom.InstanceStatus{}
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

func isInstanceAccepted(resource *instancedom.Instance) bool {
	return resource.Status == nil
}

func isInstancePending(resource *instancedom.Instance) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == commondomain.ResourceStatePending)
}

func isInstanceCreating(resource *instancedom.Instance) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateCreating
}

func instanceIsNotDeleting(resource *instancedom.Instance) bool {
	return resource.Status == nil ||
		resource.Status.State != commondomain.ResourceStateDeleting
}

func wantInstanceDelete(resource *instancedom.Instance) bool {
	return resource.DeletedAt != nil && instanceIsNotDeleting(resource)
}

func isInstanceDeleting(resource *instancedom.Instance) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateDeleting
}

func wantInstanceRetryCreate(resource *instancedom.Instance) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == commondomain.ResourceStateCreating
}
