package kubernetes

import (
	"context"
	"errors"
	"log"
	"time"

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
	// Power-state management applies only to active instances that are not being deleted.
	// It is orthogonal to the create/delete lifecycle below.
	if isInstanceActive(resource) {
		if handled, requeue, err := h.handlePowerReconcile(ctx, resource); handled {
			return requeue, err
		}

		// No power transition is pending, so the instance has no lifecycle edge left to fire and
		// takes the update path instead of the create/delete state machine below. Power ordering
		// matters: a start or stop is an explicit request and outranks reconciling the spec.
		return commonbackend.HandleUpdate(ctx, resource, &resource.Status.Status, h.plugin.Update, h.repo, h.MaxConditions)
	}

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

// handlePowerReconcile drives power-state transitions for an active instance. It returns
// handled=false when no power action is required, so the caller falls through to the
// lifecycle state machine.
//
// A pending restart (RestartPhase set) takes precedence over a plain start/stop request and
// runs as a durable, monotonically-advancing state machine: power-off -> power-on -> (cleared).
// The phase is persisted on the CR, so a controller restart resumes from the persisted phase
// rather than inferring progress from the current power state. Because there is no atomic
// transaction across the provider, the status subresource, and CR metadata, every step is
// idempotent (safe to repeat, since reconciliation is at-least-once) and each observable effect
// (status) is persisted BEFORE the phase advances, so the phase never claims progress that is
// not durable.
func (h *InstancePluginHandler) handlePowerReconcile(ctx context.Context, resource *instancedom.Instance) (handled bool, requeue bool, err error) {
	// Reject malformed power intent up front: surface an error condition and requeue rather than
	// silently doing nothing. These combinations are never produced by the gateway or delegator,
	// so they indicate corruption or manual tampering.
	if msg := validatePowerIntent(resource); msg != "" {
		return true, true, h.recordPowerCondition(ctx, resource, "InvalidPowerIntent", msg)
	}

	powerState := instancedom.PowerStateOff
	if resource.Status != nil && resource.Status.PowerState != "" {
		powerState = resource.Status.PowerState
	}

	switch {
	case resource.RestartPhase == instancedom.RestartPhasePowerOff:
		// The power-off phase always requeues to run the subsequent power-on phase.
		return true, true, h.runRestartPowerOff(ctx, resource)
	case resource.RestartPhase == instancedom.RestartPhasePowerOn:
		requeue, err = h.runRestartPowerOn(ctx, resource)
		return true, requeue, err
	case resource.DesiredPowerState == instancedom.PowerStateOn && powerState == instancedom.PowerStateOff:
		done, opErr := h.runPowerOp(ctx, resource, h.plugin.PowerOn, instancedom.PowerStateOn)
		return true, !done, opErr
	case resource.DesiredPowerState == instancedom.PowerStateOff && powerState == instancedom.PowerStateOn:
		done, opErr := h.runPowerOp(ctx, resource, h.plugin.PowerOff, instancedom.PowerStateOff)
		return true, !done, opErr
	default:
		return false, false, nil
	}
}

// validatePowerIntent returns a non-empty message when the power-related annotations are
// malformed. restart-id and restart-phase must be present together; the phase must be a known
// value; and the desired power state, if set, must be on or off.
func validatePowerIntent(resource *instancedom.Instance) string {
	switch {
	case resource.RestartID != "" && resource.RestartPhase == "":
		return "restart-id is set without restart-phase"
	case resource.RestartPhase != "" && resource.RestartID == "":
		return "restart-phase is set without restart-id"
	case resource.RestartPhase != "" &&
		resource.RestartPhase != instancedom.RestartPhasePowerOff &&
		resource.RestartPhase != instancedom.RestartPhasePowerOn:
		return "unknown restart-phase value: " + string(resource.RestartPhase)
	case resource.DesiredPowerState != "" &&
		resource.DesiredPowerState != instancedom.PowerStateOn &&
		resource.DesiredPowerState != instancedom.PowerStateOff:
		return "invalid desired-power-state value: " + string(resource.DesiredPowerState)
	default:
		return ""
	}
}

// runRestartPowerOff executes the power-off phase: power the instance down, persist
// PowerState=off, then advance the phase to power-on. The caller always requeues to run the
// next phase.
func (h *InstancePluginHandler) runRestartPowerOff(ctx context.Context, resource *instancedom.Instance) error {
	done, err := h.runPowerOp(ctx, resource, h.plugin.PowerOff, instancedom.PowerStateOff)
	if !done {
		return err
	}
	// Advance is compare-and-swap on the restart id: a newer restart may have arrived during the
	// power-off phase, and must not be overwritten with this (older) request's phase.
	return h.updateRestartIfCurrent(ctx, resource, func(inst *instancedom.Instance) {
		inst.RestartPhase = instancedom.RestartPhasePowerOn
	})
}

// runRestartPowerOn executes the power-on phase: power the instance up, persist PowerState=on,
// then clear the restart annotations. Cleanup is conditional on the restart id so a superseding
// restart is not clobbered; it never powers off again in this phase.
func (h *InstancePluginHandler) runRestartPowerOn(ctx context.Context, resource *instancedom.Instance) (requeue bool, err error) {
	done, err := h.runPowerOp(ctx, resource, h.plugin.PowerOn, instancedom.PowerStateOn)
	if !done {
		return true, err
	}
	if err := h.updateRestartIfCurrent(ctx, resource, func(inst *instancedom.Instance) {
		inst.RestartID = ""
		inst.RestartPhase = ""
	}); err != nil {
		return true, err
	}
	return false, nil
}

// runPowerOp invokes a provider power operation and, on success, persists the target power state.
// It centralizes the control flow shared by start/stop and the restart phases:
//   - ErrStillProcessing: done=false, err=nil — the caller requeues with no status change.
//   - any other provider error: recorded as a status condition and returned — the caller requeues.
//   - success: the target power state is persisted and done=true.
//
// done reports only whether the target state has been persisted, so the restart phase handlers
// know whether to proceed to phase advancement/cleanup.
func (h *InstancePluginHandler) runPowerOp(
	ctx context.Context,
	resource *instancedom.Instance,
	op backendport.DelegatedFunc[*instancedom.Instance],
	target instancedom.PowerState,
) (done bool, err error) {
	if opErr := op(ctx, resource); opErr != nil {
		if errors.Is(opErr, backendport.ErrStillProcessing) {
			return false, nil
		}
		return false, h.recordPowerError(ctx, resource, opErr)
	}
	if perr := h.persistPowerState(ctx, resource, target); perr != nil {
		return false, perr
	}
	return true, nil
}

// persistPowerState records the instance power state, refreshing PowerStateSince only on an
// actual transition. Re-persisting a state that is already recorded preserves the original
// transition timestamp, so an at-least-once retry does not rewrite it to the retry time.
// (If the very first status write failed, the transition was never recorded, so a retry stamps
// the retry time — PowerStateSince is therefore the last controller-observed transition.)
func (h *InstancePluginHandler) persistPowerState(ctx context.Context, resource *instancedom.Instance, ps instancedom.PowerState) error {
	if resource.Status == nil {
		resource.Status = &instancedom.InstanceStatus{}
	}
	if resource.Status.PowerState != ps || resource.Status.PowerStateSince == nil {
		now := time.Now()
		resource.Status.PowerStateSince = &now
	}
	resource.Status.PowerState = ps

	if _, err := h.repo.UpdateStatus(ctx, resource); err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			return nil
		}
		return err
	}

	return nil
}

// updateRestartIfCurrent applies mutate to the CR's restart annotations, but only if the
// persisted restart id still matches the one this reconcile is completing. This guards both the
// phase advance and the final cleanup so neither clobbers a newer restart that arrived in the
// meantime. The fresh load supplies the resource version, so the write uses optimistic concurrency.
func (h *InstancePluginHandler) updateRestartIfCurrent(ctx context.Context, resource *instancedom.Instance, mutate func(*instancedom.Instance)) error {
	current := &instancedom.Instance{}
	current.Name = resource.GetName()
	current.Tenant = resource.GetTenant()
	current.Workspace = resource.GetWorkspace()

	if err := h.repo.Load(ctx, &current); err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			return nil
		}
		return err
	}

	if current.RestartID != resource.RestartID {
		// A newer restart superseded this one; leave it for its own reconcile.
		return nil
	}

	mutate(current)

	if _, err := h.repo.Update(ctx, current); err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			return nil
		}
		return err
	}

	return nil
}

// recordPowerError records a failed provider power operation as a status condition and returns the
// original error, so the controller requeues (with backoff) while the failure is surfaced on the
// resource. If recording itself fails, that error is returned instead.
func (h *InstancePluginHandler) recordPowerError(ctx context.Context, resource *instancedom.Instance, opErr error) error {
	if err := h.recordPowerCondition(ctx, resource, "PowerOperationFailed", opErr.Error()); err != nil {
		return err
	}
	return opErr
}

// recordPowerCondition appends a PowerManagementError condition (preserving the lifecycle state)
// and persists it. A not-found resource is treated as already gone.
func (h *InstancePluginHandler) recordPowerCondition(ctx context.Context, resource *instancedom.Instance, reason, msg string) error {
	if resource.Status == nil {
		resource.Status = &instancedom.InstanceStatus{}
	}
	resource.Status.PushCondition(commondomain.StatusCondition{
		State:   resource.Status.State,
		Type:    "PowerManagementError",
		Reason:  reason,
		Message: msg,
	})
	commonbackend.TrimConditions(&resource.Status.Status, h.MaxConditions)

	if _, err := h.repo.UpdateStatus(ctx, resource); err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			return nil
		}
		return err
	}

	return nil
}

func isInstanceActive(resource *instancedom.Instance) bool {
	return resource.DeletedAt == nil &&
		resource.Status != nil &&
		resource.Status.State == commondomain.ResourceStateActive
}

func (h *InstancePluginHandler) setResourceState(ctx context.Context, resource *instancedom.Instance, state commondomain.ResourceState, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &instancedom.InstanceStatus{}
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

func (h *InstancePluginHandler) setResourceErrorState(ctx context.Context, resource *instancedom.Instance, err error, requeue bool) (bool, error) {
	if resource.Status == nil {
		resource.Status = &instancedom.InstanceStatus{}
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
		resource.Status.Conditions[1].State == commondomain.ResourceStateCreating
}
