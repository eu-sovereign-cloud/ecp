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

func (h *PublicIpPluginHandler) HandleReconcile(ctx context.Context, resource *publicipdom.PublicIp) error {
	// An active resource has no lifecycle transition left to make, so it takes the update
	// path instead of the create/delete state machine below. See commonbackend.HandleUpdate.
	if isPublicIpActive(resource) {
		return commonbackend.HandleUpdate(ctx, resource, &resource.Status.Status, h.plugin.Update, h.repo, h.MaxConditions)
	}

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
	case isPublicIpAccepted(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStatePending))
	case isPublicIpPending(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))
	case isPublicIpCreating(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStateActive))
	case wantPublicIpDelete(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting))
	case isPublicIpDeleting(resource):
		return nil
	case wantPublicIpRetryCreate(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))
	default:
		log.Fatal("must never achieve that condition")
	}

	return nil
}

func (h *PublicIpPluginHandler) setResourceState(ctx context.Context, resource *publicipdom.PublicIp, state commondomain.ResourceState) error {
	if resource.Status == nil {
		resource.Status = &publicipdom.PublicIpStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromState(state))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, err := h.repo.UpdateStatus(ctx, resource)

	return err
}

func (h *PublicIpPluginHandler) setResourceErrorState(ctx context.Context, resource *publicipdom.PublicIp, err error) error {
	if resource.Status == nil {
		resource.Status = &publicipdom.PublicIpStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromError(err))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, updateErr := h.repo.UpdateStatus(ctx, resource)

	return updateErr
}

func isPublicIpActive(resource *publicipdom.PublicIp) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateActive
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
		resource.Status.Conditions[1].State == commondomain.ResourceStateCreating
}
