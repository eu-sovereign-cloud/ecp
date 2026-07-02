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
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

// PublicIpPluginHandler drives the PublicIp reconciliation state machine.
type PublicIpPluginHandler struct {
	frameworkbackend.GenericPluginHandler[*publicipdom.PublicIp]
	repo   persistence.Repo[*publicipdom.PublicIp]
	plugin PublicIpPlugin
}

var _ backendport.PluginHandler[*publicipdom.PublicIp] = (*PublicIpPluginHandler)(nil)

// NewPublicIpPluginHandler creates a new PublicIpPluginHandler.
func NewPublicIpPluginHandler(
	repo persistence.Repo[*publicipdom.PublicIp],
	plugin PublicIpPlugin,
	maxConditions int,
) *PublicIpPluginHandler {
	handler := &PublicIpPluginHandler{
		repo:   repo,
		plugin: plugin,
	}
	handler.MaxConditions = maxConditions

	return handler
}

func (h *PublicIpPluginHandler) HandleReconcile(ctx context.Context, resource *publicipdom.PublicIp) (bool, error) {
	var delegate backendport.DelegatedFunc[*publicipdom.PublicIp]

	switch {
	case isPublicIpAccepted(resource):
		delegate = frameworkbackend.BypassDelegated[*publicipdom.PublicIp]
	case isPublicIpPending(resource):
		delegate = frameworkbackend.BypassDelegated[*publicipdom.PublicIp]
	case isPublicIpCreating(resource):
		delegate = h.plugin.Create
	case wantPublicIpDelete(resource):
		delegate = frameworkbackend.BypassDelegated[*publicipdom.PublicIp]
	case isPublicIpDeleting(resource):
		delegate = h.plugin.Delete
	case wantPublicIpRetryCreate(resource):
		delegate = frameworkbackend.BypassDelegated[*publicipdom.PublicIp]
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
	case isPublicIpAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending, false)
	case isPublicIpPending(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	case isPublicIpCreating(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive, false)
	case wantPublicIpDelete(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting, true)
	case isPublicIpDeleting(resource):
		return false, nil
	case wantPublicIpRetryCreate(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *PublicIpPluginHandler) setResourceState(ctx context.Context, resource *publicipdom.PublicIp, state commondomain.ResourceState, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &publicipdom.PublicIpStatus{}
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

func (h *PublicIpPluginHandler) setResourceErrorState(ctx context.Context, resource *publicipdom.PublicIp, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &publicipdom.PublicIpStatus{}
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

func isPublicIpAccepted(resource *publicipdom.PublicIp) bool {
	return resource.Status == nil
}

func isPublicIpPending(resource *publicipdom.PublicIp) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == commondomain.ResourceStatePending)
}

func isPublicIpCreating(resource *publicipdom.PublicIp) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateCreating
}

func publicIpIsNotDeleting(resource *publicipdom.PublicIp) bool {
	return resource.Status == nil ||
		resource.Status.State != commondomain.ResourceStateDeleting
}

func wantPublicIpDelete(resource *publicipdom.PublicIp) bool {
	return resource.DeletedAt != nil && publicIpIsNotDeleting(resource)
}

func isPublicIpDeleting(resource *publicipdom.PublicIp) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateDeleting
}

func wantPublicIpRetryCreate(resource *publicipdom.PublicIp) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[len(resource.Status.Conditions)-2].State == commondomain.ResourceStateCreating
}
