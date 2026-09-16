package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"time"

	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"

	frameworkbackend "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
)

// createNotSupportedConditionType marks a Create failure as permanent: the plugin has refused the
// request outright (backendport.ErrNotSupported) rather than hit a transient error, so retrying
// Create would only be refused again. wantInternetGatewayRetryCreate excludes conditions of this
// type so such a resource does not loop forever between Creating and Error.
const createNotSupportedConditionType = "CreateNotSupported"

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

func (h *InternetGatewayPluginHandler) HandleReconcile(ctx context.Context, resource *internetgatewaydom.InternetGateway) error {
	// An active resource has no lifecycle transition left to make, so it takes the update
	// path instead of the create/delete state machine below. See commonbackend.HandleUpdate.
	if isInternetGatewayActive(resource) {
		return commonbackend.HandleUpdate(ctx, resource, &resource.Status.Status, h.plugin.Update, h.repo, h.MaxConditions)
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
		return nil // Nothing to do.
	}

	if err := delegate(ctx, resource); err != nil {
		// Classified before the progress-signal check below, as the controller and
		// commonbackend.HandleUpdate both do: requeueError.Unwrap keeps a refusal visible
		// through a signal, so a plugin returning RevisitBecause(d, ...ErrNotSupported) must
		// still have its refusal recorded rather than be handed back as a reschedule the
		// controller then drops with nothing written to the status.
		if isInternetGatewayCreating(resource) && errors.Is(err, backendport.ErrNotSupported) {
			// The plugin will never satisfy this create (e.g. Spec.EgressOnly cannot be
			// enforced). Recording it through setResourceErrorState would leave
			// wantInternetGatewayRetryCreate unable to tell this apart from a transient
			// failure, so every later reconcile would send the resource back through
			// Creating and hit the same refusal forever. Record a distinct, terminal
			// condition and stop instead of requeuing.
			return commonbackend.IgnoreNotFound(h.setResourceRefusedState(ctx, resource, err))
		}

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
	case isInternetGatewayAccepted(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStatePending))
	case isInternetGatewayPending(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))
	case isInternetGatewayCreating(resource):
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStateActive))
	case wantInternetGatewayDelete(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateDeleting))
	case isInternetGatewayDeleting(resource):
		return nil
	case wantInternetGatewayRetryCreate(resource):
		return commonbackend.RequeueAfterState(h.setResourceState(ctx, resource, commondomain.ResourceStateCreating))
	default:
		return fmt.Errorf("unreachable reconcile state for internet gateway %q", resource.GetName())
	}
}

func (h *InternetGatewayPluginHandler) setResourceState(ctx context.Context, resource *internetgatewaydom.InternetGateway, state commondomain.ResourceState) error {
	if resource.Status == nil {
		resource.Status = &internetgatewaydom.InternetGatewayStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromState(state))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, err := h.repo.UpdateStatus(ctx, resource)

	return err
}

func (h *InternetGatewayPluginHandler) setResourceErrorState(ctx context.Context, resource *internetgatewaydom.InternetGateway, err error) error {
	if resource.Status == nil {
		resource.Status = &internetgatewaydom.InternetGatewayStatus{}
	}

	resource.Status.PushCondition(commonbackend.ConditionFromError(err))
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, updateErr := h.repo.UpdateStatus(ctx, resource)

	return updateErr
}

// setResourceRefusedState records a permanent Create refusal. Unlike setResourceErrorState, the
// condition it pushes carries createNotSupportedConditionType so wantInternetGatewayRetryCreate
// can recognise it and refuse to re-enter Creating.
func (h *InternetGatewayPluginHandler) setResourceRefusedState(ctx context.Context, resource *internetgatewaydom.InternetGateway, err error) error {
	if resource.Status == nil {
		resource.Status = &internetgatewaydom.InternetGatewayStatus{}
	}

	resource.Status.PushCondition(commondomain.StatusCondition{
		LastTransitionAt: time.Now(),
		Type:             createNotSupportedConditionType,
		State:            commondomain.ResourceStateError,
		Reason:           "NotSupported",
		Message:          err.Error(),
	})
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	_, updateErr := h.repo.UpdateStatus(ctx, resource)

	return updateErr
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
		resource.Status.Conditions[1].State == commondomain.ResourceStateCreating &&
		resource.Status.Conditions[0].Type != createNotSupportedConditionType
}
