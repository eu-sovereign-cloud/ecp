package backend

import (
	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
)

// RequeueAfterState turns a successful status write into a progress signal, so a lifecycle
// transition that has more work to do comes back for it. A failed write is returned as-is: the
// controller retries it with backoff, which is what a persistence failure wants.
//
// It exists because "advance the state, then reconcile again" is the shape of every intermediate
// transition, and spelling it out at each arm buries the one bit that actually differs between
// them - which state is being entered.
func RequeueAfterState(err error) error {
	if err != nil {
		return err
	}

	return backendport.StillProcessing
}
