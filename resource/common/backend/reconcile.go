package backend

import (
	"errors"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
)

// RequeueAfterState turns a successful status write into a progress signal, so a lifecycle
// transition that has more work to do comes back for it.
//
// A write that failed because the resource is already gone settles quietly instead: there is
// nothing left to reconcile, and requeuing it would just repeat the same not-found write forever.
// Any other error is returned as-is: the controller retries it with backoff, which is what a
// persistence failure wants.
//
// It exists because "advance the state, then reconcile again" is the shape of every intermediate
// transition, and spelling it out at each arm buries the one bit that actually differs between
// them - which state is being entered.
func RequeueAfterState(err error) error {
	switch {
	case err == nil:
		return backendport.StillProcessing
	case errors.Is(err, kernel.ErrNotFound):
		return nil
	default:
		return err
	}
}

// IgnoreNotFound settles a status write silently when the resource it targeted is already gone:
// there is nothing left to reconcile. Any other outcome, including nil, is returned unchanged.
func IgnoreNotFound(err error) error {
	if errors.Is(err, kernel.ErrNotFound) {
		return nil
	}

	return err
}
