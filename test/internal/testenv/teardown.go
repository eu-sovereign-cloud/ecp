package testenv

import (
	"context"
	"net/http"
	"time"
)

// deleteRetryInterval and deleteRetryTimeout bound DeleteUntilGone. The wait is for a
// finalizer to clear a child CR, which the dummy plugin does in seconds; the timeout is
// there so a genuinely stuck teardown fails fast rather than hanging the suite.
const (
	deleteRetryInterval = 500 * time.Millisecond
	deleteRetryTimeout  = 30 * time.Second
)

// DeleteUntilGone runs a best-effort teardown delete, retrying while the API answers
// 409 Conflict.
//
// A parent (workspace, network) cannot be deleted while its child namespace still holds
// resources — that refusal is a deliberate, synchronous invariant of the API. Child
// deletes, however, are asynchronous: the API accepts them and a finalizer clears the CR
// some time later. A teardown that fires child and parent deletes back to back therefore
// races the finalizer, gets a 409 on the parent, and — since teardown ignores its return
// values — silently leaks the parent and everything above it. Retrying is the whole fix:
// the conflict clears on its own once the children are gone.
//
// Errors and non-conflict statuses (including 404 for something already gone) end the
// retry: this is teardown, and there is nothing useful to do with the failure.
func DeleteUntilGone(ctx context.Context, del func() (*http.Response, error)) {
	deadline := time.Now().Add(deleteRetryTimeout)

	for {
		resp, err := del()
		if resp != nil {
			_ = resp.Body.Close()
		}
		if err == nil && resp != nil && resp.StatusCode == http.StatusConflict && time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(deleteRetryInterval):
			}
			continue
		}
		return
	}
}
