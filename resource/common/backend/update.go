package backend

import (
	"context"
	"errors"
	"slices"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// updateFailedConditionType is the Type stamped by UpdateFailedCondition. It is what tells a
// later successful update that there is a failure to clear.
const updateFailedConditionType = "UpdateFailed"

// HandleUpdate drives a CSP plugin's Update for a resource that has already been created.
//
// It is level-triggered, unlike the create and delete paths: there is no "updating" state to enter
// and leave, the plugin is simply handed the desired state on every reconcile of an active resource
// and decides for itself whether anything needs to change. That is what makes a mutable field
// reach the provider at all - the lifecycle state machine has no edge left to fire once a resource
// is active, so an edge-triggered design would need observed state in status to diff against, and
// the SECA spec publishes that for almost nothing.
//
// The resource stays active throughout, including when the update fails. It is still running and
// healthy; it merely no longer matches its spec. Holding it active is also what makes the failure
// recoverable - an "error" state matches no arm of the reconciler, so the resource would be
// stranded there and a corrected spec would never be retried.
//
// Status is written only when what it reports actually changes. The controller watches its own
// writes, so a status write on every pass would feed itself: an unchanged failure re-written each
// reconcile would keep the resource reconciling forever.
//
// It returns nil when the resource is settled, a progress signal (backendport.StillProcessing, or a
// RevisitBecause carrying the provider's error) when the pass should be retried, and a plain error
// only when persisting the outcome failed.
func HandleUpdate[D persistence.IdentifiableResource](
	ctx context.Context,
	resource D,
	status *domain.Status,
	update backendport.DelegatedFunc[D],
	repo persistence.WriterRepo[D],
	maxConditions int,
) error {
	switch updateErr := update(ctx, resource); {
	case updateErr == nil:
		return clearUpdateFailure(ctx, resource, status, repo, maxConditions)

	case errors.Is(updateErr, backendport.StillProcessing):
		// In flight, not failed. Leave the status alone and come back to it.
		return backendport.StillProcessing

	case errors.Is(updateErr, backendport.ErrNotSupported):
		// A provider that cannot apply the change at all is not retried: re-issuing an operation
		// it has already refused would spin forever, and the reason it gave is more useful to the
		// user than another attempt.
		return recordUpdateFailure(ctx, resource, status, repo, maxConditions, updateErr)

	default:
		// Assumed transient. Recording the failure is what the user sees; the signal carries the
		// cause so it also reaches the log, and keeps the controller's cadence rather than backoff.
		if recordErr := recordUpdateFailure(ctx, resource, status, repo, maxConditions, updateErr); recordErr != nil {
			return recordErr
		}

		return backendport.RevisitBecause(0, updateErr)
	}
}

// recordUpdateFailure surfaces why the provider would not apply the change, keeping the resource
// active. It writes nothing when the same failure is already the most recent condition.
func recordUpdateFailure[D persistence.IdentifiableResource](
	ctx context.Context,
	resource D,
	status *domain.Status,
	repo persistence.WriterRepo[D],
	maxConditions int,
	updateErr error,
) error {
	condition := UpdateFailedCondition(domain.ResourceStateActive, updateErr.Error())

	// PushCondition would still bump LastTransitionAt and Occurrences for an identical condition,
	// and that is enough of a change to trigger another reconcile. Report a stable failure once.
	if previous := status.PeekConditions(); previous != nil && domain.EqualStatusConditions(*previous, condition) {
		return nil
	}

	status.PushCondition(condition)
	TrimConditions(status, maxConditions)

	return persistIgnoringMissing(ctx, resource, repo)
}

// clearUpdateFailure retracts a previously reported failure once an update succeeds. It writes
// nothing in the common case, where the last update also succeeded and there is nothing to retract.
//
// The whole condition list is scanned, not just the head: a failure is only the most recent
// condition until something else is pushed over it - a resize stepping through "updating", a power
// transition on an instance - and a buried failure the resource has since recovered from is exactly
// as misleading as a fresh one.
//
// Recovery removes the stale conditions rather than layering an active one on top. Pushing alone
// would not be idempotent: the failure would still be in the list on the next pass, so the resource
// would be re-written on every reconcile and never settle.
func clearUpdateFailure[D persistence.IdentifiableResource](
	ctx context.Context,
	resource D,
	status *domain.Status,
	repo persistence.WriterRepo[D],
	maxConditions int,
) error {
	before := len(status.Conditions)

	status.Conditions = slices.DeleteFunc(status.Conditions, func(c domain.StatusCondition) bool {
		return c.Type == updateFailedConditionType
	})

	if len(status.Conditions) == before {
		return nil
	}

	status.PushCondition(ConditionFromState(domain.ResourceStateActive))
	TrimConditions(status, maxConditions)

	return persistIgnoringMissing(ctx, resource, repo)
}

// persistIgnoringMissing treats a resource that has since been deleted as nothing to report: the
// status of an object that no longer exists cannot be written and does not need to be.
func persistIgnoringMissing[D persistence.IdentifiableResource](
	ctx context.Context,
	resource D,
	repo persistence.WriterRepo[D],
) error {
	if _, err := repo.UpdateStatus(ctx, resource); err != nil && !errors.Is(err, kernel.ErrNotFound) {
		return err
	}

	return nil
}

// TrimConditions drops the oldest conditions until at most maxConditions remain. A maxConditions of
// zero or less means unbounded. Every handler that pushes a condition then persists it needs this,
// so it lives here rather than being re-inlined beside each push.
func TrimConditions(status *domain.Status, maxConditions int) {
	for maxConditions > 0 && len(status.Conditions) > maxConditions {
		status.PopCondition()
	}
}
