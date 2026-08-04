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
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

// InternetGatewayPluginHandler drives the InternetGateway reconciliation state machine.
type InternetGatewayPluginHandler struct {
	frameworkbackend.GenericPluginHandler[*internetgatewaydom.InternetGateway]
	repo   persistence.Repo[*internetgatewaydom.InternetGateway]
	plugin InternetGatewayPlugin
}

var _ backendport.PluginHandler[*internetgatewaydom.InternetGateway] = (*InternetGatewayPluginHandler)(nil)

// NewInternetGatewayPluginHandler creates a new InternetGatewayPluginHandler.
func NewInternetGatewayPluginHandler(
	repo persistence.Repo[*internetgatewaydom.InternetGateway],
	plugin InternetGatewayPlugin,
	maxConditions int,
) *InternetGatewayPluginHandler {
	handler := &InternetGatewayPluginHandler{
		repo:   repo,
		plugin: plugin,
	}
	handler.MaxConditions = maxConditions

	return handler
}

func (h *InternetGatewayPluginHandler) HandleReconcile(ctx context.Context, resource *internetgatewaydom.InternetGateway) (bool, error) {
	// An active resource has no lifecycle transition left to make, so it takes the update
	// path instead of the create/delete state machine below. See commonbackend.HandleUpdate.
	if isInternetGatewayActive(resource) {
		return commonbackend.HandleUpdate(ctx, resource, &resource.Status.Status, h.plugin.Update, h.persistStatus, h.MaxConditions)
	}

	var delegate backendport.DelegatedFunc[*internetgatewaydom.InternetGateway]

	switch {
	case isInternetGatewayAccepted(resource):
		delegate = frameworkbackend.BypassDelegated[*internetgatewaydom.InternetGateway]
	case isInternetGatewayPending(resource):
		delegate = frameworkbackend.BypassDelegated[*internetgatewaydom.InternetGateway]
	case isInternetGatewayCreating(resource):
		delegate = h.plugin.Create
	case wantInternetGatewayDelete(resource):
		delegate = frameworkbackend.BypassDelegated[*internetgatewaydom.InternetGateway]
	case isInternetGatewayDeleting(resource):
		delegate = h.plugin.Delete
	case wantInternetGatewayRetryCreate(resource):
		delegate = frameworkbackend.BypassDelegated[*internetgatewaydom.InternetGateway]
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
	case isInternetGatewayAccepted(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStatePending, false)
	case isInternetGatewayPending(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	case isInternetGatewayCreating(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateActive, false)
	case wantInternetGatewayDelete(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting, true)
	case isInternetGatewayDeleting(resource):
		return false, nil
	case wantInternetGatewayRetryCreate(resource):
		return h.setResourceState(ctx, resource, commondomain.ResourceStateCreating, true)
	default:
		log.Fatal("must never achieve that condition")
	}

	return false, nil
}

func (h *InternetGatewayPluginHandler) setResourceState(ctx context.Context, resource *internetgatewaydom.InternetGateway, state commondomain.ResourceState, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &internetgatewaydom.InternetGatewayStatus{}
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

func (h *InternetGatewayPluginHandler) setResourceErrorState(ctx context.Context, resource *internetgatewaydom.InternetGateway, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &internetgatewaydom.InternetGatewayStatus{}
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

// persistStatus writes the resource's status subresource. It is handed to the shared
// update helper, which owns the decision of when a write is warranted.
func (h *InternetGatewayPluginHandler) persistStatus(ctx context.Context, resource *internetgatewaydom.InternetGateway) error {
	_, err := h.repo.UpdateStatus(ctx, resource)

	return err
}

func isInternetGatewayActive(resource *internetgatewaydom.InternetGateway) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateActive
}

func isInternetGatewayAccepted(resource *internetgatewaydom.InternetGateway) bool {
	return resource.Status == nil
}

func isInternetGatewayPending(resource *internetgatewaydom.InternetGateway) bool {
	return resource.DeletedAt == nil && (resource.Status == nil ||
		resource.Status.State == commondomain.ResourceStatePending)
}

func isInternetGatewayCreating(resource *internetgatewaydom.InternetGateway) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateCreating
}

func internetGatewayIsNotDeleting(resource *internetgatewaydom.InternetGateway) bool {
	return resource.Status == nil ||
		resource.Status.State != commondomain.ResourceStateDeleting
}

func wantInternetGatewayDelete(resource *internetgatewaydom.InternetGateway) bool {
	return resource.DeletedAt != nil && internetGatewayIsNotDeleting(resource)
}

func isInternetGatewayDeleting(resource *internetgatewaydom.InternetGateway) bool {
	return resource.DeletedAt != nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateDeleting
}

func wantInternetGatewayRetryCreate(resource *internetgatewaydom.InternetGateway) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 1 &&
		resource.Status.Conditions[1].State == commondomain.ResourceStateCreating
}
