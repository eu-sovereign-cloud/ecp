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

// createNotSupportedConditionType marks a Create failure as a refusal: the plugin rejected the
// request outright (backendport.ErrNotSupported) rather than hit a transient error, so the same
// spec would only be refused again. wantInternetGatewayRetryCreate excludes conditions of this
// type, keeping the resource out of the Creating/Error loop a transient failure gets.
//
// A refusal is settled, not permanent. The spec that caused it is the tenant's to correct, so
// wantInternetGatewayRefusedCreateRetry hands a resource in this state back to Create - without
// first re-entering Creating, which is what would make the loop - and setResourceRefusedState
// writes nothing when the refusal is unchanged, so an uncorrected spec comes to rest after one
// more refused attempt instead of being stranded or spinning.
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

	// Whether this pass is the one attempting the create, either from Creating or as a fresh
	// attempt at a spec a previous pass refused. Both classify an ErrNotSupported below as a
	// Create refusal rather than a transient failure.
	var creating bool

	switch {
	case isInternetGatewayAccepted(resource):
		delegate = frameworkbackend.BypassDelegated[*internetgatewaydom.InternetGateway]
	case isInternetGatewayPending(resource):
		delegate = frameworkbackend.BypassDelegated[*internetgatewaydom.InternetGateway]
	case isInternetGatewayCreating(resource):
		delegate, creating = h.plugin.Create, true
	case wantInternetGatewayDelete(resource):
		delegate = frameworkbackend.BypassDelegated[*internetgatewaydom.InternetGateway]
	case isInternetGatewayDeleting(resource):
		delegate = h.plugin.Delete
	case wantInternetGatewayRetryCreate(resource):
		delegate = frameworkbackend.BypassDelegated[*internetgatewaydom.InternetGateway]
	case wantInternetGatewayRefusedCreateRetry(resource):
		delegate, creating = h.plugin.Create, true
	default:
		return nil // Nothing to do.
	}

	if err := delegate(ctx, resource); err != nil {
		// Classified before the progress-signal check below, as the controller and
		// commonbackend.HandleUpdate both do: requeueError.Unwrap keeps a refusal visible
		// through a signal, so a plugin returning RevisitBecause(d, ...ErrNotSupported) must
		// still have its refusal recorded rather than be handed back as a reschedule the
		// controller then drops with nothing written to the status.
		if creating && errors.Is(err, backendport.ErrNotSupported) {
			// The plugin will not satisfy this create as specified (e.g. Spec.EgressOnly
			// cannot be enforced). Recording it through setResourceErrorState would leave
			// wantInternetGatewayRetryCreate unable to tell this apart from a transient
			// failure, so every later reconcile would send the resource back through
			// Creating and hit the same refusal forever. Record a distinct condition and
			// stop instead of requeuing.
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
	case wantInternetGatewayRefusedCreateRetry(resource):
		// The refused spec has since been corrected and the create just succeeded, so the
		// gateway goes straight to Active. The refusal stays in the condition history as a
		// record of what the tenant had asked for.
		return commonbackend.IgnoreNotFound(h.setResourceState(ctx, resource, commondomain.ResourceStateActive))
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

// setResourceRefusedState records a Create refusal. Unlike setResourceErrorState, the condition it
// pushes carries createNotSupportedConditionType so wantInternetGatewayRetryCreate can recognise it
// and refuse to re-enter Creating.
//
// Re-reporting an unchanged refusal writes nothing. PushCondition would still bump
// LastTransitionAt and Occurrences, and that write is itself enough to trigger another reconcile,
// which wantInternetGatewayRefusedCreateRetry would answer with another attempt at the same spec -
// so a refusal that keeps being handed back settles here instead of spinning.
func (h *InternetGatewayPluginHandler) setResourceRefusedState(ctx context.Context, resource *internetgatewaydom.InternetGateway, err error) error {
	if resource.Status == nil {
		resource.Status = &internetgatewaydom.InternetGatewayStatus{}
	}

	condition := commondomain.StatusCondition{
		LastTransitionAt: time.Now(),
		Type:             createNotSupportedConditionType,
		State:            commondomain.ResourceStateError,
		Reason:           "NotSupported",
		Message:          err.Error(),
	}

	if previous := resource.Status.PeekConditions(); previous != nil &&
		commondomain.EqualStatusConditions(*previous, condition) {
		return nil
	}

	resource.Status.PushCondition(condition)
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
		(resource.Status.Conditions[1].State == commondomain.ResourceStateCreating ||
			resource.Status.Conditions[1].Type == createNotSupportedConditionType) &&
		resource.Status.Conditions[0].Type != createNotSupportedConditionType
}

// wantInternetGatewayRefusedCreateRetry reports whether a gateway is resting on a Create refusal
// and should be offered to the plugin again. The offer is what lets a corrected spec recover: the
// refusal is a judgement on what was asked for, not on the resource, and without this the only way
// out of the state would be to delete the gateway and recreate it.
//
// It delegates to Create directly rather than routing back through Creating. Writing a Creating
// condition first would change the status on every pass, and each of those writes would trigger
// the reconcile that writes the next one - the loop the refusal exists to prevent.
func wantInternetGatewayRefusedCreateRetry(resource *internetgatewaydom.InternetGateway) bool {
	return resource.DeletedAt == nil && resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateError &&
		len(resource.Status.Conditions) > 0 &&
		resource.Status.Conditions[0].Type == createNotSupportedConditionType
}
